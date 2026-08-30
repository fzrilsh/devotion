package quota

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/scheduler"
)

// RequestExpireJob is layer 2 of the two-layer request-expiry handling
// (research.md R-07): the in-process ticker that moves a candidate whose 72-hour
// reply window has lapsed to expired and tells the buyer why the request closed
// (FR-037). It is registered by serve under LockKeyRequestExpire, so only one
// instance runs during a deploy rollover. The lapse boundary is the request's
// stored reply_due_at compared to the injected Clock, so the window figure lives
// at write time and this layer only compares, matching order.IsRequestExpired.
func (s *Service) RequestExpireJob() scheduler.Job {
	return scheduler.Job{
		Name:    "quota:request-expire",
		LockKey: scheduler.LockKeyRequestExpire,
		Run: func(ctx context.Context, _ *pgxpool.Conn) error {
			return s.runRequestExpire(ctx)
		},
	}
}

// runRequestExpire does one pass: read the candidates still awaiting a reply
// whose reply window has lapsed, then expire each in its own transaction so one
// candidate's failure neither rolls back the others nor aborts the pass. now
// comes from the injected Clock (Rule 5); the cutoff is that instant, matching
// IsRequestExpired's inclusive boundary. ExpireCandidate reports the rows it
// changed, so the buyer is notified only for a candidate this pass actually
// expired: a candidate the subcontractor answered or a race already expired
// between the scan and the update is a no-op and sends no notice.
func (s *Service) runRequestExpire(ctx context.Context) error {
	now := s.clock.Now()
	rows, err := s.queries().ListCandidatesToExpire(ctx, tstz(now))
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
			q := sqlcgen.New(tx)
			affected, err := q.ExpireCandidate(ctx, sqlcgen.ExpireCandidateParams{
				ID:        row.CandidateID,
				UpdatedAt: tstz(now),
			})
			if err != nil {
				return err
			}
			if affected == 0 {
				return nil
			}
			link := "/quota-requests/" + uuidString(row.RequestID)
			return s.notifier.Enqueue(ctx, tx, row.BuyerAccount,
				sqlcgen.EventTypeRequestExpired,
				"Permintaan kuota kedaluwarsa",
				"Batas waktu balasan permintaan kuota Anda telah lewat tanpa penawaran. Anda dapat mengirim permintaan baru ke listing lain.",
				&link)
		}); err != nil {
			return err
		}
	}
	return nil
}
