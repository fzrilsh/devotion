package order

import (
	"context"
	"net/http"
	"time"

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
	s.registerContacts(r, auth)
	s.registerConfirm(r, auth)
	s.registerCancel(r, auth)
	s.registerPayment(r, auth)
	s.registerDispute(r, auth)
	s.registerLateOrder(r, auth)
	s.registerMediation(r, auth)
}

// handleWorkOrderDetail returns one work order's full detail (FR-038, FR-039). The
// route is authenticated but not role-gated: the party guard compares the caller's
// account id to the order's two parties, so a non-party (or a malformed id, or an
// order that does not exist) all collapse to the same 404, never leaking that the
// order exists to someone not on it. An admin is admitted past the party guard
// because FR-045 and FR-046 require reading the full history of an order the admin
// is not party to; the read stays read-only, since the forward status change is
// still subcontractor only and an admin changes state through dispute resolution.
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
	if row.BuyerAccount != acc.ID && row.SubcontractorAccount != acc.ID && !callerIsAdmin(r) {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Pesanan tidak ditemukan.")
		return
	}

	view, err := s.buildDetailView(r.Context(), row, acc.ID)
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
//
// callerAccount is the account reading the order. can_record_payment and
// can_review depend on who is asking: only a party (buyer or subcontractor) may
// record a statement or review, so an admin reading the order past the party
// guard gets both false. payment_mismatch does not depend on the caller: it is
// the same flag a party sees and admin reads when a dispute is reported (FR-043).
func (s *Service) buildDetailView(ctx context.Context, row sqlcgen.GetWorkOrderForViewRow, callerAccount pgtype.UUID) (workOrderView, error) {
	history, err := s.queries().ListWorkOrderStatusHistory(ctx, row.ID)
	if err != nil {
		return workOrderView{}, err
	}
	allocs, err := s.queries().ListWorkOrderAllocations(ctx, row.ID)
	if err != nil {
		return workOrderView{}, err
	}
	pays, err := s.queries().ListPaymentRecordsForWorkOrder(ctx, row.ID)
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

	payments := make([]paymentView, 0, len(pays))
	for _, p := range pays {
		var note *string
		if p.Note.Valid {
			n := p.Note.String
			note = &n
		}
		payments = append(payments, paymentView{
			PaymentID:           uuidString(p.ID),
			Direction:           string(p.Direction),
			Date:                platform.FormatDateID(p.Date.Time),
			DeclaredByProfileID: uuidString(p.ProfileID),
			Note:                note,
			CreatedAt:           p.CreatedAt.Time,
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

	// Lazy auto-confirm (research.md R-07 layer 1): a shipped order past its
	// 7-day window reads as confirmed everywhere, without waiting for the ticker
	// to write the row, so it never looks shipped on one page and confirmed on
	// another. IsAutoConfirmDue takes the same open-dispute flag the ticker's
	// NOT EXISTS guard enforces, so a disputed order reads as still shipped here
	// too: both layers agree on both the boundary instant and the dispute halt
	// (FR-068, FR-070).
	effStatus := row.Status
	if row.Status == sqlcgen.WorkOrderStatusShipped && row.ShippedAt.Valid &&
		IsAutoConfirmDue(AutoConfirmBase(row.AutoConfirmBaseAt, row.ShippedAt), s.clock.Now(), row.HasOpenDispute) {
		effStatus = sqlcgen.WorkOrderStatusConfirmed
	}

	// can_record_payment and can_review are caller-scoped: only a party may state a
	// payment or review the order, so an admin reading the order past the party
	// guard gets both false and the client offers neither button. A party may
	// record a payment on any status except cancelled (a cancelled order settles no
	// money through the platform, FR-041). It may review once the order reads as
	// confirmed received and it has not already reviewed, the same two legs
	// createReview enforces (FR-047); ReviewExistsForOrderReviewer is the
	// not-yet-reviewed leg, keyed on the caller's own profile so the button never
	// offers an action the one_review_per_order_per_reviewer constraint would reject.
	isBuyer := row.BuyerAccount == callerAccount
	isSub := row.SubcontractorAccount == callerAccount
	isParty := isBuyer || isSub

	canRecordPayment := isParty && effStatus != sqlcgen.WorkOrderStatusCancelled

	canReview := false
	if isParty && effStatus == sqlcgen.WorkOrderStatusConfirmed {
		reviewerID := row.BuyerID
		if isSub {
			reviewerID = row.SubcontractorID
		}
		reviewed, err := s.queries().ReviewExistsForOrderReviewer(ctx, sqlcgen.ReviewExistsForOrderReviewerParams{
			WorkOrderID: row.ID,
			ReviewerID:  reviewerID,
		})
		if err != nil {
			return workOrderView{}, err
		}
		canReview = !reviewed
	}

	view := workOrderView{
		WorkOrderID:            uuidString(row.ID),
		Status:                 string(effStatus),
		BuyerProfileID:         uuidString(row.BuyerID),
		SubcontractorProfileID: uuidString(row.SubcontractorID),
		ProductItemID:          uuidString(row.ProductItemID),
		Quantity:               row.Quantity,
		Deadline:               platform.FormatDateID(row.Deadline.Time),
		TotalPrice:             row.TotalPrice,
		ReadinessLeadDays:      row.ReadinessLeadDays,
		ReadinessDeadline:      platform.FormatDateID(row.ReadinessWeekStart.Time),
		AllowedTransitions:     allowedTransitions(effStatus),
		SelfCancellable:        effStatus == sqlcgen.WorkOrderStatusAccepted,
		CanRecordPayment:       canRecordPayment,
		CanReview:              canReview,
		Allocations:            allocations,
		StatusHistory:          statusHistory,
		Payments:               payments,
		PaymentMismatch:        computePaymentMismatch(pays, row.BuyerID, row.SubcontractorID),
	}
	// auto_confirm_at is shown while the order is still effectively shipped, so the
	// buyer sees when it will close; once the window has passed the order already
	// reads as confirmed and the field is dropped (FR-068).
	if effStatus == sqlcgen.WorkOrderStatusShipped && row.ShippedAt.Valid {
		at := AutoConfirmAt(AutoConfirmBase(row.AutoConfirmBaseAt, row.ShippedAt))
		view.AutoConfirmAt = &at
	}
	return view, nil
}

// computePaymentMismatch derives the FR-043 mismatch flag from the two parties'
// payment statement rows, never from money (payment_record carries no amount).
// The platform does not verify that payment happened, so what it marks is a
// contradiction between statements, not who is right. A statement is one party's
// declaration in one direction; the buyer's natural direction is 'sent' and the
// subcontractor's is 'received', so a matched pair is the buyer's sent statement
// against the subcontractor's received statement.
//
// Two shapes count as a mismatch. missing_counterpart: one party has stated and
// the other has not stated at all, so the pair is incomplete. date_differs: both
// parties have stated but on different dates, with day_difference the absolute
// day gap. When both have stated on the same date, or neither has stated at all,
// there is no mismatch and the result is nil. The flag does not depend on who is
// reading: it is the same for both parties and for admin on a dispute.
func computePaymentMismatch(pays []sqlcgen.PaymentRecord, buyerID, subID pgtype.UUID) *paymentMismatch {
	var buyerDate, subDate *time.Time
	for _, p := range pays {
		d := p.Date.Time
		switch p.ProfileID {
		case buyerID:
			if buyerDate == nil || d.Before(*buyerDate) {
				buyerDate = &d
			}
		case subID:
			if subDate == nil || d.Before(*subDate) {
				subDate = &d
			}
		}
	}

	switch {
	case buyerDate == nil && subDate == nil:
		return nil
	case buyerDate == nil || subDate == nil:
		return &paymentMismatch{Kind: "missing_counterpart"}
	default:
		days := daysBetween(*buyerDate, *subDate)
		if days == 0 {
			return nil
		}
		return &paymentMismatch{Kind: "date_differs", DayDifference: &days}
	}
}

// daysBetween is the absolute whole-day gap between two statement dates. The dates
// are stored as WIB calendar days (payment_record.date is a date column), so the
// difference is taken on the calendar dates, not on wall-clock durations.
func daysBetween(a, b time.Time) int {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	da := time.Date(ay, am, ad, 0, 0, 0, 0, time.UTC)
	dbb := time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC)
	diff := int(da.Sub(dbb).Hours() / 24)
	if diff < 0 {
		diff = -diff
	}
	return diff
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
	return s.buildDetailView(ctx, row, accountID)
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
		items = append(items, s.listItemView(wo))
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
// from the shared state machine so the list can render the same buttons. The
// same lazy auto-confirm as buildDetailView (research.md R-07 layer 1) applies,
// so a shipped order past its 7-day window reads as confirmed on the list too,
// never shipped here and confirmed on the detail page (FR-068).
func (s *Service) listItemView(wo sqlcgen.ListWorkOrdersForPartyRow) workOrderView {
	effStatus := wo.Status
	if wo.Status == sqlcgen.WorkOrderStatusShipped && wo.ShippedAt.Valid &&
		IsAutoConfirmDue(AutoConfirmBase(wo.AutoConfirmBaseAt, wo.ShippedAt), s.clock.Now(), wo.HasOpenDispute) {
		effStatus = sqlcgen.WorkOrderStatusConfirmed
	}
	view := workOrderView{
		WorkOrderID:            uuidString(wo.ID),
		Status:                 string(effStatus),
		BuyerProfileID:         uuidString(wo.BuyerID),
		SubcontractorProfileID: uuidString(wo.SubcontractorID),
		Quantity:               wo.Quantity,
		Deadline:               platform.FormatDateID(wo.Deadline.Time),
		TotalPrice:             wo.TotalPrice,
		ReadinessDeadline:      platform.FormatDateID(wo.ReadinessWeekStart.Time),
		AllowedTransitions:     allowedTransitions(effStatus),
		SelfCancellable:        effStatus == sqlcgen.WorkOrderStatusAccepted,
		// The list row carries the summary shape only. can_record_payment,
		// can_review, and payment_mismatch are computed on the detail read, which
		// loads the payment rows and the caller's review state; the list frontend
		// renders neither button nor the mismatch notice, so they stay false/nil
		// here rather than paying for a per-row payment and review lookup.
		CanRecordPayment: false,
		CanReview:        false,
		Allocations:      []allocationView{},
		StatusHistory:    []statusEntry{},
		Payments:         []paymentView{},
		PaymentMismatch:  nil,
	}
	if effStatus == sqlcgen.WorkOrderStatusShipped && wo.ShippedAt.Valid {
		at := AutoConfirmAt(AutoConfirmBase(wo.AutoConfirmBaseAt, wo.ShippedAt))
		view.AutoConfirmAt = &at
	}
	return view
}

// parseStatusFilter validates the status[] query values against the known work
// order statuses. An unknown value is rejected rather than silently dropped, so a
// typo does not quietly widen the result. An empty filter means no restriction.
// The result is []string, not []WorkOrderStatus: the query casts the parameter to
// text[] because pgx cannot encode a slice of the named enum type without the enum
// OID registered on the pool. An empty slice stays non-nil, matching the query's
// cardinality(...) = 0 "no restriction" branch.
func parseStatusFilter(values []string) ([]string, bool) {
	out := make([]string, 0, len(values))
	for _, v := range values {
		switch sqlcgen.WorkOrderStatus(v) {
		case sqlcgen.WorkOrderStatusAccepted,
			sqlcgen.WorkOrderStatusProduction,
			sqlcgen.WorkOrderStatusCompleted,
			sqlcgen.WorkOrderStatusShipped,
			sqlcgen.WorkOrderStatusConfirmed,
			sqlcgen.WorkOrderStatusCancelled,
			sqlcgen.WorkOrderStatusInMediation:
			out = append(out, v)
		default:
			return nil, false
		}
	}
	return out, true
}
