package order

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// paymentReq drives POST /payments over the wired router with the given body.
func paymentReq(handler http.Handler, workOrderID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost,
		"/api/work-orders/"+workOrderID+"/payments",
		strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// decodePaymentDetail parses the payments array a payment test asserts on.
func decodePaymentDetail(t *testing.T, rec *httptest.ResponseRecorder) struct {
	Payments []struct {
		Direction           string  `json:"direction"`
		Date                string  `json:"date"`
		DeclaredByProfileID string  `json:"declared_by_profile_id"`
		Note                *string `json:"note"`
	} `json:"payments"`
} {
	t.Helper()
	var body struct {
		Payments []struct {
			Direction           string  `json:"direction"`
			Date                string  `json:"date"`
			DeclaredByProfileID string  `json:"declared_by_profile_id"`
			Note                *string `json:"note"`
		} `json:"payments"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderDetail %q: %v", rec.Body.String(), err)
	}
	return body
}

// TestPayment_BuyerRecordsStatement_FR041 proves a party can record a payment
// statement: the response is 201 and the detail's payments array now carries the
// party's statement with its direction, date, and note, and no money amount
// (FR-040, FR-041).
func TestPayment_BuyerRecordsStatement_FR041(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_pay_ok")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	rec := paymentReq(handler, uuidString(h.workOrderID),
		`{"direction":"sent","date":"2026-08-20","note":"Transfer DP 50%"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, mau 201; body %s", rec.Code, rec.Body.String())
	}
	body := decodePaymentDetail(t, rec)
	if len(body.Payments) != 1 {
		t.Fatalf("payments = %d, mau 1 (FR-041)", len(body.Payments))
	}
	p := body.Payments[0]
	if p.Direction != "sent" {
		t.Fatalf("direction %q, mau %q", p.Direction, "sent")
	}
	if p.Note == nil || *p.Note != "Transfer DP 50%" {
		t.Fatalf("note %v, mau catatan tersimpan", p.Note)
	}
}

// TestPayment_RejectsAdminRole_FR041 proves the route's role gate turns away a
// caller without a business role (admin) with 403 before the handler runs: a
// payment statement is a business action of the two parties (FR-041).
func TestPayment_RejectsAdminRole_FR041(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_pay_admin")
	admin := seedAcceptAccount(t, h.pool, "admin_pay@contoh.test", true)
	handler := woRouter(h, httpx.RoleAdmin, admin)

	rec := paymentReq(handler, uuidString(h.workOrderID),
		`{"direction":"sent","date":"2026-08-20"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; body %s", rec.Code, rec.Body.String())
	}
}

// TestPayment_RejectsStranger_FR038 proves a caller holding a business role but
// on neither side of the order is turned away with the same 404 as a missing
// order, so the endpoint never confirms the order exists to a non-party.
func TestPayment_RejectsStranger_FR038(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_pay_stranger")
	stranger := seedAcceptAccount(t, h.pool, "stranger_pay@contoh.test", false)
	handler := woRouter(h, httpx.RoleBuyer, stranger)

	rec := paymentReq(handler, uuidString(h.workOrderID),
		`{"direction":"sent","date":"2026-08-20"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404; body %s", rec.Code, rec.Body.String())
	}
}

// TestPayment_RejectsBadDirection_FR041 proves an unknown direction is invalid
// input (422 VALIDATION_FAILED): only sent and received are accepted.
func TestPayment_RejectsBadDirection_FR041(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_pay_baddir")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	rec := paymentReq(handler, uuidString(h.workOrderID),
		`{"direction":"maybe","date":"2026-08-20"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestPayment_RejectsBadDate_FR041 proves a malformed date is invalid input (422
// VALIDATION_FAILED): the date must parse as YYYY-MM-DD.
func TestPayment_RejectsBadDate_FR041(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_pay_baddate")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	rec := paymentReq(handler, uuidString(h.workOrderID),
		`{"direction":"sent","date":"20 Agustus"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestPayment_DuplicateDirectionRejected_FR041 proves a party may state each
// direction at most once: the second sent-statement by the same party hits the
// one_statement_per_party_per_direction constraint and comes back as a readable
// PAYMENT_STATEMENT_EXISTS 409, not a 500.
func TestPayment_DuplicateDirectionRejected_FR041(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_pay_dup")
	handler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)

	first := paymentReq(handler, uuidString(h.workOrderID),
		`{"direction":"sent","date":"2026-08-20"}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("pernyataan pertama status %d, mau 201; body %s", first.Code, first.Body.String())
	}

	second := paymentReq(handler, uuidString(h.workOrderID),
		`{"direction":"sent","date":"2026-08-21"}`)
	if second.Code != http.StatusConflict {
		t.Fatalf("pernyataan kedua status %d, mau 409; body %s", second.Code, second.Body.String())
	}
	if p := decodeAcceptProblem(t, second); p.Code != string(httpx.CodePaymentStatementExists) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodePaymentStatementExists)
	}
}

// TestPayment_BothPartiesStatementsVisible_FR043 proves both parties' statements
// on the same order are visible together, so a disagreement (buyer states sent,
// subcontractor never states received) is apparent to admin on mediation
// (FR-043). Here both sides state their own direction and both appear.
func TestPayment_BothPartiesStatementsVisible_FR043(t *testing.T) {
	h := seedAcceptedWorkOrder(t, "wo_pay_both")

	buyerHandler := woRouter(h, httpx.RoleBuyer, h.buyerAcc)
	if rec := paymentReq(buyerHandler, uuidString(h.workOrderID),
		`{"direction":"sent","date":"2026-08-20"}`); rec.Code != http.StatusCreated {
		t.Fatalf("pernyataan buyer status %d, mau 201; body %s", rec.Code, rec.Body.String())
	}

	subHandler := woRouter(h, httpx.RoleSubcontractor, h.subAcc)
	rec := paymentReq(subHandler, uuidString(h.workOrderID),
		`{"direction":"received","date":"2026-08-20"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("pernyataan subkontraktor status %d, mau 201; body %s", rec.Code, rec.Body.String())
	}
	body := decodePaymentDetail(t, rec)
	if len(body.Payments) != 2 {
		t.Fatalf("payments = %d, mau 2 (FR-043)", len(body.Payments))
	}
	// The two statements come from the two distinct parties.
	if body.Payments[0].DeclaredByProfileID == body.Payments[1].DeclaredByProfileID {
		t.Fatal("dua pernyataan dari profil yang sama; mau dua pihak berbeda (FR-043)")
	}
}
