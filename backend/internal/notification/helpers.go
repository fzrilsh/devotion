package notification

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// tstz wraps a time in a valid pgtype.Timestamptz for the generated params.
func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// textOrNull maps an optional string to pgtype.Text: a nil pointer is SQL NULL,
// which is how a notification with no deep link stores its link column.
func textOrNull(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// pgText wraps a non-null string in a valid pgtype.Text, used for a channel's
// last_error where an empty string is a legitimate (if unlikely) value distinct
// from SQL NULL.
func pgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// uuidString renders a pgtype.UUID as canonical text, empty when not valid. It
// mirrors the helper in account/masterdata so the response bodies serialize uuids
// the same way across packages.
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

// textPtr maps a pgtype.Text back to an optional string: an invalid (NULL) value
// becomes nil, so the notification.link column surfaces as a null work_order_id.
func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

// parseUUID parses a canonical uuid string into a pgtype.UUID. It reports ok
// false on any unparseable input, which the handler turns into a 404 rather than
// a 500: a malformed id in the path names no notification.
func parseUUID(s string) (pgtype.UUID, bool) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, false
	}
	return u, u.Valid
}
