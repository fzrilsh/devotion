package masterdata

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// queuePageLimit bounds one admin proposal-queue page, matching the other admin
// queues so the keyset consumers page at the same size.
const queuePageLimit = 20

// registerAdmin wires the master-data admin surface (T068), all gated to the
// admin role: managing the baseline catalog (FR-059, FR-060) and deciding item
// proposals with the proposer notified (FR-061, FR-074). Every route checks the
// caller's role through RequireRole, so a non-admin never reaches a handler.
func (s *Service) registerAdmin(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleAdmin)
	r.Gated("GET /api/admin/master/items", gate, s.handleListItems)
	r.Gated("POST /api/admin/master/items", gate, s.handleCreateItem)
	r.Gated("PATCH /api/admin/master/items/{itemId}", gate, s.handleUpdateItem)
	r.Gated("GET /api/admin/proposals", gate, s.handleListProposals)
	r.Gated("POST /api/admin/proposals/{proposalId}/decision", gate, s.handleDecideProposal)
}

// createItemInput is the POST /admin/master/items body: the item kind and its
// name. Both are required; the name is trimmed and length-checked like a
// proposal.
type createItemInput struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// updateItemInput is the PATCH /admin/master/items/{itemId} body. Both fields
// are optional pointers so a rename-only or deactivate-only request leaves the
// other column untouched (FR-059). A nil field means "unchanged".
type updateItemInput struct {
	Name   *string `json:"name"`
	Active *bool   `json:"active"`
}

// decisionInput is the POST /admin/proposals/{proposalId}/decision body: approve
// or reject, with an optional reason. A rejection carries a reason the proposer
// reads; the handler requires it before the constraint fires.
type decisionInput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// handleListItems lists every item of one kind, active and inactive, for the
// admin catalog surface (FR-059). The kind comes from the required kind query
// param; an unknown kind is a 422.
func (s *Service) handleListItems(w http.ResponseWriter, r *http.Request) {
	kind := sqlcgen.ItemType(strings.TrimSpace(r.URL.Query().Get("kind")))
	if kind != sqlcgen.ItemTypeProduct && kind != sqlcgen.ItemTypeMachine {
		httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
			{Field: "kind", Message: "Jenis harus product atau machine."},
		})
		return
	}
	rows, err := s.queries().ListCatalogItemsByType(r.Context(), kind)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	out := make([]catalogItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, catalogItem{
			ItemID: uuidString(row.ID),
			Kind:   string(row.Type),
			Name:   row.Name,
			Active: row.Active,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleCreateItem adds a baseline catalog item (FR-059). A same-name item of the
// kind hits the item_name_unique_per_type constraint, which the handler catches
// as a 422 field error rather than letting the 500 surface.
func (s *Service) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var in createItemInput
	if !decodeJSON(w, r, &in) {
		return
	}
	kind := sqlcgen.ItemType(in.Kind)
	name := strings.TrimSpace(in.Name)
	var fields []httpx.FieldError
	if kind != sqlcgen.ItemTypeProduct && kind != sqlcgen.ItemTypeMachine {
		fields = append(fields, httpx.FieldError{Field: "kind", Message: "Jenis harus product atau machine."})
	}
	if n := len([]rune(name)); n < minProposedName || n > 100 {
		fields = append(fields, httpx.FieldError{Field: "name", Message: "Nama harus 2 sampai 100 karakter."})
	}
	if len(fields) > 0 {
		httpx.WriteValidation(w, "Masukan tidak sah.", fields)
		return
	}

	item, err := s.queries().InsertCatalogItem(r.Context(), sqlcgen.InsertCatalogItemParams{
		Type:      kind,
		Name:      name,
		CreatedAt: tstz(s.clock.Now()),
	})
	if err != nil {
		if isUniqueViolation(err, "item_name_unique_per_type") {
			httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
				{Field: "name", Message: "Item dengan nama itu sudah ada."},
			})
			return
		}
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusCreated, catalogItem{
		ItemID: uuidString(item.ID),
		Kind:   string(item.Type),
		Name:   item.Name,
		Active: item.Active,
	})
}

// handleUpdateItem renames or flips the active flag on an existing item (FR-059).
// It locks the row first, then applies the partial update. Deactivating an item
// only flips its active flag; the listing rows referencing it are untouched, so
// those listings stay discoverable through search (FR-060). A rename to a
// duplicate name hits the unique constraint, caught as a 422.
func (s *Service) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID, valid := parseUUID(r.PathValue("itemId"))
	if !valid {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id item tidak sah.")
		return
	}
	var in updateItemInput
	if !decodeJSON(w, r, &in) {
		return
	}

	var name pgtype.Text
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if n := len([]rune(trimmed)); n < minProposedName || n > 100 {
			httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
				{Field: "name", Message: "Nama harus 2 sampai 100 karakter."},
			})
			return
		}
		name = pgtype.Text{String: trimmed, Valid: true}
	}
	var active pgtype.Bool
	if in.Active != nil {
		active = pgtype.Bool{Bool: *in.Active, Valid: true}
	}
	if !name.Valid && !active.Valid {
		httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
			{Field: "name", Message: "Sertakan name atau active untuk diubah."},
		})
		return
	}

	var updated sqlcgen.UpdateCatalogItemRow
	err := db.WithTx(r.Context(), s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if _, err := q.LockCatalogItem(r.Context(), itemID); err != nil {
			return err
		}
		var err error
		updated, err = q.UpdateCatalogItem(r.Context(), sqlcgen.UpdateCatalogItemParams{
			ID:     itemID,
			Name:   name,
			Active: active,
		})
		return err
	})
	if err != nil {
		switch {
		case isNoRows(err):
			httpx.WriteProblem(w, httpx.CodeNotFound, "Item tidak ditemukan.")
		case isUniqueViolation(err, "item_name_unique_per_type"):
			httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
				{Field: "name", Message: "Item dengan nama itu sudah ada."},
			})
		default:
			httpx.WriteInternal(w)
		}
		return
	}
	writeJSON(w, http.StatusOK, catalogItem{
		ItemID: uuidString(updated.ID),
		Kind:   string(updated.Type),
		Name:   updated.Name,
		Active: updated.Active,
	})
}

// proposalQueueItem is one row of the admin pending-proposal queue: the proposal,
// its proposed name and kind, the proposer's business name, and the admin note
// carried as the decision reason (null while still pending).
type proposalQueueItem struct {
	ProposalID   string  `json:"proposal_id"`
	Kind         string  `json:"kind"`
	ProposedName string  `json:"proposed_name"`
	Status       string  `json:"status"`
	ProposerName string  `json:"proposer_name"`
	Reason       *string `json:"reason"`
	CreatedAt    string  `json:"created_at"`
}

// proposalQueueResponse is the pending-proposal queue page plus the keyset
// pagination envelope.
type proposalQueueResponse struct {
	Items      []proposalQueueItem `json:"items"`
	Pagination queuePagination     `json:"pagination"`
}

// handleListProposals returns one keyset page of pending item proposals oldest
// first (FR-061). The cursor is opaque and passed back verbatim; a malformed one
// falls back to the first page.
func (s *Service) handleListProposals(w http.ResponseWriter, r *http.Request) {
	params := sqlcgen.ListItemProposalsPendingParams{Limit: queuePageLimit + 1}
	if cur, ok := decodeQueueCursor(r.URL.Query().Get("cursor")); ok {
		params.AfterCreatedAt = cur.created
		params.AfterID = cur.id
	}

	rows, err := s.queries().ListItemProposalsPending(r.Context(), params)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}

	hasNext := len(rows) > queuePageLimit
	if hasNext {
		rows = rows[:queuePageLimit]
	}

	items := make([]proposalQueueItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, proposalQueueItem{
			ProposalID:   uuidString(row.ID),
			Kind:         string(row.Type),
			ProposedName: row.ProposedName,
			Status:       string(row.Status),
			ProposerName: row.ProposerName,
			Reason:       textPtr(row.AdminNote),
			CreatedAt:    row.CreatedAt.Time.Format(createProposalTimeLayout),
		})
	}

	page := queuePagination{HasNext: hasNext}
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		cursor := encodeQueueCursor(queueCursor{created: last.CreatedAt, id: last.ID})
		page.NextCursor = &cursor
	}

	writeJSON(w, http.StatusOK, proposalQueueResponse{Items: items, Pagination: page})
}

