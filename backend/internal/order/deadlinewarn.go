package order

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/scheduler"
)

// DeadlineApproachingJob is the in-process ticker that sends the FR-051 "deadline
// mendekat" notice for a delivery deadline drawing near, the notification piece
// FR-051 requires beside deadline_passed. Unlike auto-confirm and late-order there
// is no compute-on-read twin: an approaching deadline changes no status and drives
// no list, it is purely a nudge, so the ticker is the only layer. It is registered
// by serve under LockKeyDeadlineApproaching, so only one instance runs during a
// deploy rollover. The band arithmetic is the shared deadline.go (PastDeadlineCutoff,
// DeadlineApproachingCutoff), so a late order is warned by the late-order job and an
// approaching one by this job, never both.
func (s *Service) DeadlineApproachingJob() scheduler.Job {
	return scheduler.Job{
		Name:    "order:deadline-approaching",
		LockKey: scheduler.LockKeyDeadlineApproaching,
		Run: func(ctx context.Context, _ *pgxpool.Conn) error {
			return s.runDeadlineApproaching(ctx)
		},
	}
}

// runDeadlineApproaching does one pass: read the active, not-yet-shipped orders
// whose deadline is inside the 7-day warning band and not yet warned, then handle
// each in its own transaction so one order's failure neither rolls back the others
// nor aborts the pass. now comes from the injected Clock (Rule 5); the band edges
// are PastDeadlineCutoff(now) (an order already late is left to the late-order job)
// and DeadlineApproachingCutoff(now) (the far edge of the lead). Each order stamps
// deadline_warn_sent_at (IS NULL guard) and warns both parties; there is no status
// change and no history row, because a nearing deadline is a reminder, not a state
// transition (FR-051). The notice is non-transactional (FR-091), so it honors each
// recipient's channel preferences at delivery time.
func (s *Service) runDeadlineApproaching(ctx context.Context) error {
	now := s.clock.Now()
	rows, err := s.queries().ListWorkOrdersApproachingDeadlineToNotify(ctx, sqlcgen.ListWorkOrdersApproachingDeadlineToNotifyParams{
		AfterCutoff:  pgtype.Date{Time: PastDeadlineCutoff(now), Valid: true},
		BeforeCutoff: pgtype.Date{Time: DeadlineApproachingCutoff(now), Valid: true},
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
			q := sqlcgen.New(tx)
			// The IS NULL guard makes the stamp idempotent, so a repeat pass over
			// the same row (two overlapping instances) warns only once.
			if err := q.MarkDeadlineWarnSent(ctx, sqlcgen.MarkDeadlineWarnSentParams{
				ID:                 row.ID,
				DeadlineWarnSentAt: tstz(now),
			}); err != nil {
				return err
			}
			// Both parties are reminded the delivery deadline is near (FR-051).
			link := "/work-orders/" + uuidString(row.ID)
			for _, account := range [2]pgtype.UUID{row.BuyerAccount, row.SubcontractorAccount} {
				if err := s.notifier.Enqueue(ctx, tx, account,
					sqlcgen.EventTypeDeadlineApproaching,
					"Tenggat pengiriman mendekat",
					"Tenggat pengiriman pesanan tinggal beberapa hari lagi. Pastikan produksi berjalan sesuai rencana.",
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
