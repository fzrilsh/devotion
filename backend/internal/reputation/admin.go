package reputation

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// errAlreadyHidden signals a hide attempt on a review already taken down. It is
// raised inside the transaction after the row lock reveals hidden is already
// true, so a second admin does not overwrite the first admin's recorded reason
// and moment.
var errAlreadyHidden = errors.New("review already hidden")

// hideBody is the POST /admin/reviews/{reviewId}/hide request: the reason the
// admin took the review down. It is required (the hiding_complete CHECK enforces
// it too), and the handler rejects an empty or too-short one before the
// constraint fires so the admin reads a validation error, not a raw 500.
type hideBody struct {
	Reason string `json:"reason"`
}

// registerAdmin wires the review moderation surface (T069), gated to the admin
// role: hiding a review that violates the rules (FR-050). RequireRole rejects a
// non-admin before the handler runs.
func (s *Service) registerAdmin(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleAdmin)
	r.Gated("POST /api/admin/reviews/{reviewId}/hide", gate, s.handleHideReview)
}

// handleHideReview marks one review hidden with the admin's identity, the moment
// from the injected Clock, and the reason (FR-050). Hiding is the whole action:
// the public list and the average rating both already filter NOT hidden, so the
// review leaves both at once with no second rule to drift. The row is locked
// first so two admins deciding the same review serialize, and an empty or short
// reason is rejected in the handler before the hiding_complete CHECK would
// surface a 500.
func (s *Service) handleHideReview(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	reviewID, ok := parseUUID(r.PathValue("reviewId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id ulasan tidak sah.")
		return
	}
	var body hideBody
	if !decodeJSON(w, r, &body) {
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if n := len([]rune(reason)); n < 5 || n > 1000 {
		httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
			{Field: "reason", Message: "Alasan penyembunyian harus 5 sampai 1000 karakter."},
		})
		return
	}
	now := tstz(s.clock.Now())

	err := db.WithTx(r.Context(), s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		locked, err := q.LockReviewForHide(r.Context(), reviewID)
		if err != nil {
			return err
		}
		if locked.Hidden {
			return errAlreadyHidden
		}
		return q.HideReview(r.Context(), sqlcgen.HideReviewParams{
			ID:           reviewID,
			HiddenBy:     acc.ID,
			HiddenAt:     now,
			HiddenReason: pgtype.Text{String: reason, Valid: true},
		})
	})
	if err != nil {
		switch {
		case errors.Is(err, errAlreadyHidden):
			httpx.WriteProblem(w, httpx.CodeValidationFailed, "Ulasan ini sudah disembunyikan.")
		case isNoRows(err):
			httpx.WriteProblem(w, httpx.CodeNotFound, "Ulasan tidak ditemukan.")
		default:
			httpx.WriteInternal(w)
		}
		return
	}

	row, err := s.queries().GetReviewForResponse(r.Context(), reviewID)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, responseView(row))
}
