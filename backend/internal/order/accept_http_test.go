package order

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// mockAuth is a stand-in Authenticator so the accept HTTP tests exercise the
// route's role gate and handler without importing account. A nil principal is an
// absent session (401); otherwise the set roles and account flow to the handler.
type mockAuth struct {
	principal *httpx.Principal
}

func (m *mockAuth) Authenticate(_ *http.Request) (httpx.Principal, error) {
	if m.principal == nil {
		return httpx.Principal{}, httpx.ErrUnauthenticated
	}
	return *m.principal, nil
}

// newAcceptHTTPHandler wires the accept route behind its real gate with the
// given authenticator. It needs no database: both tests here reject before the
// service ever queries, so a zero-value Service is enough to Register the route.
func newAcceptHTTPHandler(auth httpx.Authenticator) http.Handler {
	r := httpx.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)))
	(&Service{}).Register(r, auth)
	return r.Handler()
}

// buyerPrincipal is a caller holding the buyer role, the one the accept route
// admits. account_id is arbitrary: these tests never reach the offer guard.
func buyerPrincipal() *httpx.Principal {
	return &httpx.Principal{Roles: httpx.RoleBuyer, Account: sqlcgen.UserAccount{}}
}

// decodeAcceptProblem reads the problem+json body into its code and detail.
func decodeAcceptProblem(t *testing.T, rec *httptest.ResponseRecorder) struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
} {
	t.Helper()
	var p struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("decode problem %q: %v", rec.Body.String(), err)
	}
	return p
}

// TestAccept_RejectsNonBusinessRole_FR005 proves the accept route's role gate
// turns away a caller holding no business role with 403 before the handler runs,
// so an admin session can never form an agreement on a party's behalf. Both
// business roles are admitted now (either party may close the negotiation,
// FR-033), so the gate is proven with a role outside that pair. The offer id is a
// syntactically valid UUID so the gate, not the id parse, is what rejects.
func TestAccept_RejectsNonBusinessRole_FR005(t *testing.T) {
	handler := newAcceptHTTPHandler(&mockAuth{principal: &httpx.Principal{
		Roles:   httpx.RoleAdmin,
		Account: sqlcgen.UserAccount{},
	}})

	req := httptest.NewRequest(http.MethodPost,
		"/api/offers/11111111-1111-1111-1111-111111111111/accept", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d, mau 403; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeForbidden) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeForbidden)
	}
}

// TestAccept_AdmitsSubcontractorRole_FR033 proves the route no longer locks the
// accept capability to the buyer. A subcontractor-only caller must clear the role
// gate and reach the service, where the offer id (a valid UUID pointing at no
// row) turns into a 404. Anything but 404 here means the gate rejected the role,
// which would leave the subcontractor unable to approve a buyer's counter (spec
// skenario 4).
func TestAccept_AdmitsSubcontractorRole_FR033(t *testing.T) {
	h := newAcceptHarness(t, "accept_admits_sub")

	r := httpx.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)))
	h.svc.Register(r, &mockAuth{principal: &httpx.Principal{
		Roles:   httpx.RoleSubcontractor,
		Account: sqlcgen.UserAccount{},
	}})

	req := httptest.NewRequest(http.MethodPost,
		"/api/offers/11111111-1111-1111-1111-111111111111/accept", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404 (peran lolos gate, penawaran tidak ada); body %s",
			rec.Code, rec.Body.String())
	}
}

// TestAccept_RejectsInvalidOfferID_ContractValidationFailed proves a malformed
// offer id is a 422 VALIDATION_FAILED, not a 404 or a 500. No FR governs this;
// what is enforced is the contract's 422 response for a syntactically invalid
// path parameter. The caller is a buyer so it clears the role gate; parseUUID
// then rejects the path value before any query.
func TestAccept_RejectsInvalidOfferID_ContractValidationFailed(t *testing.T) {
	handler := newAcceptHTTPHandler(&mockAuth{principal: buyerPrincipal()})

	req := httptest.NewRequest(http.MethodPost, "/api/offers/bukan-uuid/accept", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, mau 422; body %s", rec.Code, rec.Body.String())
	}
	if p := decodeAcceptProblem(t, rec); p.Code != string(httpx.CodeValidationFailed) {
		t.Fatalf("code %q, mau %q", p.Code, httpx.CodeValidationFailed)
	}
}

