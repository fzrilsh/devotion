package quota

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// rejectInput is the OfferReject body: a reason a buyer can read on the request
// detail page (FR-030). The reason is required, 5..500 characters.
type rejectInput struct {
	Reason string `json:"reason"`
}

// validateRejectInput enforces the OfferReject rule (422 with a field name):
// the reason is 5..500 characters after trimming is not applied, matching the
// contract's minLength/maxLength on the raw string.
func validateRejectInput(in rejectInput) *validationError {
	n := len([]rune(in.Reason))
	if n < 5 || n > 500 {
		return &validationError{fields: []httpx.FieldError{
			{Field: "reason", Message: "Alasan penolakan antara 5 dan 500 karakter."},
		}}
	}
	return nil
}

// rejectCandidate declines a candidate on behalf of its subcontractor (FR-031).
// It loads the candidate, verifies the caller owns its listing, then marks the
// candidate rejected with the reason. The buyer sees the outcome and reason on
// the request detail page (FR-030); FR-051 lists no rejection notification, so
// none is sent.
func (s *Service) rejectCandidate(ctx context.Context, accountID, candidateID pgtype.UUID, in rejectInput) error {
	if verr := validateRejectInput(in); verr != nil {
		return verr
	}

	cand, err := s.queries().GetCandidateForOffer(ctx, candidateID)
	if err != nil {
		if isNoRows(err) {
			return &conflictError{code: httpx.CodeNotFound, detail: "Kandidat tidak ditemukan."}
		}
		return err
	}

	if cand.SubcontractorAccount != accountID {
		return &conflictError{code: httpx.CodeForbidden, detail: "Anda bukan pemilik listing kandidat ini."}
	}

	now := s.clock.Now()
	return s.queries().RejectCandidate(ctx, sqlcgen.RejectCandidateParams{
		ID:              candidateID,
		RejectionReason: pgtype.Text{String: in.Reason, Valid: true},
		UpdatedAt:       tstz(now),
	})
}
