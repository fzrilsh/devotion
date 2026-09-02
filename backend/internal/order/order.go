// Package order owns work orders: their formation from an accepted offer and the
// capacity allocation that backs them. It changes listing capacity by inserting
// allocation rows and raising used_capacity, so the dependency runs
// order -> listing, never the reverse (CLAUDE.md). The allocation transaction
// follows research.md R-04: lock every candidate period ascending by week_start,
// sum the remaining capacity, then fill the earliest weeks first. The row lock
// order is the deadlock preventer, not tidy housekeeping.
package order

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// Notifier queues an in-app notification (and its channel rows) inside the
// caller's transaction. order depends on this narrow interface rather than the
// whole notification service, matching quota, so the enqueue stays inside the
// agreement transaction and a test can record it. notification.Service satisfies
// it.
type Notifier interface {
	Enqueue(ctx context.Context, tx pgx.Tx, accountID pgtype.UUID, event sqlcgen.EventType, title, body string, link *string) error
}

// HorizonEnsurer extends a listing's calendar so every week up to a target
// exists before the accept path locks the range (FR-088). listing.Service
// satisfies it. order takes this narrow interface, not the whole listing
// service, so the dependency stays one-directional and testable.
type HorizonEnsurer interface {
	EnsureHorizon(ctx context.Context, tx pgx.Tx, listingID pgtype.UUID, untilWeek time.Time) (time.Time, error)
}

// Service owns the accept endpoint. It holds the pool, the injected Clock (Rule
// 5: the agreement instant comes from the Clock, never time.Now), the notifier
// it calls for the losing candidates and the two parties, and the horizon
// ensurer it calls to grow the calendar before allocating. It carries no
// per-request state.
type Service struct {
	pool     *pgxpool.Pool
	clock    platform.Clock
	notifier Notifier
	horizon  HorizonEnsurer
}

// New builds a Service over pool. clock supplies the agreement instant so a test
// drives "now"; notifier queues the agreement_formed notifications inside the
// formation transaction; horizon grows the listing calendar up to the deadline
// week before the periods are locked.
func New(pool *pgxpool.Pool, clock platform.Clock, notifier Notifier, horizon HorizonEnsurer) *Service {
	return &Service{pool: pool, clock: clock, notifier: notifier, horizon: horizon}
}

// queries returns a Queries bound to the pool for the read paths. The write path
// binds its own Queries to its transaction inside WithTx.
func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }
