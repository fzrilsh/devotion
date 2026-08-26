// Package reputation owns reviews: writing one on a finished work order and
// reading the reviews a business profile has received (FR-047 to FR-050). The
// derived figures (average rating, completion rate) are not computed here as a
// second implementation; they come from the SearchReputation query the search
// page already uses, so a profile and a search result can never show different
// numbers for the same business (FR-071, FR-073).
//
// The dependency runs account -> reputation -> sqlcgen. reputation never imports
// account. It does import order for the auto-confirm predicate: whether a
// shipped order already counts as confirmed is one rule with one implementation
// (order.IsAutoConfirmDue), and a review must be accepted on an order that reads
// as confirmed on the order pages (FR-068). order does not import reputation, so
// the direction stays acyclic.
package reputation

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// reviewPageLimit caps one page of a profile's reviews. The keyset query asks for
// one extra row to detect whether a next page exists (FR-048).
const reviewPageLimit = 20

// Service owns the review write and the public review list. It holds the pool
// and the injected Clock (Rule 5: created_at comes from the Clock, never
// time.Now). It carries no per-request state.
type Service struct {
	pool  *pgxpool.Pool
	clock platform.Clock
}

// New builds a Service over pool. clock stamps created_at so a test drives "now"
// instead of reading the wall clock, which the auto-confirm precondition depends
// on being controllable.
func New(pool *pgxpool.Pool, clock platform.Clock) *Service {
	return &Service{pool: pool, clock: clock}
}

// queries returns a Queries bound to the pool for the read paths. The write path
// binds its own Queries to its transaction inside WithTx.
func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }

// Register wires the package's routes. The write is gated to the two business
// roles (only a party to an order can review it, and the handler further guards
// party membership); the list is public, matching the contract, because a
// prospective buyer must be able to read a subcontractor's reputation before
// signing in.
func (s *Service) Register(r *httpx.Router, auth httpx.Authenticator) {
	gate := httpx.RequireRole(auth, httpx.RoleBuyer, httpx.RoleSubcontractor)
	r.Gated("POST /api/work-orders/{workOrderId}/reviews", gate, s.handleCreateReview)
	r.Public("GET /api/profile/{profileId}/reviews", s.handleListReviews)
}
