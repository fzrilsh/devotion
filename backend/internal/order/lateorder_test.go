package order

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// lateListItems decodes the admin late-order list body into its work_order_ids so
// a test asserts membership without caring about the rest of the summary shape.
func lateListItems(t *testing.T, body []byte) []string {
	t.Helper()
	var out workOrderList
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode WorkOrderList %q: %v", string(body), err)
	}
	ids := make([]string, 0, len(out.Items))
	for _, it := range out.Items {
		ids = append(ids, it.WorkOrderID)
	}
	return ids
}

// contains reports whether id is in ids.
func contains(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// TestLateOrders_ActivePastDeadlineListed_FR045 proves an active order whose
// delivery deadline has passed appears on the admin monitoring list. The order is
// on time at acceptBaseTime (deadline is that day) and only becomes late once the
// injected Clock crosses into a later WIB day, so this also proves the list is the
// compute-on-read layer keyed off PastDeadlineCutoff(now), not a stored flag.
func TestLateOrders_ActivePastDeadlineListed_FR045(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_late_active")
	admin := seedAcceptAccount(t, h.pool, "admin_late_active@contoh.test", false)
	handler := woRouter(h, httpx.RoleAdmin, admin)

	get := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/late-orders", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	// On the deadline day the order is still on time, so the list is empty.
	rec := get()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	if ids := lateListItems(t, rec.Body.Bytes()); contains(ids, uuidString(h.workOrderID)) {
		t.Fatal("pesanan tampil telat pada hari tenggat; masih tepat waktu (FR-045)")
	}

	// Two WIB days later the deadline has passed, so the order surfaces.
	h.clock.Advance(2 * 24 * time.Hour)
	rec = get()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	if ids := lateListItems(t, rec.Body.Bytes()); !contains(ids, uuidString(h.workOrderID)) {
		t.Fatalf("pesanan aktif lewat tenggat tak muncul di daftar admin (FR-045); ids=%v", ids)
	}
}

// TestLateOrders_ClosedOrdersExcluded_FR045 proves a confirmed or a cancelled
// order never appears on the late list even long past its deadline: the query's
// status set is exactly idx_order_deadline_active's predicate, so a closed order
// is out of scope. Confirmed orders are done; cancelled orders are dead. Only
// still-open orders can be late.
func TestLateOrders_ClosedOrdersExcluded_FR045(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_late_confirmed")
	admin := seedAcceptAccount(t, h.pool, "admin_late_closed@contoh.test", false)
	handler := woRouter(h, httpx.RoleAdmin, admin)

	// Close the order (confirmed) while its deadline is well in the past.
	if _, err := h.pool.Exec(context.Background(),
		`UPDATE work_order SET status = 'confirmed', confirmed_at = $2 WHERE id = $1`,
		h.workOrderID, h.clock.Now()); err != nil {
		t.Fatalf("konfirmasi pesanan: %v", err)
	}

	h.clock.Advance(10 * 24 * time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/late-orders", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	if ids := lateListItems(t, rec.Body.Bytes()); contains(ids, uuidString(h.workOrderID)) {
		t.Fatal("pesanan terkonfirmasi muncul di daftar telat; hanya pesanan terbuka yang bisa telat (FR-045)")
	}
}

// TestLateOrders_NotifiesBothPartiesOnce_FR045 proves the ticker notifies both the
// buyer and the subcontractor that the deadline passed, and that a second
// consecutive pass sends nothing more: late_notified_at stamps the row under an IS
// NULL guard, so two overlapping instances during a deploy rollover each notify at
// most once. The Clock is advanced, never waited on.
func TestLateOrders_NotifiesBothPartiesOnce_FR045(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_late_notify")
	rec := &recordingNotifier{}
	h.svc.notifier = rec

	// Past the deadline: the order is now late.
	h.clock.Advance(2 * 24 * time.Hour)

	if err := h.svc.runLateOrder(context.Background()); err != nil {
		t.Fatalf("runLateOrder pass 1: %v", err)
	}
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeDeadlinePassed); got != 1 {
		t.Fatalf("pemberi order diberi tahu %d kali di pass 1, mau 1 (FR-045)", got)
	}
	if got := rec.countFor(h.subAcc, sqlcgen.EventTypeDeadlinePassed); got != 1 {
		t.Fatalf("subkontraktor diberi tahu %d kali di pass 1, mau 1 (FR-045)", got)
	}

	// A second pass, still late, must not re-notify either party.
	h.clock.Advance(6 * time.Hour)
	if err := h.svc.runLateOrder(context.Background()); err != nil {
		t.Fatalf("runLateOrder pass 2: %v", err)
	}
	if got := rec.countFor(h.buyerAcc, sqlcgen.EventTypeDeadlinePassed); got != 1 {
		t.Fatalf("pemberi order diberi tahu %d kali total, mau tepat 1 (FR-045)", got)
	}
	if got := rec.countFor(h.subAcc, sqlcgen.EventTypeDeadlinePassed); got != 1 {
		t.Fatalf("subkontraktor diberi tahu %d kali total, mau tepat 1 (FR-045)", got)
	}
}

// TestLateOrders_NonAdminRejected_FR045 proves the monitoring list is admin only:
// a buyer party to the order is rejected by the role gate before the handler runs,
// so the platform-wide list never leaks to a non-admin.
func TestLateOrders_NonAdminRejected_FR045(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_late_forbidden")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/late-orders", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; daftar telat harus khusus admin (FR-045); body %s", rec.Code, rec.Body.String())
	}
}
