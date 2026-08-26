package reputation

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/order"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// reviewBody is the POST /reviews request: a 1..5 rating and an optional free
// text up to 2000 characters (FR-047).
type reviewBody struct {
	Rating int     `json:"rating"`
	Text   *string `json:"text"`
}

// reviewView is the Review body. Reviews are never anonymous: the author's
// profile id and business name always ride along, as does the transaction date
// of the order being reviewed (FR-049).
type reviewView struct {
	ReviewID           string    `json:"review_id"`
	WorkOrderID        string    `json:"work_order_id"`
	AuthorProfileID    string    `json:"author_profile_id"`
	AuthorBusinessName string    `json:"author_business_name"`
	TargetProfileID    string    `json:"target_profile_id"`
	Rating             int       `json:"rating"`
	Text               *string   `json:"text"`
	Hidden             bool      `json:"hidden"`
	TransactionDate    string    `json:"transaction_date"`
	CreatedAt          time.Time `json:"created_at"`
}

// reviewList is the ReviewList body: a page of reviews plus the keyset cursor.
type reviewList struct {
	Items      []reviewView `json:"items"`
	Pagination pagination   `json:"pagination"`
}

// handleCreateReview records the caller's review of the counterparty on one work
// order (FR-047, FR-049). The route admits both business roles; the handler
// validates the rating and the text length, then the service guards party
// membership, the confirmed-received precondition, and the one-per-party rule.
func (s *Service) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(r.PathValue("workOrderId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}
	var body reviewBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Rating < 1 || body.Rating > 5 {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Rating harus bilangan 1 sampai 5.")
		return
	}
	if body.Text != nil && len([]rune(*body.Text)) > 2000 {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Ulasan paling banyak 2000 karakter.")
		return
	}

	view, err := s.createReview(r.Context(), acc.ID, id, body.Rating, body.Text)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// createReview writes the review in one transaction (FR-047). It loads the order,
// guards that the caller is one of its two parties (a non-party, an unknown
// order, and a business the caller never transacted with all collapse to the same
// 404), then enforces the precondition the database cannot: the order must read
// as confirmed received. That check is the reason this cannot be a CHECK
// constraint, since it references work_order from review, so the application owns
// it (data-model.md).
//
// The confirmed test uses the same lazy auto-confirm the order pages use
// (order.IsAutoConfirmDue): an order past its 7-day window reads as confirmed
// there, so it must be reviewable here even before the ticker writes the row.
// Reusing the predicate rather than restating the arithmetic keeps the two from
// drifting apart (FR-068).
//
// One review per order per party: the application checks first for a readable
// 409, and the one_review_per_order_per_reviewer constraint closes the race
// between two concurrent submissions.
func (s *Service) createReview(ctx context.Context, accountID, workOrderID pgtype.UUID, rating int, text *string) (reviewView, error) {
	now := s.clock.Now()
	var reviewID pgtype.UUID
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		wo, err := q.GetWorkOrderForView(ctx, workOrderID)
		if err != nil {
			if isNoRows(err) {
				return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
			}
			return err
		}
		if wo.BuyerAccount != accountID && wo.SubcontractorAccount != accountID {
			return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
		}

		if !confirmedReceived(wo, now) {
			return &conflictError{
				code:   httpx.CodeWorkOrderNotCompleted,
				detail: "Ulasan hanya dapat diberikan setelah pesanan dikonfirmasi diterima.",
			}
		}

		// The reviewer is the caller's side of the order, the reviewee the other.
		reviewerID, revieweeID := wo.BuyerID, wo.SubcontractorID
		if accountID == wo.SubcontractorAccount {
			reviewerID, revieweeID = wo.SubcontractorID, wo.BuyerID
		}

		var textCol pgtype.Text
		if text != nil {
			textCol = pgtype.Text{String: *text, Valid: true}
		}
		row, err := q.InsertReview(ctx, sqlcgen.InsertReviewParams{
			WorkOrderID: workOrderID,
			ReviewerID:  reviewerID,
			RevieweeID:  revieweeID,
			Rating:      int16(rating),
			Text:        textCol,
			CreatedAt:   tstz(now),
		})
		if err != nil {
			if isReviewDuplicate(err) {
				return &conflictError{
					code:   httpx.CodeReviewAlreadySubmitted,
					detail: "Anda sudah memberi ulasan pada pesanan ini.",
				}
			}
			return err
		}
		reviewID = row.ID
		return nil
	})
	if err != nil {
		return reviewView{}, err
	}

	row, err := s.queries().GetReviewForResponse(ctx, reviewID)
	if err != nil {
		return reviewView{}, err
	}
	return responseView(row), nil
}

