package notification

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// pageLimit caps one page of the notification feed. Keyset pagination on
// (created_at DESC, id DESC) keeps the order stable across pages even as new
// notifications arrive, so a fixed page size is enough; the client follows
// next_cursor until has_next is false.
const pageLimit = 20

// Register wires the four notification routes. All sit behind RequireAuth: a
// notification feed is per-account, so every route needs an authenticated
// caller but no particular business role. Gating them (rather than Public)
// keeps them out of the router's uncovered set, so the coverage test that
// forbids an unguarded /api route still passes.
func (s *Service) Register(r *httpx.Router) {
	auth := httpx.RequireAuth(s.auth)
	r.Gated("GET /api/notifications", auth, s.handleList)
	r.Gated("POST /api/notifications/{notificationId}/read", auth, s.handleMarkRead)
	r.Gated("GET /api/notifications/preferences", auth, s.handleGetPreferences)
	r.Gated("PUT /api/notifications/preferences", auth, s.handlePutPreferences)
}

// notificationBody is the Notification response body. work_order_id is backed by
// the DB link column (a generic deep link); the field name follows the contract.
type notificationBody struct {
	NotificationID string  `json:"notification_id"`
	Event          string  `json:"event"`
	Title          string  `json:"title"`
	Body           string  `json:"body"`
	WorkOrderID    *string `json:"work_order_id"`
	Read           bool    `json:"read"`
	CreatedAt      string  `json:"created_at"`
}

// notificationList is the NotificationList response body.
type notificationList struct {
	Items       []notificationBody `json:"items"`
	UnreadCount int64              `json:"unread_count"`
	Pagination  pagination         `json:"pagination"`
}

// pagination is the keyset Pagination body. NextCursor is opaque: the client
// passes it back verbatim and never parses it.
type pagination struct {
	HasNext    bool    `json:"has_next"`
	NextCursor *string `json:"next_cursor"`
}

// preferencesBody is the NotificationPreferences body. Only the non-transactional
// channels are configurable (FR-054); transactional notifications ignore them.
type preferencesBody struct {
	NonTransactional channelPrefs `json:"non_transactional"`
}

type channelPrefs struct {
	Email    bool `json:"email"`
	Whatsapp bool `json:"whatsapp"`
}

// principalAccount pulls the authenticated account out of the request context.
// The route is gated, so a missing principal is an invariant violation (500),
// matching how the account handlers treat it.
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

// handleList returns the caller's notifications newest first, one keyset page at
// a time, plus the total unread count (FR-051, FR-055). unread=true filters to
// still-unread rows; cursor resumes after the last row of the previous page. An
// unparseable cursor is treated as no cursor (first page) rather than an error,
// so a stale client cannot wedge the feed.
func (s *Service) handleList(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}

	unreadOnly := r.URL.Query().Get("unread") == "true"

	params := sqlcgen.ListNotificationsParams{
		AccountID:  acc.ID,
		UnreadOnly: unreadOnly,
		PageLimit:  pageLimit + 1, // one extra row detects a next page
	}
	if cur, ok := decodeCursor(r.URL.Query().Get("cursor")); ok {
		params.BeforeCreated = cur.created
		params.BeforeID = cur.id
	}

	rows, err := s.queries().ListNotifications(r.Context(), params)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}

	hasNext := len(rows) > pageLimit
	if hasNext {
		rows = rows[:pageLimit]
	}

	unread, err := s.queries().CountUnreadNotifications(r.Context(), acc.ID)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}

	items := make([]notificationBody, 0, len(rows))
	for _, row := range rows {
		items = append(items, notificationBody{
			NotificationID: uuidString(row.ID),
			Event:          string(row.Event),
			Title:          row.Title,
			Body:           row.Body,
			WorkOrderID:    textPtr(row.Link),
			Read:           row.ReadAt.Valid,
			CreatedAt:      row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	page := pagination{HasNext: hasNext}
	if hasNext {
		last := rows[len(rows)-1]
		c := encodeCursor(cursor{created: last.CreatedAt, id: last.ID})
		page.NextCursor = &c
	}

	writeJSON(w, http.StatusOK, notificationList{Items: items, UnreadCount: unread, Pagination: page})
}

// handleMarkRead stamps read_at on one notification the caller owns (FR-055).
// A malformed id or one that names no notification the caller owns affects zero
// rows, which becomes a 404, so the endpoint never confirms a notification the
// caller cannot see. Re-marking an already-read notification still matches one
// row (idempotent 204).
func (s *Service) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}

	id, ok := parseUUID(r.PathValue("notificationId"))
	if !ok {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Notifikasi tidak ditemukan.")
		return
	}

	n, err := s.queries().MarkNotificationRead(r.Context(), sqlcgen.MarkNotificationReadParams{
		ID:        id,
		AccountID: acc.ID,
		ReadAt:    tstz(s.clock.Now()),
	})
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	if n == 0 {
		httpx.WriteProblem(w, httpx.CodeNotFound, "Notifikasi tidak ditemukan.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetPreferences returns the caller's two non-transactional channel
// toggles (FR-054).
func (s *Service) handleGetPreferences(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	prefs, err := s.queries().GetNotifPreferences(r.Context(), acc.ID)
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, preferencesBody{
		NonTransactional: channelPrefs{Email: prefs.NotifNontxEmail, Whatsapp: prefs.NotifNontxWhatsapp},
	})
}

// handlePutPreferences writes the two non-transactional channel toggles and
// returns the stored values. Transactional notifications ignore these flags, so
// only the non-transactional channels are affected (FR-091).
func (s *Service) handlePutPreferences(w http.ResponseWriter, r *http.Request) {
	acc, ok := principalAccount(w, r)
	if !ok {
		return
	}
	var body preferencesBody
	if !decodeJSON(w, r, &body) {
		return
	}
	prefs, err := s.queries().UpdateNotifPreferences(r.Context(), sqlcgen.UpdateNotifPreferencesParams{
		ID:                 acc.ID,
		NotifNontxEmail:    body.NonTransactional.Email,
		NotifNontxWhatsapp: body.NonTransactional.Whatsapp,
		UpdatedAt:          tstz(s.clock.Now()),
	})
	if err != nil {
		httpx.WriteInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, preferencesBody{
		NonTransactional: channelPrefs{Email: prefs.NotifNontxEmail, Whatsapp: prefs.NotifNontxWhatsapp},
	})
}

// cursor is the decoded keyset position: the (created_at, id) of the last row of
// the previous page. It is serialized opaquely; the client never sees its shape.
type cursor struct {
	created pgtype.Timestamptz
	id      pgtype.UUID
}

// cursorPayload is the on-the-wire cursor before base64: an RFC3339 timestamp and
// a uuid string. Encoding it as JSON then base64url keeps the token opaque while
// staying trivially reversible on the server.
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

// writeJSON encodes v as a 2xx body. Error bodies go through httpx.WriteProblem.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// decodeJSON reads a JSON request body into dst, rejecting unknown fields and
// oversized payloads. On any error it writes a validation problem and returns
// false, matching the account package's request decoding.
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
