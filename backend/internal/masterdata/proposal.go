package masterdata

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// Proposal input bounds, from the POST /master/proposals contract: proposed_name
// is 2..80 characters and kind is one of the two catalog item types.
const (
	minProposedName = 2
	maxProposedName = 80
)

// Sentinel errors the proposal path returns; the handler maps them to problem
// responses. errProfileMissing is an invariant break (a profile has existed
// since registration, T026), so it becomes a 500 rather than something the
// caller can act on.
var (
	errProfileMissing = errors.New("masterdata: profil tidak ditemukan")
)

// validationError carries per-field rejections in the shape
// httpx.WriteValidation renders, so the handler translates it without restating
// the field rules.
type validationError struct {
	fields []httpx.FieldError
}

func (e *validationError) Error() string { return "masterdata: masukan tidak sah" }

// proposalInput is the POST /master/proposals request body. kind is validated
// against the catalog item types; proposed_name is trimmed and length-checked.
type proposalInput struct {
	Kind         string `json:"kind"`
	ProposedName string `json:"proposed_name"`
}

// proposalView is the ItemProposal response body. Contract field names differ
// from the DB columns (proposal_id from id, kind from type, reason from
// admin_note), so the mapping is by hand. reason is nullable, so it is a
// pointer that is null while the proposal is pending.
type proposalView struct {
	ProposalID   string  `json:"proposal_id"`
	Kind         string  `json:"kind"`
	ProposedName string  `json:"proposed_name"`
	Status       string  `json:"status"`
	Reason       *string `json:"reason"`
	CreatedAt    string  `json:"created_at"`
}

// createProposalTimeLayout is RFC3339 for the created_at field, matching the
// date-time format the ItemProposal schema declares.
const createProposalTimeLayout = "2006-01-02T15:04:05Z07:00"

// newProposalView maps a proposal row to its response body.
func newProposalView(p sqlcgen.ItemProposal) proposalView {
	v := proposalView{
		ProposalID:   uuidString(p.ID),
		Kind:         string(p.Type),
		ProposedName: p.ProposedName,
		Status:       string(p.Status),
		CreatedAt:    p.CreatedAt.Time.Format(createProposalTimeLayout),
	}
	if p.AdminNote.Valid {
		note := p.AdminNote.String
		v.Reason = &note
	}
	return v
}

// validateProposalInput enforces the request field rules (422 with a field
// name): kind is product or machine, proposed_name is 2..80 characters after
// trimming surrounding whitespace. It returns the normalized values so the
// caller inserts them without re-trimming.
func (s *Service) validateProposalInput(in proposalInput) (sqlcgen.ItemType, string, *validationError) {
	var fields []httpx.FieldError

	kind := sqlcgen.ItemType(in.Kind)
	if kind != sqlcgen.ItemTypeProduct && kind != sqlcgen.ItemTypeMachine {
		fields = append(fields, httpx.FieldError{Field: "kind", Message: "Jenis harus product atau machine."})
	}

	name := strings.TrimSpace(in.ProposedName)
	if n := len([]rune(name)); n < minProposedName || n > maxProposedName {
		fields = append(fields, httpx.FieldError{Field: "proposed_name", Message: "Nama usulan harus 2 sampai 80 karakter."})
	}

	if len(fields) > 0 {
		return "", "", &validationError{fields: fields}
	}
	return kind, name, nil
}

// createProposal records a user's proposal for a new catalog item (FR-061). It
// resolves the caller's profile, validates the input, and inserts a pending
// proposal timestamped from the Clock (Rule 5). It runs as a single statement,
// not a transaction: nothing else is written and no notification fires until an
// admin decides.
func (s *Service) createProposal(ctx context.Context, accountID pgtype.UUID, in proposalInput) (proposalView, error) {
	kind, name, verr := s.validateProposalInput(in)
	if verr != nil {
		return proposalView{}, verr
	}

	q := s.queries()
	profileID, err := q.GetProfileIDByAccount(ctx, accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return proposalView{}, errProfileMissing
		}
		return proposalView{}, err
	}

	p, err := q.InsertItemProposal(ctx, sqlcgen.InsertItemProposalParams{
		ProfileID:    profileID,
		Type:         kind,
		ProposedName: name,
		CreatedAt:    tstz(s.clock.Now()),
	})
	if err != nil {
		return proposalView{}, err
	}
	return newProposalView(p), nil
}