// TestAccept_RejectsUnauthenticated_FR005 proves an absent session is 401, not
// 403: a logged-out caller is told to sign in, not that they are forbidden.
func TestAccept_RejectsUnauthenticated_FR005(t *testing.T) {
	handler := newAcceptHTTPHandler(&mockAuth{principal: nil})

	req := httptest.NewRequest(http.MethodPost,
		"/api/offers/11111111-1111-1111-1111-111111111111/accept", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, mau 401; body %s", rec.Code, rec.Body.String())
	}
}

// TestAccept_FormsAgreement_FR034_FR036 is the one success path that runs the
// endpoint end to end: a real buyer POSTs a valid offer through the router, the
// gate, principalAccount, parseUUID, the R-04 transaction, and the 201 mapping.
// The four T042 tests call s.accept() directly, so nothing else proves the HTTP
// contract: that the route answers 201 Created and that WorkOrderDetail
// serializes. This is the first WorkOrderDetail to cross HTTP, so it asserts the
// fields the contract marks required and the two the frontend renders its state
// machine from (FR-039): allowed_transitions must be present and non-empty, and
// self_cancellable must be present. If either serializes wrong, T059 breaks later
// and more expensively.
func TestAccept_FormsAgreement_FR034_FR036(t *testing.T) {
	h := newAcceptHarness(t, "accept_http_success")
	week := platform.WeekStart(acceptBaseTime)

	listingID, subAcc := seedListing(t, h, "alfa", 100, week, week)
	subProf := subProfileID(t, h, subAcc)

	buyer := seedAcceptProfile(t, h.pool, seedAcceptAccount(t, h.pool, "buyer@contoh.test", false), "Pembeli")
	req := seedRequest(t, h, buyer, 50, week)
	offer := seedOfferedCandidate(t, h, req, listingID, subProf, 1_000_000)
	buyerAcc := buyerAccountOf(t, h, buyer)

	r := httpx.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)))
	h.svc.Register(r, &mockAuth{principal: &httpx.Principal{
		Roles:   httpx.RoleBuyer,
		Account: sqlcgen.UserAccount{ID: buyerAcc},
	}})
	handler := r.Handler()

	httpReq := httptest.NewRequest(http.MethodPost,
		"/api/offers/"+uuidString(offer)+"/accept", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, mau 201; body %s", rec.Code, rec.Body.String())
	}

	// Decode against the contract's field names (openapi.yaml WorkOrderDetail),
	// not guessed ones. allowed_transitions and self_cancellable are pointers so a
	// missing key is distinguishable from an empty or false value.
	var body struct {
		WorkOrderID        string    `json:"work_order_id"`
		Status             string    `json:"status"`
		AllowedTransitions *[]string `json:"allowed_transitions"`
		SelfCancellable    *bool     `json:"self_cancellable"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode WorkOrderDetail %q: %v", rec.Body.String(), err)
	}

	if body.WorkOrderID == "" {
		t.Fatal("work_order_id kosong, mau id pesanan yang terbentuk")
	}
	// The first status after an agreement forms is 'accepted', the head of
	// accepted -> production -> completed -> shipped -> confirmed (openapi.yaml
	// WorkOrderStatus).
	if body.Status != string(sqlcgen.WorkOrderStatusAccepted) {
		t.Fatalf("status %q, mau %q", body.Status, sqlcgen.WorkOrderStatusAccepted)
	}
	if body.AllowedTransitions == nil {
		t.Fatal("allowed_transitions tidak ada di respons; frontend merender tombol dari sini (FR-039)")
	}
	if len(*body.AllowedTransitions) == 0 {
		t.Fatal("allowed_transitions kosong, mau minimal satu transisi dari status accepted")
	}
	if body.SelfCancellable == nil {
		t.Fatal("self_cancellable tidak ada di respons (FR-066)")
	}
}
