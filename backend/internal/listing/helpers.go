package listing

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// maxBodyBytes caps a request body. Listing payloads are small (a capacity, a
// lead time, a handful of item ids, up to 26 periods); this stops a client from
// streaming an unbounded body into the decoder.
const maxBodyBytes = 64 << 10

// writeJSON encodes v as the 2xx body. Error bodies go through httpx.WriteProblem;
// this is only for the success payloads the handlers own.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// decodeJSON reads a JSON body into dst, rejecting unknown fields and oversized
// bodies. It returns false and writes a validation problem on failure, so a
// handler can `if !decodeJSON(...) { return }`.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		httpx.WriteProblem(w, httpx.CodeValidationFailed, "Format permintaan tidak sah.")
		return false
	}
	return true
}

// uuidString renders a pgtype.UUID as canonical text, empty when not valid.
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

// tstz wraps a time as a pgtype.Timestamptz for event columns.
func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// pgdate wraps a time as a pgtype.Date for week_start and horizon columns. Only
// the calendar date matters; the time of day is dropped by the date type.
func pgdate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

// isNoRows reports whether err is pgx.ErrNoRows, so a "no listing yet" lookup is
// a 404 rather than a 500.
func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// principalAccount pulls the authenticated UserAccount off the request context.
// The route is gated, so a missing Principal or a wrong Account type is an
// invariant violation and becomes a 500. It is a free function, not a method,
// because the listing package holds no account service. The bool is false when
// it already wrote the 500, so the handler returns early.
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

// itoa32 renders an int32 as base-10 text for the Indonesian 409 details, kept
// here so the conflict messages read as plain string concatenation.
func itoa32(n int32) string { return strconv.FormatInt(int64(n), 10) }

// uniqueUUIDs parses each string to a pgtype.UUID and drops duplicates and
// unparseable values, preserving first-seen order. The service compares the
// result count to CountActiveCatalogItemsOfType to catch unknown ids.
func uniqueUUIDs(ss []string) []pgtype.UUID {
	seen := make(map[string]struct{}, len(ss))
	out := make([]pgtype.UUID, 0, len(ss))
	for _, s := range ss {
		u, ok := parseUUID(s)
		if !ok {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, u)
	}
	return out
}

// allocatedInt64 coerces the ListPeriodsInRange.Allocated column to int64. sqlc
// types it as interface{} because it cannot infer the COALESCE(sum(...)::bigint)
// result, so the driver hands back an int64, an int32, or nil; anything else
// reads as zero.
func allocatedInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case nil:
		return 0
	default:
		return 0
	}
}
