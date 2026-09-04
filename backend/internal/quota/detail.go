package quota

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// incomingQuery holds the validated inputs of one incoming page: an optional
// opaque cursor, a page size 1..50 defaulting to 20, and an optional
// candidate-status filter.
type incomingQuery struct {
	cursor *cursor
	size   int32
	status sqlcgen.NullCandidateStatus
}

// parseIncomingQuery validates the query string for GET /quota-requests/incoming:
// cursor opaque, size 1..50 default 20, status one of the candidate statuses when
// present. A garbled value is a 422 naming its field.
func parseIncomingQuery(r *http.Request) (incomingQuery, *validationError) {
	qv := r.URL.Query()
	var fields []httpx.FieldError
	var q incomingQuery

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

	if raw := qv.Get("status"); raw != "" {
		if !validCandidateStatus(raw) {
			fields = append(fields, httpx.FieldError{Field: "status", Message: "Status kandidat tidak dikenal."})
		} else {
			q.status = sqlcgen.NullCandidateStatus{CandidateStatus: sqlcgen.CandidateStatus(raw), Valid: true}
		}
	}

	if len(fields) > 0 {
		return incomingQuery{}, &validationError{fields: fields}
	}
	return q, nil
}

// validCandidateStatus reports whether raw is one of the candidate_status enum
// values, so an unknown filter is rejected before it reaches SQL.
func validCandidateStatus(raw string) bool {
	switch sqlcgen.CandidateStatus(raw) {
	case sqlcgen.CandidateStatusAwaitingReply,
		sqlcgen.CandidateStatusOffered,
		sqlcgen.CandidateStatusRejected,
		sqlcgen.CandidateStatusExpired,
		sqlcgen.CandidateStatusNotContinued,
		sqlcgen.CandidateStatusAgreed:
		return true
	}
	return false
}

// detailCandidateView is one candidate on the request-detail and incoming views:
// the shared candidate fields, the whole offer chain ordered by round, and the
// latest offer in that chain. The buyer sees every round side by side (FR-032)
// and compares each candidate's newest offer. offers is the full sequence-asc
// chain; latest_offer is its last element, omitted when the candidate has no
// reply yet. offers is omitted on the incoming list, which carries no chain.
type detailCandidateView struct {
	CandidateID     string      `json:"candidate_id"`
	ListingID       string      `json:"listing_id"`
	ProfileID       string      `json:"profile_id"`
	BusinessName    string      `json:"business_name"`
	Status          string      `json:"status"`
	RejectionReason *string     `json:"rejection_reason"`
	Offers          []offerView `json:"offers,omitempty"`
	LatestOffer     *offerView  `json:"latest_offer,omitempty"`
}

// detailView is the QuotaRequestDetail response: the request fields plus every
// candidate with its latest offer attached (FR-030, FR-032).
type detailView struct {
	RequestID     string                `json:"request_id"`
	ProductItemID string                `json:"product_item_id"`
	Quantity      int32                 `json:"quantity"`
	Material      string                `json:"material"`
	Deadline      string                `json:"deadline"`
	Note          *string               `json:"note"`
	Candidates    []detailCandidateView `json:"candidates"`
	CreatedAt     time.Time             `json:"created_at"`
	ExpiresAt     time.Time             `json:"expires_at"`
}

// incomingCandidateView is one candidate on the subcontractor's incoming list:
// the shared candidate fields plus what the subcontractor needs to judge the
// request before replying (FR-031, FR-035, FR-090), and the offer chain so a
// pending negotiation is answerable. quantity and deadline are the request's own;
// capacity_in_range is the subcontractor's remaining capacity summed across the
// readiness..deadline week range, and can_fulfill is whether that covers the
// requested quantity. The readiness lead is the listing's own
// (readiness_lead_days), so the marker reflects the earliest the subcontractor
// could start without an offer being sent yet. offers is the full sequence-asc
// chain and latest_offer its last element, so after the buyer counters the
// subcontractor sees the buyer's round on reload and can counter back (FR-032,
// FR-033); both are omitted when the candidate has no offer yet.
type incomingCandidateView struct {
	CandidateID     string      `json:"candidate_id"`
	ListingID       string      `json:"listing_id"`
	ProfileID       string      `json:"profile_id"`
	BusinessName    string      `json:"business_name"`
	Status          string      `json:"status"`
	RejectionReason *string     `json:"rejection_reason"`
	Quantity        int32       `json:"quantity"`
	Material        string      `json:"material"`
	Deadline        string      `json:"deadline"`
	Note            *string     `json:"note"`
	CapacityInRange int64       `json:"capacity_in_range"`
	CanFulfill      bool        `json:"can_fulfill"`
	Offers          []offerView `json:"offers,omitempty"`
	LatestOffer     *offerView  `json:"latest_offer,omitempty"`
}

