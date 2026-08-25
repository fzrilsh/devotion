package order

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// decodeCancelDetail parses the WorkOrderDetail fields the cancel tests assert on.
func decodeCancelDetail(t *testing.T, rec *httptest.ResponseRecorder) struct {
	Status string `json:"status"`
} {
	t.Helper()
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderDetail %q: %v", rec.Body.String(), err)
	}
	return body
}

// cancelReq drives POST /cancel over the wired router with the given reason body.
func cancelReq(handler http.Handler, workOrderID, reason string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost,
		"/api/work-orders/"+workOrderID+"/cancel",
		strings.NewReader(`{"reason":`+jsonString(reason)+`}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// jsonString quotes a string as a JSON literal for a small inline body.
func jsonString(s string) string {
	b := strings.Builder{}
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// TestWorkOrderCancel_BuyerCancelsPreProduction_FR020_FR065 proves a party can
// cancel while the order is still 'accepted': the status flips to cancelled and
// the allocation the agreement took is reversed, so the period's used capacity
// returns to zero (FR-020). The reason rides the cancellation (FR-065).
func TestWorkOrderCancel_BuyerCancelsPreProduction_FR020_FR065(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_cancel_ok")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	rec := cancelReq(handler, uuidString(h.workOrderID), "Pembeli membatalkan pesanan ini.")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	body := decodeCancelDetail(t, rec)
	if body.Status != string(sqlcgen.WorkOrderStatusCancelled) {
		t.Fatalf("status %q, mau %q", body.Status, sqlcgen.WorkOrderStatusCancelled)
	}

	// FR-020: every allocation reversed, so no live allocation remains and the
	// period's used capacity is back to zero.
	allocs, err := h.svc.queries().ListActiveAllocationsForReversal(context.Background(), h.workOrderID)
	if err != nil {
		t.Fatalf("list allocations: %v", err)
	}
	if len(allocs) != 0 {
		t.Fatalf("masih ada %d alokasi hidup setelah pembatalan, mau 0", len(allocs))
	}
}

// TestWorkOrderCancel_RejectsStranger_FR038 proves a caller who is on neither
// side of the order is turned away with the same 404 as a missing order, so the
// endpoint never confirms the order exists to a non-party. This is FR-038 (an
// order is shown only to its parties), not a role restriction: the stranger here
// still holds a business role, they just are not a party to this order.
func TestWorkOrderCancel_RejectsStranger_FR038(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_cancel_stranger")
	stranger := seedAcceptAccount(t, h.pool, "stranger_cancel@contoh.test", false)
	handler := woRouter(h, httpx.RoleBuyer, stranger)

	rec := cancelReq(handler, uuidString(h.workOrderID), "Bukan pihak pada pesanan ini.")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404; body %s", rec.Code, rec.Body.String())
	}
}

// TestWorkOrderCancel_RejectsAdminRole_FR005 proves the route's role gate turns
// away an admin with 403 before the handler runs. Cancellation is a business
// action of the two parties (FR-065); an admin has no business_profile, so
// letting the request reach the handler would fault on the profile lookup rather
// than reject cleanly. The gate is RequireRole(RoleBuyer, RoleSubcontractor), so
// the admin never reaches it.
func TestWorkOrderCancel_RejectsAdminRole_FR005(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_cancel_admin")
	admin := seedAcceptAccount(t, h.pool, "admin_cancel@contoh.test", true)
	handler := woRouter(h, httpx.RoleAdmin, admin)

	rec := cancelReq(handler, uuidString(h.workOrderID), "Admin mencoba membatalkan.")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; body %s", rec.Code, rec.Body.String())
	}
}

// TestWorkOrderCancel_RejectsShortReason_FR065 proves a reason under the minimum
// length is invalid input (422 VALIDATION_FAILED), since the reason both parties
// read must be substantive.
func TestWorkOrderCancel_RejectsShortReason_FR065(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_cancel_short")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	rec := cancelReq(handler, uuidString(h.workOrderID), "no")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestWorkOrderCancel_RejectsAfterProduction_FR066 proves that once the order has
// left 'accepted' (here, moved to production by the subcontractor), self
// cancellation is no longer available: the caller gets CANCELLATION_AFTER_PRODUCTION
// and is pointed at the dispute path via the problem meta.
func TestWorkOrderCancel_RejectsAfterProduction_FR066(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_cancel_afterprod")

	// Advance to production through the real status endpoint.
	if _, err := h.svc.changeStatus(context.Background(), h.subAcc, h.workOrderID,
		sqlcgen.WorkOrderStatusProduction, nil); err != nil {
		t.Fatalf("maju ke produksi: %v", err)
	}

	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)
	rec := cancelReq(handler, uuidString(h.workOrderID), "Ingin membatalkan setelah produksi.")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d, mau 409; body %s", rec.Code, rec.Body.String())
	}
	p := decodeAcceptProblem(t, rec)
	if p.Code != string(httpx.CodeCancellationAfterProduction) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeCancellationAfterProduction)
	}
}
