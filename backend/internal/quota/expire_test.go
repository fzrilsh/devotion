package quota

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// candidateStatus reads the stored status of a candidate row, so a test asserts
// what the expiry job persisted, not what a read layer would recompute.
func candidateStatus(t *testing.T, h *harness, candidateID pgtype.UUID) string {
	t.Helper()
	var status string
	if err := h.pool.QueryRow(context.Background(),
		`SELECT status FROM request_candidate WHERE id = $1`, candidateID).Scan(&status); err != nil {
		t.Fatalf("baca status kandidat: %v", err)
	}
	return status
}

// countEvent counts how many times the recording notifier saw an event.
func countEvent(events []sqlcgen.EventType, want sqlcgen.EventType) int {
	c := 0
	for _, e := range events {
		if e == want {
			c++
		}
	}
	return c
}

// TestRequestExpire_LapsedWindowExpiresAndNotifies_FR037 proves the ticker moves
// a candidate whose 72-hour reply window has lapsed from awaiting_reply to
// expired and notifies the buyer once. reply_due_at is seeded at baseTime + 72h,
// so advancing the clock one hour past it crosses the inclusive boundary.
func TestRequestExpire_LapsedWindowExpiresAndNotifies_FR037(t *testing.T) {
	h := newHarness(t, "quota_expire_lapsed")
	f := h.seedCandidate(t, "alfa", 50, platform.WeekStart(baseTime).AddDate(0, 0, 7*2))

	// One hour past reply_due_at (baseTime + 72h): the window has lapsed.
	h.clock.Set(baseTime.Add(72*time.Hour + time.Hour))
	if err := h.svc.runRequestExpire(context.Background()); err != nil {
		t.Fatalf("runRequestExpire: %v", err)
	}

	if got := candidateStatus(t, h, f.candidateID); got != "expired" {
		t.Fatalf("status kandidat %q, mau %q (FR-037)", got, "expired")
	}
	if got := countEvent(h.notifier.events, sqlcgen.EventTypeRequestExpired); got != 1 {
		t.Fatalf("buyer diberi tahu %d kali, mau 1 (FR-037)", got)
	}
}

// TestRequestExpire_WithinWindowUntouched_FR037 proves a candidate whose reply
// window has not yet lapsed is left awaiting_reply and no notice goes out, so the
// job never expires a request the subcontractor can still answer.
func TestRequestExpire_WithinWindowUntouched_FR037(t *testing.T) {
	h := newHarness(t, "quota_expire_within")
	f := h.seedCandidate(t, "alfa", 50, platform.WeekStart(baseTime).AddDate(0, 0, 7*2))

	// One hour before reply_due_at: still inside the window.
	h.clock.Set(baseTime.Add(72*time.Hour - time.Hour))
	if err := h.svc.runRequestExpire(context.Background()); err != nil {
		t.Fatalf("runRequestExpire: %v", err)
	}

	if got := candidateStatus(t, h, f.candidateID); got != "awaiting_reply" {
		t.Fatalf("status kandidat %q, mau %q; belum lewat tenggat (FR-037)", got, "awaiting_reply")
	}
	if got := countEvent(h.notifier.events, sqlcgen.EventTypeRequestExpired); got != 0 {
		t.Fatalf("buyer diberi tahu %d kali sebelum tenggat, mau 0 (FR-037)", got)
	}
}

// TestRequestExpire_AnsweredCandidateNotExpired_FR037 proves a candidate the
// subcontractor already answered (status offered) is never expired even past the
// window: the awaiting_reply guard makes it a no-op, so an answered candidate
// keeps its status and the buyer gets no expiry notice.
func TestRequestExpire_AnsweredCandidateNotExpired_FR037(t *testing.T) {
	h := newHarness(t, "quota_expire_answered")
	f := h.seedCandidate(t, "alfa", 50, platform.WeekStart(baseTime).AddDate(0, 0, 7*2))

	// The subcontractor replied before the window lapsed.
	if _, err := h.pool.Exec(context.Background(),
		`UPDATE request_candidate SET status = 'offered', updated_at = $2 WHERE id = $1`,
		f.candidateID, tstz(baseTime.Add(time.Hour))); err != nil {
		t.Fatalf("balas kandidat: %v", err)
	}

	h.clock.Set(baseTime.Add(72*time.Hour + time.Hour))
	if err := h.svc.runRequestExpire(context.Background()); err != nil {
		t.Fatalf("runRequestExpire: %v", err)
	}

	if got := candidateStatus(t, h, f.candidateID); got != "offered" {
		t.Fatalf("status kandidat %q, mau %q; kandidat terjawab tak boleh kedaluwarsa (FR-037)", got, "offered")
	}
	if got := countEvent(h.notifier.events, sqlcgen.EventTypeRequestExpired); got != 0 {
		t.Fatalf("buyer diberi tahu %d kali untuk kandidat terjawab, mau 0 (FR-037)", got)
	}
}

