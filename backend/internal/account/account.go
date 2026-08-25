// Package account owns the authentication surface: registration, six-digit
// email and phone verification, login and logout, password recovery, and the
// two self-service endpoints GET /me and PATCH /me/roles. It sits on top of the
// session store (opaque cookie sessions), the rate limiter (login and OTP
// budgets), and the generated queries. Passwords are bcrypt digests; the
// plaintext verification codes are delivered out of band and only their SHA-256
// hashes are stored, so a database read reveals neither a password nor a code.
package account

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
	"github.com/fzrilsh/devotion/backend/internal/platform/ratelimit"
	"github.com/fzrilsh/devotion/backend/internal/platform/session"
)

// bcryptCost is fixed at 10 (the plan's chosen cost): strong enough for the
// 2GB box while keeping login latency bounded, since login runs the rate limit
// before ever reaching the compare.
const bcryptCost = 10

// codeTTL is how long a six-digit verification or recovery code stays valid.
// No source document pins this, so it is chosen here: long enough to arrive by
// email or WhatsApp and be typed, short enough that a leaked code is stale
// fast. Expiry is set from the Clock and checked in Go, never with a DB
// default, so tests move the clock instead of waiting.
const codeTTL = 15 * time.Minute

// deliverTimeout bounds a single out-of-band code send. The send runs detached
// from the request context (R-09), so it needs its own ceiling: a hung SMTP or
// WhatsApp dial cannot leak a goroutine forever.
const deliverTimeout = 30 * time.Second

// CodeDelivery hands a freshly minted plaintext code to its out-of-band
// channel. The account service never persists the plaintext; it stores only the
// hash and passes the plaintext here exactly once. Delivery failures must not
// fail the request that issued the code (resend and recover are best-effort by
// contract), so callers log and move on. The concrete email/WhatsApp senders
// arrive with the notification work; a nil delivery is treated as "do not send"
// so early wiring and tests stay simple.
type CodeDelivery interface {
	SendEmailCode(ctx context.Context, email, code string) error
	SendPhoneCode(ctx context.Context, phone, code string) error
	SendRecoveryCode(ctx context.Context, email, code string) error
}

// Service carries the dependencies every handler needs. It holds no per-request
// state, so one instance is shared across the server.
type Service struct {
	pool     *pgxpool.Pool
	clock    platform.Clock
	sessions *session.Store
	limiter  *ratelimit.Limiter
	delivery CodeDelivery
	log      *slog.Logger
	// devMode logs the plaintext verification code to slog so local development
	// can read a code that is otherwise only stored as a hash. It must never be
	// true in production: the code would leak into the log stream.
	devMode bool
}

// New builds a Service. delivery may be nil, in which case issued codes are
// stored but not sent anywhere (useful before the notification channels exist
// and in tests that read the hash directly). log may be nil (the CLI admin
// subcommand does not send codes); a nil logger falls back to slog.Default so a
// send failure is never silently dropped. devMode must be false in production.
func New(pool *pgxpool.Pool, clock platform.Clock, sessions *session.Store, limiter *ratelimit.Limiter, delivery CodeDelivery, log *slog.Logger, devMode bool) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{pool: pool, clock: clock, sessions: sessions, limiter: limiter, delivery: delivery, log: log, devMode: devMode}
}

// queries returns a Queries bound to the pool for a standalone statement.
func (s *Service) queries() *sqlcgen.Queries { return sqlcgen.New(s.pool) }

// myAccount is the MyAccount response body from openapi.yaml. ProfileID is a
// pointer so it serializes to null when the account has no business profile
// yet, matching the nullable contract field.
type myAccount struct {
	AccountID     string    `json:"account_id"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	EmailVerified bool      `json:"email_verified"`
	PhoneVerified bool      `json:"phone_verified"`
	Roles         rolesBody `json:"roles"`
	ProfileID     *string   `json:"profile_id"`
	IsAdmin       bool      `json:"is_admin"`
}

type rolesBody struct {
	Subcontractor bool `json:"subcontractor"`
	Buyer         bool `json:"buyer"`
}

// buildMyAccount assembles the MyAccount body, resolving the optional business
// profile id. A missing profile is not an error; it is a null profile_id.
func (s *Service) buildMyAccount(ctx context.Context, acc sqlcgen.UserAccount) (myAccount, error) {
	out := myAccount{
		AccountID:     uuidString(acc.ID),
		Email:         acc.Email,
		Phone:         acc.Phone,
		EmailVerified: acc.EmailVerified,
		PhoneVerified: acc.PhoneVerified,
		Roles:         rolesBody{Subcontractor: acc.RoleSubcontractor, Buyer: acc.RoleBuyer},
		IsAdmin:       acc.RoleAdmin,
	}
	profileID, err := s.queries().GetProfileIDByAccount(ctx, acc.ID)
	if err != nil {
		if isNoRows(err) {
			return out, nil
		}
		return myAccount{}, err
	}
	id := uuidString(profileID)
	out.ProfileID = &id
	return out, nil
}

// hashPassword returns the bcrypt digest of a plaintext password.
func hashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// passwordMatches reports whether plain matches the stored bcrypt digest.
func passwordMatches(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// uuidString renders a pgtype.UUID as canonical text.
func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b, _ := u.MarshalJSON()
	// MarshalJSON wraps the value in quotes; strip them.
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return ""
}

func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
