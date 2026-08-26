package reputation

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// conflictError carries a stable machine code and the already-composed
// Indonesian detail a 4xx must quote. Copied from order rather than imported so
// the two packages keep separate error surfaces.
type conflictError struct {
	code   httpx.Code
	detail string
}

func (e *conflictError) Error() string { return e.detail }

// writeErr maps a service error to its problem response. A conflictError carries
// its own code and detail; anything else is an unexpected fault surfaced as 500.
func writeErr(w http.ResponseWriter, err error) {
	var cerr *conflictError
	if errors.As(err, &cerr) {
		httpx.WriteProblem(w, cerr.code, cerr.detail)
		return
	}
	httpx.WriteInternal(w)
}

// writeJSON encodes v as the 2xx body. Error bodies go through
// httpx.WriteProblem.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
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

// tstz wraps a time as a pgtype.Timestamptz for the created_at column.
func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// isNoRows reports whether err is pgx.ErrNoRows.
func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// isReviewDuplicate reports whether err is the unique violation on
// one_review_per_order_per_reviewer, the signal that this party already reviewed
// this order (FR-047). The application check runs first; this closes the race
// between two concurrent submissions, which the check alone cannot.
func isReviewDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == "one_review_per_order_per_reviewer"
}

// principalAccount pulls the authenticated UserAccount off the request context.
// The route is gated, so a missing Principal or a wrong Account type is an
// invariant violation and becomes a 500.
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
// cannot exhaust memory.
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

// pagination is the keyset Pagination body. NextCursor is opaque: the client
// passes it back verbatim and never parses it.
type pagination struct {
	HasNext    bool    `json:"has_next"`
	NextCursor *string `json:"next_cursor"`
}

// cursor is the decoded keyset position: the (created_at, id) of the last row of
// the previous page.
type cursor struct {
	created pgtype.Timestamptz
	id      pgtype.UUID
}

// cursorPayload is the on-the-wire cursor before base64: an RFC3339 timestamp
// and a uuid string.
type cursorPayload struct {
	Created string `json:"c"`
	ID      string `json:"i"`
}

// encodeCursor builds the opaque next_cursor from a row's keyset position.
func encodeCursor(c cursor) string {
	b, _ := json.Marshal(cursorPayload{
		Created: c.created.Time.Format(time.RFC3339Nano),
		ID:      uuidString(c.id),
	})
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeCursor reverses encodeCursor. It returns ok false on any malformed input
// so the caller falls back to the first page rather than erroring.
func decodeCursor(s string) (cursor, bool) {
	if strings.TrimSpace(s) == "" {
		return cursor{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return cursor{}, false
	}
	var p cursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return cursor{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, p.Created)
	if err != nil {
		return cursor{}, false
	}
	id, ok := parseUUID(p.ID)
	if !ok {
		return cursor{}, false
	}
	return cursor{created: tstz(t), id: id}, true
}
