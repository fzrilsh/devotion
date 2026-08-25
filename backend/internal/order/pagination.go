package order

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// pagination is the keyset Pagination body shared by the work-order list.
// NextCursor is opaque: the client passes it back verbatim and never parses it,
// so the stable ordering guarantee across pages is the server's alone.
type pagination struct {
	HasNext    bool    `json:"has_next"`
	NextCursor *string `json:"next_cursor"`
}

// cursor is the decoded keyset position: the (created_at, id) of the last row of
// the previous page. It is serialized opaquely; the client never sees its shape.
type cursor struct {
	created pgtype.Timestamptz
	id      pgtype.UUID
}

// cursorPayload is the on-the-wire cursor before base64: an RFC3339 timestamp and
// a uuid string. Encoding it as JSON then base64url keeps the token opaque while
// staying trivially reversible on the server. Mirrors notification's cursor so
// order keeps no dependency on that package.
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