// TestRequestExpire_SecondPassNoRewarn_FR037 proves a second pass after a
// candidate is expired sends no further notice: the awaiting_reply guard makes
// the re-run a no-op, so the buyer is notified once per expiry, not once per tick.
func TestRequestExpire_SecondPassNoRewarn_FR037(t *testing.T) {
	h := newHarness(t, "quota_expire_secondpass")
	h.seedCandidate(t, "alfa", 50, platform.WeekStart(baseTime).AddDate(0, 0, 7*2))

	h.clock.Set(baseTime.Add(72*time.Hour + time.Hour))
	if err := h.svc.runRequestExpire(context.Background()); err != nil {
		t.Fatalf("runRequestExpire pass 1: %v", err)
	}
	if got := countEvent(h.notifier.events, sqlcgen.EventTypeRequestExpired); got != 1 {
		t.Fatalf("pass 1: buyer diberi tahu %d kali, mau 1 (FR-037)", got)
	}

	if err := h.svc.runRequestExpire(context.Background()); err != nil {
		t.Fatalf("runRequestExpire pass 2: %v", err)
	}
	if got := countEvent(h.notifier.events, sqlcgen.EventTypeRequestExpired); got != 1 {
		t.Fatalf("pass 2: buyer diberi tahu %d kali total, mau tetap 1 (FR-037)", got)
	}
}

// TestRequestExpire_MultiCandidateNoReplyNotifiesOnce_FR037 proves a request sent
// to three candidates that all fall silent expires every candidate but notifies
// the buyer exactly once, not once per candidate: the "tanpa penawaran" notice is
// request-level (AS-7), and the deep link points at the one request.
func TestRequestExpire_MultiCandidateNoReplyNotifiesOnce_FR037(t *testing.T) {
	h := newHarness(t, "quota_expire_multi")
	deadline := platform.WeekStart(baseTime).AddDate(0, 0, 7*2)
	// One request to three candidates: seed the first, then two more on the same
	// request so all three share one reply window.
	f := h.seedCandidate(t, "alfa", 50, deadline)
	b := seedListingNamed(t, h, "beta")
	c := seedListingNamed(t, h, "gamma")
	addCandidate(t, h, f.requestID, b)
	addCandidate(t, h, f.requestID, c)

	h.clock.Set(baseTime.Add(72*time.Hour + time.Hour))
	if err := h.svc.runRequestExpire(context.Background()); err != nil {
		t.Fatalf("runRequestExpire: %v", err)
	}

	if n := expiredCount(t, h, f.requestID); n != 3 {
		t.Fatalf("kandidat kedaluwarsa %d, mau 3 (FR-037)", n)
	}
	if got := countEvent(h.notifier.events, sqlcgen.EventTypeRequestExpired); got != 1 {
		t.Fatalf("buyer diberi tahu %d kali untuk satu request, mau 1 (FR-037/AS-7)", got)
	}
}

// TestRequestExpire_StandingOfferSuppressesNotice_FR037 proves a request where one
// candidate replied with an offer and the others fell silent expires the silent
// candidates without telling the buyer the request lapsed "tanpa penawaran": AS-7
// conditions the notice on no candidate having replied, and one did.
func TestRequestExpire_StandingOfferSuppressesNotice_FR037(t *testing.T) {
	h := newHarness(t, "quota_expire_standing")
	deadline := platform.WeekStart(baseTime).AddDate(0, 0, 7*2)
	f := h.seedCandidate(t, "alfa", 50, deadline)
	b := seedListingNamed(t, h, "beta")
	silent := addCandidate(t, h, f.requestID, b)

	// The first candidate replied with an offer before the window lapsed.
	if _, err := h.pool.Exec(context.Background(),
		`UPDATE request_candidate SET status = 'offered', updated_at = $2 WHERE id = $1`,
		f.candidateID, tstz(baseTime.Add(time.Hour))); err != nil {
		t.Fatalf("balas kandidat: %v", err)
	}

	h.clock.Set(baseTime.Add(72*time.Hour + time.Hour))
	if err := h.svc.runRequestExpire(context.Background()); err != nil {
		t.Fatalf("runRequestExpire: %v", err)
	}

	if got := candidateStatus(t, h, silent); got != "expired" {
		t.Fatalf("kandidat diam %q, mau expired (FR-037)", got)
	}
	if got := candidateStatus(t, h, f.candidateID); got != "offered" {
		t.Fatalf("kandidat penawar %q, mau offered (FR-037)", got)
	}
	if got := countEvent(h.notifier.events, sqlcgen.EventTypeRequestExpired); got != 0 {
		t.Fatalf("buyer diberi tahu %d kali padahal ada penawaran, mau 0 (FR-037/AS-7)", got)
	}
}

// seedListingNamed seeds a fresh subcontractor listing and returns its id, so a
// test can place an extra candidate on an existing request.
func seedListingNamed(t *testing.T, h *harness, name string) pgtype.UUID {
	t.Helper()
	listingID, _ := seedListing(t, h, name)
	return listingID
}

// addCandidate places one more candidate on an existing request and returns the
// candidate id.
func addCandidate(t *testing.T, h *harness, requestID, listingID pgtype.UUID) pgtype.UUID {
	t.Helper()
	var subconProf pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`SELECT profile_id FROM capacity_listing WHERE id = $1`, listingID).Scan(&subconProf); err != nil {
		t.Fatalf("baca profil listing: %v", err)
	}
	var candidateID pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`INSERT INTO request_candidate (request_id, listing_id, subcontractor_id, updated_at)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		requestID, listingID, subconProf, tstz(baseTime)).Scan(&candidateID); err != nil {
		t.Fatalf("tambah kandidat: %v", err)
	}
	return candidateID
}

// expiredCount counts the candidates of a request now in status expired.
func expiredCount(t *testing.T, h *harness, requestID pgtype.UUID) int {
	t.Helper()
	var n int
	if err := h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM request_candidate WHERE request_id = $1 AND status = 'expired'`,
		requestID).Scan(&n); err != nil {
		t.Fatalf("hitung kandidat kedaluwarsa: %v", err)
	}
	return n
}
