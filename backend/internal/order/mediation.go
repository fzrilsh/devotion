package order

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// disputeView is the Dispute contract body the admin queue and the mediate and
// resolve endpoints return. reporter_id and liable_party_id are exposed as the
// contract's *_profile_id names; result, liable_profile_id, allocation_reversed,
// and resolved_at are null until the dispute is resolved.
type disputeView struct {
	DisputeID          string     `json:"dispute_id"`
	WorkOrderID        string     `json:"work_order_id"`
	ReporterProfileID  string     `json:"reporter_profile_id"`
	Status             string     `json:"status"`
	ReportBody         string     `json:"report_body"`
	Result             *string    `json:"result"`
	LiableProfileID    *string    `json:"liable_profile_id"`
	AllocationReversed *bool      `json:"allocation_reversed"`
	CreatedAt          time.Time  `json:"created_at"`
	ResolvedAt         *time.Time `json:"resolved_at"`
}

// disputeToView renders a stored dispute row as the Dispute contract shape. The
// three post-resolution fields (result, liable_profile_id, allocation_reversed)
// and resolved_at come through only when set, matching the contract's nullable
// declarations so a still-open dispute serializes them as null.
func disputeToView(d sqlcgen.Dispute) disputeView {
	v := disputeView{
		DisputeID:         uuidString(d.ID),
		WorkOrderID:       uuidString(d.WorkOrderID),
		ReporterProfileID: uuidString(d.ReporterID),
		Status:            string(d.Status),
		ReportBody:        d.ReportBody,
		CreatedAt:         d.CreatedAt.Time,
	}
	if d.Result.Valid {
		r := string(d.Result.DisputeResult)
		v.Result = &r
	}
	if d.LiablePartyID.Valid {
		id := uuidString(d.LiablePartyID)
		v.LiableProfileID = &id
	}
	if d.AllocationReversed.Valid {
		ar := d.AllocationReversed.Bool
		v.AllocationReversed = &ar
	}
	if d.ResolvedAt.Valid {
		at := d.ResolvedAt.Time
		v.ResolvedAt = &at
	}
	return v
}

// resolveBody is the resolve request. result is the mandatory outcome enum
// (cancelled / continued / confirmed). allocation_reversed and liable_profile_id
// are meaningful only on the cancelled branch, where the admin decides whether
// capacity returns and which party bears the cancellation in the completion rate
// (FR-067, FR-072); the non-cancelled branches ignore them and force
// allocation_reversed false. note is the mandated admin note, up to 2000 chars.
type resolveBody struct {
	Result             string  `json:"result"`
	AllocationReversed bool    `json:"allocation_reversed"`
	LiableProfileID    *string `json:"liable_profile_id"`
	Note               string  `json:"note"`
}

// registerMediation wires the three admin-only mediation routes (FR-046, FR-067,
// FR-072): the dispute queue, the move-into-mediation action, and the resolve
// decision. All three are admin-gated, so a non-admin is rejected with 403 before
// the handler runs; the handlers carry no party guard because an admin mediates
// every party's disputes.
func (s *Service) registerMediation(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleAdmin)
	r.Gated("GET /api/admin/disputes", gate, s.handleListDisputes)
	r.Gated("POST /api/admin/disputes/{disputeId}/mediate", gate, s.handleMediateDispute)
	r.Gated("POST /api/admin/disputes/{disputeId}/resolve", gate, s.handleResolveDispute)
}

