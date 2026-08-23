package listing

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// EnsureHorizon makes sure every weekly period up to untilWeek exists, then
// raises horizon_until to cover it. It is idempotent and safe to call
// concurrently: duplicate weeks are prevented by one_period_per_week, horizon
// regression by GREATEST in RaiseHorizonUntil, and deadlock by the fixed lock
// order (the caller already holds the listing row lock). It returns the
// resulting horizon.
//
// It never touches total_capacity, used_capacity, or marked_full on existing
// rows, and never touches calendar_updated_at: a listing whose calendar the
// owner never edited must keep reading as stale so FR-021 keeps working.
func (s *Service) EnsureHorizon(ctx context.Context, tx pgx.Tx, listingID pgtype.UUID, untilWeek time.Time) (time.Time, error) {
	q := sqlcgen.New(tx)

	// The single place a week boundary is computed for the horizon, so
	// week_start_is_monday and horizon_is_monday cannot be violated structurally.
	untilWeek = platform.WeekStart(untilWeek)

	l, err := q.GetListingByID(ctx, listingID)
	if err != nil {
		return time.Time{}, err
	}
	now := s.clock.Now()
	weekNow := platform.WeekStart(now)

	horizon := l.HorizonUntil.Time
	if l.HorizonUntil.Valid {
		horizon = platform.WeekStart(l.HorizonUntil.Time)
	}

	// Nothing to do when the calendar already reaches untilWeek. The horizon
	// never moves backward.
	if !l.HorizonUntil.Valid || untilWeek.After(horizon) {
		// Series starts one week past the current horizon, but never before the
		// current week, so past weeks are never generated. When there is no
		// horizon yet, start at the current week.
		start := horizon.AddDate(0, 0, 7)
		if !l.HorizonUntil.Valid || start.Before(weekNow) {
			start = weekNow
		}
		if !start.After(untilWeek) {
			if err := q.InsertPeriodsUpToWeek(ctx, sqlcgen.InsertPeriodsUpToWeekParams{
				ListingID:     listingID,
				Column2:       pgdate(start),
				Column3:       pgdate(untilWeek),
				TotalCapacity: l.WeeklyCapacity,
				CreatedAt:     tstz(now),
			}); err != nil {
				return time.Time{}, err
			}
		}
		if err := q.RaiseHorizonUntil(ctx, sqlcgen.RaiseHorizonUntilParams{
			ID:           listingID,
			HorizonUntil: pgdate(untilWeek),
			UpdatedAt:    tstz(now),
		}); err != nil {
			return time.Time{}, err
		}
		return untilWeek, nil
	}
	return horizon, nil
}

// listPeriods returns the calendar of the caller's listing within [from, to].
// from defaults to the current week's Monday and to defaults to horizon_until,
// so an unparameterized request returns the whole live calendar. remaining is
// derived in Go: max(0, capacity - allocated), forced to zero when the week is
// marked full.
func (s *Service) listPeriods(ctx context.Context, accountID pgtype.UUID, from, to *time.Time) ([]periodView, error) {
	profileID, err := s.profileID(ctx, s.queries(), accountID)
	if err != nil {
		return nil, err
	}
	l, err := s.queries().GetListingByProfile(ctx, profileID)
	if err != nil {
		if isNoRows(err) {
			return nil, errListingNotFound
		}
		return nil, err
	}

	lo := platform.WeekStart(s.clock.Now())
	if from != nil {
		lo = platform.WeekStart(*from)
	}
	hi := l.HorizonUntil.Time
	if to != nil {
		hi = platform.WeekStart(*to)
	}

	rows, err := s.queries().ListPeriodsInRange(ctx, sqlcgen.ListPeriodsInRangeParams{
		ListingID:   l.ID,
		WeekStart:   pgdate(lo),
		WeekStart_2: pgdate(hi),
	})
	if err != nil {
		return nil, err
	}
	return periodViews(rows), nil
}