// confirmedReceived reports whether an order counts as confirmed received, the
// precondition a review requires (FR-047). A stored 'confirmed' status is the
// manual case; a shipped order past its auto-confirm instant is the lazy case,
// decided by order.IsAutoConfirmDue with the same open-dispute halt the order
// pages and the ticker use, so a disputed order is not reviewable while its
// count is stopped (FR-068, FR-070).
func confirmedReceived(wo sqlcgen.GetWorkOrderForViewRow, now time.Time) bool {
	if wo.Status == sqlcgen.WorkOrderStatusConfirmed {
		return true
	}
	return wo.Status == sqlcgen.WorkOrderStatusShipped && wo.ShippedAt.Valid &&
		order.IsAutoConfirmDue(wo.ShippedAt.Time, now, wo.HasOpenDispute)
}

// handleListReviews returns one profile's received reviews, newest first, one
// keyset page at a time (FR-048). The route is public: a prospective buyer reads
// a subcontractor's reputation before signing in. Hidden reviews are excluded by
// the query, matching what the average rating counts (FR-050). An unparseable
// cursor falls back to the first page; an unknown profile is an empty page, not a
// 404, so the endpoint does not enumerate which profile ids exist.
func (s *Service) handleListReviews(w http.ResponseWriter, r *http.Request) {
	profileID, ok := parseUUID(r.PathValue("profileId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Profil tidak ditemukan.")
		return
	}

	params := sqlcgen.ListReviewsForProfileParams{
		RevieweeID: profileID,
		PageLimit:  reviewPageLimit + 1, // one extra row detects a next page
	}
	if cur, ok := decodeCursor(r.URL.Query().Get("cursor")); ok {
		params.BeforeCreated = cur.created
		params.BeforeID = cur.id
	}

	rows, err := s.queries().ListReviewsForProfile(r.Context(), params)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}

	hasNext := len(rows) > reviewPageLimit
	if hasNext {
		rows = rows[:reviewPageLimit]
	}

	items := make([]reviewView, 0, len(rows))
	for _, row := range rows {
		items = append(items, listView(row))
	}

	page := pagination{HasNext: hasNext}
	if hasNext {
		last := rows[len(rows)-1]
		c := encodeCursor(cursor{created: last.CreatedAt, id: last.ID})
		page.NextCursor = &c
	}
	writeJSON(w, http.StatusOK, reviewList{Items: items, Pagination: page})
}

// responseView renders the freshly inserted review for the 201 body.
func responseView(row sqlcgen.GetReviewForResponseRow) reviewView {
	return reviewView{
		ReviewID:           uuidString(row.ID),
		WorkOrderID:        uuidString(row.WorkOrderID),
		AuthorProfileID:    uuidString(row.ReviewerID),
		AuthorBusinessName: row.AuthorBusinessName,
		TargetProfileID:    uuidString(row.RevieweeID),
		Rating:             int(row.Rating),
		Text:               textOrNil(row.Text),
		Hidden:             row.Hidden,
		TransactionDate:    platform.FormatDateID(row.TransactionDate.Time),
		CreatedAt:          row.CreatedAt.Time,
	}
}

// listView renders one row of the public review list. It is the same shape as
// responseView over the list row type, so the 201 body and the list entry for
// the same review are identical.
func listView(row sqlcgen.ListReviewsForProfileRow) reviewView {
	return reviewView{
		ReviewID:           uuidString(row.ID),
		WorkOrderID:        uuidString(row.WorkOrderID),
		AuthorProfileID:    uuidString(row.ReviewerID),
		AuthorBusinessName: row.AuthorBusinessName,
		TargetProfileID:    uuidString(row.RevieweeID),
		Rating:             int(row.Rating),
		Text:               textOrNil(row.Text),
		Hidden:             row.Hidden,
		TransactionDate:    platform.FormatDateID(row.TransactionDate.Time),
		CreatedAt:          row.CreatedAt.Time,
	}
}

// textOrNil converts a nullable text column to a pointer, so an absent review
// text serializes as JSON null rather than an empty string.
func textOrNil(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	v := t.String
	return &v
}
