package order

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// workOrderPageLimit caps one page of the work-order list. The keyset query asks
// for one extra row to detect whether a next page exists (FR-038).
const workOrderPageLimit = 20

// statusChangeBody is the POST /status request: the target status and an optional
// note. new_status is validated against the allowed forward moves; note is capped
// by decodeJSON's body limit and echoed into the history trail.
type statusChangeBody struct {
	NewStatus string  `json:"new_status"`
	Note      *string `json:"note"`
}

// registerWorkOrder wires US5's read and status routes. The detail and list reads
// are open to any authenticated party (the handler enforces the party guard, so a
// non-party sees a 404, not a 403). The forward status change is subcontractor
// only (FR-005): only the maker advances production. Cancellation (T054) wires its
// own both-party route.
func (s *Service) registerWorkOrder(r *httpx.Router, auth httpx.Authenticator) {
	authed := httpx.RequireAuth(auth)
	subGate := httpx.RequireRole(auth, httpx.RoleSubcontractor)
	r.Gated("GET /api/work-orders", authed, s.handleListWorkOrders)
	r.Gated("GET /api/work-orders/{workOrderId}", authed, s.handleWorkOrderDetail)
	r.Gated("POST /api/work-orders/{workOrderId}/status", subGate, s.handleWorkOrderStatus)
	s.registerCancel(r, auth)
}