// handleListDisputes returns the admin dispute queue, newest first, one keyset
// page at a time (FR-046). An optional status query narrows to one DisputeStatus;
// an unknown value is rejected as invalid input rather than silently ignored. An
// unparseable cursor falls back to the first page. The list is not wrapped in a
// pagination envelope: the contract returns a bare Dispute array here, so the
// next page is reached by passing the last row forward (the admin queue is small).
func (s *Service) handleListDisputes(w http.ResponseWriter, r *http.Request) {
	params := sqlcgen.ListDisputesForAdminParams{
		PageLimit: workOrderPageLimit,
	}
	if raw := r.URL.Query().Get("status"); raw != "" {
		st, ok := parseDisputeStatus(raw)
		if !ok {
			httpx.WriteProblem(w, httpx.CodeValidationFailed,
				"Nilai status pada filter tidak sah. Nilai yang diterima: reported, in_mediation, resolved.")
			return
		}
		params.StatusFilter = pgtype.Text{String: string(st), Valid: true}
	}
	if cur, ok := decodeCursor(r.URL.Query().Get("cursor")); ok {
		params.BeforeCreated = cur.created
		params.BeforeID = cur.id
	}

	rows, err := s.queries().ListDisputesForAdmin(r.Context(), params)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}

	items := make([]disputeView, 0, len(rows))
	for _, d := range rows {
		items = append(items, disputeToView(d))
	}
	writeJSON(w, http.StatusOK, items)
}

// parseDisputeStatus validates the status filter against the DisputeStatus enum,
// rejecting an unknown value so a typo does not silently widen the queue.
func parseDisputeStatus(s string) (sqlcgen.DisputeStatus, bool) {
	switch sqlcgen.DisputeStatus(s) {
	case sqlcgen.DisputeStatusReported,
		sqlcgen.DisputeStatusInMediation,
		sqlcgen.DisputeStatusResolved:
		return sqlcgen.DisputeStatus(s), true
	default:
		return "", false
	}
}

// handleMediateDispute moves a reported dispute's order into "Dalam Mediasi"
// (FR-046). The route is admin-gated. The dispute id comes from the path; a
// malformed id or missing dispute is a 404. The order transition and the dispute
// status update commit together, so an order in mediation always has a dispute in
// mediation.
func (s *Service) handleMediateDispute(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(r.PathValue("disputeId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Sengketa tidak ditemukan.")
		return
	}
	view, err := s.mediateDispute(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// mediateDispute moves the order behind a reported dispute into 'in_mediation'
// and marks the dispute in_mediation, in one transaction (FR-046). It locks the
// dispute, rejects one already resolved, moves the order via MoveWorkOrderToMediation
// (which halts the 7-day auto-confirm count by leaving the 'shipped' scan set,
// FR-070), records a system history row on the order, then notifies both parties
// that admin has taken up the case. A dispute already in_mediation moves the order
// again harmlessly (the order status guard is idempotent for an order already
// there) and re-stamps in_mediation.
func (s *Service) mediateDispute(ctx context.Context, disputeID pgtype.UUID) (disputeView, error) {
	now := s.clock.Now()
	var out sqlcgen.Dispute
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		d, err := q.GetDisputeForAdmin(ctx, disputeID)
		if err != nil {
			if isNoRows(err) {
				return &conflictError{code: httpx.CodeNotFound, detail: "Sengketa tidak ditemukan."}
			}
			return err
		}
		if d.Status == sqlcgen.DisputeStatusResolved {
			return &conflictError{
				code:   httpx.CodeInvalidStatusTransition,
				detail: "Sengketa sudah diputuskan sehingga tidak dapat dikembalikan ke mediasi.",
			}
		}

		// Read the pre-move status for the history trail before the update flips it.
		// MoveWorkOrderToMediation is an UPDATE ... RETURNING, so its returned row
		// already carries 'in_mediation'; the 'continued' resolution restores from
		// this old_status, so it must record the status the order actually left, not
		// the mediation status it is entering.
		before, err := q.GetWorkOrderForView(ctx, d.WorkOrderID)
		if err != nil {
			return err
		}

		if _, err := q.MoveWorkOrderToMediation(ctx, d.WorkOrderID); err != nil {
			if isNoRows(err) {
				// The order is terminal (confirmed/cancelled): it cannot enter mediation.
				return &conflictError{
					code:   httpx.CodeInvalidStatusTransition,
					detail: "Pesanan sudah dalam keadaan akhir sehingga tidak dapat masuk mediasi.",
				}
			}
			return err
		}

		if err := q.MarkDisputeInMediation(ctx, disputeID); err != nil {
			return err
		}

		// A system-driven move into mediation: by_system true, no human actor.
		if err := q.InsertOrderStatusHistory(ctx, sqlcgen.InsertOrderStatusHistoryParams{
			WorkOrderID: d.WorkOrderID,
			OldStatus:   sqlcgen.NullWorkOrderStatus{WorkOrderStatus: before.Status, Valid: true},
			NewStatus:   sqlcgen.WorkOrderStatusInMediation,
			ChangedBy:   pgtype.UUID{},
			BySystem:    true,
			Note:        pgtype.Text{},
			CreatedAt:   tstz(now),
		}); err != nil {
			return err
		}

		if err := s.notifyBothParties(ctx, tx, d.WorkOrderID,
			sqlcgen.EventTypeOrderStatusChanged,
			"Sengketa masuk mediasi",
			"Admin sedang menengahi sengketa pada pesanan ini. Tunggu keputusan mediasi."); err != nil {
			return err
		}

		out, err = q.GetDisputeForAdmin(ctx, disputeID)
		return err
	})
	if err != nil {
		return disputeView{}, err
	}
	return disputeToView(out), nil
}

