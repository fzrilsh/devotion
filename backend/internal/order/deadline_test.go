package order

import (
	"testing"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// jkt is the zone the domain reasons in; test instants are built here so a
// boundary check is not thrown off by the host zone.
var jkt = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic(err)
	}
	return loc
}()

// TestAutoConfirm_BoundaryInstant_FR068_FR070 proves the read predicate and the
// job agree at the exact auto-confirm instant, and that an open dispute halts the
// count regardless of the instant. One nanosecond before the boundary the order
// is not yet due, at it and after it the order is due; but with an open dispute
// the order is never due, even well past the boundary. This is the boundary the
// two scheduler layers must not disagree on, dispute included (FR-070).
func TestAutoConfirm_BoundaryInstant_FR068_FR070(t *testing.T) {
	shipped := time.Date(2026, 8, 1, 9, 0, 0, 0, jkt)
	at := AutoConfirmAt(shipped)
	if got := at.Sub(shipped); got != AutoConfirmWindow {
		t.Fatalf("AutoConfirmAt lead = %v, mau %v", got, AutoConfirmWindow)
	}
	if IsAutoConfirmDue(shipped, at.Add(-time.Nanosecond), false) {
		t.Fatal("due sebelum instan batas, mau belum")
	}
	if !IsAutoConfirmDue(shipped, at, false) {
		t.Fatal("belum due pada instan batas, mau due")
	}
	if !IsAutoConfirmDue(shipped, at.Add(time.Nanosecond), false) {
		t.Fatal("belum due setelah instan batas, mau due")
	}
	// An open dispute stops the count: even a week past the boundary the order is
	// not due, so the lazy read layer never flips a disputed order to confirmed.
	if IsAutoConfirmDue(shipped, at.Add(7*24*time.Hour), true) {
		t.Fatal("due meski ada sengketa terbuka, mau belum (FR-070)")
	}
}

// TestAutoConfirmApproaching_WarnWindow_FR069 proves the warning fires inside
// the two-day lead but not before it and not once the order is already due.
func TestAutoConfirmApproaching_WarnWindow_FR069(t *testing.T) {
	shipped := time.Date(2026, 8, 1, 9, 0, 0, 0, jkt)
	at := AutoConfirmAt(shipped)
	warnStart := at.Add(-AutoConfirmWarnLead)

	if IsAutoConfirmApproaching(shipped, warnStart.Add(-time.Nanosecond)) {
		t.Fatal("approaching sebelum jendela peringatan, mau belum")
	}
	if !IsAutoConfirmApproaching(shipped, warnStart) {
		t.Fatal("belum approaching pada awal jendela, mau approaching")
	}
	if !IsAutoConfirmApproaching(shipped, at.Add(-time.Nanosecond)) {
		t.Fatal("belum approaching tepat sebelum tenggat, mau approaching")
	}
	if IsAutoConfirmApproaching(shipped, at) {
		t.Fatal("masih approaching pada tenggat, mau tidak (sudah due)")
	}
}

// TestReadinessDeadline_RoundsToMonday_FR087 proves the readiness week is the
// Monday of the week that contains agreement date plus the lead, matching the
// week_start used everywhere else.
func TestReadinessDeadline_RoundsToMonday_FR087(t *testing.T) {
	// Wednesday 2026-08-05; lead 5 days lands on Monday 2026-08-10.
	agreed := time.Date(2026, 8, 5, 14, 0, 0, 0, jkt)
	got := ReadinessDeadline(agreed, 5)
	want := platform.WeekStart(time.Date(2026, 8, 10, 0, 0, 0, 0, jkt))
	if !got.Equal(want) {
		t.Fatalf("ReadinessDeadline = %v, mau %v", got, want)
	}
	if got.Weekday() != time.Monday {
		t.Fatalf("readiness week bukan Senin: %v", got.Weekday())
	}
}

// TestIsCalendarStale_Boundary_FR021 proves a calendar exactly at the stale
// window edge is not yet stale and one instant past it is.
func TestIsCalendarStale_Boundary_FR021(t *testing.T) {
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, jkt)
	edge := now.Add(-CalendarStaleWindow)
	if IsCalendarStale(edge, now) {
		t.Fatal("stale tepat pada batas, mau belum")
	}
	if !IsCalendarStale(edge.Add(-time.Nanosecond), now) {
		t.Fatal("belum stale melewati batas, mau stale")
	}
}

// TestIsRequestExpired_Boundary_FR037 proves the quota reply window is
// inclusive at its due instant.
func TestIsRequestExpired_Boundary_FR037(t *testing.T) {
	due := time.Date(2026, 8, 22, 10, 0, 0, 0, jkt)
	if IsRequestExpired(due, due.Add(-time.Nanosecond)) {
		t.Fatal("expired sebelum reply_due_at, mau belum")
	}
	if !IsRequestExpired(due, due) {
		t.Fatal("belum expired pada reply_due_at, mau expired")
	}
}
