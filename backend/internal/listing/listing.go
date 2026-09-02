// Package listing owns capacity listings and their weekly availability
// calendar. A subcontractor publishes one listing carrying a single weekly
// capacity number (FR-012/FR-014/FR-076); the calendar is a run of weekly
// periods, each stored as the Monday date beginning its week. This package
// never imports internal/order: order changes listing capacity through
// allocation rows, so the dependency runs order -> listing, never the reverse.
package listing

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

const (
	// InitialHorizonWeeks is how far ahead CreateListing seeds the calendar. It
	// must satisfy two requirements at once: at least 13 periods (FR-088) and at
	// least three months ahead (FR-017). The longest three calendar months span
	// 92 days, so 84 or 91 days could fall short on some start dates; 14 weeks is
	// 98 days (15 rows counting the current week), clearing both.
	InitialHorizonWeeks = 14

	// MaxPeriodBatch caps how many periods one PUT /listing/me/periods request may
	// touch, matching the contract's maxItems and bounding a single transaction.
	MaxPeriodBatch = 26
)

// Notifier queues an in-app notification (and its channel rows) inside the
// caller's transaction. listing depends on this narrow interface rather than
// the whole notification service so the two packages stay loosely coupled and
// the enqueue can be recorded in a test. notification.Service satisfies it, and
// it mirrors quota.Notifier so both write packages queue the same way.
type Notifier interface {
	Enqueue(ctx context.Context, tx pgx.Tx, accountID pgtype.UUID, event sqlcgen.EventType, title, body string, link *string) error
}

// Service owns the listing and calendar endpoints. It holds the pool and the
// Clock; the Clock supplies every timestamp and the current week's Monday (Rule
// 5), never time.Now. It also holds the notifier for the stale-calendar
// reminder job (FR-021). It carries no per-request state.
type Service struct {
	pool     *pgxpool.Pool
	clock    platform.Clock
	notifier Notifier
}

// New builds a Service over pool. clock is injected so a test drives the
// calendar by advancing time rather than reading the wall clock; notifier
// queues the FR-021 stale-calendar reminder inside the marking transaction.
func New(pool *pgxpool.Pool, clock platform.Clock, notifier Notifier) *Service {
	return &Service{pool: pool, clock: clock, notifier: notifier}
}

// queries returns a Queries bound to the pool for a standalone statement.
// Transactional work binds its own Queries to the pgx.Tx instead.
func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }
