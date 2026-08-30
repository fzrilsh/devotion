package listing

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/scheduler"
)

// calendarStaleWindow is how long a published calendar may go untouched before
// its owner is reminded to refresh it (FR-021). It matches order.CalendarStaleWindow
// exactly, but is inlined here because listing never imports order (the dependency
// runs order -> listing). The search read layer applies the same seven-day window
// when it flags a candidate stale, so the reminder and the marker agree.
const calendarStaleWindow = 7 * 24 * time.Hour

// CalendarStaleJob is layer 2 of the two-layer stale-calendar handling
// (research.md R-07): the search read layer already flags a stale calendar for
// the searcher, and this in-process ticker sends the owner the FR-021 reminder to
// refresh it. It is registered by serve under LockKeyCalendarStale, so only one
// instance runs during a deploy rollover. The staleness cutoff is the shared
// seven-day window, so the reminder never fires on a calendar the search layer
// would still call fresh.
func (s *Service) CalendarStaleJob() scheduler.Job {
	return scheduler.Job{
		Name:    "listing:calendar-stale",
		LockKey: scheduler.LockKeyCalendarStale,
		Run: func(ctx context.Context, _ *pgxpool.Conn) error {
			return s.runCalendarStale(ctx)
		},
	}
}

// runCalendarStale does one pass: read the published listings whose calendar has
// gone untouched past the stale window and whose owner has not yet been reminded
// for the current episode, then handle each in its own transaction so one
// listing's failure neither rolls back the others nor aborts the pass. now comes
// from the injected Clock (Rule 5); the cutoff is now - calendarStaleWindow,
// matching the search layer's staleBefore. Each listing stamps stale_notified_at
// (guarded so a fresh owner edit re-arms it) and reminds the owner; there is no
// status change, because a stale calendar is a nudge, not a state transition.
func (s *Service) runCalendarStale(ctx context.Context) error {
	now := s.clock.Now()
	cutoff := now.Add(-calendarStaleWindow)
	rows, err := s.queries().ListStaleListingsToNotify(ctx, tstz(cutoff))
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
			q := sqlcgen.New(tx)
			// The re-arm guard makes the stamp idempotent for the current episode, so
			// a repeat pass over the same row (two overlapping instances) reminds once.
			if err := q.MarkStaleNotified(ctx, sqlcgen.MarkStaleNotifiedParams{
				ID:              row.ID,
				StaleNotifiedAt: tstz(now),
			}); err != nil {
				return err
			}
			link := "/listing/me"
			return s.notifier.Enqueue(ctx, tx, row.OwnerAccount,
				sqlcgen.EventTypeCalendarStale,
				"Perbarui kalender kapasitas Anda",
				"Kalender kapasitas Anda belum diperbarui lebih dari tujuh hari. Perbarui agar pencari melihat ketersediaan terkini.",
				&link)
		}); err != nil {
			return err
		}
	}
	return nil
}
