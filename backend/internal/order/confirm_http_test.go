package order

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// confirmReq drives POST /confirm over the wired router. Confirmation carries no
// body, so the request is sent without one.
func confirmReq(handler http.Handler, workOrderID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost,
		"/api/work-orders/"+workOrderID+"/confirm", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestConfirm_BuyerConfirmsShipped_FR047_FR068 proves the buyer can manually
// confirm receipt of a shipped order: the status flips to confirmed, the closure
// is recorded as the buyer's (auto_confirmed=false, by_system=false in history),
// not the system's 7-day auto-confirm, and it happens well inside the window so
// the manual path is what closed it, not the ticker.
func TestConfirm_BuyerConfirmsShipped_FR047_FR068(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_confirm_ok")
	shipWorkOrder(t, h)

	// One day after shipment: far inside the 7-day window, so only the manual
	// confirmation can close the order here.
	h.clock.Advance(24 * time.Hour)

	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)
	rec := confirmReq(handler, uuidString(h.workOrderID))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200; body %s", rec.Code, rec.Body.String())
	}
	if body := decodeCancelDetail(t, rec); body.Status != string(sqlcgen.WorkOrderStatusConfirmed) {
		t.Fatalf("status %q, mau %q", body.Status, sqlcgen.WorkOrderStatusConfirmed)
	}

	// Stored as the buyer's manual closure, not the system's: auto_confirmed stays
	// false so the trail distinguishes it from a 7-day auto-confirm (FR-068).
	wo := loadWorkOrder(t, h)
	if wo.Status != sqlcgen.WorkOrderStatusConfirmed {
		t.Fatalf("status tersimpan %q, mau %q", wo.Status, sqlcgen.WorkOrderStatusConfirmed)
	}
	if wo.AutoConfirmed {
		t.Fatal("auto_confirmed true pada konfirmasi manual buyer; harus false (FR-068)")
	}

	// The closing history row records the human actor, not the system: the
	// read query does not expose by_system/changed_by, so read them raw. A
	// manual confirmation is the buyer's act (by_system=false, changed_by=buyer),
	// which distinguishes it from the 7-day auto-confirm (FR-039, FR-068).
	var bySystem bool
	var changedBy pgtype.UUID
	if err := h.pool.QueryRow(context.Background(),
		`SELECT by_system, changed_by
		   FROM work_order_status_history
		  WHERE work_order_id = $1 AND new_status = 'confirmed'
		  ORDER BY created_at DESC, id DESC
		  LIMIT 1`, h.workOrderID).Scan(&bySystem, &changedBy); err != nil {
		t.Fatalf("baca riwayat konfirmasi: %v", err)
	}
	if bySystem {
		t.Fatal("by_system true pada konfirmasi manual buyer; harus false (FR-039)")
	}
	if changedBy != h.buyerAcc {
		t.Fatal("changed_by bukan buyer yang mengonfirmasi (FR-039)")
	}
}

// TestConfirm_RejectsSubcontractorRole_FR047 proves the route's role gate turns
// away the subcontractor with 403: confirmation is the buyer's act (FR-047). The
// subcontractor advances the order up to 'shipped' and the buyer confirms receipt;
// the gate is RequireRole(RoleBuyer), so a subcontractor never reaches the handler.
func TestConfirm_RejectsSubcontractorRole_FR047(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_confirm_sub")
	shipWorkOrder(t, h)

	handler := woRouter(h, httpx.RoleSubcontractor, h.subAcc)
	rec := confirmReq(handler, uuidString(h.workOrderID))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; body %s", rec.Code, rec.Body.String())
	}
}

// TestConfirm_RejectsNonPartyBuyer_FR038 proves a buyer who is not this order's
// buyer is turned away with the same 404 as a missing order, so the endpoint never
// confirms the order exists to a non-party.
func TestConfirm_RejectsNonPartyBuyer_FR038(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_confirm_stranger")
	shipWorkOrder(t, h)
	stranger := seedAcceptAccount(t, h.pool, "stranger_confirm@contoh.test", false)

	handler := woRouter(h, httpx.RoleBuyer, stranger)
	rec := confirmReq(handler, uuidString(h.workOrderID))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404; body %s", rec.Code, rec.Body.String())
	}
}

// TestConfirm_RejectsNotShipped_FR047 proves confirming an order that is not yet
// shipped (here still 'accepted') is rejected with INVALID_STATUS_TRANSITION: the
// buyer can only confirm receipt once the maker declared shipment.
func TestConfirm_RejectsNotShipped_FR047(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_confirm_notshipped")

	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)
	rec := confirmReq(handler, uuidString(h.workOrderID))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d, mau 409; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeInvalidStatusTransition) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeInvalidStatusTransition)
	}
}

// TestConfirm_OpenDisputeDoesNotConfirm_FR070 proves an open dispute halts manual
// confirmation the same way it halts auto-confirm: the order stays shipped and the
// buyer's confirm attempt is rejected rather than closing the order out from under
// an unresolved dispute.
func TestConfirm_OpenDisputeDoesNotConfirm_FR070(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_confirm_disputed")
	shipWorkOrder(t, h)

	// The subcontractor opens a dispute while the order sits at 'shipped'.
	if _, err := h.svc.createDispute(context.Background(), h.subAcc, h.workOrderID,
		"Pembayaran belum diterima sesuai kesepakatan."); err != nil {
		t.Fatalf("buka sengketa: %v", err)
	}

	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)
	rec := confirmReq(handler, uuidString(h.workOrderID))
	if rec.Code == http.StatusOK {
		t.Fatalf("konfirmasi berhasil padahal sengketa terbuka; body %s", rec.Body.String())
	}

	// The order must remain shipped: the dispute row, not the confirm, decides.
	wo := loadWorkOrder(t, h)
	if wo.Status != sqlcgen.WorkOrderStatusShipped {
		t.Fatalf("status tersimpan %q, mau %q (sengketa menahan konfirmasi, FR-070)",
			wo.Status, sqlcgen.WorkOrderStatusShipped)
	}
}
