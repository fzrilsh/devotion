// Package search ranks published capacity listings for a buyer against the four
// hard criteria (FR-022..FR-026) and returns one keyset page (FR-080). It is a
// read package: the SearchCandidates query only reads, and the sole write, the
// horizon extension for passing candidates (FR-088), happens in a separate
// transaction outside that query so a search request stays a read. The
// dependency runs search -> listing (search calls EnsureHorizon), never the
// reverse; search never imports internal/order.
package search

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// HorizonExtender materializes a listing's calendar up to a week, raising its
// horizon_until. listing.Service satisfies it; search depends on this narrow
// interface rather than the whole listing service so the two packages stay
// loosely coupled and the extension can be stubbed in a test. The method runs
// inside the caller's transaction (FR-088: the extension is a write kept out of
// the read query, in its own transaction).
type HorizonExtender interface {
	EnsureHorizon(ctx context.Context, tx pgx.Tx, listingID pgtype.UUID, untilWeek time.Time) (time.Time, error)
}

// Service owns GET /api/search. It holds the pool, the injected Clock (Rule 5:
// the search date comes from the Clock, never time.Now), and the horizon
// extender it calls for passing candidates. It carries no per-request state.
type Service struct {
	pool    *pgxpool.Pool
	clock   platform.Clock
	horizon HorizonExtender
}

// New builds a Service over pool. clock supplies the search date so a test
// drives "today" instead of reading the wall clock; horizon is the listing
// service whose EnsureHorizon search calls for passing candidates (FR-088).
func New(pool *pgxpool.Pool, clock platform.Clock, horizon HorizonExtender) *Service {
	return &Service{pool: pool, clock: clock, horizon: horizon}
}

// queries returns a Queries bound to the pool for the read query. The horizon
// extension binds its own Queries to its transaction inside EnsureHorizon.
func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }
