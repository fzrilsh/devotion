package order

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// disputeReq drives POST /disputes over the wired router with the given body.
func disputeReq(handler http.Handler, workOrderID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost,
		"/api/work-orders/"+workOrderID+"/disputes",
		strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// decodeDisputeStatus parses the status a dispute test asserts on.
func decodeDisputeStatus(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderDetail %q: %v", rec.Body.String(), err)
	}
	return body.Status
}

// TestDispute_PartyOpensDispute_FR046 proves a party can report a dispute: the
// response is 201 and the work order keeps its current status (accepted here). It
// does not jump to in_mediation on report: the open dispute row is what stops the
// auto-confirm clock, and admin moves the order to in_mediation when mediating
// (T071, FR-046).
func TestDispute_PartyOpensDispute_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_disp_ok")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	rec := disputeReq(handler, uuidString(h.workOrderID),
		`{"report_body":"Barang tidak sesuai contoh yang disepakati."}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, mau 201; body %s", rec.Code, rec.Body.String())
	}
	if got := decodeDisputeStatus(t, rec); got != string(sqlcgen.WorkOrderStatusAccepted) {
		t.Fatalf("status %q, mau %q; laporan sengketa tidak memindahkan work_order (FR-046)",
			got, sqlcgen.WorkOrderStatusAccepted)
	}
}

// TestDispute_RejectsAdminRole_FR046 proves the route's role gate turns away a
// caller without a business role (admin) with 403 before the handler runs:
// reporting a dispute is a business action of the two parties (FR-046).
func TestDispute_RejectsAdminRole_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_disp_admin")
	admin := seedAcceptAccount(t, h.pool, "admin_disp@contoh.test", true)
	handler := woRouter(h, httpx.RoleAdmin, admin)

	rec := disputeReq(handler, uuidString(h.workOrderID),
		`{"report_body":"Barang tidak sesuai contoh yang disepakati."}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; body %s", rec.Code, rec.Body.String())
	}
}

// TestDispute_RejectsStranger_FR046 proves a caller holding a business role but
// on neither side of the order is turned away with the same 404 as a missing
// order, so the endpoint never confirms the order exists to a non-party.
func TestDispute_RejectsStranger_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_disp_stranger")
	stranger := seedAcceptAccount(t, h.pool, "stranger_disp@contoh.test", false)
	handler := woRouter(h, httpx.RoleBuyer, stranger)

	rec := disputeReq(handler, uuidString(h.workOrderID),
		`{"report_body":"Barang tidak sesuai contoh yang disepakati."}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404; body %s", rec.Code, rec.Body.String())
	}
}

// TestDispute_RejectsShortReport_FR046 proves too short a report body is invalid
// input (422 VALIDATION_FAILED): the report must be at least 10 characters so
// admin has something to mediate on.
func TestDispute_RejectsShortReport_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_disp_short")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	rec := disputeReq(handler, uuidString(h.workOrderID), `{"report_body":"buruk"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestDispute_SecondReportRejected_FR046 proves at most one dispute may be open
// per order: a second report while the first is still unresolved hits the
// idx_one_open_dispute index and comes back as a readable DISPUTE_ALREADY_OPEN
// 409, not a 500.
func TestDispute_SecondReportRejected_FR046(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_disp_dup")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	first := disputeReq(handler, uuidString(h.workOrderID),
		`{"report_body":"Barang tidak sesuai contoh yang disepakati."}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("laporan pertama status %d, mau 201; body %s", first.Code, first.Body.String())
	}

	second := disputeReq(handler, uuidString(h.workOrderID),
		`{"report_body":"Saya laporkan lagi karena belum ditangani."}`)
	if second.Code != http.StatusConflict {
		t.Fatalf("laporan kedua status %d, mau 409; body %s", second.Code, second.Body.String())
	}
	if p := decodeAcceptProblem(t, second); p.Code != string(httpx.CodeDisputeAlreadyOpen) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeDisputeAlreadyOpen)
	}
}

// TestDispute_StopsAutoConfirm_FR070 proves reporting a dispute through the
// endpoint stops the 7-day auto-confirm count: after the report the order still
// reads as shipped (the report does not move the work order), but the open dispute
// row makes the auto-confirm scan skip it, so even eight days on the ticker leaves
// it shipped rather than closing it (FR-070).
func TestDispute_StopsAutoConfirm_FR070(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_disp_stopac")
	shipWorkOrder(t, h)

	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)
	rec := disputeReq(handler, uuidString(h.workOrderID),
		`{"report_body":"Barang belum sampai, saya laporkan sengketa."}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("laporan sengketa status %d, mau 201; body %s", rec.Code, rec.Body.String())
	}

	// Past the 7-day window the ticker must not close a disputed order.
	h.clock.Advance(8 * 24 * time.Hour)
	if err := h.svc.runAutoConfirm(context.Background()); err != nil {
		t.Fatalf("runAutoConfirm: %v", err)
	}

	wo := loadWorkOrder(t, h)
	if wo.Status != sqlcgen.WorkOrderStatusShipped {
		t.Fatalf("status %q, mau %q; sengketa terbuka harus menghentikan konfirmasi otomatis tanpa memindahkan status (FR-070)",
			wo.Status, sqlcgen.WorkOrderStatusShipped)
	}
	if wo.AutoConfirmed {
		t.Fatal("auto_confirmed true pada pesanan bersengketa (FR-070)")
	}
}

// TestDispute_DetailReadsNotConfirmed_FR070 reads a disputed shipped order's
// detail back through the ROUTER, past the 7-day window, and proves the status
// the buyer sees is not 'confirmed'. This is the layer the divergence used to
// hide in: the lazy read path recomputed 'confirmed' from shipped_at and now
// alone, so the GET response flipped a disputed order to confirmed while the
// scheduler correctly left it shipped. Now GetWorkOrderForView carries
// has_open_dispute into IsAutoConfirmDue, so the endpoint agrees with the
// ticker: a disputed order still reads as shipped (FR-070).
func TestDispute_DetailReadsNotConfirmed_FR070(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_disp_detail")
	shipWorkOrder(t, h)

	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)
	rec := disputeReq(handler, uuidString(h.workOrderID),
		`{"report_body":"Barang belum sampai, saya laporkan sengketa."}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("laporan sengketa status %d, mau 201; body %s", rec.Code, rec.Body.String())
	}

	// Well past the deadline, the detail GET must not report the order confirmed.
	h.clock.Advance(8 * 24 * time.Hour)
	req := httptest.NewRequest(http.MethodGet,
		"/api/work-orders/"+uuidString(h.workOrderID), nil)
	detail := httptest.NewRecorder()
	handler.ServeHTTP(detail, req)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status %d, mau 200; body %s", detail.Code, detail.Body.String())
	}
	if got := decodeDisputeStatus(t, detail); got == string(sqlcgen.WorkOrderStatusConfirmed) {
		t.Fatalf("detail membaca %q pada pesanan bersengketa; lapisan baca dan penjadwal harus sepakat bukan 'confirmed' (FR-070)", got)
	}
	if got := decodeDisputeStatus(t, detail); got != string(sqlcgen.WorkOrderStatusShipped) {
		t.Fatalf("detail membaca %q, mau %q (FR-070)", got, sqlcgen.WorkOrderStatusShipped)
	}
}
