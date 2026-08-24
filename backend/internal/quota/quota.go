// Package quota owns the buyer's quota requests (FR-029, FR-030): sending one
// request to several candidate listings in a single action, and listing the
// buyer's own requests. It is a write package over quota_request and
// request_candidate. The 72-hour reply window (FR-082) is computed here from the
// injected Clock, never from time.Now, and the request table carries no column
// the buyer can set to control it. A request that targets the buyer's own
// listing is rejected before any insert (FR-083); the database trigger
// trg_reject_self_request is only the safety net.
package quota

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// replyWindowHours is the fixed reply window a candidate has before a request
// lapses (FR-082). It is applied to the injected Clock, so the deadline is
// deterministic in tests and identical across the two scheduler layers.
const replyWindowHours = 72

// Notifier queues an in-app notification (and its channel rows) inside the
// caller's transaction. quota depends on this narrow interface rather than the
// whole notification service so the two packages stay loosely coupled and the
// enqueue can be recorded in a test. notification.Service satisfies it.
type Notifier interface {
	Enqueue(ctx context.Context, tx pgx.Tx, accountID pgtype.UUID, event sqlcgen.EventType, title, body string, link *string) error
}

// Service owns POST and GET /api/quota-requests. It holds the pool, the injected
// Clock (Rule 5: created_at and reply_due_at both come from the Clock, never
// time.Now), and the notifier it calls for each candidate. It carries no
// per-request state.
type Service struct {
	pool     *pgxpool.Pool
	clock    platform.Clock
	notifier Notifier
}

// New builds a Service over pool. clock supplies both time columns so a test
// drives "now" instead of reading the wall clock; notifier queues the
// request_received notification for each candidate inside the request's
// transaction.
func New(pool *pgxpool.Pool, clock platform.Clock, notifier Notifier) *Service {
	return &Service{pool: pool, clock: clock, notifier: notifier}
}

// queries returns a Queries bound to the pool for the read paths. The write path
// binds its own Queries to its transaction inside WithTx.
func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }
