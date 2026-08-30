package order

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
)

// recordingNotifier captures every Enqueue so a test can assert who was notified
// of what. It satisfies Notifier without a queue; the auto-confirm tests assert
// on the closure and warning notices, not on delivery.
type recordingNotifier struct {
	mu    sync.Mutex
	calls []recordedNotice
}

type recordedNotice struct {
	account pgtype.UUID
	event   sqlcgen.EventType
}

func (n *recordingNotifier) Enqueue(_ context.Context, _ pgx.Tx, account pgtype.UUID, event sqlcgen.EventType, _, _ string, _ *string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.calls = append(n.calls, recordedNotice{account: account, event: event})
	return nil
}

func (n *recordingNotifier) countFor(account pgtype.UUID, event sqlcgen.EventType) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	c := 0
	for _, call := range n.calls {
		if call.account == account && call.event == event {
			c++
		}
	}
	return c
}

// shipWorkOrder drives a freshly accepted order forward to 'shipped' through the
// real state machine, so shipped_at is stamped from the Clock exactly as
// production would stamp it. It leaves the clock at acceptBaseTime.
func shipWorkOrder(t *testing.T, h *woHarness) {
	t.Helper()
	ctx := context.Background()
	for _, s := range []sqlcgen.WorkOrderStatus{
		sqlcgen.WorkOrderStatusProduction,
		sqlcgen.WorkOrderStatusCompleted,
		sqlcgen.WorkOrderStatusShipped,
	} {
		if _, err := h.svc.changeStatus(ctx, h.subAcc, h.workOrderID, s, nil); err != nil {
			t.Fatalf("majukan status ke %s: %v", s, err)
		}
	}
}

// loadWorkOrder reads the raw row so a test can assert status and auto_confirmed
// as stored, not as the lazy read layer would recompute them.
func loadWorkOrder(t *testing.T, h *woHarness) sqlcgen.WorkOrder {
	t.Helper()
	wo, err := h.svc.queries().LockWorkOrderForReversal(context.Background(), h.workOrderID)
	if err != nil {
		t.Fatalf("baca work_order: %v", err)
	}
	return wo
}

// TestAutoConfirm_SixDaysNotYetDue_FR068 proves a shipped order six days on is not
// auto-confirmed by the ticker and still reads as shipped with its auto_confirm_at
// deadline visible, so the buyer sees when it will close.
func TestAutoConfirm_SixDaysNotYetDue_FR068(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_ac_sixdays")
	shipWorkOrder(t, h)

	// Six days after shipment: inside the 7-day window, so nothing closes.
	h.clock.Advance(6 * 24 * time.Hour)
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm: %v", err)
	}

	wo := loadWorkOrder(t, h)
	if wo.Status != sqlcgen.WorkOrderStatusShipped {
		t.Fatalf("status tersimpan %q, mau %q", wo.Status, sqlcgen.WorkOrderStatusShipped)
	}
	if wo.AutoConfirmed {
		t.Fatal("auto_confirmed true sebelum tenggat; pesanan belum boleh ditutup (FR-068)")
	}

	// The detail view still carries the auto-confirm deadline so the buyer sees it.
	row, err := h.svc.queries().GetWorkOrderForView(context.Background(), h.workOrderID)
	if err != nil {
		t.Fatalf("GetWorkOrderForView: %v", err)
	}
	view, err := h.svc.buildDetailView(context.Background(), row)
	if err != nil {
		t.Fatalf("buildDetailView: %v", err)
	}
	if view.Status != string(sqlcgen.WorkOrderStatusShipped) {
		t.Fatalf("status tampilan %q, mau %q", view.Status, sqlcgen.WorkOrderStatusShipped)
	}
	if view.AutoConfirmAt == nil {
		t.Fatal("auto_confirm_at tidak ada; buyer tak melihat tenggat konfirmasi (FR-068)")
	}
}

// TestAutoConfirm_EightDaysCloses_FR068 proves the ticker closes a shipped order
// past its 7-day window: status becomes confirmed, auto_confirmed marks it as the
// system's closure, and both parties are notified the order closed automatically.
func TestAutoConfirm_EightDaysCloses_FR068(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_ac_eightdays")
	rec := &recordingNotifier{}
	h.svc.notifier = rec
	shipWorkOrder(t, h)

	h.clock.Advance(8 * 24 * time.Hour)
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm: %v", err)
	}

	wo := loadWorkOrder(t, h)
	if wo.Status != sqlcgen.WorkOrderStatusConfirmed {
		t.Fatalf("status %q, mau %q", wo.Status, sqlcgen.WorkOrderStatusConfirmed)
	}
	if !wo.AutoConfirmed {
		t.Fatal("auto_confirmed false; penutupan otomatis harus menandai dirinya (FR-068)")
	}
	if !wo.ConfirmedAt.Valid {
		t.Fatal("confirmed_at kosong pada pesanan terkonfirmasi (FR-068)")
	}
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeOrderAutoClosed); got != 1 {
		t.Fatalf("pemberi order diberi tahu %d kali, mau 1 (FR-068)", got)
	}
	if got := rec.countFor(h.subAcc, sqlcgen.EventTypeOrderAutoClosed); got != 1 {
		t.Fatalf("subkontraktor diberi tahu %d kali, mau 1 (FR-068)", got)
	}

	// Both parties are invited to rate each other once the order auto-closes
	// (FR-051 "permintaan rating", US5 AS-1).
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeRatingRequest); got != 1 {
		t.Fatalf("pemberi order diminta ulasan %d kali, mau 1 (FR-051)", got)
	}
	if got := rec.countFor(h.subAcc, sqlcgen.EventTypeRatingRequest); got != 1 {
		t.Fatalf("subkontraktor diminta ulasan %d kali, mau 1 (FR-051)", got)
	}

	// The system closure records a by_system history row with no human actor.
	history, err := h.svc.queries().ListWorkOrderStatusHistory(context.Background(), h.workOrderID)
	if err != nil {
		t.Fatalf("ListWorkOrderStatusHistory: %v", err)
	}
	if last := history[len(history)-1]; last.NewStatus != sqlcgen.WorkOrderStatusConfirmed {
		t.Fatalf("riwayat terakhir %q, mau %q", last.NewStatus, sqlcgen.WorkOrderStatusConfirmed)
	}
}

// TestAutoConfirm_OpenDisputeNotClosed_FR070 proves a disputed order is never
// auto-closed even past the window: an open (unresolved) dispute row makes the
// auto-confirm scan skip the order, so the count stops on the dispute's existence,
// not on the work-order status. The order stays shipped (admin moves it to
// in_mediation separately, T071) and is never auto_confirmed.
func TestAutoConfirm_OpenDisputeNotClosed_FR070(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_ac_dispute")
	shipWorkOrder(t, h)

	// A party reports a dispute before the deadline. The report leaves the order
	// on 'shipped'; the open dispute row is what stops the clock.
	if _, err := h.pool.Exec(context.Background(),
		`INSERT INTO dispute (work_order_id, reporter_id, report_body, created_at)
		 VALUES ($1, (SELECT buyer_id FROM work_order WHERE id = $1), $2, $3)`,
		h.workOrderID, "Barang tidak sesuai contoh yang disepakati.", h.clock.Now()); err != nil {
		t.Fatalf("buka sengketa: %v", err)
	}

	h.clock.Advance(8 * 24 * time.Hour)
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm: %v", err)
	}

	wo := loadWorkOrder(t, h)
	if wo.Status != sqlcgen.WorkOrderStatusShipped {
		t.Fatalf("status %q, mau %q; sengketa terbuka menghentikan hitung mundur tanpa memindahkan status (FR-070)", wo.Status, sqlcgen.WorkOrderStatusShipped)
	}
	if wo.AutoConfirmed {
		t.Fatal("auto_confirmed true pada pesanan bersengketa (FR-070)")
	}
}

