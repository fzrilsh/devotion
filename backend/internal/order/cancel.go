package order

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// cancelBody is the cancel request: a required reason both parties can read on
// the order. reason is 5..500 characters, matching the OpenAPI contract.
type cancelBody struct {
	Reason string `json:"reason"`
}

// registerCancel wires the self-cancellation route. Both parties may cancel
// while the order is still "accepted" (FR-065), so the gate admits either the
// buyer or the subcontractor role; the handler further guards that the caller is
// actually a party to this particular order.
func (s *Service) registerCancel(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleBuyer, httpx.RoleSubcontractor)
	r.Gated("POST /api/work-orders/{workOrderId}/cancel", gate, s.handleCancel)
}

// handleCancel cancels a pre-production order on either party's own initiative
// (FR-065). The route admits both business roles; the handler validates the
// reason (5..500), then the service guards party membership and the "accepted"
// status. Past production the order is no longer self-cancellable and the caller
// is directed to open a dispute (FR-066).
func (s *Service) handleCancel(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(r.PathValue("workOrderId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}
	var body cancelBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if n := len([]rune(body.Reason)); n < 5 || n > 500 {
		httpx.WriteProblem(w, httpx.CodeValidationFailed,
			"Alasan pembatalan antara 5 dan 500 karakter.")
		return
	}

	view, err := s.cancel(r.Context(), acc.ID, id, body.Reason)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// cancel reverses a pre-production order in one transaction (FR-020, FR-065). It
// locks the order, guards that the caller is one of its two parties (a non-party
// sees a 404, never a leak), and rejects a cancellation once the order has left
// "accepted": past production self-cancellation is unavailable and the caller is
// pointed at the dispute path (FR-066). On success it records who cancelled (the
// party's profile id), the reason, and the moment, flips the status to
// "cancelled", writes the history row with the human actor, reverses every
// allocation row under the same transaction (reusing reverseAllocationInTx so the
// status flip and the capacity return commit together), then notifies the other
// party with the reason (FR-065). The reversal is idempotent, so a repeat call
// does not double-refund capacity.
func (s *Service) cancel(ctx context.Context, accountID, workOrderID pgtype.UUID, reason string) (workOrderView, error) {
	now := s.clock.Now()
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		wo, err := q.LockWorkOrderForStatusChange(ctx, workOrderID)
		if err != nil {
			if isNoRows(err) {
				return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
			}
			return err
		}

		// Resolve both parties' accounts for the party guard and the notice.
		view, err := q.GetWorkOrderForView(ctx, workOrderID)
		if err != nil {
			return err
		}
		if view.BuyerAccount != accountID && view.SubcontractorAccount != accountID {
			// Not a party to this order: a 404, not a leak that it exists.
			return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
		}

		// Self-cancellation is available only while "accepted". Once the order has
		// entered production (or moved past it), the caller must open a dispute
		// instead; the alternative path rides the problem meta (FR-066).
		if wo.Status != sqlcgen.WorkOrderStatusAccepted {
			return &metaError{
				code: httpx.CodeCancellationAfterProduction,
				detail: "Pesanan sudah masuk tahap Produksi sehingga tidak dapat dibatalkan sendiri. " +
					"Laporkan sengketa agar ditengahi admin.",
				meta: map[string]any{
					"alternative_path": "POST /api/work-orders/" + uuidString(workOrderID) + "/disputes",
				},
			}
		}

		// cancelled_by_id stores the cancelling party's business profile id, not the
		// account id, so the reputation split (FR-072) can charge that party.
		profileID, err := q.GetProfileIDByAccount(ctx, accountID)
		if err != nil {
			if isNoRows(err) {
				return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
			}
			return err
		}

		if _, err := q.CancelWorkOrder(ctx, sqlcgen.CancelWorkOrderParams{
			ID:                 workOrderID,
			CancelledByID:      profileID,
			CancellationReason: pgtype.Text{String: reason, Valid: true},
			CancelledAt:        tstz(now),
		}); err != nil {
			return err
		}

		note := pgtype.Text{String: reason, Valid: true}
		if err := q.InsertOrderStatusHistory(ctx, sqlcgen.InsertOrderStatusHistoryParams{
			WorkOrderID: workOrderID,
			OldStatus:   sqlcgen.NullWorkOrderStatus{WorkOrderStatus: wo.Status, Valid: true},
			NewStatus:   sqlcgen.WorkOrderStatusCancelled,
			ChangedBy:   accountID,
			BySystem:    false,
			Note:        note,
			CreatedAt:   tstz(now),
		}); err != nil {
			return err
		}

		// Reverse every allocation under this same transaction so the status flip
		// and the capacity return are atomic (FR-020).
		if err := s.reverseAllocationInTx(ctx, q, workOrderID, now); err != nil {
			return err
		}

		// Notify the other party with the reason (FR-065). A cancellation is an
		// order status change (accepted -> cancelled), which FR-051 enumerates as
		// "perubahan status pesanan", so it rides EventTypeOrderStatusChanged rather
		// than a separate event outside the nine FR-051 lists. The canceller is one
		// of the two accounts; the notice goes to whichever is not the caller.
		other := view.BuyerAccount
		if accountID == view.BuyerAccount {
			other = view.SubcontractorAccount
		}
		link := "/work-orders/" + uuidString(workOrderID)
		if err := s.notifier.Enqueue(ctx, tx, other,
			sqlcgen.EventTypeOrderStatusChanged,
			"Pesanan dibatalkan",
			"Pesanan dibatalkan dengan alasan: "+reason,
			&link); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return workOrderView{}, err
	}

	// Reload the fresh detail so the response reflects the cancelled status, the
	// empty live allocations, and the recomputed (empty) allowed_transitions.
	row, err := s.queries().GetWorkOrderForView(ctx, workOrderID)
	if err != nil {
		return workOrderView{}, err
	}
	return s.buildDetailView(ctx, row)
}
