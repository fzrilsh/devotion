package order

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// writeJSON encodes v as the 2xx body. Error bodies go through httpx.WriteProblem;
// this is only for the success payload the accept handler owns.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// uuidString renders a pgtype.UUID as canonical text, empty when not valid. It
// is copied from quota rather than imported so order keeps no dependency on that
// package.
func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b, _ := u.MarshalJSON()
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return ""
}

// parseUUID parses canonical UUID text into a pgtype.UUID, reporting validity.
func parseUUID(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, false
	}
	return u, u.Valid
}

// tstz wraps a time as a pgtype.Timestamptz for the event columns.
func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// pgdate wraps a time as a pgtype.Date for the week and deadline columns. Only
// the calendar date matters; the time of day is dropped by the date type.
func pgdate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

// isNoRows reports whether err is pgx.ErrNoRows.
func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// isUniqueViolation reports whether err is a Postgres unique-violation on the
// one-agreement-per-request partial index. That is the concurrency safety net of
// FR-036: the losing agreement of two racing accepts hits it and is turned into
// the CAPACITY_ALREADY_TAKEN reason.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == "idx_one_agreement_per_request"
}

// principalAccount pulls the authenticated UserAccount off the request context.
// The route is gated, so a missing Principal or a wrong Account type is an
// invariant violation and becomes a 500. The bool is false when it already wrote
// the 500, so the handler returns early.
func principalAccount(w http.ResponseWriter, r *http.Request) (sqlcgen.UserAccount, bool) {
	p, ok := httpx.PrincipalFromContext(r.Context())
	if !ok {
		httpx.WriteInternal(w)
		return sqlcgen.UserAccount{}, false
	}
	acc, ok := p.Account.(sqlcgen.UserAccount)
	if !ok {
		httpx.WriteInternal(w)
		return sqlcgen.UserAccount{}, false
	}
	return acc, true
}

// decodeJSON reads the request body into dst, rejecting an unknown field or a
// malformed body with VALIDATION_FAILED. The body is capped so a large payload
// cannot exhaust memory. It returns false when it already wrote the problem, so
// the handler returns early. Modeled on notification's decodeJSON.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Format permintaan tidak sah.")
		return false
	}
	return true
}

// itoa64 formats an int64 in base 10, for the capacity and quantity figures the
// Indonesian detail strings quote.
func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

// itoa32 formats an int32 in base 10.
func itoa32(n int32) string { return strconv.FormatInt(int64(n), 10) }