// TestAutoConfirm_ManualConfirmBeforeDeadline_FR068 proves an order confirmed by a
// party before the window is not marked auto_confirmed: the auto-close guard is a
// no-op on a non-shipped order, so the manual confirmation stands as the party's.
func TestAutoConfirm_ManualConfirmBeforeDeadline_FR068(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_ac_manual")
	shipWorkOrder(t, h)

	// A party confirms manually (auto_confirmed left false) before the window.
	if _, err := h.pool.Exec(context.Background(),
		`UPDATE work_order SET status = 'confirmed', confirmed_at = $2 WHERE id = $1`,
		h.workOrderID, h.clock.Now()); err != nil {
		t.Fatalf("konfirmasi manual: %v", err)
	}

	h.clock.Advance(8 * 24 * time.Hour)
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm: %v", err)
	}

	wo := loadWorkOrder(t, h)
	if wo.Status != sqlcgen.WorkOrderStatusConfirmed {
		t.Fatalf("status %q, mau %q", wo.Status, sqlcgen.WorkOrderStatusConfirmed)
	}
	if wo.AutoConfirmed {
		t.Fatal("auto_confirmed true pada konfirmasi manual; hanya penutupan sistem yang menandai (FR-068)")
	}
}

// TestAutoConfirm_WarnsBuyerOnce_FR069 proves the buyer is warned once when the
// deadline is inside the lead, and a second pass does not re-warn: the
// confirm_warn_sent_at stamp dedups the notice.
func TestAutoConfirm_WarnsBuyerOnce_FR069(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_ac_warn")
	rec := &recordingNotifier{}
	h.svc.notifier = rec
	shipWorkOrder(t, h)

	// Six days on: inside the 2-day warning lead (deadline at day 7), not yet due.
	h.clock.Advance(6 * 24 * time.Hour)
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm pass 1: %v", err)
	}
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeConfirmationDueApproaching); got != 1 {
		t.Fatalf("buyer diperingatkan %d kali di pass 1, mau 1 (FR-069)", got)
	}

	// A later pass still inside the lead must not re-warn.
	h.clock.Advance(6 * time.Hour) // +6 hours, still day 6
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm pass 2: %v", err)
	}
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeConfirmationDueApproaching); got != 1 {
		t.Fatalf("buyer diperingatkan %d kali total, mau tepat 1 (FR-069)", got)
	}
}

// TestAutoConfirm_BothLayersShareDomainFunction_FR068_FR070 proves the lazy read
// layer and the ticker agree because both call IsAutoConfirmDue: at one hour
// before the deadline the read view still shows shipped and the job closes
// nothing; one hour after, the read view shows confirmed and the job also closes
// the row. If the two layers used different arithmetic they would disagree here.
// The disputed case is the one this used to miss: a disputed order well past the
// deadline must read as shipped in the detail view AND be left shipped by the
// ticker. Before has_open_dispute was threaded into IsAutoConfirmDue the read
// layer flipped it to confirmed while the ticker left it shipped, so the same
// order looked confirmed to the buyer and shipped to the scheduler (FR-070).
func TestAutoConfirm_BothLayersShareDomainFunction_FR068_FR070(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_ac_bothlayers")
	shipWorkOrder(t, h)
	wo := loadWorkOrder(t, h)
	shippedAt := wo.ShippedAt.Time

	// One hour before the shared AutoConfirmAt: neither layer treats it as due.
	h.clock.Set(AutoConfirmAt(shippedAt).Add(-time.Hour))
	row, err := h.svc.queries().GetWorkOrderForView(context.Background(), h.workOrderID)
	if err != nil {
		t.Fatalf("GetWorkOrderForView: %v", err)
	}
	view, err := h.svc.buildDetailView(context.Background(), row)
	if err != nil {
		t.Fatalf("buildDetailView: %v", err)
	}
	if view.Status != string(sqlcgen.WorkOrderStatusShipped) {
		t.Fatalf("lapisan baca menutup lebih awal: %q, mau %q", view.Status, sqlcgen.WorkOrderStatusShipped)
	}
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm sebelum tenggat: %v", err)
	}
	if loadWorkOrder(t, h).AutoConfirmed {
		t.Fatal("ticker menutup sebelum tenggat; dua lapisan tidak sepakat (FR-068)")
	}

	// One hour after: both layers now treat it as confirmed.
	h.clock.Set(AutoConfirmAt(shippedAt).Add(time.Hour))
	row, err = h.svc.queries().GetWorkOrderForView(context.Background(), h.workOrderID)
	if err != nil {
		t.Fatalf("GetWorkOrderForView: %v", err)
	}
	view, err = h.svc.buildDetailView(context.Background(), row)
	if err != nil {
		t.Fatalf("buildDetailView: %v", err)
	}
	if view.Status != string(sqlcgen.WorkOrderStatusConfirmed) {
		t.Fatalf("lapisan baca belum menutup setelah tenggat: %q, mau %q", view.Status, sqlcgen.WorkOrderStatusConfirmed)
	}
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm setelah tenggat: %v", err)
	}
	if !loadWorkOrder(t, h).AutoConfirmed {
		t.Fatal("ticker tak menutup setelah tenggat; dua lapisan tidak sepakat (FR-068)")
	}
}

