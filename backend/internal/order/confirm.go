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

// registerConfirm wires the buyer's manual confirmation route. Only the buyer of
// an order confirms receipt (FR-047, US5 AS-2), so the gate admits the buyer role
// alone; the handler further guards that the caller is actually the buyer of this
// particular order. The subcontractor advances the order up to "shipped" through
// the /status endpoint, and the system closes it after 7 days (FR-068), but the
// move to "confirmed" is the buyer's, or the system's on their behalf.
func (s *Service) registerConfirm(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleBuyer)
	r.Gated("POST /api/work-orders/{workOrderId}/confirm", gate, s.handleConfirm)
}

// handleConfirm confirms receipt of a shipped order on the buyer's initiative
// (FR-047, FR-068). The route is buyer-gated; the handler guards that the caller
// is the buyer of this order (a non-party sees a 404), and the service rejects a
// confirmation on an order that is not "shipped" with INVALID_STATUS_TRANSITION.
// There is no request body: confirmation carries no data beyond the act itself.
func (s *Service) handleConfirm(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(r.PathValue("workOrderId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}

	view, err := s.confirm(r.Context(), acc.ID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// confirm closes a shipped order as buyer-confirmed in one transaction (FR-047,
// FR-068). It locks the order, guards that the caller is its buyer (a non-party,
// including the subcontractor, sees a 404, never a leak), validates the move
// against the shared state machine so an order that is not "shipped" is rejected
// with INVALID_STATUS_TRANSITION quoting the allowed order, then flips the status
// to "confirmed" with auto_confirmed=false (the buyer's manual act, not the
// system's 7-day closure). An open dispute halts the confirmation the same way it
// halts auto-confirm (FR-070): PartyConfirmWorkOrder's NOT EXISTS guard makes the
// write a no-op, so the order stays "shipped" and the state-machine check on the
// reloaded row surfaces the block rather than reporting a phantom success. The
// history row records the human actor (changed_by = account id, by_system = false),
// and the subcontractor is notified.
func (s *Service) confirm(ctx context.Context, accountID, workOrderID pgtype.UUID) (workOrderView, error) {
	now := s.clock.Now()
	var subcontractorAccount pgtype.UUID
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
		if view.BuyerAccount != accountID {
			// Not the buyer of this order (the subcontractor included): a 404, not a
			// leak that the order exists.
			return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
		}
		subcontractorAccount = view.SubcontractorAccount

		// The move must be one the state machine allows: only "shipped" -> "confirmed".
		// From any other status this quotes the current status and the forward order,
		// the same INVALID_STATUS_TRANSITION body the /status endpoint uses.
		if !transitionAllowed(wo.Status, sqlcgen.WorkOrderStatusConfirmed) {
			return &metaError{
				code:   httpx.CodeInvalidStatusTransition,
				detail: invalidTransitionDetail(wo.Status, sqlcgen.WorkOrderStatusConfirmed),
				meta: map[string]any{
					"current_status":      string(wo.Status),
					"allowed_transitions": allowedTransitions(wo.Status),
				},
			}
		}

		// PartyConfirmWorkOrder guards status='shipped' AND no open dispute. wo.Status
		// is 'shipped' here (transitionAllowed passed), so a no-op means a dispute was
		// reported between the read and this write: report it as still not shipped-
		// confirmable rather than a phantom success (FR-070).
		if _, err := q.PartyConfirmWorkOrder(ctx, sqlcgen.PartyConfirmWorkOrderParams{
			ID:          workOrderID,
			ConfirmedAt: tstz(now),
		}); err != nil {
			if isNoRows(err) {
				return &metaError{
					code:   httpx.CodeInvalidStatusTransition,
					detail: invalidTransitionDetail(wo.Status, sqlcgen.WorkOrderStatusConfirmed),
					meta: map[string]any{
						"current_status":      string(wo.Status),
						"allowed_transitions": allowedTransitions(wo.Status),
					},
				}
			}
			return err
		}

		if err := q.InsertOrderStatusHistory(ctx, sqlcgen.InsertOrderStatusHistoryParams{
			WorkOrderID: workOrderID,
			OldStatus:   sqlcgen.NullWorkOrderStatus{WorkOrderStatus: wo.Status, Valid: true},
			NewStatus:   sqlcgen.WorkOrderStatusConfirmed,
			ChangedBy:   accountID,
			BySystem:    false,
			CreatedAt:   tstz(now),
		}); err != nil {
			return err
		}

		// Notify the subcontractor that the buyer confirmed receipt (FR-051 "perubahan
		// status pesanan"). A confirmation is an order status change, so it rides
		// EventTypeOrderStatusChanged like the other status moves.
		link := "/work-orders/" + uuidString(workOrderID)
		if err := s.notifier.Enqueue(ctx, tx, subcontractorAccount,
			sqlcgen.EventTypeOrderStatusChanged,
			"Pesanan dikonfirmasi diterima",
			"Pemberi order mengonfirmasi telah menerima barang. Pesanan kini selesai.",
			&link); err != nil {
			return err
		}

		// A confirmed order is reviewable, so invite both parties to rate each other
		// (FR-051 "permintaan rating", US5 AS-1). The request rides the confirmation
		// transaction so it never outlives a rolled-back close; it is non-transactional,
		// so each party still honors its channel preferences (FR-091).
		reviewLink := "/work-orders/" + uuidString(workOrderID) + "/review"
		for _, account := range [2]pgtype.UUID{accountID, subcontractorAccount} {
			if err := s.notifier.Enqueue(ctx, tx, account,
				sqlcgen.EventTypeRatingRequest,
				"Beri ulasan untuk pesanan ini",
				"Pesanan telah selesai. Beri rating dan ulasan untuk pihak lain agar reputasi keduanya terbentuk.",
				&reviewLink); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return workOrderView{}, err
	}

	// Reload the fresh detail so the response carries the confirmed status, its
	// history, and the recomputed (empty) allowed_transitions.
	row, err := s.queries().GetWorkOrderForView(ctx, workOrderID)
	if err != nil {
		return workOrderView{}, err
	}
	return s.buildDetailView(ctx, row, accountID)
}