// handleDecideProposal applies an admin's approve or reject decision to a pending
// proposal and notifies the proposer (FR-061, FR-074). A rejection must carry a
// reason, rejected in the handler before the constraint fires; an already-decided
// proposal is a 422 and an unknown one a 404.
func (s *Service) handleDecideProposal(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	proposalID, valid := parseUUID(r.PathValue("proposalId"))
	if !valid {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id usulan tidak sah.")
		return
	}
	var in decisionInput
	if !decodeJSON(w, r, &in) {
		return
	}

	approved, ok := parseProposalDecision(in.Decision)
	if !ok {
		httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
			{Field: "decision", Message: "Keputusan harus approved atau rejected."},
		})
		return
	}
	reason := strings.TrimSpace(in.Reason)
	if !approved && reason == "" {
		httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
			{Field: "reason", Message: "Alasan penolakan wajib diisi."},
		})
		return
	}
	if len(reason) > 1000 {
		httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
			{Field: "reason", Message: "Alasan maksimal 1000 karakter."},
		})
		return
	}
	var note *string
	if reason != "" {
		note = &reason
	}

	view, err := s.DecideProposal(r.Context(), proposalID, acc.ID, approved, note)
	if err != nil {
		switch {
		case errors.Is(err, errAlreadyDecided):
			httpx.WriteProblem(w, httpx.CodeValidationFailed, "Usulan ini sudah diputuskan.")
		case isNoRows(err):
			httpx.WriteProblem(w, httpx.CodeNotFound, "Usulan tidak ditemukan.")
		default:
			httpx.WriteInternal(w)
		}
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// parseProposalDecision maps the decision body value to approved (true) or
// rejected (false). Only these two are valid; pending is not a decision.
func parseProposalDecision(v string) (approved, ok bool) {
	switch v {
	case string(sqlcgen.ProposalStatusApproved):
		return true, true
	case string(sqlcgen.ProposalStatusRejected):
		return false, true
	default:
		return false, false
	}
}

// parseUUID parses a canonical UUID string into a pgtype.UUID, false when it is
// not a valid UUID.
func parseUUID(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, false
	}
	return u, u.Valid
}

// isNoRows reports whether err is pgx.ErrNoRows, used to turn a missing row into
// a 404.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// isUniqueViolation reports whether err is a Postgres unique-violation (23505) on
// the named constraint, so a duplicate catalog name becomes a field error.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

// queuePagination is the keyset pagination envelope the admin proposal queue
// returns. next_cursor is opaque: the client passes it back verbatim.
type queuePagination struct {
	HasNext    bool    `json:"has_next"`
	NextCursor *string `json:"next_cursor"`
}

// queueCursor is the decoded keyset position: the (created_at, id) of the last
// row of the previous page.
type queueCursor struct {
	created pgtype.Timestamptz
	id      pgtype.UUID
}

// queueCursorPayload is the on-the-wire cursor before base64: an RFC3339 stamp
// and a uuid string, kept opaque behind base64url.
type queueCursorPayload struct {
	Created string `json:"c"`
	ID      string `json:"i"`
}

// encodeQueueCursor builds the opaque next_cursor from a row's keyset position.
func encodeQueueCursor(c queueCursor) string {
	b, _ := json.Marshal(queueCursorPayload{
		Created: c.created.Time.Format(time.RFC3339Nano),
		ID:      uuidString(c.id),
	})
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeQueueCursor reverses encodeQueueCursor, returning ok false on any
// malformed input so the caller falls back to the first page.
func decodeQueueCursor(s string) (queueCursor, bool) {
	if strings.TrimSpace(s) == "" {
		return queueCursor{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return queueCursor{}, false
	}
	var p queueCursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return queueCursor{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, p.Created)
	if err != nil {
		return queueCursor{}, false
	}
	id, ok := parseUUID(p.ID)
	if !ok {
		return queueCursor{}, false
	}
	return queueCursor{created: pgtype.Timestamptz{Time: t, Valid: true}, id: id}, true
}
