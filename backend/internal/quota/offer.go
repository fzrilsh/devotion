package quota

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// offerInput is the OfferCreate body a subcontractor sends to reply to a
// candidate (FR-031): a whole-rupiah total price, a readiness lead in days, and
// an optional note. Money is int64 rupiah (Rule 3); there is no float field.
type offerInput struct {
	TotalPrice        int64   `json:"total_price"`
	ReadinessLeadDays int32   `json:"readiness_lead_days"`
	Note              *string `json:"note"`
}

// offerView is the Offer response body. party is the proposing side
// (subcontractor or buyer); note is nullable.
type offerView struct {
	OfferID           string    `json:"offer_id"`
	Party             string    `json:"party"`
	TotalPrice        int64     `json:"total_price"`
	ReadinessLeadDays int32     `json:"readiness_lead_days"`
	Note              *string   `json:"note"`
	CreatedAt         time.Time `json:"created_at"`
}

// validateOfferInput enforces the OfferCreate rules (all 422 with a field
// name): total_price a whole rupiah of at least 1, readiness_lead_days within
// 0..365, note at most 500 characters when present.
func validateOfferInput(in offerInput) *validationError {
	var fields []httpx.FieldError

	if in.TotalPrice < 1 {
		fields = append(fields, httpx.FieldError{Field: "total_price", Message: "Harga minimal 1 rupiah."})
	}
	if in.ReadinessLeadDays < 0 || in.ReadinessLeadDays > 365 {
		fields = append(fields, httpx.FieldError{Field: "readiness_lead_days", Message: "Kesiapan antara 0 dan 365 hari."})
	}
	if in.Note != nil && len([]rune(*in.Note)) > 500 {
		fields = append(fields, httpx.FieldError{Field: "note", Message: "Catatan maksimal 500 karakter."})
	}

	if len(fields) > 0 {
		return &validationError{fields: fields}
	}
	return nil
}

// createOffer records a subcontractor's reply to a candidate (FR-031). It loads
// the candidate, verifies the caller owns the candidate's listing, rejects a
// readiness start past the deadline (FR-090) and a quantity beyond the remaining
// capacity in the readiness..deadline range (FR-035), then inserts the offer,
// moves the candidate to offered, and notifies the buyer, all in one
// transaction. The capacity range is keyed on the offer's own submission date
// and readiness lead (Rule 4: week rounding stays in platform.WeekStart).
func (s *Service) createOffer(ctx context.Context, accountID pgtype.UUID, candidateID pgtype.UUID, in offerInput) (offerView, error) {
	if verr := validateOfferInput(in); verr != nil {
		return offerView{}, verr
	}

	cand, err := s.queries().GetCandidateForOffer(ctx, candidateID)
	if err != nil {
		if isNoRows(err) {
			return offerView{}, &conflictError{code: httpx.CodeNotFound, detail: "Kandidat tidak ditemukan."}
		}
		return offerView{}, err
	}

	if cand.SubcontractorAccount != accountID {
		return offerView{}, &conflictError{code: httpx.CodeForbidden, detail: "Anda bukan pemilik listing kandidat ini."}
	}

	now := s.clock.Now()
	if !now.Before(cand.ReplyDueAt.Time) {
		return offerView{}, &conflictError{
			code:   httpx.CodeRequestExpired,
			detail: "Batas waktu balasan request ini sudah lewat pada " + platform.FormatDateID(cand.ReplyDueAt.Time) + ".",
		}
	}

	readinessWeek := platform.WeekStart(now.AddDate(0, 0, int(in.ReadinessLeadDays)))
	deadlineWeek := platform.WeekStart(cand.Deadline.Time)

	if readinessWeek.After(deadlineWeek) {
		return offerView{}, &conflictError{
			code:   httpx.CodeReadinessAfterDeadline,
			detail: "Kesiapan mulai melewati tenggat pesanan, tidak ada minggu yang tersisa.",
		}
	}

	remaining, err := s.queries().RemainingCapacityForOffer(ctx, sqlcgen.RemainingCapacityForOfferParams{
		ListingID:      cand.ListingID,
		ReadinessWeek:  pgdate(readinessWeek),
		DeadlineWeek:   pgdate(deadlineWeek),
		WeeklyCapacity: cand.WeeklyCapacity,
		HorizonUntil:   cand.HorizonUntil,
	})
	if err != nil {
		return offerView{}, err
	}

	if int64(cand.Quantity) > remaining {
		untilWeek := deadlineWeek.Format(dateLayout)
		return offerView{}, &metaError{
			code: httpx.CodeInsufficientCapacity,
			detail: "Kapasitas tersisa " + itoa64(remaining) + " potong sampai minggu " +
				platform.FormatDateID(deadlineWeek) + ", kurang dari " + itoa64(int64(cand.Quantity)) + " potong yang diminta.",
			meta: map[string]any{
				"quantity_requested": cand.Quantity,
				"remaining_capacity": remaining,
				"until_week":         untilWeek,
			},
		}
	}

	note := pgtype.Text{}
	if in.Note != nil {
		note = pgtype.Text{String: *in.Note, Valid: true}
	}

	var offer sqlcgen.Offer
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		offer, err = q.InsertOffer(ctx, sqlcgen.InsertOfferParams{
			CandidateID:       candidateID,
			ProposedBy:        sqlcgen.OfferPartySubcontractor,
			TotalPrice:        in.TotalPrice,
			ReadinessLeadDays: in.ReadinessLeadDays,
			Note:              note,
			CreatedAt:         tstz(now),
		})
		if err != nil {
			return err
		}

		if err := q.SetCandidateStatus(ctx, sqlcgen.SetCandidateStatusParams{
			ID:        candidateID,
			Status:    sqlcgen.CandidateStatusOffered,
			UpdatedAt: tstz(now),
		}); err != nil {
			return err
		}

		link := "/quota-requests/" + uuidString(cand.RequestID)
		return s.notifier.Enqueue(ctx, tx, cand.BuyerAccount,
			sqlcgen.EventTypeOfferReceived,
			"Penawaran baru",
			"Subkontraktor mengirim penawaran untuk permintaan kuota Anda.",
			&link)
	})
	if err != nil {
		return offerView{}, err
	}

	return offerViewOf(offer), nil
}

// offerViewOf renders one offer row as its response body.
func offerViewOf(o sqlcgen.Offer) offerView {
	var note *string
	if o.Note.Valid {
		n := o.Note.String
		note = &n
	}
	return offerView{
		OfferID:           uuidString(o.ID),
		Party:             string(o.ProposedBy),
		TotalPrice:        o.TotalPrice,
		ReadinessLeadDays: o.ReadinessLeadDays,
		Note:              note,
		CreatedAt:         o.CreatedAt.Time,
	}
}

// itoa64 renders a base-10 int64 for the FR-035 detail string.
func itoa64(n int64) string { return strconv.FormatInt(n, 10) }
