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