// errAlreadyDecided signals a decision attempted on a proposal that is no longer
// pending. It is raised inside the decision transaction after the row lock
// reveals a terminal status, rolling the transaction back so a second decision
// never overwrites the first (the same pattern the verification admin path uses).
var errAlreadyDecided = errors.New("masterdata: usulan sudah diputuskan")

// DecideProposal applies an admin decision to a pending proposal and notifies
// the proposer, satisfying the FR-061 done-condition that the proposer is told
// when their proposal is decided. The decision and the notification row share
// one transaction (FR-086): the row is written inside tx via notif.Enqueue, so
// a rolled-back decision leaves no orphan notification, and delivery to email
// and WhatsApp happens later in the scheduler where a failure cannot fail this
// transaction. On approval the catalog item is resolved (inserted or an existing
// same-name item reactivated) inside the same transaction so approved_yields_item
// holds without the caller pre-creating the row. note is the optional admin
// reason stored in admin_note. The proposal is locked FOR UPDATE first so two
// admins deciding the same proposal serialize and the second sees the terminal
// status the first committed, returning errAlreadyDecided.
func (s *Service) DecideProposal(ctx context.Context, proposalID, adminAccountID pgtype.UUID, approved bool, note *string) (proposalView, error) {
	status := sqlcgen.ProposalStatusRejected
	if approved {
		status = sqlcgen.ProposalStatusApproved
	}

	var view proposalView
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		loaded, err := q.LockItemProposal(ctx, proposalID)
		if err != nil {
			return err
		}
		if loaded.Status != sqlcgen.ProposalStatusPending {
			return errAlreadyDecided
		}

		// On approval, resolve the catalog item the proposal yields: insert the
		// proposed name, or reactivate an existing same-name item, and carry its id
		// so approved_yields_item is satisfied. A rejection leaves item_id null.
		var itemID pgtype.UUID
		if approved {
			item, err := q.UpsertCatalogItemReturning(ctx, sqlcgen.UpsertCatalogItemReturningParams{
				Type:      loaded.Type,
				Name:      loaded.ProposedName,
				CreatedAt: tstz(s.clock.Now()),
			})
			if err != nil {
				return err
			}
			itemID = item.ID
		}

		decided, err := q.DecideItemProposal(ctx, sqlcgen.DecideItemProposalParams{
			ID:        proposalID,
			Status:    status,
			AdminNote: textOrNull(note),
			DecidedBy: adminAccountID,
			DecidedAt: tstz(s.clock.Now()),
			ItemID:    itemID,
		})
		if err != nil {
			return err
		}

		title, body := proposalDecisionMessage(decided)
		if err := s.notif.Enqueue(ctx, tx, loaded.ProposerAccountID, sqlcgen.EventTypeItemProposalDecision, title, body, nil); err != nil {
			return err
		}

		view = newProposalView(decided)
		return nil
	})
	if err != nil {
		return proposalView{}, err
	}
	return view, nil
}

// proposalDecisionMessage composes the Indonesian notification title and body
// for a decided proposal. An approval and a rejection read differently; a
// rejection quotes the admin note when one was given.
func proposalDecisionMessage(p sqlcgen.ItemProposal) (title, body string) {
	if p.Status == sqlcgen.ProposalStatusApproved {
		return "Usulan item disetujui",
			"Usulan Anda \"" + p.ProposedName + "\" telah disetujui dan masuk daftar baku."
	}
	body = "Usulan Anda \"" + p.ProposedName + "\" ditolak."
	if p.AdminNote.Valid && strings.TrimSpace(p.AdminNote.String) != "" {
		body += " Alasan: " + p.AdminNote.String
	}
	return "Usulan item ditolak", body
}

// textOrNull wraps an optional string as a pgtype.Text, null when the pointer
// is nil, for the admin_note column.
func textOrNull(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
