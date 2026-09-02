package order

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// woHarness bundles the accept harness with a formed agreement so the work-order
// tests start from a real 'accepted' order rather than hand-inserted rows. It
// exposes both parties' account ids and the created work order id.
type woHarness struct {
	*acceptHarness
	workOrderID pgtype.UUID
	buyerAcc    pgtype.UUID
	subAcc      pgtype.UUID
}

// seedAcceptedWorkOrder forms an agreement through the accept path, so the order
// exists in 'accepted' with its allocation rows, and returns the ids the tests
// drive from.
func seedAcceptedWorkOrder(t *testing.T, name string) *woHarness {
	t.Helper()
	h := newAcceptHarness(t, name)
	week := platform.WeekStart(acceptBaseTime)

	listingID, subAcc := seedListing(t, h, "alfa", 200, week, week)
	subProf := subProfileID(t, h, subAcc)

	buyer := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyer@contoh.test", false), "Pembeli")
	buyerAcc := buyerAccountOf(t, h, buyer)
	req := seedRequest(t, h, buyer, 50, week)
	offer := seedOfferedCandidate(t, h, req, listingID, subProf, 1_000_000)

	view, err := h.svc.accept(context.Background(), buyerAcc, offer)
	if err != nil {
		t.Fatalf("bentuk kesepakatan: %v", err)
	}
	woID, ok := parseUUID(view.WorkOrderID)
	if !ok {
		t.Fatalf("work_order_id tidak sah: %q", view.WorkOrderID)
	}
	return &woHarness{acceptHarness: h, workOrderID: woID, buyerAcc: buyerAcc, subAcc: subAcc}
}

// woRouter wires the order routes with a principal holding the given roles and
// account, so a test drives the real gate and party guards over HTTP.
func woRouter(h *woHarness, roles httpx.Role, account pgtype.UUID) http.Handler {
	r := httpx.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)))
	h.svc.Register(r, &mockAuth{principal: &httpx.Principal{
		Roles:   roles,
		Account: sqlcgen.UserAccount{ID: account},
	}})
	return r.Handler()
}

