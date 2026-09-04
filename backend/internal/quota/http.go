package quota

import (
	"errors"
	"net/http"
	"time"

	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// dateLayout is the wire format for the deadline field, a bare calendar date.
const dateLayout = "2006-01-02"

// requestInput is the QuotaRequestCreate body for POST /quota-requests. One
// action targets several candidate listings (listing_ids, FR-029); the material
// is required so a candidate can judge the request. note is optional.
type requestInput struct {
	ListingIDs    []string `json:"listing_ids"`
	ProductItemID string   `json:"product_item_id"`
	Quantity      int32    `json:"quantity"`
	Material      string   `json:"material"`
	Deadline      string   `json:"deadline"`
	Note          *string  `json:"note"`
}

// candidateView is one candidate of a request as returned to the buyer: the
// candidate row id, its listing and owning profile, the business name, the
// candidate's own status, and the rejection reason when it was rejected (FR-029,
// FR-035). rejection_reason is null for any status other than rejected.
type candidateView struct {
	CandidateID     string  `json:"candidate_id"`
	ListingID       string  `json:"listing_id"`
	ProfileID       string  `json:"profile_id"`
	BusinessName    string  `json:"business_name"`
	Status          string  `json:"status"`
	RejectionReason *string `json:"rejection_reason"`
}

// requestView is the QuotaRequest response body. expires_at is the reply window
// deadline (reply_due_at), computed from the injected Clock (FR-082). note is
// nullable.
type requestView struct {
	RequestID     string          `json:"request_id"`
	ProductItemID string          `json:"product_item_id"`
	Quantity      int32           `json:"quantity"`
	Material      string          `json:"material"`
	Deadline      string          `json:"deadline"`
	Note          *string         `json:"note"`
	Candidates    []candidateView `json:"candidates"`
	CreatedAt     time.Time       `json:"created_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
}

// pagination is the keyset page marker. next_cursor is opaque and nil when there
// is no next page (FR-080).
type pagination struct {
	HasNext    bool    `json:"has_next"`
	NextCursor *string `json:"next_cursor"`
}

// listView is the paginated list of the buyer's own requests (FR-030).
type listView struct {
	Items      []requestView `json:"items"`
	Pagination pagination    `json:"pagination"`
}

// Register wires the two buyer-only quota routes. Both are Gated behind
// RoleBuyer, so each stays out of the router's uncovered set and a wrong role is
// rejected before the handler runs. The Register(r, auth) shape mirrors the
// listing and search services, so Service holds no Authenticator.
func (s *Service) Register(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleBuyer)
	r.Gated("POST /api/quota-requests", gate, s.handleCreate)
	r.Gated("GET /api/quota-requests", gate, s.handleList)

	subGate := httpx.RequireRole(auth, httpx.RoleSubcontractor)
	r.Gated("POST /api/candidates/{candidateId}/offers", subGate, s.handleOffer)
	r.Gated("POST /api/candidates/{candidateId}/reject", subGate, s.handleReject)

	// A counter-offer can come from either party (FR-033), so the route admits
	// both roles; counterOffer checks the caller is a party to the negotiation
	// and alternates with the last proposer.
	counterGate := httpx.RequireRole(auth, httpx.RoleSubcontractor, httpx.RoleBuyer)
	r.Gated("POST /api/offers/{offerId}/counter", counterGate, s.handleCounter)

	// The incoming list is the subcontractor's side of FR-030; it must be
	// registered before the {requestId} detail route so the literal path wins.
	r.Gated("GET /api/quota-requests/incoming", subGate, s.handleIncoming)
	r.Gated("GET /api/candidates/{candidateId}", subGate, s.handleIncomingDetail)
	r.Gated("GET /api/quota-requests/{requestId}", gate, s.handleDetail)
}

// handleDetail returns one of the buyer's own requests with every candidate and
// its latest offer (FR-030, FR-032). The route is buyer-gated; the service loads
// the request under a buyer-account guard so another buyer's id is a 404.
func (s *Service) handleDetail(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	requestID, ok := parseUUID(r.PathValue("requestId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id permintaan tidak sah.")
		return
	}
	view, err := s.requestDetail(r.Context(), acc.ID, requestID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// handleIncoming returns one keyset page of the subcontractor's incoming
// candidates (FR-030), optionally filtered by candidate status (FR-031). The
// route is gated behind RoleSubcontractor.
func (s *Service) handleIncoming(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	q, verr := parseIncomingQuery(r)
	if verr != nil {
		httpx.WriteValidation(w, "Masukan tidak sah.", verr.fields)
		return
	}
	view, err := s.listIncoming(r.Context(), acc.ID, q)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// handleIncomingDetail returns one incoming candidate for its subcontractor
// owner. Unlike the paginated list, this endpoint is usable from a copied link
// or after a browser refresh because it does not depend on a client-side cache.
func (s *Service) handleIncomingDetail(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	candidateID, ok := parseUUID(r.PathValue("candidateId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id kandidat tidak sah.")
		return
	}
	view, err := s.incomingDetail(r.Context(), acc.ID, candidateID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// handleCounter chains a counter-offer onto an existing offer (FR-033). The
// route admits either party; the service checks the caller is a party to the
// negotiation and did not make the last offer, then records a new offer row.
func (s *Service) handleCounter(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	offerID, ok := parseUUID(r.PathValue("offerId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id penawaran tidak sah.")
		return
	}
	var in counterInput
	if !decodeJSON(w, r, &in) {
		return
	}
	view, err := s.counterOffer(r.Context(), acc.ID, offerID, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// handleReject declines a candidate on behalf of its subcontractor (FR-031).
// The route is gated behind RoleSubcontractor; the service also checks the
// caller owns the candidate's listing. On success it returns 204 with no body;
// the buyer sees the outcome and reason on the request detail page (FR-030).
func (s *Service) handleReject(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	candidateID, ok := parseUUID(r.PathValue("candidateId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id kandidat tidak sah.")
		return
	}
	var in rejectInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := s.rejectCandidate(r.Context(), acc.ID, candidateID, in); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleOffer records a subcontractor's reply to a candidate (FR-031). The route
// is gated behind RoleSubcontractor; the service also checks the caller owns the
// candidate's listing so one subcontractor cannot reply for another.
func (s *Service) handleOffer(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	candidateID, ok := parseUUID(r.PathValue("candidateId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id kandidat tidak sah.")
		return
	}
	var in offerInput
	if !decodeJSON(w, r, &in) {
		return
	}
	view, err := s.createOffer(r.Context(), acc.ID, candidateID, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (s *Service) handleCreate(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	var in requestInput
	if !decodeJSON(w, r, &in) {
		return
	}
	view, err := s.createRequest(r.Context(), acc.ID, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (s *Service) handleList(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	q, verr := parseListQuery(r)
	if verr != nil {
		httpx.WriteValidation(w, "Masukan tidak sah.", verr.fields)
		return
	}
	view, err := s.listRequests(r.Context(), acc.ID, q)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// listQuery holds the validated inputs of one list page: an optional opaque
// cursor and a page size 1..50 defaulting to 20.
type listQuery struct {
	cursor *cursor
	size   int32
}

// parseListQuery validates the query string for GET /quota-requests: cursor
// opaque, size 1..50 default 20. A garbled cursor or an out-of-range size is a
// 422 naming its field.
func parseListQuery(r *http.Request) (listQuery, *validationError) {
	qv := r.URL.Query()
	var fields []httpx.FieldError
	var q listQuery

	if raw := qv.Get("cursor"); raw != "" {
		c, ok := decodeCursor(raw)
		if !ok {
			fields = append(fields, httpx.FieldError{Field: "cursor", Message: "Kursor paginasi tidak sah."})
		} else {
			q.cursor = &c
		}
	}

	if n, ok := atoiDefault(qv.Get("size"), 20); !ok || n < 1 || n > 50 {
		fields = append(fields, httpx.FieldError{Field: "size", Message: "Ukuran halaman harus antara 1 dan 50."})
	} else {
		q.size = int32(n)
	}

	if len(fields) > 0 {
		return listQuery{}, &validationError{fields: fields}
	}
	return q, nil
}

// validatedInput is a requestInput whose fields have been checked and parsed:
// the deadline as a time and the product id as a UUID. The service uses it
// without re-parsing.
type validatedInput struct {
	listingIDs  []string
	productItem string
	quantity    int32
	material    string
	deadline    time.Time
	note        *string
}

// validateRequestInput enforces the QuotaRequestCreate field rules (all 422 with
// a field name): at least one listing id and every id a valid UUID, product id
// required uuid, quantity >= 1, material non-empty, deadline required date and
// not earlier than the current Jakarta date.
func (s *Service) validateRequestInput(in requestInput) (validatedInput, *validationError) {
	var fields []httpx.FieldError
	var v validatedInput
	now := s.clock.Now().In(platform.Jakarta)
	currentDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, platform.Jakarta)

	if len(in.ListingIDs) == 0 {
		fields = append(fields, httpx.FieldError{Field: "listing_ids", Message: "Pilih minimal satu kandidat."})
	} else {
		for _, id := range in.ListingIDs {
			if _, ok := parseUUID(id); !ok {
				fields = append(fields, httpx.FieldError{Field: "listing_ids", Message: "Ada id listing yang tidak sah."})
				break
			}
		}
	}
	v.listingIDs = in.ListingIDs

	if in.ProductItemID == "" {
		fields = append(fields, httpx.FieldError{Field: "product_item_id", Message: "Pilih produk yang dipesan."})
	} else if _, ok := parseUUID(in.ProductItemID); !ok {
		fields = append(fields, httpx.FieldError{Field: "product_item_id", Message: "Id produk tidak sah."})
	} else {
		v.productItem = in.ProductItemID
	}

	if in.Quantity < 1 {
		fields = append(fields, httpx.FieldError{Field: "quantity", Message: "Jumlah minimal 1 potong."})
	} else {
		v.quantity = in.Quantity
	}

	if len(in.Material) == 0 {
		fields = append(fields, httpx.FieldError{Field: "material", Message: "Isi bahan yang diminta."})
	} else {
		v.material = in.Material
	}

	if in.Deadline == "" {
		fields = append(fields, httpx.FieldError{Field: "deadline", Message: "Isi tenggat pesanan."})
	} else if t, err := platform.ParseDate(in.Deadline); err != nil {
		fields = append(fields, httpx.FieldError{Field: "deadline", Message: "Tenggat harus berformat YYYY-MM-DD."})
	} else if t.Before(currentDate) {
		fields = append(fields, httpx.FieldError{Field: "deadline", Message: "Tenggat tidak boleh berada di masa lampau."})
	} else {
		v.deadline = t
	}

	v.note = in.Note

	if len(fields) > 0 {
		return validatedInput{}, &validationError{fields: fields}
	}
	return v, nil
}

// writeErr maps a service error to its problem response. A validationError
// renders per field; a conflictError carries its own code and detail; a missing
// profile is an invariant break and becomes a 500; anything else is a 500.
func writeErr(w http.ResponseWriter, err error) {
	var verr *validationError
	if errors.As(err, &verr) {
		httpx.WriteValidation(w, "Masukan tidak sah.", verr.fields)
		return
	}
	var cerr *conflictError
	if errors.As(err, &cerr) {
		httpx.WriteProblem(w, cerr.code, cerr.detail)
		return
	}
	var merr *metaError
	if errors.As(err, &merr) {
		httpx.WriteProblemMeta(w, merr.code, merr.detail, merr.meta)
		return
	}
	httpx.WriteInternal(w)
}
