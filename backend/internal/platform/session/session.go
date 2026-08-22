// Package session issues and validates opaque login sessions. A session is a
// 32-byte random token delivered only in the devotion_session cookie; the
// database stores its SHA-256 hash, never the token itself, so a database read
// cannot reconstruct a usable session. Lookups happen on every authenticated
// request, so the hash is a plain SHA-256 (fast, constant-length) rather than
// bcrypt: the token already carries full entropy and does not need a slow KDF.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// CookieName is the session cookie. It carries the opaque token, never the
// account id or any claim: the token is a lookup key, and everything else is
// read from the row it points to.
const CookieName = "devotion_session"

// TTL is the session lifetime. Each authenticated request slides it forward
// (rolling renewal), so an active user is never logged out while a token that
// has gone quiet for a week expires on its own.
const TTL = 7 * 24 * time.Hour

// tokenBytes is the raw token length before base64url encoding. 32 bytes is
// 256 bits of entropy, well past any guessing budget.
const tokenBytes = 32

// ErrNotFound is returned when no live session matches a token (absent,
// expired, or logged out). Callers translate it into a 401, never a 500.
var ErrNotFound = errors.New("session: tidak ditemukan")

// Session is a validated live session and the account it belongs to.
type Session struct {
	ID        pgtype.UUID
	AccountID pgtype.UUID
	ExpiresAt time.Time
}

// Store creates, validates, renews, and deletes sessions. Time comes from the
// injected Clock so expiry is testable without waiting.
type Store struct {
	pool   *pgxpool.Pool
	clock  platform.Clock
	secure bool
}

// New returns a Store. secure controls the cookie Secure attribute: it stays
// true everywhere except when configuration explicitly opts into plain HTTP for
// local development, and that exception is logged loudly by the caller.
func New(pool *pgxpool.Pool, clock platform.Clock, secure bool) *Store {
	return &Store{pool: pool, clock: clock, secure: secure}
}

// Issue mints a new token, stores its hash for accountID, and returns the raw
// token to place in the cookie. sourceAddr is recorded for audit; a nil value
// stores NULL.
func (s *Store) Issue(ctx context.Context, accountID pgtype.UUID, sourceAddr *netip.Addr) (string, error) {
	raw, err := newToken()
	if err != nil {
		return "", err
	}
	now := s.clock.Now()
	_, err = sqlcgen.New(s.pool).CreateSession(ctx, sqlcgen.CreateSessionParams{
		AccountID:     accountID,
		TokenHash:     hashToken(raw),
		SourceAddress: sourceAddr,
		ExpiresAt:     tstz(now.Add(TTL)),
		CreatedAt:     tstz(now),
	})
	if err != nil {
		return "", err
	}
	return raw, nil
}

// Validate looks up the live session for a raw token and slides its expiry
// forward. An absent, expired, or deleted session returns ErrNotFound. The
// expiry check is in SQL, so an expired row is treated as absent.
func (s *Store) Validate(ctx context.Context, raw string) (Session, error) {
	now := s.clock.Now()
	q := sqlcgen.New(s.pool)
	row, err := q.GetSessionByTokenHash(ctx, sqlcgen.GetSessionByTokenHashParams{
		TokenHash: hashToken(raw),
		ExpiresAt: tstz(now),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	expires := now.Add(TTL)
	if err := q.RenewSession(ctx, sqlcgen.RenewSessionParams{
		ID:         row.ID,
		ExpiresAt:  tstz(expires),
		AccessedAt: tstz(now),
	}); err != nil {
		return Session{}, err
	}
	return Session{ID: row.ID, AccountID: row.AccountID, ExpiresAt: expires}, nil
}

// Revoke deletes the session for a raw token, backing logout. Deleting an
// already-absent token is a no-op, so a double logout is harmless.
func (s *Store) Revoke(ctx context.Context, raw string) error {
	return sqlcgen.New(s.pool).DeleteSession(ctx, hashToken(raw))
}

// RevokeOthers deletes every session for the account except the one behind raw.
// Recovery confirmation uses it to end all other sessions after a password
// reset without logging the current caller out mid-request.
func (s *Store) RevokeOthers(ctx context.Context, accountID pgtype.UUID, raw string) error {
	return sqlcgen.New(s.pool).DeleteOtherSessions(ctx, sqlcgen.DeleteOtherSessionsParams{
		AccountID: accountID,
		TokenHash: hashToken(raw),
	})
}

// RevokeAll deletes every session for the account, used when no caller session
// is retained.
func (s *Store) RevokeAll(ctx context.Context, accountID pgtype.UUID) error {
	return sqlcgen.New(s.pool).DeleteAllSessions(ctx, accountID)
}

// SetCookie writes the session cookie holding the raw token. httpOnly keeps it
// out of JavaScript, SameSite=Lax blocks cross-site sends, and Secure is set
// unless the store was built for plain-HTTP local development.
func (s *Store) SetCookie(w http.ResponseWriter, raw string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(TTL / time.Second),
	})
}

// ClearCookie expires the session cookie in the browser, so logout removes it
// even before the next request.
func (s *Store) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// TokenFromRequest returns the raw token from the session cookie, or "" and
// false when the cookie is absent.
func TokenFromRequest(r *http.Request) (string, bool) {
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	return c.Value, true
}

// newToken returns a base64url-encoded 32-byte random token.
func newToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken returns the SHA-256 of a raw token. Only this hash is stored, so a
// database read cannot recover the token.
func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