// TestWorkOrderDetail_PartySeesOrder_FR038 proves a party to the order can read
// its full detail over HTTP, and that the state-machine fields the frontend
// renders from are present (FR-039): allowed_transitions non-empty and
// self_cancellable set for an 'accepted' order.
func TestWorkOrderDetail_PartySeesOrder_FR038(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_detail_party")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	req := httptest.NewRequest(http.MethodGet, "/api/work-orders/"+uuidString(h.workOrderID), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var body struct {
		WorkOrderID        string    `json:"work_order_id"`
		Status             string    `json:"status"`
		AllowedTransitions *[]string `json:"allowed_transitions"`
		SelfCancellable    *bool     `json:"self_cancellable"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderDetail %q: %v", rec.Body.String(), err)
	}
	if body.Status != string(sqlcgen.WorkOrderStatusAccepted) {
		t.Fatalf("status %q, mau %q", body.Status, sqlcgen.WorkOrderStatusAccepted)
	}
	if body.AllowedTransitions == nil || len(*body.AllowedTransitions) == 0 {
		t.Fatal("allowed_transitions kosong; frontend merender tombol dari sini (FR-039)")
	}
	if body.SelfCancellable == nil {
		t.Fatal("self_cancellable tidak ada di respons (FR-066)")
	}
}

// TestWorkOrderDetail_DatesAreISO_FR038_FR039 pins the wire format of every date
// field on the detail body. deadline, readiness_deadline, and each allocation's
// week_start are `format: date` in the contract, and week_start is the same value
// the availability calendar takes back as YYYY-MM-DD, so the long Indonesian form
// made the field unparseable by any client that reads it. The seeded order is
// accepted at acceptBaseTime, a Monday, with readiness_lead_days 0 and its
// request deadline on that same week, so all three name that Monday.
func TestWorkOrderDetail_DatesAreISO_FR038_FR039(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_detail_iso")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	req := httptest.NewRequest(http.MethodGet, "/api/work-orders/"+uuidString(h.workOrderID), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var body workOrderView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderDetail %q: %v", rec.Body.String(), err)
	}

	want := platform.FormatDate(platform.WeekStart(acceptBaseTime))
	if body.Deadline != want {
		t.Fatalf("deadline = %q, mau %q; kontrak menyatakan format: date", body.Deadline, want)
	}
	if body.ReadinessDeadline != want {
		t.Fatalf("readiness_deadline = %q, mau %q; kontrak menyatakan format: date", body.ReadinessDeadline, want)
	}
	if len(body.Allocations) == 0 {
		t.Fatal("allocations kosong; pesanan yang disepakati punya minimal satu baris alokasi")
	}
	for i, a := range body.Allocations {
		if _, err := platform.ParseDate(a.WeekStart); err != nil {
			t.Fatalf("allocations[%d].week_start = %q, tidak lolos ParseDate: %v; kalender ketersediaan menerima kembali nilai ini", i, a.WeekStart, err)
		}
		if a.WeekStart != want {
			t.Fatalf("allocations[%d].week_start = %q, mau %q", i, a.WeekStart, want)
		}
	}
	// A date the response gives must be a date the API takes back, or the same
	// field is two formats depending on direction.
	for name, got := range map[string]string{"deadline": body.Deadline, "readiness_deadline": body.ReadinessDeadline} {
		if _, err := platform.ParseDate(got); err != nil {
			t.Fatalf("%s = %q tidak lolos ParseDate: %v", name, got, err)
		}
	}
}

// TestWorkOrderDetail_NonPartyGets404_FR038 proves a caller who is on neither
// side of the order gets the same 404 as a missing order, so the endpoint never
// confirms an order exists to someone not party to it.
func TestWorkOrderDetail_NonPartyGets404_FR038(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_detail_nonparty")
	stranger := seedAcceptAccount(t, h.pool, "stranger@contoh.test", false)
	handler := woRouter(h, httpx.RoleBuyer, stranger)

	req := httptest.NewRequest(http.MethodGet, "/api/work-orders/"+uuidString(h.workOrderID), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404; body %s", rec.Code, rec.Body.String())
	}
}

// TestWorkOrderDetail_AdminReadsNonPartyOrder_FR045_FR046 proves an admin reads
// the full detail of an order it is on neither side of. FR-046 requires an admin
// mediating a dispute to see the whole history, and FR-045 puts a late order in
// front of an admin who is not a party either. The admin late list and the dispute
// queue both carry only a work_order_id, so this is the only route to the history.
func TestWorkOrderDetail_AdminReadsNonPartyOrder_FR045_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_detail_admin")
	admin := seedAcceptAccount(t, h.pool, "admin_detail@contoh.test", false)
	handler := woRouter(h, httpx.RoleAdmin, admin)

	req := httptest.NewRequest(http.MethodGet, "/api/work-orders/"+uuidString(h.workOrderID), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; admin harus dapat membaca riwayat pesanan yang bukan pihaknya (FR-046); body %s",
			rec.Code, rec.Body.String())
	}
	var body workOrderView
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderDetail: %v", err)
	}
	if body.WorkOrderID != uuidString(h.workOrderID) {
		t.Fatalf("work_order_id %q, mau %q", body.WorkOrderID, uuidString(h.workOrderID))
	}
	if len(body.StatusHistory) == 0 {
		t.Fatal("status_history kosong; admin membuka detail justru untuk riwayatnya (FR-046)")
	}
	if len(body.Allocations) == 0 {
		t.Fatal("allocations kosong; FR-046 menuntut alokasi kapasitas terbaca admin")
	}
}

// TestWorkOrderDetail_AdminCannotAdvanceStatus_FR045_FR046 proves widening the
// detail read did not widen the write: the forward status change stays
// subcontractor only (FR-005), and an admin moves an order through dispute
// resolution instead.
func TestWorkOrderDetail_AdminCannotAdvanceStatus_FR045_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_status_admin")
	admin := seedAcceptAccount(t, h.pool, "admin_status@contoh.test", false)
	handler := woRouter(h, httpx.RoleAdmin, admin)

	req := httptest.NewRequest(http.MethodPost,
		"/api/work-orders/"+uuidString(h.workOrderID)+"/status",
		strings.NewReader(`{"new_status":"production"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; admin membaca saja, transisi maju tetap milik subkontraktor (FR-005); body %s",
			rec.Code, rec.Body.String())
	}
}

