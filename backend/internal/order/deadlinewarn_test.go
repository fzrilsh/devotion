package order

import (
	"context"
	"testing"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
)

// setDeadline overwrites a work order's delivery deadline to a fixed date so a
// test controls where the deadline falls relative to the FR-051 warning band,
// independent of the week the accept path derived. It also clears
// deadline_warn_sent_at so the order starts un-warned.
func setDeadline(t *testing.T, h *woHarness, at time.Time) {
	t.Helper()
	if _, err := h.pool.Exec(context.Background(),
		`UPDATE work_order SET deadline = $2, deadline_warn_sent_at = NULL WHERE id = $1`,
		h.workOrderID, at); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
}

// TestDeadlineApproaching_WithinLeadWarnsBothPartiesOnce_FR051 proves the ticker
// warns both the buyer and the subcontractor once when an active order's delivery
// deadline is inside the seven-day lead, and that a second consecutive pass sends
// nothing more: deadline_warn_sent_at stamps the row under an IS NULL guard, so two
// overlapping instances during a deploy rollover each warn at most once. The Clock
// is advanced, never waited on.
func TestDeadlineApproaching_WithinLeadWarnsBothPartiesOnce_FR051(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_dl_within")
	rec := &recordingNotifier{}
	h.svc.notifier = rec

	// Deadline three days out: inside the 7-day lead, not yet past.
	setDeadline(t, h, acceptBaseTime.Add(3*24*time.Hour))

	if err := h.svc.runDeadlineApproaching(context.Background()); err != nil {
		t.Fatalf("runDeadlineApproaching pass 1: %v", err)
	}
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeDeadlineApproaching); got != 1 {
		t.Fatalf("pemberi order diberi tahu %d kali di pass 1, mau 1 (FR-051)", got)
	}
	if got := rec.countFor(h.subAcc, sqlcgen.EventTypeDeadlineApproaching); got != 1 {
		t.Fatalf("subkontraktor diberi tahu %d kali di pass 1, mau 1 (FR-051)", got)
	}

	// A second pass, still approaching, must not re-warn either party.
	h.clock.Advance(6 * time.Hour)
	if err := h.svc.runDeadlineApproaching(context.Background()); err != nil {
		t.Fatalf("runDeadlineApproaching pass 2: %v", err)
	}
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeDeadlineApproaching); got != 1 {
		t.Fatalf("pemberi order diberi tahu %d kali total, mau tetap 1 (FR-051)", got)
	}
	if got := rec.countFor(h.subAcc, sqlcgen.EventTypeDeadlineApproaching); got != 1 {
		t.Fatalf("subkontraktor diberi tahu %d kali total, mau tetap 1 (FR-051)", got)
	}
}

// TestDeadlineApproaching_BeyondLeadUntouched_FR051 proves an order whose deadline
// is further out than the seven-day lead earns no warning, so the job never nudges
// a party about a deadline that is still weeks away.
func TestDeadlineApproaching_BeyondLeadUntouched_FR051(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_dl_beyond")
	rec := &recordingNotifier{}
	h.svc.notifier = rec

	// Deadline ten days out: past the far edge of the 7-day lead.
	setDeadline(t, h, acceptBaseTime.Add(10*24*time.Hour))

	if err := h.svc.runDeadlineApproaching(context.Background()); err != nil {
		t.Fatalf("runDeadlineApproaching: %v", err)
	}
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeDeadlineApproaching); got != 0 {
		t.Fatalf("pemberi order diberi tahu %d kali sebelum masuk masa peringatan, mau 0 (FR-051)", got)
	}
	if got := rec.countFor(h.subAcc, sqlcgen.EventTypeDeadlineApproaching); got != 0 {
		t.Fatalf("subkontraktor diberi tahu %d kali sebelum masuk masa peringatan, mau 0 (FR-051)", got)
	}
}

// TestDeadlineApproaching_PastDeadlineNotWarned_FR051 proves an order already past
// its deadline is not warned by this job: the past-deadline cutoff is the near edge
// of the band, so a late order is left to the late-order job (FR-045) and the two
// notices never both fire for one order.
func TestDeadlineApproaching_PastDeadlineNotWarned_FR051(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_dl_past")
	rec := &recordingNotifier{}
	h.svc.notifier = rec

	// Deadline two days in the past: this is the late-order job's territory.
	setDeadline(t, h, acceptBaseTime.Add(-2*24*time.Hour))

	if err := h.svc.runDeadlineApproaching(context.Background()); err != nil {
		t.Fatalf("runDeadlineApproaching: %v", err)
	}
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeDeadlineApproaching); got != 0 {
		t.Fatalf("pemberi order diberi tahu mendekat %d kali untuk pesanan yang sudah telat, mau 0 (FR-051)", got)
	}
	if got := rec.countFor(h.subAcc, sqlcgen.EventTypeDeadlineApproaching); got != 0 {
		t.Fatalf("subkontraktor diberi tahu mendekat %d kali untuk pesanan yang sudah telat, mau 0 (FR-051)", got)
	}
}

// TestDeadlineApproaching_ShippedOrderNotWarned_FR051 proves a shipped order inside
// the lead is not warned: once shipped, the relevant clock is the auto-confirm
// warning (FR-069), not the delivery deadline, so the status filter excludes it.
func TestDeadlineApproaching_ShippedOrderNotWarned_FR051(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_dl_shipped")
	rec := &recordingNotifier{}
	h.svc.notifier = rec

	shipWorkOrder(t, h)
	// Deadline three days out, but the order has already shipped.
	setDeadline(t, h, acceptBaseTime.Add(3*24*time.Hour))

	if err := h.svc.runDeadlineApproaching(context.Background()); err != nil {
		t.Fatalf("runDeadlineApproaching: %v", err)
	}
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeDeadlineApproaching); got != 0 {
		t.Fatalf("pemberi order diberi tahu %d kali untuk pesanan terkirim, mau 0 (FR-051)", got)
	}
	if got := rec.countFor(h.subAcc, sqlcgen.EventTypeDeadlineApproaching); got != 0 {
		t.Fatalf("subkontraktor diberi tahu %d kali untuk pesanan terkirim, mau 0 (FR-051)", got)
	}
}