// notifyBothParties queues one notification for the buyer and one for the
// subcontractor of an order, inside the caller's transaction. The mediation
// endpoints tell both sides when admin takes up or closes a case, so the notice
// goes to both accounts rather than "the other party" as a self-cancel does. It
// reads the two accounts through GetWorkOrderForView, the same read the detail
// path uses, so it needs only the order id.
func (s *Service) notifyBothParties(ctx context.Context, tx pgx.Tx, workOrderID pgtype.UUID, event sqlcgen.EventType, title, body string) error {
	q := sqlcgen.New(tx)
	view, err := q.GetWorkOrderForView(ctx, workOrderID)
	if err != nil {
		return err
	}
	link := "/work-orders/" + uuidString(workOrderID)
	for _, account := range [2]pgtype.UUID{view.BuyerAccount, view.SubcontractorAccount} {
		if err := s.notifier.Enqueue(ctx, tx, account, event, title, body, &link); err != nil {
			return err
		}
	}
	return nil
}

// handleResolveDispute closes a dispute in mediation with the admin's explicit
// decision (FR-067, FR-072). The route is admin-gated. The body carries the
// mandatory result enum and, on the cancelled branch, the allocation and liability
// choices the admin must make explicitly. Input faults (bad result, over-long
// note, missing liability on a cancellation) are 422; a missing dispute is 404; a
// dispute not in mediation is a conflict.
func (s *Service) handleResolveDispute(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(r.PathValue("disputeId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Sengketa tidak ditemukan.")
		return
	}

	var body resolveBody
	if !decodeJSON(w, r, &body) {
		return
	}
	result, ok := parseDisputeResult(body.Result)
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed,
			"Nilai result tidak sah. Nilai yang diterima: cancelled, continued, confirmed.")
		return
	}
	if len(body.Note) > 2000 {
		httpx.WriteProblem(w, httpx.CodeValidationFailed,
			"Catatan mediasi maksimal 2000 karakter.")
		return
	}

	// The cancelled branch is the only one that touches the completion rate, so the
	// admin MUST name the liable party and MUST state whether capacity is returned
	// (FR-067). A note is required so the cancellation carries its reason. The other
	// two branches do not charge anyone and do not return capacity, so those fields
	// are ignored and allocation_reversed is forced false.
	var liableProfile pgtype.UUID
	if result == sqlcgen.DisputeResultCancelled {
		if body.LiableProfileID == nil {
			httpx.WriteProblem(w, httpx.CodeValidationFailed,
				"Pihak yang menanggung pembatalan wajib ditentukan saat hasil mediasi dibatalkan.")
			return
		}
		lp, ok := parseUUID(*body.LiableProfileID)
		if !ok {
			httpx.WriteProblem(w, httpx.CodeValidationFailed,
				"liable_profile_id bukan UUID yang sah.")
			return
		}
		liableProfile = lp
		if body.Note == "" {
			httpx.WriteProblem(w, httpx.CodeValidationFailed,
				"Catatan wajib diisi saat hasil mediasi dibatalkan.")
			return
		}
	}

	view, err := s.resolveDispute(r.Context(), acc.ID, id, result, body.AllocationReversed, liableProfile, body.Note)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// parseDisputeResult validates the resolve outcome against the DisputeResult enum.