// TestAutoConfirm_BothLayersAgreeOnDisputed_FR070 proves the read layer and the
// ticker agree that a disputed order stays shipped even well past the deadline.
// This is the divergence the has_open_dispute flag closes: the ticker's NOT
// EXISTS guard has always skipped disputed orders, but the read layer used to
// recompute "confirmed" from shipped_at and now alone, so the buyer saw a
// confirmed order the scheduler still treated as shipped (FR-070). Both layers
// now feed the same open-dispute flag into IsAutoConfirmDue, so both say shipped.
func TestAutoConfirm_BothLayersAgreeOnDisputed_FR070(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_ac_bothdispute")
	shipWorkOrder(t, h)

	// A party reports a dispute; the order stays 'shipped', the open dispute row
	// is what halts the count in both layers.
	if _, err := h.pool.Exec(context.Background(),
		`INSERT INTO dispute (work_order_id, reporter_id, report_body, created_at)
		 VALUES ($1, (SELECT buyer_id FROM work_order WHERE id = $1), $2, $3)`,
		h.workOrderID, "Barang tidak sesuai contoh yang disepakati.", h.clock.Now()); err != nil {
		t.Fatalf("buka sengketa: %v", err)
	}

	// Well past the 7-day deadline: without the dispute this would read confirmed.
	h.clock.Advance(8 * 24 * time.Hour)

	// Read layer: the detail view must still show shipped, not confirmed.
	row, err := h.svc.queries().GetWorkOrderForView(context.Background(), h.workOrderID)
	if err != nil {
		t.Fatalf("GetWorkOrderForView: %v", err)
	}
	if !row.HasOpenDispute {
		t.Fatal("has_open_dispute false pada pesanan bersengketa; lapisan baca tak akan tahu (FR-070)")
	}
	view, err := h.svc.buildDetailView(context.Background(), row)
	if err != nil {
		t.Fatalf("buildDetailView: %v", err)
	}
	if view.Status != string(sqlcgen.WorkOrderStatusShipped) {
		t.Fatalf("lapisan baca membaca pesanan bersengketa sebagai %q, mau %q (FR-070)", view.Status, sqlcgen.WorkOrderStatusShipped)
	}

	// Ticker: the scheduler must also leave it shipped, never auto-confirmed.
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm: %v", err)
	}
	stored := loadWorkOrder(t, h)
	if stored.Status != sqlcgen.WorkOrderStatusShipped {
		t.Fatalf("penjadwal menutup pesanan bersengketa: status %q, mau %q (FR-070)", stored.Status, sqlcgen.WorkOrderStatusShipped)
	}
	if stored.AutoConfirmed {
		t.Fatal("auto_confirmed true pada pesanan bersengketa; dua lapisan harus sepakat 'belum dikonfirmasi' (FR-070)")
	}
}
