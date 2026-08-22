package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
	"github.com/fzrilsh/devotion/backend/internal/platform/httpx"
)

// seedNotification inserts one notification for the harness account at the given
// offset from baseTime and returns its id. read controls whether read_at is set.
func (h *harness) seedNotification(t *testing.T, offset time.Duration, event sqlcgen.EventType, read bool) pgtype.UUID {
	t.Helper()
	created := baseTime.Add(offset)
	var readAt any
	if read {
		readAt = created
	}
	var id pgtype.UUID
	err := h.pool.QueryRow(context.Background(),
		`INSERT INTO notification (account_id, event, transactional, title, body, created_at, read_at)
		 VALUES ($1, $2, true, 'judul', 'isi', $3, $4) RETURNING id`,
		h.acc, event, created, readAt).Scan(&id)
	if err != nil {
		t.Fatalf("seed notification: %v", err)
	}
	return id
}

// TestList_PaginatesAndCountsUnread proves the feed returns a keyset page newest
// first, reports has_next when a further page exists, and carries the total
// unread count independent of the page window. FR-051 (list), FR-055 (unread).
func TestList_PaginatesAndCountsUnread_FR051_FR055(t *testing.T) {
	h := newHarness(t, "notif_list")
	// 25 notifications, oldest first by offset; two marked read.
	for i := 0; i < 25; i++ {
		h.seedNotification(t, time.Duration(i)*time.Minute, sqlcgen.EventTypeOrderStatusChanged, i < 2)
	}

	rec := h.do("GET", "/api/notifications", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, mau 200", rec.Code)
	}
	var page notificationList
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(page.Items) != pageLimit {
		t.Fatalf("items = %d, mau %d", len(page.Items), pageLimit)
	}
	if !page.Pagination.HasNext || page.Pagination.NextCursor == nil {
		t.Fatal("mau has_next true dengan next_cursor")
	}
	if page.UnreadCount != 23 {
		t.Fatalf("unread_count = %d, mau 23", page.UnreadCount)
	}
	// Newest first: the first item is the highest offset (24 minutes). The API
	// renders created_at in WIB, so compare the instant, not the zoned string.
	first, err := time.Parse(time.RFC3339, page.Items[0].CreatedAt)
	if err != nil {
		t.Fatalf("parse created_at: %v", err)
	}
	if !first.Equal(baseTime.Add(24 * time.Minute)) {
		t.Fatalf("urutan salah, item pertama created_at = %s", page.Items[0].CreatedAt)
	}

	// Second page via the opaque cursor returns the remaining 5 rows.
	rec = h.do("GET", "/api/notifications?cursor="+*page.Pagination.NextCursor, nil)
	var page2 notificationList
	_ = json.Unmarshal(rec.Body.Bytes(), &page2)
	if len(page2.Items) != 5 {
		t.Fatalf("halaman dua items = %d, mau 5", len(page2.Items))
	}
	if page2.Pagination.HasNext {
		t.Fatal("halaman dua mau has_next false")
	}
}

// TestList_UnreadOnlyFilter proves unread=true drops rows already read. FR-051.
func TestList_UnreadOnlyFilter_FR051(t *testing.T) {
	h := newHarness(t, "notif_unread")
	h.seedNotification(t, 0, sqlcgen.EventTypeOrderStatusChanged, true)
	h.seedNotification(t, time.Minute, sqlcgen.EventTypeOrderStatusChanged, false)

	rec := h.do("GET", "/api/notifications?unread=true", nil)
	var page notificationList
	_ = json.Unmarshal(rec.Body.Bytes(), &page)
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, mau 1 (hanya yang belum dibaca)", len(page.Items))
	}
	if page.Items[0].Read {
		t.Fatal("item yang lolos filter unread masih read")
	}
}

// TestMarkRead_Idempotent proves marking a notification read is 204 and stays
// 204 on a repeat (COALESCE keeps the first read time). FR-055.
func TestMarkRead_Idempotent_FR055(t *testing.T) {
	h := newHarness(t, "notif_markread")
	id := h.seedNotification(t, 0, sqlcgen.EventTypeOrderStatusChanged, false)

	for i := 0; i < 2; i++ {
		rec := h.do("POST", "/api/notifications/"+uuidString(id)+"/read", nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("percobaan %d: status %d, mau 204", i, rec.Code)
		}
	}
}

// TestMarkRead_NotOwned_NotFound proves a notification belonging to another
// account is 404, never confirmed to a caller who does not own it. FR-055.
func TestMarkRead_NotOwned_NotFound_FR055(t *testing.T) {
	h := newHarness(t, "notif_markread_other")
	other := seedAccount(t, h.pool, "other@example.com", "628999888777")
	var id pgtype.UUID
	err := h.pool.QueryRow(context.Background(),
		`INSERT INTO notification (account_id, event, transactional, title, body, created_at)
		 VALUES ($1, 'order_status_changed', true, 'j', 'b', $2) RETURNING id`,
		other, baseTime).Scan(&id)
	if err != nil {
		t.Fatalf("seed other: %v", err)
	}

	rec := h.do("POST", "/api/notifications/"+uuidString(id)+"/read", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404", rec.Code)
	}
}

// TestMarkRead_MalformedID_NotFound proves an unparseable id in the path is a
// 404, not a 500: a bad id names no notification. FR-055.
func TestMarkRead_MalformedID_NotFound_FR055(t *testing.T) {
	h := newHarness(t, "notif_markread_bad")
	rec := h.do("POST", "/api/notifications/bukan-uuid/read", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, mau 404", rec.Code)
	}
}

// TestPreferences_GetThenPut proves the two non-transactional toggles round-trip
// through GET and PUT. FR-054.
func TestPreferences_GetThenPut_FR054(t *testing.T) {
	h := newHarness(t, "notif_prefs")

	rec := h.do("GET", "/api/notifications/preferences", nil)
	var got preferencesBody
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if !got.NonTransactional.Email || !got.NonTransactional.Whatsapp {
		t.Fatal("default preferensi mau kedua kanal aktif")
	}

	rec = h.do("PUT", "/api/notifications/preferences", preferencesBody{
		NonTransactional: channelPrefs{Email: true, Whatsapp: false},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status %d, mau 200", rec.Code)
	}
	var after preferencesBody
	_ = json.Unmarshal(rec.Body.Bytes(), &after)
	if !after.NonTransactional.Email || after.NonTransactional.Whatsapp {
		t.Fatalf("preferensi tersimpan = %+v, mau email true whatsapp false", after.NonTransactional)
	}
}

// TestList_NoSession_Unauthorized proves the feed rejects an unauthenticated
// caller, so a notification feed is never served without a principal. The gate
// runs before any query, so no database is needed. FR-051.
func TestList_NoSession_Unauthorized_FR051(t *testing.T) {
	svc := New(nil, nil, stubAuth{fail: true}, nil, nil)
	r := httpx.NewRouter(quietLogger())
	svc.Register(r)

	req := httptest.NewRequest("GET", "/api/notifications", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, mau 401", rec.Code)
	}
}