func parseDisputeResult(s string) (sqlcgen.DisputeResult, bool) {
	switch sqlcgen.DisputeResult(s) {
	case sqlcgen.DisputeResultCancelled,
		sqlcgen.DisputeResultContinued,
		sqlcgen.DisputeResultConfirmed:
		return sqlcgen.DisputeResult(s), true
	default:
		return "", false
	}
}

// resolveDispute applies the admin's mediation decision to the order and closes
// the dispute, in one transaction (FR-067, FR-072). It locks the dispute, requires
// it be in mediation, then branches on result:
//
//   - cancelled: the order is cancelled with cancelled_by_id set to the liable
//     party's profile id, so only that party carries the cancellation in the
//     completion rate (FR-072); capacity is reversed only when the admin chose it,
//     reusing reverseAllocationInTx so there is one reversal path, not two.
//   - continued: the order is restored to the status it held before mediation,
//     read from the mediation history row. When that status is 'shipped' the 7-day
//     auto-confirm clock restarts from this close instant (not the original
//     shipment) and the buyer's warning flag is cleared, so a disputed-then-
//     continued shipment does not auto-close on the next tick (FR-068).
//   - confirmed: the admin confirms the order on the buyer's behalf; auto_confirmed
//     stays false so an admin confirmation is distinguishable from the 7-day system
//     closure in the trail.
//
// Every branch records a history row with the admin as the human actor
// (by_system=false), then ResolveDispute stamps the outcome and closes the
// dispute, and both parties are notified. allocation_reversed is stored as the
// admin's choice on cancelled and forced false otherwise.
func (s *Service) resolveDispute(ctx context.Context, adminAccount, disputeID pgtype.UUID, result sqlcgen.DisputeResult, allocationReversed bool, liableProfile pgtype.UUID, note string) (disputeView, error) {
	now := s.clock.Now()
	var out sqlcgen.Dispute
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		d, err := q.GetDisputeForAdmin(ctx, disputeID)
		if err != nil {
			if isNoRows(err) {
				return &conflictError{code: httpx.CodeNotFound, detail: "Sengketa tidak ditemukan."}
			}
			return err
		}
		if d.Status != sqlcgen.DisputeStatusInMediation {
			return &conflictError{
				code:   httpx.CodeInvalidStatusTransition,
				detail: "Hanya sengketa yang sedang dalam mediasi yang dapat diputuskan.",
			}
		}

		storedReversed := false
		switch result {
		case sqlcgen.DisputeResultCancelled:
			// The order is cancelled and the liable party's profile id is written to
			// cancelled_by_id so the completion-rate divisor charges only that party
			// (FR-072). CancelWorkOrder has no status guard, so it moves the order
			// from 'in_mediation' to 'cancelled' directly.
			if _, err := q.CancelWorkOrder(ctx, sqlcgen.CancelWorkOrderParams{
				ID:                 d.WorkOrderID,
				CancelledByID:      liableProfile,
				CancellationReason: pgtype.Text{String: note, Valid: true},
				CancelledAt:        tstz(now),
			}); err != nil {
				return err
			}
			if err := s.recordMediationHistory(ctx, q, d.WorkOrderID, sqlcgen.WorkOrderStatusCancelled, adminAccount, note, now); err != nil {
				return err
			}
			if allocationReversed {
				if err := s.reverseAllocationInTx(ctx, q, d.WorkOrderID, now); err != nil {
					return err
				}
				storedReversed = true
			}

		case sqlcgen.DisputeResultContinued:
			// Restore the order to the status it held before mediation, read straight
			// off the mediation history row so the restore lands exactly where the
			// order left (Decision 1).
			prev, err := q.GetStatusBeforeMediation(ctx, d.WorkOrderID)
			if err != nil {
				return err
			}
			if !prev.Valid {
				return &conflictError{
					code:   httpx.CodeInvalidStatusTransition,
					detail: "Status pesanan sebelum mediasi tidak ditemukan sehingga tidak dapat dilanjutkan.",
				}
			}
			if _, err := q.RestoreWorkOrderStatus(ctx, sqlcgen.RestoreWorkOrderStatusParams{
				ID:     d.WorkOrderID,
				Status: prev.WorkOrderStatus,
			}); err != nil {
				return err
			}
			// Only a shipment restarts the auto-confirm clock: the window runs from
			// this mediation close, not the original shipment, and the buyer is warned
			// afresh before the new deadline (FR-068). A non-shipped restore has no
			// auto-confirm clock to restart.
			if prev.WorkOrderStatus == sqlcgen.WorkOrderStatusShipped {
				if err := q.RestartAutoConfirmClock(ctx, sqlcgen.RestartAutoConfirmClockParams{
					ID:                d.WorkOrderID,
					AutoConfirmBaseAt: tstz(now),
				}); err != nil {
					return err
				}
			}
			if err := s.recordMediationHistory(ctx, q, d.WorkOrderID, prev.WorkOrderStatus, adminAccount, note, now); err != nil {
				return err
			}

		case sqlcgen.DisputeResultConfirmed:
			// The admin confirms the order on the buyer's behalf. auto_confirmed stays
			// false so this admin decision is distinct from the 7-day system closure.
			if _, err := q.ForceConfirmWorkOrder(ctx, sqlcgen.ForceConfirmWorkOrderParams{
				ID:          d.WorkOrderID,
				ConfirmedAt: tstz(now),
			}); err != nil {
				return err
			}
			if err := s.recordMediationHistory(ctx, q, d.WorkOrderID, sqlcgen.WorkOrderStatusConfirmed, adminAccount, note, now); err != nil {
				return err
			}
		}

		reversedParam := pgtype.Bool{Bool: storedReversed, Valid: true}
		liableParam := pgtype.UUID{}
		if result == sqlcgen.DisputeResultCancelled {
			liableParam = liableProfile
		}
		if _, err := q.ResolveDispute(ctx, sqlcgen.ResolveDisputeParams{
			ID:                 disputeID,
			Result:             sqlcgen.NullDisputeResult{DisputeResult: result, Valid: true},
			AllocationReversed: reversedParam,
			LiablePartyID:      liableParam,
			AdminNote:          pgtype.Text{String: note, Valid: true},
			HandledBy:          adminAccount,
			ResolvedAt:         tstz(now),
		}); err != nil {
			return err
		}

		if err := s.notifyBothParties(ctx, tx, d.WorkOrderID,
			sqlcgen.EventTypeOrderStatusChanged,
			"Sengketa diputuskan",
			"Admin telah memutuskan hasil mediasi pada pesanan ini."); err != nil {
			return err
		}

		out, err = q.GetDisputeForAdmin(ctx, disputeID)
		return err
	})
	if err != nil {
		return disputeView{}, err
	}
	return disputeToView(out), nil
}

// recordMediationHistory writes the order status-history row for a mediation
// close. The old status is always 'in_mediation' (the order is leaving mediation),
// the actor is the admin (by_system=false), and the note is the admin's mediation
// note.
func (s *Service) recordMediationHistory(ctx context.Context, q *sqlcgen.Queries, workOrderID pgtype.UUID, newStatus sqlcgen.WorkOrderStatus, adminAccount pgtype.UUID, note string, now time.Time) error {
	return q.InsertOrderStatusHistory(ctx, sqlcgen.InsertOrderStatusHistoryParams{
		WorkOrderID: workOrderID,
		OldStatus:   sqlcgen.NullWorkOrderStatus{WorkOrderStatus: sqlcgen.WorkOrderStatusInMediation, Valid: true},
		NewStatus:   newStatus,
		ChangedBy:   adminAccount,
		BySystem:    false,
		Note:        pgtype.Text{String: note, Valid: note != ""},
		CreatedAt:   tstz(now),
	})
}
