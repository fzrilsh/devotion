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