// updatePeriods validates the whole batch first, then writes it in one
// transaction, so a 26-element batch is never left half applied. The
// validation rules mirror the contract (PeriodUpdateItem): a Monday week_start
// no earlier than the current week and no more than 26 weeks ahead, capacity
// >= 0, no duplicate weeks. The transaction locks the listing, extends the
// horizon to the furthest week in the batch, then locks and upserts each period
// in ascending week order, and finally stamps calendar_updated_at (the sole
// path that advances it, FR-021).
func (s *Service) updatePeriods(ctx context.Context, accountID pgtype.UUID, items []periodInput) ([]periodView, error) {
	weekNow := platform.WeekStart(s.clock.Now())
	parsed, verr := s.validatePeriodInput(items, weekNow)
	if verr != nil {
		return nil, verr
	}

	// Ascending by week keeps the period lock order fixed within the batch.
	sort.Slice(parsed, func(i, j int) bool { return parsed[i].week.Before(parsed[j].week) })

	var out []periodView
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		profileID, err := s.profileID(ctx, q, accountID)
		if err != nil {
			return err
		}
		l, err := q.LockListingByProfile(ctx, profileID)
		if err != nil {
			if isNoRows(err) {
				return errListingNotFound
			}
			return err
		}

		furthest := parsed[len(parsed)-1].week
		if _, err := s.EnsureHorizon(ctx, tx, l.ID, furthest); err != nil {
			return err
		}

		now := s.clock.Now()
		for _, p := range parsed {
			period, err := q.LockPeriodByWeek(ctx, sqlcgen.LockPeriodByWeekParams{
				ListingID: l.ID,
				WeekStart: pgdate(p.week),
			})
			if err != nil {
				if isNoRows(err) {
					return errPeriodOutsideCal
				}
				return err
			}
			if p.capacity < period.UsedCapacity {
				return &conflictError{
					code:   httpx.CodeCapacityAlreadyAllocated,
					detail: "Kapasitas minggu " + platform.FormatDateID(p.week) + " tidak dapat diturunkan ke " + itoa32(p.capacity) + " potong. Sudah memakai " + itoa32(period.UsedCapacity) + " potong untuk pesanan berjalan.",
					week:   p.week,
					used:   period.UsedCapacity,
					want:   p.capacity,
				}
			}
			if p.markedFull {
				active, err := q.PeriodHasActiveAllocation(ctx, period.ID)
				if err != nil {
					return err
				}
				if active {
					return &conflictError{
						code:   httpx.CodePeriodAlreadyAllocated,
						detail: "Minggu " + platform.FormatDateID(p.week) + " tidak dapat ditandai penuh karena sudah memakai " + itoa32(period.UsedCapacity) + " potong untuk pesanan berjalan.",
						week:   p.week,
						used:   period.UsedCapacity,
						want:   p.capacity,
					}
				}
			}
			if err := q.UpsertPeriod(ctx, sqlcgen.UpsertPeriodParams{
				ListingID:     l.ID,
				WeekStart:     pgdate(p.week),
				TotalCapacity: p.capacity,
				MarkedFull:    p.markedFull,
				CreatedAt:     tstz(now),
			}); err != nil {
				return err
			}
		}

		if err := q.TouchCalendarUpdatedAt(ctx, sqlcgen.TouchCalendarUpdatedAtParams{
			ID:                l.ID,
			CalendarUpdatedAt: tstz(now),
		}); err != nil {
			return err
		}

		lo := parsed[0].week
		hi := parsed[len(parsed)-1].week
		rows, err := q.ListPeriodsInRange(ctx, sqlcgen.ListPeriodsInRangeParams{
			ListingID:   l.ID,
			WeekStart:   pgdate(lo),
			WeekStart_2: pgdate(hi),
		})
		if err != nil {
			return err
		}
		out = periodViews(rows)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
