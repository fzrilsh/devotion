// Package ratelimit enforces the domain-data rate limits from research.md R-10:
// login attempts per account, one-time codes per number and per source
// address, and quota requests per user. State lives in the rate_limit table,
// never in memory, so a redeploy is not a way around a limit. Every timestamp
// comes from an injected Clock, so window-expiry tests move the clock instead
// of sleeping.
package ratelimit

import (
	"context"
	"hash/fnv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fzrilsh/devotion/backend/internal/db"
	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform"
)

// Target names one rate-limited action. The values match the
// rate_limit_target enum in migration 000014.
type Target string

const (
	// TargetLoginAccount limits password-guessing that hops between IPs.
	TargetLoginAccount Target = "login_account"
	// TargetOTPPhone limits repeated resend presses to one number.
	TargetOTPPhone Target = "otp_phone"
	// TargetOTPAddress limits how many distinct numbers one source address may
	// trigger codes to, so a number-cycler cannot provoke a WhatsApp block.
	TargetOTPAddress Target = "otp_address"
	// TargetQuotaRequest limits a single user flooding every subcontractor.
	TargetQuotaRequest Target = "quota_request"
)

// window describes one fixed-window limit: at most max events per window.
type window struct {
	max    int
	length time.Duration
}

// limits is the single source of the four R-10 windows. Nothing else may set
// a limit, so the numbers live in exactly one place.
var limits = map[Target]window{
	TargetLoginAccount: {max: 5, length: 15 * time.Minute},
	TargetOTPPhone:     {max: 3, length: time.Hour},
	TargetOTPAddress:   {max: 10, length: time.Hour},
	TargetQuotaRequest: {max: 20, length: time.Hour},
}

// Result is the outcome of a check. When Allowed is false, RetryAfter is the
// time until the current window rolls over, which the HTTP layer renders as a
// Retry-After header.
type Result struct {
	Allowed    bool
	RetryAfter time.Duration
}

// Limiter enforces the R-10 limits against the rate_limit table.
type Limiter struct {
	pool  *pgxpool.Pool
	clock platform.Clock
}

// New returns a Limiter backed by pool, reading time from clock.
func New(pool *pgxpool.Pool, clock platform.Clock) *Limiter {
	return &Limiter{pool: pool, clock: clock}
}

// Check counts one event against target for key (an account, number, or user
// id) and reports whether it is allowed. The increment and the comparison
// happen in one transaction; the ON CONFLICT row lock means two concurrent
// callers cannot both read the same count and both pass. otp_address must go
// through CheckAddress instead, which counts distinct numbers.
func (l *Limiter) Check(ctx context.Context, target Target, key string) (Result, error) {
	if target == TargetOTPAddress {
		panic("ratelimit: otp_address memakai CheckAddress, bukan Check")
	}
	lim := limits[target]
	now := l.clock.Now()
	start := windowStart(now, lim.length)

	var count int32
	err := db.WithTx(ctx, l.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		c, err := q.TouchRateLimit(ctx, sqlcgen.TouchRateLimitParams{
			Target:      sqlcgen.RateLimitTarget(target),
			Key:         key,
			WindowStart: tstz(start),
		})
		count = c
		return err
	})
	if err != nil {
		return Result{}, err
	}
	if int(count) > lim.max {
		return Result{Allowed: false, RetryAfter: retryAfter(start, lim.length, now)}, nil
	}
	return Result{Allowed: true}, nil
}

// CheckAddress enforces the per-address distinct-number limit. member (the
// phone number) is counted only once per window, so re-sending a code to a
// number already counted does not consume more of the address budget; only a
// new distinct number does. A transaction-scoped advisory lock per address
// serializes the count-then-record so two new numbers cannot both slip past
// the last slot.
func (l *Limiter) CheckAddress(ctx context.Context, address, member string) (Result, error) {
	lim := limits[TargetOTPAddress]
	now := l.clock.Now()
	start := windowStart(now, lim.length)
	memberKey := address + memberSep + member
	pattern := likeEscape(address) + memberSep + "%"

	var allowed bool
	err := db.WithTx(ctx, l.pool, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		if err := q.LockRateLimitKey(ctx, advisoryKey(address)); err != nil {
			return err
		}
		recorded, err := q.MemberRecorded(ctx, sqlcgen.MemberRecordedParams{
			Target:      sqlcgen.RateLimitTargetOtpAddress,
			Key:         memberKey,
			WindowStart: tstz(start),
		})
		if err != nil {
			return err
		}
		if recorded {
			allowed = true
			return nil
		}
		n, err := q.CountDistinctMembers(ctx, sqlcgen.CountDistinctMembersParams{
			Target:      sqlcgen.RateLimitTargetOtpAddress,
			Key:         pattern,
			WindowStart: tstz(start),
		})
		if err != nil {
			return err
		}
		if int(n) >= lim.max {
			allowed = false
			return nil
		}
		if err := q.RecordMember(ctx, sqlcgen.RecordMemberParams{
			Target:      sqlcgen.RateLimitTargetOtpAddress,
			Key:         memberKey,
			WindowStart: tstz(start),
		}); err != nil {
			return err
		}
		allowed = true
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	if !allowed {
		return Result{Allowed: false, RetryAfter: retryAfter(start, lim.length, now)}, nil
	}
	return Result{Allowed: true}, nil
}

// memberSep separates address from number in the composite key. The unit
// separator never appears in an IP address or phone number.
const memberSep = "\x1f"

// windowStart truncates now to the start of its fixed window. Truncate rounds
// on the absolute instant; for 15-minute and hourly windows that aligns to the
// same wall-clock boundaries in Asia/Jakarta, which has no DST.
func windowStart(now time.Time, length time.Duration) time.Time {
	return now.Truncate(length)
}

// retryAfter is the time until the current window ends.
func retryAfter(start time.Time, length time.Duration, now time.Time) time.Duration {
	d := start.Add(length).Sub(now)
	if d < 0 {
		return 0
	}
	return d
}

// tstz wraps a time for a timestamptz parameter.
func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// advisoryKey hashes an address to a stable int64 for pg_advisory_xact_lock.
func advisoryKey(address string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(address))
	return int64(h.Sum64())
}

// likeEscape escapes the LIKE metacharacters so an address is matched
// literally. Cloudflare-derived addresses do not contain these, but the prefix
// match must not silently widen if one ever does.
func likeEscape(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '%', '_':
			out = append(out, '\\')
		}
		out = append(out, s[i])
	}
	return string(out)
}
