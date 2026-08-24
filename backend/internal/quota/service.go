package quota

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// createRequest sends one quota request to several candidate listings in a
// single action (FR-029). It validates the body, resolves the buyer's profile,
// and resolves every listing so an unknown or unpublished id is a 422 and a
// listing owned by the buyer is a 409 SELF_REQUEST before any insert (FR-083).
// The request row and its candidate rows are written in one transaction, with
// created_at and reply_due_at both taken from the injected Clock (FR-082, Rule
// 5); a request_received notification is enqueued per candidate inside that
// transaction so a queue failure rolls the whole thing back.
func (s *Service) createRequest(ctx context.Context, accountID pgtype.UUID, in requestInput) (requestView, error) {
	v, verr := validateRequestInput(in)
	if verr != nil {
		return requestView{}, verr
	}

	buyerProfile, err := s.queries().GetProfileIDByAccount(ctx, accountID)
	if err != nil {
		if isNoRows(err) {
			return requestView{}, errProfileMissing
		}
		return requestView{}, err
	}

	listingIDs, allValid := uniqueUUIDs(v.listingIDs)
	if !allValid {
		return requestView{}, &validationError{fields: []httpx.FieldError{
			{Field: "listing_ids", Message: "Ada id listing yang tidak sah."},
		}}
	}

	rows, err := s.queries().GetCandidateListings(ctx, listingIDs)
	if err != nil {
		return requestView{}, err
	}

	// Index the resolved listings so an unknown or unpublished id is a 422 and a
	// listing owned by the buyer is a 409 before any write (FR-083). The order
	// follows the buyer's deduplicated input so the response is deterministic.
	byListing := make(map[string]sqlcgen.GetCandidateListingsRow, len(rows))
	for _, row := range rows {
		byListing[uuidString(row.ListingID)] = row
	}

	resolved := make([]sqlcgen.GetCandidateListingsRow, 0, len(listingIDs))
	for _, id := range listingIDs {
		row, ok := byListing[uuidString(id)]
		if !ok || !row.Published {
			return requestView{}, &validationError{fields: []httpx.FieldError{
				{Field: "listing_ids", Message: "Ada listing yang tidak ditemukan atau belum tayang."},
			}}
		}
		if row.ProfileID == buyerProfile {
			return requestView{}, &conflictError{code: httpx.CodeSelfRequest, detail: "Kandidat tidak sah."}
		}
		resolved = append(resolved, row)
	}

	productItem, _ := parseUUID(v.productItem)
	now := s.clock.Now()
	replyDue := now.Add(replyWindowHours * time.Hour)

	note := pgtype.Text{}
	if v.note != nil {
		note = pgtype.Text{String: *v.note, Valid: true}
	}

	var reqRow sqlcgen.QuotaRequest
	candidates := make([]candidateView, 0, len(resolved))
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		reqRow, err = q.InsertQuotaRequest(ctx, sqlcgen.InsertQuotaRequestParams{
			BuyerID:       buyerProfile,
			ProductItemID: productItem,
			Quantity:      v.quantity,
			Material:      v.material,
			Deadline:      pgdate(v.deadline),
			Note:          note,
			ReplyDueAt:    tstz(replyDue),
			CreatedAt:     tstz(now),
		})
		if err != nil {
			return err
		}

		for _, row := range resolved {
			cand, err := q.InsertRequestCandidate(ctx, sqlcgen.InsertRequestCandidateParams{
				RequestID:       reqRow.ID,
				ListingID:       row.ListingID,
				SubcontractorID: row.ProfileID,
				UpdatedAt:       tstz(now),
			})
			if err != nil {
				return err
			}
			candidates = append(candidates, candidateView{
				CandidateID:  uuidString(cand.ID),
				ListingID:    uuidString(row.ListingID),
				ProfileID:    uuidString(row.ProfileID),
				BusinessName: row.BusinessName,
				Status:       string(cand.Status),
			})

			link := "/quota-requests/" + uuidString(reqRow.ID)
			if err := s.notifier.Enqueue(ctx, tx, row.AccountID,
				sqlcgen.EventTypeRequestReceived,
				"Permintaan kuota baru",
				"Ada pembeli yang mengirim permintaan kuota ke listing Anda.",
				&link); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return requestView{}, err
	}

	return viewOf(reqRow, candidates), nil
}

