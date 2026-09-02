package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// seeder holds the pool, the injected clock, and the demo region and catalog
// ids it looks up once. Every timestamp it writes is derived from clock.Now(),
// never time.Now(), so a test drives the fixtures to a fixed instant and the
// deadline-bound rows land on a known side of their windows (Rule 5, T075).
type seeder struct {
	pool      *pgxpool.Pool
	clock     platform.Clock
	now       time.Time
	thisWeek  time.Time
	cityCode  string
	productID string
	machineID string
	phoneSeq  int
	fileSeq   int
}

// seedTestData creates the full demo fixture set inside the given pool and
// returns the counts for the summary line. It first clears the fixture tables so
// a second run does not collide on the fixed .test emails, the same idempotency
// seed:regions and seed:master-data give through upserts. It then ensures the
// region and catalog rows it needs exist (idempotent), and builds each scenario
// in turn.
func seedTestData(ctx context.Context, pool *pgxpool.Pool, clock platform.Clock) (seedResult, error) {
	if err := resetTestData(ctx, pool); err != nil {
		return seedResult{}, err
	}
	s := &seeder{
		pool:     pool,
		clock:    clock,
		now:      clock.Now(),
		thisWeek: platform.WeekStart(clock.Now()),
	}
	if err := s.ensurePrerequisites(ctx); err != nil {
		return seedResult{}, err
	}

	var res seedResult
	if err := s.seedGenericBusinesses(ctx, &res); err != nil {
		return seedResult{}, err
	}
	if err := s.seedShowcaseListings(ctx, &res); err != nil {
		return seedResult{}, err
	}
	if err := s.seedStaleCalendar(ctx, &res); err != nil {
		return seedResult{}, err
	}
	if err := s.seedExpiredRequest(ctx, &res); err != nil {
		return seedResult{}, err
	}
	if err := s.seedShippedAndLateOrders(ctx, &res); err != nil {
		return seedResult{}, err
	}
	if err := s.seedVerifications(ctx, &res); err != nil {
		return seedResult{}, err
	}
	if err := s.seedFewOrdersBusiness(ctx, &res); err != nil {
		return seedResult{}, err
	}
	return res, nil
}
