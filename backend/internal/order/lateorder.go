package order

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
	"github.com/fzrilsh/devotion/backend/internal/platform/scheduler"
)

// registerLateOrder wires the admin-only late-order monitoring list (FR-045).
// The route lives in the order package, not admin, because the past-deadline
// predicate is order-domain arithmetic (order.PastDeadlineCutoff) shared with the
// notifier job below, the same way auto-confirm's read and ticker layers share
// deadline.go. It is admin-gated so only admins see the platform-wide list; a
// non-admin is rejected before the handler runs.
func (s *Service) registerLateOrder(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleAdmin)
	r.Gated("GET /api/admin/late-orders", gate, s.handleListLateOrders)
}

// handleListLateOrders lists every active order past its delivery deadline,
// newest first, one keyset page at a time (FR-045). It is the compute-on-read
// layer: "late" is computed from the deadline against PastDeadlineCutoff(now) at
// read time, so the list is always current without waiting for the ticker. The
// list is platform-wide (admin sees all parties), not party-scoped like the
// buyer/subcontractor list. An unparseable cursor falls back to the first page.
func (s *Service) handleListLateOrders(w http.ResponseWriter, r *http.Request) {
	cutoff := PastDeadlineCutoff(s.clock.Now())
	params := sqlcgen.ListLateWorkOrdersForAdminParams{
		BeforeCutoff: pgtype.Date{Time: cutoff, Valid: true},
		PageLimit:    workOrderPageLimit + 1, // one extra row detects a next page
	}
	if cur, ok := decodeCursor(r.URL.Query().Get("cursor")); ok {
		params.BeforeCreated = cur.created
		params.BeforeID = cur.id
	}

	rows, err := s.queries().ListLateWorkOrdersForAdmin(r.Context(), params)
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
		items = append(items, s.lateItemView(wo))
	}

	page := pagination{HasNext: hasNext}
	if hasNext {
		last := rows[len(rows)-1]
		c := encodeCursor(cursor{created: last.CreatedAt, id: last.ID})
		page.NextCursor = &c
	}
	writeJSON(w, http.StatusOK, workOrderList{Items: items, Pagination: page})
}

// lateItemView renders one admin late-order row as the same summary WorkOrderDetail
// shape the party list uses, so the frontend renders the admin list from the shared
// contract. It applies the same lazy auto-confirm as listItemView (research.md R-07
// layer 1): a shipped order past its 7-day window would read as confirmed and so
// leave the active set, but the late query's status filter has already excluded
// confirmed rows, so this only keeps the two layers' status text consistent.
func (s *Service) lateItemView(wo sqlcgen.ListLateWorkOrdersForAdminRow) workOrderView {
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
		Allocations:            []allocationView{},
		StatusHistory:          []statusEntry{},
		Payments:               []paymentView{},
	}
	if effStatus == sqlcgen.WorkOrderStatusShipped && wo.ShippedAt.Valid {
		at := AutoConfirmAt(wo.ShippedAt.Time)
		view.AutoConfirmAt = &at
	}
	return view
}

// LateOrderJob is layer 2 of the two-layer late-order monitoring (research.md
// R-07): the in-process ticker that sends the FR-045 late-delivery notice the
// compute-on-read list only surfaces. It is registered by serve under
// LockKeyLateOrder, so only one instance runs during a deploy rollover. The
// past-deadline arithmetic is the shared PastDeadlineCutoff, so this layer and the
// admin list never disagree on which orders are late.
func (s *Service) LateOrderJob() scheduler.Job {
	return scheduler.Job{
		Name:    "order:late-order",
		LockKey: scheduler.LockKeyLateOrder,
		Run: func(ctx context.Context, _ *pgxpool.Conn) error {
			return s.runLateOrder(ctx)
		},
	}
}

// runLateOrder does one pass: read the active orders past their deadline that have
// not yet been notified, then handle each in its own transaction so one order's
// failure neither rolls back the others nor aborts the pass. now comes from the
// injected Clock (Rule 5), and the cutoff is PastDeadlineCutoff(now), matching the
// admin list's predicate exactly. Each order stamps late_notified_at (IS NULL
// guard) and notifies both parties; there is no status change and no history row,
// because a passed deadline is a warning, not a state transition (FR-045).
func (s *Service) runLateOrder(ctx context.Context) error {
	cutoff := PastDeadlineCutoff(s.clock.Now())
	rows, err := s.queries().ListLateWorkOrdersToNotify(ctx, pgtype.Date{Time: cutoff, Valid: true})
	if err != nil {
		return err
	}
	now := s.clock.Now()
	for _, row := range rows {
		if err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
			q := sqlcgen.New(tx)
			// The IS NULL guard makes the stamp idempotent, so a repeat pass over
			// the same row (two overlapping instances) notifies only once.
			if err := q.MarkLateNotified(ctx, sqlcgen.MarkLateNotifiedParams{
				ID:             row.ID,
				LateNotifiedAt: tstz(now),
			}); err != nil {
				return err
			}
			// Both parties are alerted the deadline has passed (FR-045).
			link := "/work-orders/" + uuidString(row.ID)
			for _, account := range [2]pgtype.UUID{row.BuyerAccount, row.SubcontractorAccount} {
				if err := s.notifier.Enqueue(ctx, tx, account,
					sqlcgen.EventTypeDeadlinePassed,
					"Pesanan melewati tenggat",
					"Pesanan telah melewati tenggat pengiriman yang disepakati. Segera tindak lanjuti dengan pihak lain.",
					&link); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