// handleWorkOrderDetail returns one work order's full detail (FR-038, FR-039). The
// route is authenticated but not role-gated: the party guard compares the caller's
// account id to the order's two parties, so a non-party (or a malformed id, or an
// order that does not exist) all collapse to the same 404, never leaking that the
// order exists to someone not on it.
func (s *Service) handleWorkOrderDetail(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(r.PathValue("workOrderId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}

	row, err := s.queries().GetWorkOrderForView(r.Context(), id)
	if err != nil {
		if isNoRows(err) {
			httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
			return
		}
		httpx.WriteInternal(w)
		return
	}
	if row.BuyerAccount != acc.ID && row.SubcontractorAccount != acc.ID {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}

	view, err := s.buildDetailView(r.Context(), row)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// buildDetailView assembles the WorkOrderDetail body from the loaded row plus its
// status history and live allocation periods. allowed_transitions and
// self_cancellable come from the shared state machine (allowedTransitions), so the
// client renders from the array instead of duplicating the machine (FR-039).
// auto_confirm_at is present only once shipped, when the 7-day clock has a start
// (FR-068).
func (s *Service) buildDetailView(ctx context.Context, row sqlcgen.GetWorkOrderForViewRow) (workOrderView, error) {
	history, err := s.queries().ListWorkOrderStatusHistory(ctx, row.ID)
	if err != nil {
		return workOrderView{}, err
	}
	allocs, err := s.queries().ListWorkOrderAllocations(ctx, row.ID)
	if err != nil {
		return workOrderView{}, err
	}

	allocations := make([]allocationView, 0, len(allocs))
	for _, a := range allocs {
		allocations = append(allocations, allocationView{
			WeekStart:  platform.FormatDateID(a.WeekStart.Time),
			Capacity:   a.TotalCapacity,
			Allocated:  a.UsedCapacity,
			Remaining:  allocationRemaining(a),
			MarkedFull: a.MarkedFull,
		})
	}

	statusHistory := make([]statusEntry, 0, len(history))
	for _, h := range history {
		var note *string
		if h.Note.Valid {
			n := h.Note.String
			note = &n
		}
		statusHistory = append(statusHistory, statusEntry{
			Status: string(h.NewStatus),
			At:     h.CreatedAt.Time,
			Note:   note,
		})
	}

	view := workOrderView{
		WorkOrderID:            uuidString(row.ID),
		Status:                 string(row.Status),
		BuyerProfileID:         uuidString(row.BuyerID),
		SubcontractorProfileID: uuidString(row.SubcontractorID),
		ProductItemID:          uuidString(row.ProductItemID),
		Quantity:               row.Quantity,
		Deadline:               platform.FormatDateID(row.Deadline.Time),
		TotalPrice:             row.TotalPrice,
		ReadinessLeadDays:      row.ReadinessLeadDays,
		ReadinessDeadline:      platform.FormatDateID(row.ReadinessWeekStart.Time),
		AllowedTransitions:     allowedTransitions(row.Status),
		SelfCancellable:        row.Status == sqlcgen.WorkOrderStatusAccepted,
		Allocations:            allocations,
		StatusHistory:          statusHistory,
		Payments:               []paymentView{},
	}
	if row.Status == sqlcgen.WorkOrderStatusShipped && row.ShippedAt.Valid {
		at := AutoConfirmAt(row.ShippedAt.Time)
		view.AutoConfirmAt = &at
	}
	return view, nil
}

// allocationRemaining is capacity - allocated, floored at zero and forced to zero
// on a full week, matching the AvailabilityPeriod contract. It mirrors
// remainingCapacity but over the list row shape.
func allocationRemaining(a sqlcgen.ListWorkOrderAllocationsRow) int32 {
	if a.MarkedFull {
		return 0
	}
	r := a.TotalCapacity - a.UsedCapacity
	if r < 0 {
		return 0
	}
	return r
}

// handleWorkOrderStatus advances a work order along the forward chain (FR-039,
// FR-044). The route is subcontractor-gated (FR-005); the handler further guards
// that the caller is the subcontractor of this particular order, so another
// subcontractor's order is a 404. new_status must be one of the forward targets;
// the transition itself is validated against the shared state machine, and an
// out-of-order jump is rejected with INVALID_STATUS_TRANSITION quoting the
// allowed order. The change records the actor (changed_by = account id,
// by_system = false) and notifies the buyer.
func (s *Service) handleWorkOrderStatus(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	id, ok := parseUUID(r.PathValue("workOrderId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}
	var body statusChangeBody
	if !decodeJSON(w, r, &body) {
		return
	}

	target, ok := parseForwardStatus(body.NewStatus)
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed,
			"Status tujuan tidak sah. Nilai yang diterima: production, completed, shipped.")
		return
	}

	view, err := s.changeStatus(r.Context(), acc.ID, id, target, body.Note)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// parseForwardStatus accepts only the forward targets a subcontractor may drive
// through this endpoint: production, completed, shipped. Cancellation, mediation,
// and confirmation move through their own endpoints, so they are rejected here as
// invalid input rather than as a status transition.
func parseForwardStatus(s string) (sqlcgen.WorkOrderStatus, bool) {
	switch sqlcgen.WorkOrderStatus(s) {
	case sqlcgen.WorkOrderStatusProduction:
		return sqlcgen.WorkOrderStatusProduction, true
	case sqlcgen.WorkOrderStatusCompleted:
		return sqlcgen.WorkOrderStatusCompleted, true
	case sqlcgen.WorkOrderStatusShipped:
		return sqlcgen.WorkOrderStatusShipped, true
	default:
		return "", false
	}
}

// changeStatus performs the forward transition in one transaction: lock the order,
// guard the caller is its subcontractor, validate the move against the state
// machine, update the status (stamping shipped_at on the move to shipped), record
// the history row with the human actor, then notify the buyer. The whole thing is
// one tx so the status and its history row commit together; the notification row
// rides the same tx while delivery runs after commit.
func (s *Service) changeStatus(ctx context.Context, accountID, workOrderID pgtype.UUID, target sqlcgen.WorkOrderStatus, note *string) (workOrderView, error) {
	now := s.clock.Now()
	var buyerAccount pgtype.UUID
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)

		wo, err := q.LockWorkOrderForStatusChange(ctx, workOrderID)
		if err != nil {
			if isNoRows(err) {
				return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
			}
			return err
		}

		// Resolve both parties' accounts for the party guard and the buyer notice.
		view, err := q.GetWorkOrderForView(ctx, workOrderID)
		if err != nil {
			return err
		}
		if view.SubcontractorAccount != accountID {
			// A subcontractor who is not this order's maker is treated as a
			// non-party: a 404, not a leak that the order exists.
			return &conflictError{code: httpx.CodeNotFound, detail: "Pesanan tidak ditemukan."}
		}
		buyerAccount = view.BuyerAccount

		// The move must be one the state machine allows from the current status.
		// An out-of-order jump quotes the current status and the forward order.
		if !transitionAllowed(wo.Status, target) {
			return &metaError{
				code:   httpx.CodeInvalidStatusTransition,
				detail: invalidTransitionDetail(wo.Status, target),
				meta: map[string]any{
					"current_status":      string(wo.Status),
					"allowed_transitions": allowedTransitions(wo.Status),
				},
			}
		}

		if _, err := q.UpdateWorkOrderStatus(ctx, sqlcgen.UpdateWorkOrderStatusParams{
			ID:        workOrderID,
			Status:    target,
			ShippedAt: tstz(now),
		}); err != nil {
			return err
		}

		var noteText pgtype.Text
		if note != nil {
			noteText = pgtype.Text{String: *note, Valid: true}
		}
		if err := q.InsertOrderStatusHistory(ctx, sqlcgen.InsertOrderStatusHistoryParams{
			WorkOrderID: workOrderID,
			OldStatus:   sqlcgen.NullWorkOrderStatus{WorkOrderStatus: wo.Status, Valid: true},
			NewStatus:   target,
			ChangedBy:   accountID,
			BySystem:    false,
			Note:        noteText,
			CreatedAt:   tstz(now),
		}); err != nil {
			return err
		}

		link := "/work-orders/" + uuidString(workOrderID)
		if err := s.notifier.Enqueue(ctx, tx, buyerAccount,
			sqlcgen.EventTypeOrderStatusChanged,
			"Status pesanan berubah",
			"Pesanan Anda kini berstatus "+statusLabelID(target)+".",
			&link); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return workOrderView{}, err
	}

	// Reload the fresh detail so the response carries the new status, its history,
	// and the recomputed allowed_transitions.
	row, err := s.queries().GetWorkOrderForView(ctx, workOrderID)
	if err != nil {
		return workOrderView{}, err
	}
	return s.buildDetailView(ctx, row)
}

// transitionAllowed reports whether target is in the current status's allowed set.
// It reads from the same allowedTransitions table the client renders from, so the
// server never accepts a move the client would not have offered (FR-044).
func transitionAllowed(current, target sqlcgen.WorkOrderStatus) bool {
	for _, t := range allowedTransitions(current) {
		if t == string(target) {
			return true
		}
	}
	return false
}

// invalidTransitionDetail composes the INVALID_STATUS_TRANSITION detail in
// Indonesian, quoting the current and attempted labels and the forward order the
// caller must follow. The forward chain is the FR-039 sequence
// accepted -> production -> completed -> shipped, spelled in labels.
func invalidTransitionDetail(current, target sqlcgen.WorkOrderStatus) string {
	return "Pesanan berstatus \"" + statusLabelID(current) +
		"\" tidak dapat langsung menjadi \"" + statusLabelID(target) +
		"\". Urutan yang diizinkan: " + statusLabelID(sqlcgen.WorkOrderStatusAccepted) +
		", " + statusLabelID(sqlcgen.WorkOrderStatusProduction) +
		", " + statusLabelID(sqlcgen.WorkOrderStatusCompleted) +
		", " + statusLabelID(sqlcgen.WorkOrderStatusShipped) + "."
}

// handleListWorkOrders lists the caller's work orders newest first, one keyset
// page at a time (FR-038). It resolves the caller's business profile id, so the
// list is scoped to orders that party is on. role narrows to as_buyer or
// as_subcontractor; status[] filters by status; cursor resumes after the previous
// page. An unparseable cursor falls back to the first page.
func (s *Service) handleListWorkOrders(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	profileID, err := s.queries().GetProfileIDByAccount(r.Context(), acc.ID)
	if err != nil {
		if isNoRows(err) {
			// A caller with no business profile is on no order: an empty first page.
			writeJSON(w, http.StatusOK, workOrderList{Items: []workOrderView{}, Pagination: pagination{}})
			return
		}
		httpx.WriteInternal(w)
		return
	}

	roleFilter := r.URL.Query().Get("role")
	statuses, ok := parseStatusFilter(r.URL.Query()["status"])
	if !ok {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Nilai status pada filter tidak sah.")
		return
	}

	params := sqlcgen.ListWorkOrdersForPartyParams{
		RoleFilter:   roleFilter,
		ProfileID:    profileID,
		StatusFilter: statuses,
		PageLimit:    workOrderPageLimit + 1, // one extra row detects a next page
	}
	if cur, ok := decodeCursor(r.URL.Query().Get("cursor")); ok {
		params.BeforeCreated = cur.created
		params.BeforeID = cur.id
	}

	rows, err := s.queries().ListWorkOrdersForParty(r.Context(), params)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}

	hasNext := len(rows) > workOrderPageLimit
	if hasNext {
		rows = rows[:workOrderPageLimit]
	}

	items := make([]workOrderView, 0, len(rows))
	for _, wo := range rows {
		items = append(items, listItemView(wo))
	}

	page := pagination{HasNext: hasNext}
	if hasNext {
		last := rows[len(rows)-1]
		c := encodeCursor(cursor{created: last.CreatedAt, id: last.ID})
		page.NextCursor = &c
	}
	writeJSON(w, http.StatusOK, workOrderList{Items: items, Pagination: page})
}

// workOrderList is the WorkOrderList body: a page of details plus the keyset
// pagination cursor.
type workOrderList struct {
	Items      []workOrderView `json:"items"`
	Pagination pagination      `json:"pagination"`
}

// listItemView renders a list row as a WorkOrderDetail without its history,
// allocations, or payments: the list carries the summary shape, and the detail
// endpoint fills the rest. allowed_transitions and self_cancellable still come
// from the shared state machine so the list can render the same buttons.
func listItemView(wo sqlcgen.WorkOrder) workOrderView {
	view := workOrderView{
		WorkOrderID:            uuidString(wo.ID),
		Status:                 string(wo.Status),
		BuyerProfileID:         uuidString(wo.BuyerID),
		SubcontractorProfileID: uuidString(wo.SubcontractorID),
		Quantity:               wo.Quantity,
		Deadline:               platform.FormatDateID(wo.Deadline.Time),
		TotalPrice:             wo.TotalPrice,
		ReadinessDeadline:      platform.FormatDateID(wo.ReadinessWeekStart.Time),
		AllowedTransitions:     allowedTransitions(wo.Status),
		SelfCancellable:        wo.Status == sqlcgen.WorkOrderStatusAccepted,
		Allocations:            []allocationView{},
		StatusHistory:          []statusEntry{},
		Payments:               []paymentView{},
	}
	if wo.Status == sqlcgen.WorkOrderStatusShipped && wo.ShippedAt.Valid {
		at := AutoConfirmAt(wo.ShippedAt.Time)
		view.AutoConfirmAt = &at
	}
	return view
}

// parseStatusFilter validates the status[] query values against the known work
// order statuses. An unknown value is rejected rather than silently dropped, so a
// typo does not quietly widen the result. An empty filter means no restriction.
func parseStatusFilter(values []string) ([]sqlcgen.WorkOrderStatus, bool) {
	out := make([]sqlcgen.WorkOrderStatus, 0, len(values))
	for _, v := range values {
		switch sqlcgen.WorkOrderStatus(v) {
		case sqlcgen.WorkOrderStatusAccepted,
			sqlcgen.WorkOrderStatusProduction,
			sqlcgen.WorkOrderStatusCompleted,
			sqlcgen.WorkOrderStatusShipped,
			sqlcgen.WorkOrderStatusConfirmed,
			sqlcgen.WorkOrderStatusCancelled,
			sqlcgen.WorkOrderStatusInMediation:
			out = append(out, sqlcgen.WorkOrderStatus(v))
		default:
			return nil, false
		}
	}
	return out, true
}
