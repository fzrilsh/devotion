package quota

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// counterInput is the OfferCounter body: a new whole-rupiah price and an
// optional note. Readiness carries forward from the prior offer, so the body
// omits it (FR-033). Money is int64 rupiah (Rule 3).
type counterInput struct {
	TotalPrice int64   `json:"total_price"`
	Note       *string `json:"note"`
}

// validateCounterInput enforces the OfferCounter rules (422 with a field name):
// total_price a whole rupiah of at least 1, note at most 1000 characters when
// present.
func validateCounterInput(in counterInput) *validationError {
	var fields []httpx.FieldError

	if in.TotalPrice < 1 {
		fields = append(fields, httpx.FieldError{Field: "total_price", Message: "Harga minimal 1 rupiah."})
	}
	if in.Note != nil && len([]rune(*in.Note)) > 1000 {
		fields = append(fields, httpx.FieldError{Field: "note", Message: "Catatan maksimal 1000 karakter."})
	}

	if len(fields) > 0 {
		return &validationError{fields: fields}
	}
	return nil
}

// counterOffer chains a counter-offer onto an existing offer (FR-033). It loads
// the latest offer in the candidate's chain, checks the caller is a party to the
// negotiation and alternates with the last proposer, refuses a candidate no
// longer open, then inserts a new offer row (never an update) carrying the prior
// readiness forward, and notifies the other side. Every round stays on record.
func (s *Service) counterOffer(ctx context.Context, accountID, offerID pgtype.UUID, in counterInput) (offerView, error) {
	if verr := validateCounterInput(in); verr != nil {
		return offerView{}, verr
	}

	prev, err := s.queries().GetOfferForCounter(ctx, offerID)
	if err != nil {
		if isNoRows(err) {
			return offerView{}, &conflictError{code: httpx.CodeNotFound, detail: "Penawaran tidak ditemukan."}
		}
		return offerView{}, err
	}

	isSub := prev.SubcontractorAccount == accountID
	isBuyer := prev.BuyerAccount == accountID
	if !isSub && !isBuyer {
		return offerView{}, &conflictError{code: httpx.CodeForbidden, detail: "Anda bukan pihak dalam negosiasi ini."}
	}

	// The counter must come from the side that did not make the last offer;
	// negotiation alternates (FR-033).
	var caller sqlcgen.OfferParty
	if isSub {
		caller = sqlcgen.OfferPartySubcontractor
	} else {
		caller = sqlcgen.OfferPartyBuyer
	}
	if caller == prev.ProposedBy {
		return offerView{}, &conflictError{
			code:   httpx.CodeForbidden,
			detail: "Menunggu balasan pihak lain sebelum Anda bisa menawar lagi.",
		}
	}

	if prev.CandidateStatus != sqlcgen.CandidateStatusOffered {
		return offerView{}, &conflictError{
			code:   httpx.CodeRequestAlreadyAgreed,
			detail: "Negosiasi kandidat ini sudah tidak terbuka.",
		}
	}

	note := pgtype.Text{}
	if in.Note != nil {
		note = pgtype.Text{String: *in.Note, Valid: true}
	}

	now := s.clock.Now()

	target := prev.SubcontractorAccount
	if caller == sqlcgen.OfferPartySubcontractor {
		target = prev.BuyerAccount
	}

	var offer sqlcgen.Offer
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		offer, err = q.InsertOffer(ctx, sqlcgen.InsertOfferParams{
			CandidateID:       prev.CandidateID,
			ProposedBy:        caller,
			TotalPrice:        in.TotalPrice,
			ReadinessLeadDays: prev.ReadinessLeadDays,
			Note:              note,
			CreatedAt:         tstz(now),
		})
		if err != nil {
			return err
		}

		link := "/quota-requests/" + uuidString(prev.RequestID)
		if caller == sqlcgen.OfferPartyBuyer {
			link = "/requests/incoming/" + uuidString(prev.CandidateID)
		}
		return s.notifier.Enqueue(ctx, tx, target,
			sqlcgen.EventTypeCounterOffer,
			"Penawaran balik",
			"Ada penawaran balik untuk negosiasi kuota Anda.",
			&link)
	})
	if err != nil {
		return offerView{}, err
	}

	return offerViewOf(offer), nil
}
