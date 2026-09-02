package search

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// writeJSON encodes v as the 2xx body. Error bodies go through httpx.WriteProblem;
// this is only for the SearchResult payload the handler owns.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// uuidString renders a pgtype.UUID as canonical text, empty when not valid. It
// is copied from listing rather than imported so search keeps no dependency on
// that package beyond the HorizonExtender interface.
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

// isNoRows reports whether err is pgx.ErrNoRows, so an account with no business
// profile searches as if it has no own listings to exclude rather than erroring.
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

// cursor is the opaque keyset position: the five ordering columns of the last
// row of the previous page. It is JSON-encoded then base64url-wrapped so the
// client treats it as opaque and passes it back verbatim (FR-080). readiness
// lead is stored positive; the query negates it into cursor_neg_lead.
type cursor struct {
	Score     int32  `json:"s"`
	Remaining int64  `json:"r"`
	LeadDays  int32  `json:"l"`
	Name      string `json:"n"`
	Listing   string `json:"i"`
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
	if _, ok := parseUUID(c.Listing); !ok {
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

// strPtrText renders a plain string column as an optional string, nil when
// empty so an unset city_code serializes to null rather than "".
func strPtrText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// pgTextPtr renders a pgtype.Text as an optional string: NULL becomes nil so a
// city with no name join serializes to null.
func pgTextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

// dateString renders a pgtype.Date as YYYY-MM-DD, empty when NULL.
func dateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

// floatFromNumeric renders a pgtype.Numeric as an optional float64, nil for a
// NULL or unrepresentable value so average_rating serializes to null when there
// are no reviews to average.
func floatFromNumeric(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	v, err := n.Float64Value()
	if err != nil || !v.Valid {
		return nil
	}
	f := v.Float64
	return &f
}
