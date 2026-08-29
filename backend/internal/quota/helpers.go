package quota

import (
	"encoding/base64"
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

// maxBodyBytes caps a request body. A quota request payload is small (a handful
// of listing ids, a product id, a quantity, a material, a deadline, a short
// note); this stops a client from streaming an unbounded body into the decoder.
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

// uuidString renders a pgtype.UUID as canonical text, empty when not valid. It
// is copied from listing rather than imported so quota keeps no dependency on
// that package.
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

// textPtr maps a pgtype.Text back to an optional string: an invalid (NULL) value
// becomes nil, so an unset rejection_reason serializes as null.
func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

// pgdate wraps a time as a pgtype.Date for the deadline column. Only the
// calendar date matters; the time of day is dropped by the date type.
func pgdate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

// isNoRows reports whether err is pgx.ErrNoRows.
func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

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

// uniqueUUIDs parses each string to a pgtype.UUID and drops duplicates and
// unparseable values, preserving first-seen order. It reports whether every
// input string parsed, so the caller turns a malformed listing id into a 422
// rather than silently sending to fewer candidates than the buyer chose.
func uniqueUUIDs(ss []string) ([]pgtype.UUID, bool) {
	seen := make(map[string]struct{}, len(ss))
	out := make([]pgtype.UUID, 0, len(ss))
	allValid := true
	for _, s := range ss {
		u, ok := parseUUID(s)
		if !ok {
			allValid = false
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, u)
	}
	return out, allValid
}

// cursor is the opaque keyset position: the two ordering columns of the last row
// of the previous page (created_at, id). It is JSON-encoded then base64url-
// wrapped so the client treats it as opaque and passes it back verbatim
// (FR-080).
type cursor struct {
	CreatedAt time.Time `json:"c"`
	Request   string    `json:"i"`
}

// encodeCursor renders a cursor as an opaque base64url token.
func encodeCursor(c cursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeCursor parses an opaque token back into a cursor, reporting validity so
// a garbled cursor is a 422 rather than a silent reset to the first page.
func decodeCursor(s string) (cursor, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return cursor{}, false
	}
	var c cursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return cursor{}, false
	}
	if _, ok := parseUUID(c.Request); !ok {
		return cursor{}, false
	}
	return c, true
}

// atoiDefault parses a base-10 int, returning def when s is empty.
func atoiDefault(s string, def int) (int, bool) {
	if s == "" {
		return def, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
