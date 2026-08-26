package order

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/scheduler"
)

// AutoConfirmJob is layer 2 of the two-layer auto-confirm (research.md R-07):
// the in-process ticker that writes the closure the lazy read layer only
// computes, and sends the FR-069 approaching notice the lazy layer cannot. It is
// registered by serve under LockKeyAutoConfirm, so only one instance runs during
// a deploy rollover. Each tick warns buyers whose deadline is near (FR-069) and
// closes orders past their window (FR-068); a disputed order is excluded by the
// status='shipped' scan, so mediation stops the clock (FR-070). The deadline
// arithmetic is the shared package (AutoConfirmWindow, IsAutoConfirm*), so this
// layer and the read layer never disagree on a boundary.
func (s *Service) AutoConfirmJob() scheduler.Job {
	return scheduler.Job{
		Name:    "order:auto-confirm",
		LockKey: scheduler.LockKeyAutoConfirm,
		Run: func(ctx context.Context, _ *pgxpool.Conn) error {
			return s.runAutoConfirm(ctx)
		},
	}
}

// runAutoConfirm does one pass: first the FR-069 warnings, then the FR-068
// closures. It reads both work lists off the pool, then handles each order in
// its own transaction, so one order's failure neither rolls back the others nor
// aborts the pass. now comes from the injected Clock (Rule 5), and the window
// boundaries are derived from it so they match the read-layer predicates exactly.
func (s *Service) runAutoConfirm(ctx context.Context) error {
	now := s.clock.Now()
	// due_before is the shipped_at on or before which a 7-day order is due:
	// now - AutoConfirmWindow. warn_before is now - AutoConfirmWindow +
	// AutoConfirmWarnLead, the shipped_at up to which an order is inside the
	// warning lead. Together (due_before, warn_before] is the approaching band,
	// the SQL form of IsAutoConfirmApproaching.
	dueBefore := now.Add(-AutoConfirmWindow)
	warnBefore := dueBefore.Add(AutoConfirmWarnLead)

	if err := s.sendAutoConfirmWarnings(ctx, tstz(dueBefore), tstz(warnBefore)); err != nil {
		return err
	}
	return s.closeDueOrders(ctx, tstz(dueBefore))
}

// sendAutoConfirmWarnings enqueues the FR-069 "deadline mendekat" notice to the
// buyer of each shipped order inside the warning lead that has not been warned,
// then stamps confirm_warn_sent_at so the notice is sent once, not on every tick.
// The notice and the stamp share one transaction per order, so a warned order is
// never left unstamped (which would re-warn) nor stamped without a notice.
func (s *Service) sendAutoConfirmWarnings(ctx context.Context, dueBefore, warnBefore pgtype.Timestamptz) error {
	rows, err := s.queries().ListShippedApproachingAutoConfirm(ctx, sqlcgen.ListShippedApproachingAutoConfirmParams{
		DueBefore:  dueBefore,
		WarnBefore: warnBefore,
	})
	if err != nil {
		return err
	}
	now := s.clock.Now()
	for _, row := range rows {
		if err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
			q := sqlcgen.New(tx)
			// The IS NULL guard makes the stamp idempotent, so a repeat pass over
			// the same row (two overlapping instances) warns only once.
			if err := q.MarkConfirmWarnSent(ctx, sqlcgen.MarkConfirmWarnSentParams{
				ID:                row.ID,
				ConfirmWarnSentAt: tstz(now),
			}); err != nil {
				return err
			}
			// The notice names the date the order will close, computed from the same
			// AutoConfirmAt the closure uses, so the buyer sees the exact deadline.
			deadline := platform.FormatDateID(AutoConfirmAt(row.ShippedAt.Time))
			link := "/work-orders/" + uuidString(row.ID)
			return s.notifier.Enqueue(ctx, tx, row.BuyerAccount,
				sqlcgen.EventTypeConfirmationDueApproaching,
				"Pesanan akan dikonfirmasi otomatis",
				"Pesanan Anda akan dianggap diterima secara otomatis pada "+deadline+" bila tidak ada tindakan sebelum tenggat.",
				&link)
		}); err != nil {
			return err
		}
	}
	return nil
}

// closeDueOrders auto-confirms each shipped order past its 7-day window (FR-068)
// in its own transaction: the status flips to confirmed with auto_confirmed set,
// a by_system history row records the system as actor, and both parties are
// notified the order closed automatically. The AutoConfirmWorkOrder guard means a
// concurrent party confirmation or dispute since the scan makes the update a
// no-op, and the returned-rows check skips the notice in that case, so the closure
// and its notices fire exactly once.
func (s *Service) closeDueOrders(ctx context.Context, dueBefore pgtype.Timestamptz) error {
	rows, err := s.queries().ListShippedDueForAutoConfirm(ctx, dueBefore)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	for _, row := range rows {
		if err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
			q := sqlcgen.New(tx)

			updated, err := q.AutoConfirmWorkOrder(ctx, sqlcgen.AutoConfirmWorkOrderParams{
				ID:          row.ID,
				ConfirmedAt: tstz(now),
			})
			if err != nil {
				if isNoRows(err) {
					// A party confirmed or a dispute moved the order between the scan
					// and this update: nothing to close, and no notice.
					return nil
				}
				return err
			}

			// The closure was done by the system, so the history row carries no
			// human actor: by_system=true, changed_by null (the actor_clear CHECK
			// allows exactly this).
			if err := q.InsertOrderStatusHistory(ctx, sqlcgen.InsertOrderStatusHistoryParams{
				WorkOrderID: row.ID,
				OldStatus:   sqlcgen.NullWorkOrderStatus{WorkOrderStatus: sqlcgen.WorkOrderStatusShipped, Valid: true},
				NewStatus:   sqlcgen.WorkOrderStatusConfirmed,
				ChangedBy:   pgtype.UUID{},
				BySystem:    true,
				Note:        pgtype.Text{},
				CreatedAt:   tstz(now),
			}); err != nil {
				return err
			}

			// Both parties are told the closure happened automatically (FR-068).
			link := "/work-orders/" + uuidString(updated.ID)
			for _, account := range [2]pgtype.UUID{row.BuyerAccount, row.SubcontractorAccount} {
				if err := s.notifier.Enqueue(ctx, tx, account,
					sqlcgen.EventTypeOrderAutoClosed,
					"Pesanan dikonfirmasi otomatis",
					"Pesanan telah dianggap diterima secara otomatis tujuh hari setelah dikirim.",
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