// incomingView is the IncomingCandidateList response: one keyset page of the
// subcontractor's incoming candidates plus the page marker (FR-030).
type incomingView struct {
	Items      []incomingCandidateView `json:"items"`
	Pagination pagination              `json:"pagination"`
}

// requestDetail loads one of the buyer's own requests with every candidate and
// each candidate's latest offer (FR-030, FR-032). The buyer account guard makes
// a request that is not the caller's a 404 rather than leaking its existence.
func (s *Service) requestDetail(ctx context.Context, accountID, requestID pgtype.UUID) (detailView, error) {
	req, err := s.queries().GetRequestForBuyer(ctx, sqlcgen.GetRequestForBuyerParams{
		ID:        requestID,
		AccountID: accountID,
	})
	if err != nil {
		if isNoRows(err) {
			return detailView{}, &conflictError{code: httpx.CodeNotFound, detail: "Permintaan kuota tidak ditemukan."}
		}
		return detailView{}, err
	}

	candRows, err := s.queries().ListCandidatesByRequests(ctx, []pgtype.UUID{requestID})
	if err != nil {
		return detailView{}, err
	}

	offers, err := s.queries().ListOffersByRequest(ctx, requestID)
	if err != nil {
		return detailView{}, err
	}

	// Offers arrive ordered by candidate then sequence ascending. Group the whole
	// chain per candidate so the buyer sees every round (FR-032); the last row of
	// each chain is that candidate's latest round.
	chainByCandidate := map[string][]offerView{}
	for _, o := range offers {
		key := uuidString(o.CandidateID)
		chainByCandidate[key] = append(chainByCandidate[key], offerViewOf(o))
	}

	candidates := make([]detailCandidateView, 0, len(candRows))
	for _, c := range candRows {
		view := detailCandidateView{
			CandidateID:     uuidString(c.CandidateID),
			ListingID:       uuidString(c.ListingID),
			ProfileID:       uuidString(c.SubcontractorID),
			BusinessName:    c.BusinessName,
			Status:          string(c.Status),
			RejectionReason: textPtr(c.RejectionReason),
		}
		if chain, ok := chainByCandidate[uuidString(c.CandidateID)]; ok && len(chain) > 0 {
			view.Offers = chain
			last := chain[len(chain)-1]
			view.LatestOffer = &last
		}
		candidates = append(candidates, view)
	}

	var note *string
	if req.Note.Valid {
		n := req.Note.String
		note = &n
	}

	return detailView{
		RequestID:     uuidString(req.ID),
		ProductItemID: uuidString(req.ProductItemID),
		Quantity:      req.Quantity,
		Material:      req.Material,
		Deadline:      req.Deadline.Time.Format(dateLayout),
		Note:          note,
		Candidates:    candidates,
		CreatedAt:     req.CreatedAt.Time,
		ExpiresAt:     req.ReplyDueAt.Time,
	}, nil
}

// incomingDetail loads one candidate for the subcontractor that owns its listing.
// It returns the same fields as an incoming-list item, including capacity and the
// complete offer chain, so a direct browser navigation has no cache prerequisite.
func (s *Service) incomingDetail(ctx context.Context, accountID, candidateID pgtype.UUID) (incomingCandidateView, error) {
	c, err := s.queries().GetIncomingCandidate(ctx, sqlcgen.GetIncomingCandidateParams{
		ID:        candidateID,
		AccountID: accountID,
	})
	if err != nil {
		if isNoRows(err) {
			return incomingCandidateView{}, &conflictError{code: httpx.CodeNotFound, detail: "Kandidat tidak ditemukan."}
		}
		return incomingCandidateView{}, err
	}

	now := s.clock.Now()
	view := incomingCandidateView{
		CandidateID:     uuidString(c.CandidateID),
		ListingID:       uuidString(c.ListingID),
		ProfileID:       uuidString(c.SubcontractorID),
		BusinessName:    c.BusinessName,
		Status:          string(c.Status),
		RejectionReason: textPtr(c.RejectionReason),
		Quantity:        c.Quantity,
		Material:        c.Material,
		Deadline:        c.Deadline.Time.Format(dateLayout),
		Note:            textPtr(c.Note),
	}

	readinessWeek := platform.WeekStart(now.AddDate(0, 0, int(c.ReadinessLeadDays)))
	deadlineWeek := platform.WeekStart(c.Deadline.Time)
	if !readinessWeek.After(deadlineWeek) {
		remaining, err := s.queries().RemainingCapacityForOffer(ctx, sqlcgen.RemainingCapacityForOfferParams{
			ListingID:      c.ListingID,
			ReadinessWeek:  pgdate(readinessWeek),
			DeadlineWeek:   pgdate(deadlineWeek),
			WeeklyCapacity: c.WeeklyCapacity,
			HorizonUntil:   c.HorizonUntil,
		})
		if err != nil {
			return incomingCandidateView{}, err
		}
		view.CapacityInRange = remaining
		view.CanFulfill = int64(c.Quantity) <= remaining
	}

	offers, err := s.queries().ListOffersByCandidates(ctx, []pgtype.UUID{candidateID})
	if err != nil {
		return incomingCandidateView{}, err
	}
	if len(offers) > 0 {
		view.Offers = make([]offerView, 0, len(offers))
		for _, offer := range offers {
			view.Offers = append(view.Offers, offerViewOf(offer))
		}
		last := view.Offers[len(view.Offers)-1]
		view.LatestOffer = &last
	}
	return view, nil
}