// TestWorkOrderStatus_SubcontractorAdvances_FR039 proves the subcontractor can
// drive the order forward one legal step (accepted -> production), the response
// reflects the new status, and the allowed transitions recompute from it.
func TestWorkOrderStatus_SubcontractorAdvances_FR039(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_status_advance")
	handler := woRouter(h, httpx.RoleSubcontractor, h.subAcc)

	req := httptest.NewRequest(http.MethodPost,
		"/api/work-orders/"+uuidString(h.workOrderID)+"/status",
		strings.NewReader(`{"new_status":"production"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	if body.Status != string(sqlcgen.WorkOrderStatusProduction) {
		t.Fatalf("status %q, mau %q", body.Status, sqlcgen.WorkOrderStatusProduction)
	}
}

// TestWorkOrderStatus_RejectsNonSubcontractor_FR005 proves the buyer cannot drive
// the forward status change: the route's role gate turns them away with 403
// before the handler runs.
func TestWorkOrderStatus_RejectsNonSubcontractor_FR005(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_status_role")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	req := httptest.NewRequest(http.MethodPost,
		"/api/work-orders/"+uuidString(h.workOrderID)+"/status",
		strings.NewReader(`{"new_status":"production"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; body %s", rec.Code, rec.Body.String())
	}
}

// TestWorkOrderStatus_RejectsIllegalJump_FR044 proves an out-of-order jump
// (accepted -> shipped, skipping production and completed) is rejected with
// INVALID_STATUS_TRANSITION, and the detail quotes the allowed forward order so
// the tester can read what is permitted.
func TestWorkOrderStatus_RejectsIllegalJump_FR044(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_status_jump")
	handler := woRouter(h, httpx.RoleSubcontractor, h.subAcc)

	req := httptest.NewRequest(http.MethodPost,
		"/api/work-orders/"+uuidString(h.workOrderID)+"/status",
		strings.NewReader(`{"new_status":"shipped"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d, mau 409; body %s", rec.Code, rec.Body.String())
	}
	p := decodeAcceptProblem(t, rec)
	if p.Code != string(httpx.CodeInvalidStatusTransition) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeInvalidStatusTransition)
	}
}

// TestWorkOrderStatus_RejectsUnknownStatus_ContractValidationFailed proves a
// new_status outside the forward set (production, completed, shipped) is a 422
// VALIDATION_FAILED, not a status-transition conflict. Cancellation and
// confirmation move through their own endpoints, so 'cancelled' is invalid input
// here.
func TestWorkOrderStatus_RejectsUnknownStatus_ContractValidationFailed(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_status_badinput")
	handler := woRouter(h, httpx.RoleSubcontractor, h.subAcc)

	req := httptest.NewRequest(http.MethodPost,
		"/api/work-orders/"+uuidString(h.workOrderID)+"/status",
		strings.NewReader(`{"new_status":"cancelled"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestWorkOrderList_PartyListsOwnOrders_FR038 proves the list returns the caller's
// order under the WorkOrderList shape (items + pagination), scoped to a party.
func TestWorkOrderList_PartyListsOwnOrders_FR038(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_list_party")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	req := httptest.NewRequest(http.MethodGet, "/api/work-orders", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []struct {
			WorkOrderID string `json:"work_order_id"`
		} `json:"items"`
		Pagination *struct {
			HasNext bool `json:"has_next"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderList %q: %v", rec.Body.String(), err)
	}
	if body.Pagination == nil {
		t.Fatal("pagination tidak ada di WorkOrderList")
	}
	if len(body.Items) != 1 || body.Items[0].WorkOrderID != uuidString(h.workOrderID) {
		t.Fatalf("items = %+v, mau satu pesanan milik pemanggil", body.Items)
	}
}

// TestWorkOrderList_DatesAreISO_FR038 pins the wire format on the list shape.
// listItemView builds its own literal, separate from the detail view, so the two
// can drift: the list is what the orders index renders, and a date it cannot parse
// shows as an empty cell there even when the detail page is right. Both date
// fields must carry the same ISO form the detail body does.
func TestWorkOrderList_DatesAreISO_FR038(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_list_iso")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	req := httptest.NewRequest(http.MethodGet, "/api/work-orders", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []workOrderView `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderList %q: %v", rec.Body.String(), err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("items = %d, mau 1 pesanan milik pemanggil", len(body.Items))
	}

	want := platform.FormatDate(platform.WeekStart(acceptBaseTime))
	it := body.Items[0]
	if it.Deadline != want {
		t.Fatalf("items[0].deadline = %q, mau %q; kontrak menyatakan format: date", it.Deadline, want)
	}
	if it.ReadinessDeadline != want {
		t.Fatalf("items[0].readiness_deadline = %q, mau %q; kontrak menyatakan format: date", it.ReadinessDeadline, want)
	}
}

// TestWorkOrderStatus_FullForwardChainRecordsActor_FR039_FR044 walks every legal
// forward step the subcontractor can drive over HTTP (accepted -> production ->
// completed -> shipped), proves each one is accepted and the response reflects
// the new status, then reads the history rows directly to prove every human move
// is attributed to the acting subcontractor: changed_by is that account and
// by_system is false. The read query does not expose those columns, so the actor
// assertion queries work_order_status_history straight from the pool. This is the
// counterpart to the auto-confirm test that proves the system closure writes a
// by_system row with no human actor: forward moves are the human's, closure is
// the system's, and neither is mislabeled.
func TestWorkOrderStatus_FullForwardChainRecordsActor_FR039_FR044(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_status_fullchain")
	handler := woRouter(h, httpx.RoleSubcontractor, h.subAcc)

	for _, step := range []sqlcgen.WorkOrderStatus{
		sqlcgen.WorkOrderStatusProduction,
		sqlcgen.WorkOrderStatusCompleted,
		sqlcgen.WorkOrderStatusShipped,
	} {
		req := httptest.NewRequest(http.MethodPost,
			"/api/work-orders/"+uuidString(h.workOrderID)+"/status",
			strings.NewReader(`{"new_status":"`+string(step)+`"}`))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("majukan ke %q: status %d, mau 200; body %s", step, rec.Code, rec.Body.String())
		}
		var body struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %q: %v", rec.Body.String(), err)
		}
		if body.Status != string(step) {
			t.Fatalf("status %q, mau %q", body.Status, step)
		}
	}

	// Every forward move must carry the acting subcontractor as changed_by and
	// leave by_system false. The opening 'accepted' row (written by accept) is
	// also the subcontractor's, so no forward row may be attributed to the system.
	rows, err := h.pool.Query(context.Background(),
		`SELECT new_status, changed_by, by_system
		   FROM work_order_status_history
		  WHERE work_order_id = $1 AND new_status <> 'accepted'
		  ORDER BY created_at, id`, h.workOrderID)
	if err != nil {
		t.Fatalf("baca riwayat: %v", err)
	}
	defer rows.Close()

	var seen int
	for rows.Next() {
		var status sqlcgen.WorkOrderStatus
		var changedBy pgtype.UUID
		var bySystem bool
		if err := rows.Scan(&status, &changedBy, &bySystem); err != nil {
			t.Fatalf("scan riwayat: %v", err)
		}
		if bySystem {
			t.Fatalf("langkah %q ditandai by_system; perpindahan maju adalah tindakan subkontraktor (FR-039)", status)
		}
		if changedBy != h.subAcc {
			t.Fatalf("langkah %q changed_by bukan subkontraktor yang memindahkan (FR-039)", status)
		}
		seen++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterasi riwayat: %v", err)
	}
	if seen != 3 {
		t.Fatalf("baris riwayat maju = %d, mau 3 (production, completed, shipped)", seen)
	}
}