// listRequests returns one keyset page of the buyer's own requests, newest
// first (FR-030, FR-080). The cursor tuple is (created_at, id); the first page
// starts above every real row via sentinels. It fetches size+1 rows to decide
// has_next, then loads every page row's candidates in one query.
func (s *Service) listRequests(ctx context.Context, accountID pgtype.UUID, q listQuery) (listView, error) {
	buyerProfile, err := s.queries().GetProfileIDByAccount(ctx, accountID)
	if err != nil {
		if isNoRows(err) {
			return requestView{}.emptyList(), nil
		}
		return listView{}, err
	}

	params := sqlcgen.ListQuotaRequestsByBuyerParams{
		BuyerID: buyerProfile,
		Limit:   q.size + 1,
	}
	if q.cursor == nil {
		params.Column2 = tstz(maxTime())
		params.Column3 = maxUUIDValue()
	} else {
		params.Column2 = tstz(q.cursor.CreatedAt)
		cur, _ := parseUUID(q.cursor.Request)
		params.Column3 = cur
	}

	rows, err := s.queries().ListQuotaRequestsByBuyer(ctx, params)
	if err != nil {
		return listView{}, err
	}

	hasNext := int32(len(rows)) > q.size
	if hasNext {
		rows = rows[:q.size]
	}

	requestIDs := make([]pgtype.UUID, 0, len(rows))
	for _, row := range rows {
		requestIDs = append(requestIDs, row.ID)
	}

	candByRequest := map[string][]candidateView{}
	if len(requestIDs) > 0 {
		candRows, err := s.queries().ListCandidatesByRequests(ctx, requestIDs)
		if err != nil {
			return listView{}, err
		}
		for _, c := range candRows {
			rid := uuidString(c.RequestID)
			candByRequest[rid] = append(candByRequest[rid], candidateView{
				CandidateID:  uuidString(c.CandidateID),
				ListingID:    uuidString(c.ListingID),
				ProfileID:    uuidString(c.SubcontractorID),
				BusinessName: c.BusinessName,
				Status:       string(c.Status),
			})
		}
	}

	items := make([]requestView, 0, len(rows))
	for _, row := range rows {
		items = append(items, viewOf(row, candByRequest[uuidString(row.ID)]))
	}

	view := listView{Items: items, Pagination: pagination{HasNext: hasNext}}
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		tok := encodeCursor(cursor{
			CreatedAt: last.CreatedAt.Time,
			Request:   uuidString(last.ID),
		})
		view.Pagination.NextCursor = &tok
	}
	return view, nil
}

// viewOf assembles a requestView from a request row and its candidate views.
// expires_at is reply_due_at, the reply window the Clock computed (FR-082).
func viewOf(row sqlcgen.QuotaRequest, candidates []candidateView) requestView {
	if candidates == nil {
		candidates = []candidateView{}
	}
	var note *string
	if row.Note.Valid {
		n := row.Note.String
		note = &n
	}
	return requestView{
		RequestID:     uuidString(row.ID),
		ProductItemID: uuidString(row.ProductItemID),
		Quantity:      row.Quantity,
		Material:      row.Material,
		Deadline:      row.Deadline.Time.Format(dateLayout),
		Note:          note,
		Candidates:    candidates,
		CreatedAt:     row.CreatedAt.Time,
		ExpiresAt:     row.ReplyDueAt.Time,
	}
}

// emptyList is the zero page a buyer with no profile yet sees: no items, no next
// page. It exists so listRequests returns a well-formed body without a nil slice.
func (requestView) emptyList() listView {
	return listView{Items: []requestView{}, Pagination: pagination{HasNext: false}}
}

// maxTime is the created_at ceiling for the first list page: a timestamp above
// any real row so the strict "<" tuple comparison admits the whole set.
func maxTime() time.Time {
	return time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
}

// maxUUIDValue is the id ceiling paired with maxTime for the first page, the
// all-ones UUID that sorts above every generated id.
func maxUUIDValue() pgtype.UUID {
	var u pgtype.UUID
	for i := range u.Bytes {
		u.Bytes[i] = 0xff
	}
	u.Valid = true
	return u
}
