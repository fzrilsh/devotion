package verification

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// queuePagination is the keyset Pagination body the admin verification queue
// returns. next_cursor is opaque: the client passes it back verbatim and never
// parses it, so the stable ordering guarantee across pages is the server's
// alone. It mirrors order's pagination shape rather than importing it, since
// verification keeps no dependency on that package.
type queuePagination struct {
	HasNext    bool    `json:"has_next"`
	NextCursor *string `json:"next_cursor"`
}

// queueCursor is the decoded keyset position: the (created_at, id) of the last
// row of the previous page. It is serialized opaquely; the client never sees its
// shape.
type queueCursor struct {
	created pgtype.Timestamptz
	id      pgtype.UUID
}

// queueCursorPayload is the on-the-wire cursor before base64: an RFC3339 stamp
// and a uuid string. JSON then base64url keeps the token opaque while staying
// trivially reversible on the server.
type queueCursorPayload struct {
	Created string `json:"c"`
	ID      string `json:"i"`
}

// encodeQueueCursor builds the opaque next_cursor from a row's keyset position.
func encodeQueueCursor(c queueCursor) string {
	b, _ := json.Marshal(queueCursorPayload{
		Created: c.created.Time.Format(time.RFC3339Nano),
		ID:      uuidString(c.id),
	})
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodeQueueCursor reverses encodeQueueCursor. It returns ok false on any
// malformed input so the caller falls back to the first page rather than
// erroring, matching the opaque-cursor contract.
func decodeQueueCursor(s string) (queueCursor, bool) {
	if strings.TrimSpace(s) == "" {
		return queueCursor{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return queueCursor{}, false
	}
	var p queueCursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return queueCursor{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, p.Created)
	if err != nil {
		return queueCursor{}, false
	}
	id, ok := parseUUID(p.ID)
	if !ok {
		return queueCursor{}, false
	}
	return queueCursor{created: pgtype.Timestamptz{Time: t, Valid: true}, id: id}, true
}