// listIncoming returns one keyset page of candidates whose listing the
// subcontractor account owns, newest request first (FR-030). An optional status
// filter narrows to one candidate_status (FR-031). The cursor tuple is
// (created_at, id) of the request, matching the buyer-side list.
func (s *Service) listIncoming(ctx context.Context, accountID pgtype.UUID, q incomingQuery) (incomingView, error) {
	params := sqlcgen.ListIncomingCandidatesParams{
		AccountID: accountID,
		Limit:     q.size + 1,
		Status:    q.status,
	}
	if q.cursor == nil {
		params.Column2 = tstz(maxTime())
		params.Column3 = maxUUIDValue()
	} else {
		params.Column2 = tstz(q.cursor.CreatedAt)
		cur, _ := parseUUID(q.cursor.Request)
		params.Column3 = cur
	}

	rows, err := s.queries().ListIncomingCandidates(ctx, params)
	if err != nil {
		return incomingView{}, err
	}

	hasNext := int32(len(rows)) > q.size
	if hasNext {
		rows = rows[:q.size]
	}

	now := s.clock.Now()
	items := make([]incomingCandidateView, 0, len(rows))
	candidateIDs := make([]pgtype.UUID, 0, len(rows))
	for _, c := range rows {
		item := incomingCandidateView{
			CandidateID:     uuidString(c.CandidateID),
			ListingID:       uuidString(c.ListingID),
			ProfileID:       uuidString(c.SubcontractorID),
			BusinessName:    c.BusinessName,
			Status:          string(c.Status),
			RejectionReason: textPtr(c.RejectionReason),
			Quantity:        c.Quantity,
			Material:        c.Material,
			Deadline:        c.Deadline.Time.Format(dateLayout),
			Note:            textPtr(c.Note),
		}

		// Remaining capacity in the readiness..deadline window, keyed on the
		// listing's own readiness lead so the marker reflects the earliest the
		// subcontractor could start (FR-035, FR-090). Week rounding stays in
		// platform.WeekStart (Rule 4). If readiness runs past the deadline there
		// is no week to fill, so capacity is zero and can_fulfill is false.
		readinessWeek := platform.WeekStart(now.AddDate(0, 0, int(c.ReadinessLeadDays)))
		deadlineWeek := platform.WeekStart(c.Deadline.Time)
		if !readinessWeek.After(deadlineWeek) {
			remaining, err := s.queries().RemainingCapacityForOffer(ctx, sqlcgen.RemainingCapacityForOfferParams{
				ListingID:      c.ListingID,
				ReadinessWeek:  pgdate(readinessWeek),
				DeadlineWeek:   pgdate(deadlineWeek),
				WeeklyCapacity: c.WeeklyCapacity,
				HorizonUntil:   c.HorizonUntil,
			})
			if err != nil {
				return incomingView{}, err
			}
			item.CapacityInRange = remaining
			item.CanFulfill = int64(c.Quantity) <= remaining
		}

		items = append(items, item)
		candidateIDs = append(candidateIDs, c.CandidateID)
	}

	// Attach each candidate's offer chain so a subcontractor who reloads after the
	// buyer counters sees the buyer's round as latest_offer and can counter back
	// (FR-032, FR-033). Offers arrive ordered by candidate then sequence ascending;
	// the last row of each chain is that candidate's latest round.
	if len(candidateIDs) > 0 {
		offers, err := s.queries().ListOffersByCandidates(ctx, candidateIDs)
		if err != nil {
			return incomingView{}, err
		}
		chainByCandidate := map[string][]offerView{}
		for _, o := range offers {
			key := uuidString(o.CandidateID)
			chainByCandidate[key] = append(chainByCandidate[key], offerViewOf(o))
		}
		for i := range items {
			if chain, ok := chainByCandidate[items[i].CandidateID]; ok && len(chain) > 0 {
				items[i].Offers = chain
				last := chain[len(chain)-1]
				items[i].LatestOffer = &last
			}
		}
	}

	view := incomingView{Items: items, Pagination: pagination{HasNext: hasNext}}
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		tok := encodeCursor(cursor{
			CreatedAt: last.CreatedAt.Time,
			Request:   uuidString(last.RequestID),
		})
		view.Pagination.NextCursor = &tok
	}
	return view, nil
}
