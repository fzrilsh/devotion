package quota

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
// whose reply window has lapsed, grouped by request, then handle each request in
// its own transaction so one request's failure neither rolls back the others nor
// aborts the pass. now comes from the injected Clock (Rule 5); the cutoff is that
// instant, matching IsRequestExpired's inclusive boundary. Every lapsed candidate
// of the request is expired, but the buyer is told the request lapsed "tanpa
// penawaran" only once per request and only when no candidate made a standing
// offer (AS-7, FR-037): a request with one offer and one silent candidate expires
// the silent one without a false "no offers" notice, and a request all three
// ignored notifies the buyer a single time, not once per candidate.
func (s *Service) runRequestExpire(ctx context.Context) error {
	now := s.clock.Now()
	rows, err := s.queries().ListCandidatesToExpire(ctx, tstz(now))
	if err != nil {
		return err
	}
	// ListCandidatesToExpire is ordered by request id, so a run of rows sharing
	// a request id is contiguous; collapse each run to one request-level unit.
	for i := 0; i < len(rows); {
		requestID := rows[i].RequestID
		buyerAccount := rows[i].BuyerAccount
		j := i
		var candidateIDs []pgtype.UUID
		for j < len(rows) && rows[j].RequestID == requestID {
			candidateIDs = append(candidateIDs, rows[j].CandidateID)
			j++
		}
		i = j

		if err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
			q := sqlcgen.New(tx)
			expired := 0
			for _, candidateID := range candidateIDs {
				affected, err := q.ExpireCandidate(ctx, sqlcgen.ExpireCandidateParams{
					ID:        candidateID,
					UpdatedAt: tstz(now),
				})
				if err != nil {
					return err
				}
				expired += int(affected)
			}
			// Nothing this pass actually expired (a race or an already-answered
			// candidate): no notice. This also makes a repeat pass a no-op, so the
			// buyer hears about a lapsed request once, not once per tick.
			if expired == 0 {
				return nil
			}
			// A candidate that replied keeps the request alive; the "tanpa
			// penawaran" notice would be false, so it is suppressed (AS-7).
			hasOffer, err := q.RequestHasStandingOffer(ctx, requestID)
			if err != nil {
				return err
			}
			if hasOffer {
				return nil
			}
			link := "/quota-requests/" + uuidString(requestID)
			return s.notifier.Enqueue(ctx, tx, buyerAccount,
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
