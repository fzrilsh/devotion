package listing

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
)

// recordNotifier captures every Enqueue so a stale-calendar test can assert who
// was reminded and how often, without touching the notification queue.
type recordNotifier struct {
	mu    sync.Mutex
	calls []recordedNotice
}

type recordedNotice struct {
	account pgtype.UUID
	event   sqlcgen.EventType
}

func (r *recordNotifier) Enqueue(_ context.Context, _ pgx.Tx, account pgtype.UUID, event sqlcgen.EventType, _, _ string, _ *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, recordedNotice{account: account, event: event})
	return nil
}

func (r *recordNotifier) countFor(account pgtype.UUID, event sqlcgen.EventType) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := 0
	for _, n := range r.calls {
		if n.account == account && n.event == event {
			c++
		}
	}
	return c
}

// staleHarness wires a Service whose notifier records, so the stale-calendar
// reminder job's output is observable. It reuses the standard newHarness seed
// (region, subcontractor account, profile, catalog items) and swaps the notifier.
func staleHarness(t *testing.T, name string) (*harness, *recordNotifier) {
	t.Helper()
	h := newHarness(t, name)
	rec := &recordNotifier{}
	h.svc.notifier = rec
	return h, rec
}

// backdateCalendar sets a listing's calendar_updated_at to a fixed instant so a
// test controls whether the stale window has lapsed, independent of the create
// stamp. It clears stale_notified_at so the listing starts un-reminded.
func backdateCalendar(t *testing.T, h *harness, listingID pgtype.UUID, at time.Time) {
	t.Helper()
	if _, err := h.pool.Exec(context.Background(),
		`UPDATE capacity_listing SET calendar_updated_at = $2, stale_notified_at = NULL WHERE id = $1`,
		listingID, at); err != nil {
		t.Fatalf("backdate calendar: %v", err)
	}
}

// TestCalendarStale_LewatTujuhHari_MengingatkanPemilik_FR021 proves a published
// listing whose calendar has gone untouched past the seven-day window earns its
// owner exactly one reminder, and that the reminder is a nudge with no status
// change.
func TestCalendarStale_LewatTujuhHari_MengingatkanPemilik_FR021(t *testing.T) {
	h, rec := staleHarness(t, "stale_lewat")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	listingID := h.listingRow(t)

	// Calendar last touched at baseTime; move the clock eight days on so the
	// seven-day window has clearly lapsed.
	backdateCalendar(t, h, listingID, baseTime)
	h.clock.Set(baseTime.Add(8 * 24 * time.Hour))
	if err := h.svc.runCalendarStale(context.Background()); err != nil {
		t.Fatalf("runCalendarStale: %v", err)
	}

	if got := rec.countFor(h.accountID, sqlcgen.EventTypeCalendarStale); got != 1 {
		t.Fatalf("pemilik diingatkan %d kali, mau 1 (FR-021)", got)
	}
}

// TestCalendarStale_MasihSegar_TidakMengingatkan_FR021 proves a calendar touched
// within the window earns no reminder, so the job never nags an owner keeping
// their calendar current.
func TestCalendarStale_MasihSegar_TidakMengingatkan_FR021(t *testing.T) {
	h, rec := staleHarness(t, "stale_segar")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	listingID := h.listingRow(t)

	// Only six days on: still inside the seven-day window.
	backdateCalendar(t, h, listingID, baseTime)
	h.clock.Set(baseTime.Add(6 * 24 * time.Hour))
	if err := h.svc.runCalendarStale(context.Background()); err != nil {
		t.Fatalf("runCalendarStale: %v", err)
	}

	if got := rec.countFor(h.accountID, sqlcgen.EventTypeCalendarStale); got != 0 {
		t.Fatalf("pemilik diingatkan %d kali sebelum tenggat, mau 0 (FR-021)", got)
	}
}

// TestCalendarStale_TidakTayang_TidakMengingatkan_FR021 proves a hidden listing
// is never reminded even when its calendar is long stale: an unpublished listing
// is invisible to searchers, so its freshness does not matter.
func TestCalendarStale_TidakTayang_TidakMengingatkan_FR021(t *testing.T) {
	h, rec := staleHarness(t, "stale_hidden")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	listingID := h.listingRow(t)

	mustStatus(t, h.do("PUT", "/api/listing/me/visibility", map[string]any{"published": false}), http.StatusOK)
	backdateCalendar(t, h, listingID, baseTime)
	h.clock.Set(baseTime.Add(30 * 24 * time.Hour))
	if err := h.svc.runCalendarStale(context.Background()); err != nil {
		t.Fatalf("runCalendarStale: %v", err)
	}

	if got := rec.countFor(h.accountID, sqlcgen.EventTypeCalendarStale); got != 0 {
		t.Fatalf("listing tak tayang diingatkan %d kali, mau 0 (FR-021)", got)
	}
}

// TestCalendarStale_PassKedua_TidakMengingatkanUlang_FR021 proves the reminder
// fires once per staleness episode: a second pass over the same still-stale
// listing sends nothing, because MarkStaleNotified stamped stale_notified_at past
// calendar_updated_at.
func TestCalendarStale_PassKedua_TidakMengingatkanUlang_FR021(t *testing.T) {
	h, rec := staleHarness(t, "stale_passkedua")
	mustStatus(t, h.do("POST", "/api/listing/me", h.validCreate(500)), http.StatusCreated)
	listingID := h.listingRow(t)

	backdateCalendar(t, h, listingID, baseTime)
	h.clock.Set(baseTime.Add(8 * 24 * time.Hour))
	if err := h.svc.runCalendarStale(context.Background()); err != nil {
		t.Fatalf("runCalendarStale pass 1: %v", err)
	}
	if got := rec.countFor(h.accountID, sqlcgen.EventTypeCalendarStale); got != 1 {
		t.Fatalf("pass 1: pemilik diingatkan %d kali, mau 1 (FR-021)", got)
	}

	// A later pass, still stale and un-edited, must not remind again.
	h.clock.Set(baseTime.Add(9 * 24 * time.Hour))
	if err := h.svc.runCalendarStale(context.Background()); err != nil {
		t.Fatalf("runCalendarStale pass 2: %v", err)
	}
	if got := rec.countFor(h.accountID, sqlcgen.EventTypeCalendarStale); got != 1 {
		t.Fatalf("pass 2: pemilik diingatkan %d kali total, mau tetap 1 (FR-021)", got)
	}
}
