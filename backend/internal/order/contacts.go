package order

import (
	"net/http"

	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// contactParty is one side's off-platform contact: the business name, email, and
// WhatsApp number a party needs to arrange payment and coordination directly with
// the other (FR-040, FR-092). role marks which side of the order this is.
type contactParty struct {
	Role         string `json:"role"`
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	WhatsApp     string `json:"whatsapp"`
}

// workOrderContacts is the GET /contacts body: only the counterparty's block, so
// each party sees the other's contacts and never its own echoed back.
type workOrderContacts struct {
	Counterparty contactParty `json:"counterparty"`
}

// registerContacts wires the party-gated contact exchange route. Authenticated
// but not role-gated: the handler's party guard decides who may read, so a
// non-party sees a 404, not a 403.
func (s *Service) registerContacts(r *httpx.Router, auth httpx.Authenticator) {
	authed := httpx.RequireAuth(auth)
	r.Gated("GET /api/work-orders/{workOrderId}/contacts", authed, s.handleWorkOrderContacts)
}

// handleWorkOrderContacts returns the counterparty's contact block for one work
// order (FR-092). Once an order exists, payment and coordination happen directly
// between the two parties off-platform (FR-040), so each party must be able to
// reach the other. The party guard compares the caller's account id to the
// order's two parties: a non-party, a malformed id, or a missing order all
// collapse to the same 404, so the endpoint never leaks that an order exists to
// someone not on it. Unlike the detail read, an admin is NOT admitted here: the
// admin holds no business role (admin_has_no_business_role) and is never a
// transacting party, and reads the full order through FR-045/FR-046 rather than
// through the parties' private contacts.
func (s *Service) handleWorkOrderContacts(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(r.PathValue("workOrderId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}

	row, err := s.queries().GetWorkOrderContacts(r.Context(), id)
	if err != nil {
		if isNoRows(err) {
			httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
			return
		}
		httpx.WriteInternal(w)
		return
	}

	// The caller must be one of the two parties. The counterparty is the other
	// side. An admin (no business role) matches neither and falls through to 404.
	var counterparty contactParty
	switch acc.ID {
	case row.BuyerAccount:
		counterparty = contactParty{
			Role:         "subcontractor",
			BusinessName: row.SubcontractorBusinessName,
			Email:        row.SubcontractorEmail,
			WhatsApp:     row.SubcontractorPhone,
		}
	case row.SubcontractorAccount:
		counterparty = contactParty{
			Role:         "buyer",
			BusinessName: row.BuyerBusinessName,
			Email:        row.BuyerEmail,
			WhatsApp:     row.BuyerPhone,
		}
	default:
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}

	writeJSON(w, http.StatusOK, workOrderContacts{Counterparty: counterparty})
}
