package verification

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

// errAlreadyDecided signals that a decision was attempted on a request that is
// no longer pending. It is raised inside the decision transaction after the row
// lock reveals a terminal status, and rolls the transaction back so a second
// decision never overwrites the first.
var errAlreadyDecided = errors.New("verification request already decided")

// queuePageLimit bounds one admin queue page. Mirrors the work-order list limit
// so the two keyset consumers page at the same size.
const queuePageLimit = 20

// queueResponse is the contract's VerificationRequestList: one keyset page of
// requests plus the pagination envelope. The admin queue returns this envelope,
// unlike the applicant list which returns a bare array.
type queueResponse struct {
	Items      []verificationResponse `json:"items"`
	Pagination queuePagination        `json:"pagination"`
}

// decisionRequest is the POST /decision body: approve or reject, with an
// optional reason. A rejection carries a reason that the applicant reads, so the
// handler requires it before the rejection_needs_reason CHECK would fire
// (FR-007). An approval ignores any reason.
type decisionRequest struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// listQueue handles GET /api/admin/verification: one keyset page of verification
// requests for the admin queue, newest first (FR-007). An optional status query
// param narrows to a single status; absent, it lists every status. The cursor is
// opaque and passed back verbatim; a malformed one falls back to the first page.
func (s *Service) listQueue(w http.ResponseWriter, r *http.Request) {
	params := sqlcgen.ListVerificationQueueParams{PageLimit: queuePageLimit + 1}

	if raw := strings.TrimSpace(r.URL.Query().Get("status")); raw != "" {
		st, ok := parseVerificationStatus(raw)
		if !ok {
			httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
				{Field: "status", Message: "Status harus pending, approved, atau rejected."},
			})
			return
		}
		params.StatusFilter = sqlcgen.NullVerificationStatus{VerificationStatus: st, Valid: true}
	}

	if cur, ok := decodeQueueCursor(r.URL.Query().Get("cursor")); ok {
		params.BeforeCreated = cur.created
		params.BeforeID = cur.id
	}

	rows, err := s.queries().ListVerificationQueue(r.Context(), params)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}

	hasNext := len(rows) > queuePageLimit
	if hasNext {
		rows = rows[:queuePageLimit]
	}

	items := make([]verificationResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, verificationResponse{
			RequestID:      uuidString(row.ID),
			Status:         string(row.Status),
			ProfileID:      uuidString(row.ProfileID),
			BusinessName:   row.BusinessName,
			IdentityNumber: strPtr(row.IdentityNumber),
			IdentityFileID: uuidPtr(row.IdentityFileID),
			LocationFileID: uuidPtr(row.LocationFileID),
			Reason:         textPtr(row.AdminNote),
			SubmittedAt:    tstzPtr(row.CreatedAt),
			DecidedAt:      tstzPtr(row.DecidedAt),
		})
	}

	page := queuePagination{HasNext: hasNext}
	if hasNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		cursor := encodeQueueCursor(queueCursor{created: last.CreatedAt, id: last.ID})
		page.NextCursor = &cursor
	}

	writeJSON(w, http.StatusOK, queueResponse{Items: items, Pagination: page})
}

// decide handles POST /api/admin/verification/{requestId}/decision: an admin
// approves or rejects a pending request, recording the decision with the admin's
// identity and the decision time (FR-007). Approval flips the profile's verified
// badge in the same transaction, so the badge propagates to the profile and to
// search (FR-008); a rejection leaves the listing untouched (FR-010, FR-011).
func (s *Service) decide(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}

	requestID, valid := parseUUID(r.PathValue("requestId"))
	if !valid {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Id pengajuan tidak sah.")
		return
	}

	var body decisionRequest
	if !decodeJSON(w, r, &body) {
		return
	}

	status, ok := parseDecision(body.Decision)
	if !ok {
		httpx.WriteValidation(w, "Masukan tidak sah.", []httpx.FieldError{
			{Field: "decision", Message: "Keputusan harus approved atau rejected."},
		})
		return
	}

	reason := strings.TrimSpace(body.Reason)
	// A rejection must carry a reason the applicant reads (FR-007). Reject it in
	// the handler so the applicant gets a field error, not the 500 the
	// rejection_needs_reason CHECK would otherwise surface.
	if status == sqlcgen.VerificationStatusRejected && reason == "" {
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

	var note pgtype.Text
	if reason != "" {
		note = pgtype.Text{String: reason, Valid: true}
	}
	now := pgtype.Timestamptz{Time: s.clock.Now(), Valid: true}

	var decided sqlcgen.DecideVerificationRequestRow
	err := db.WithTx(r.Context(), s.pool, func(tx pgx.Tx) error {
		q := s.queries().WithTx(tx)

		// Lock the request first: two admins deciding the same request serialize,
		// and the second reads the status the first already committed.
		locked, err := q.LockVerificationRequest(r.Context(), requestID)
		if err != nil {
			return err
		}
		if locked.Status != sqlcgen.VerificationStatusPending {
			return errAlreadyDecided
		}

		decided, err = q.DecideVerificationRequest(r.Context(), sqlcgen.DecideVerificationRequestParams{
			Status:    status,
			AdminNote: note,
			DecidedBy: acc.ID,
			DecidedAt: now,
			ID:        requestID,
		})
		if err != nil {
			return err
		}

		if status == sqlcgen.VerificationStatusApproved {
			if err := q.MarkProfileVerified(r.Context(), sqlcgen.MarkProfileVerifiedParams{
				UpdatedAt: now,
				ID:        decided.ProfileID,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		switch {
		case err == errAlreadyDecided:
			httpx.WriteProblem(w, httpx.CodeValidationFailed, "Pengajuan ini sudah diputuskan.")
		case isNoRows(err):
			httpx.WriteProblem(w, httpx.CodeNotFound, "Pengajuan verifikasi tidak ditemukan.")
		default:
			httpx.WriteInternal(w)
		}
		return
	}

	writeJSON(w, http.StatusOK, verificationResponse{
		RequestID:      uuidString(decided.ID),
		Status:         string(decided.Status),
		ProfileID:      uuidString(decided.ProfileID),
		BusinessName:   decided.BusinessName,
		IdentityNumber: strPtr(decided.IdentityNumber),
		IdentityFileID: uuidPtr(decided.IdentityFileID),
		LocationFileID: uuidPtr(decided.LocationFileID),
		Reason:         textPtr(decided.AdminNote),
		SubmittedAt:    tstzPtr(decided.CreatedAt),
		DecidedAt:      tstzPtr(decided.DecidedAt),
	})
}

// parseVerificationStatus maps the status query param to the sqlc enum.
func parseVerificationStatus(v string) (sqlcgen.VerificationStatus, bool) {
	switch v {
	case string(sqlcgen.VerificationStatusPending):
		return sqlcgen.VerificationStatusPending, true
	case string(sqlcgen.VerificationStatusApproved):
		return sqlcgen.VerificationStatusApproved, true
	case string(sqlcgen.VerificationStatusRejected):
		return sqlcgen.VerificationStatusRejected, true
	default:
		return "", false
	}
}

// parseDecision maps the decision body value to a terminal status. Only approved
// and rejected are valid decisions; pending is not a decision an admin can make.
func parseDecision(v string) (sqlcgen.VerificationStatus, bool) {
	switch v {
	case string(sqlcgen.VerificationStatusApproved):
		return sqlcgen.VerificationStatusApproved, true
	case string(sqlcgen.VerificationStatusRejected):
		return sqlcgen.VerificationStatusRejected, true
	default:
		return "", false
	}
}
