package notification

import (
	"context"
	"testing"

	"github.com/fzrilsh/devotion/backend/internal/db/sqlcgen"
)

// channelRows reads the (channel, status, attempts) of every channel row for the
// harness account, ordered by channel, for asserting delivery outcomes.
func (h *harness) channelRows(t *testing.T) []sqlcgen.NotificationChannel {
	t.Helper()
	rows, err := h.pool.Query(context.Background(),
		`SELECT id, notification_id, channel, status, attempts, last_error, attempted_at, sent_at
		 FROM notification_channel ORDER BY channel`)
	if err != nil {
		t.Fatalf("query channels: %v", err)
	}
	defer rows.Close()
	var out []sqlcgen.NotificationChannel
	for rows.Next() {
		var c sqlcgen.NotificationChannel
		if err := rows.Scan(&c.ID, &c.NotificationID, &c.Channel, &c.Status, &c.Attempts,
			&c.LastError, &c.AttemptedAt, &c.SentAt); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, c)
	}
	return out
}

// enqueue runs Enqueue inside a real transaction and commits, mirroring how a
// domain package calls it inside its event transaction.
func (h *harness) enqueue(t *testing.T, event sqlcgen.EventType) {
	t.Helper()
	ctx := context.Background()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := h.svc.Enqueue(ctx, tx, h.acc, event, "judul", "isi", nil); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// TestEnqueue_TransactionalQueuesBothChannels proves a transactional event
// queues email and WhatsApp unconditionally, ignoring the account's toggles, so
// a core notification cannot be silenced. FR-091.
func TestEnqueue_TransactionalQueuesBothChannels_FR091(t *testing.T) {
	h := newHarness(t, "notif_enq_tx")
	// Disable both non-transactional toggles: a transactional event still fans out.
	_, err := h.pool.Exec(context.Background(),
		`UPDATE user_account SET notif_nontx_email = false, notif_nontx_whatsapp = false WHERE id = $1`, h.acc)
	if err != nil {
		t.Fatalf("update prefs: %v", err)
	}

	h.enqueue(t, sqlcgen.EventTypeOrderStatusChanged)

	rows := h.channelRows(t)
	if len(rows) != 2 {
		t.Fatalf("kanal = %d, mau 2 (email + whatsapp)", len(rows))
	}
}

// TestEnqueue_NonTransactionalHonorsPreferences proves a non-transactional event
// queues only the channels the account left enabled. FR-091, FR-054.
func TestEnqueue_NonTransactionalHonorsPreferences_FR091_FR054(t *testing.T) {
	h := newHarness(t, "notif_enq_nontx")
	// Leave only email enabled.
	_, err := h.pool.Exec(context.Background(),
		`UPDATE user_account SET notif_nontx_email = true, notif_nontx_whatsapp = false WHERE id = $1`, h.acc)
	if err != nil {
		t.Fatalf("update prefs: %v", err)
	}

	h.enqueue(t, sqlcgen.EventTypeCalendarStale)

	rows := h.channelRows(t)
	if len(rows) != 1 {
		t.Fatalf("kanal = %d, mau 1 (hanya email)", len(rows))
	}
	if rows[0].Channel != sqlcgen.NotificationChannelTypeEmail {
		t.Fatalf("kanal = %s, mau email", rows[0].Channel)
	}
}

// TestEnqueue_InAppPersistsRegardlessOfChannels proves the in-app notification is
// written even when a non-transactional event has every channel disabled: the
// feed is always visible. FR-054.
func TestEnqueue_InAppPersists_FR054(t *testing.T) {
	h := newHarness(t, "notif_enq_inapp")
	_, err := h.pool.Exec(context.Background(),
		`UPDATE user_account SET notif_nontx_email = false, notif_nontx_whatsapp = false WHERE id = $1`, h.acc)
	if err != nil {
		t.Fatalf("update prefs: %v", err)
	}

	h.enqueue(t, sqlcgen.EventTypeRatingRequest)

	var n int
	_ = h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM notification WHERE account_id = $1`, h.acc).Scan(&n)
	if n != 1 {
		t.Fatalf("notifikasi in-app = %d, mau 1 meski semua kanal mati", n)
	}
	if len(h.channelRows(t)) != 0 {
		t.Fatal("mau nol kanal saat preferensi mematikan keduanya")
	}
}

// TestDeliver_Success marks a channel sent when the transport succeeds, and
// leaves the in-app notification untouched. FR-086, FR-054.
func TestDeliver_Success_FR086(t *testing.T) {
	h := newHarness(t, "notif_deliver_ok")
	h.svc.email = &okSender{}
	h.enqueueEmailOnly(t)

	h.runDeliver(t)

	rows := h.channelRows(t)
	if len(rows) != 1 || rows[0].Status != sqlcgen.DeliveryStatusSent {
		t.Fatalf("status = %v, mau sent", rows)
	}
	if rows[0].Attempts != 1 {
		t.Fatalf("attempts = %d, mau 1", rows[0].Attempts)
	}
}

// TestDeliver_ThreeFailuresThenPermanent proves a channel whose transport keeps
// failing is retried up to three times and then marked failed_permanent, and the
// in-app notification is never affected. FR-085.
func TestDeliver_ThreeFailuresThenPermanent_FR085(t *testing.T) {
	h := newHarness(t, "notif_deliver_fail")
	h.svc.email = errSender{}
	h.enqueueEmailOnly(t)

	// Three ticks: pending, pending, failed_permanent.
	for i := 1; i <= 3; i++ {
		h.runDeliver(t)
		rows := h.channelRows(t)
		if rows[0].Attempts != int16(i) {
			t.Fatalf("tick %d: attempts = %d, mau %d", i, rows[0].Attempts, i)
		}
		want := sqlcgen.DeliveryStatusPending
		if i == 3 {
			want = sqlcgen.DeliveryStatusFailedPermanent
		}
		if rows[0].Status != want {
			t.Fatalf("tick %d: status = %s, mau %s", i, rows[0].Status, want)
		}
	}

	// A fourth tick claims nothing (attempts < 3 guard), so it stays permanent.
	h.runDeliver(t)
	if got := h.channelRows(t)[0].Attempts; got != 3 {
		t.Fatalf("attempts setelah exhaust = %d, mau tetap 3", got)
	}

	// The in-app notification survived every failed send.
	var n int
	_ = h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM notification WHERE account_id = $1`, h.acc).Scan(&n)
	if n != 1 {
		t.Fatalf("notifikasi in-app = %d, mau 1 meski pengiriman gagal", n)
	}
}

// enqueueEmailOnly writes one transactional notification with only the email
// channel, so a delivery test drives a single channel through the job.
func (h *harness) enqueueEmailOnly(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	var notifID any
	err := h.pool.QueryRow(ctx,
		`INSERT INTO notification (account_id, event, transactional, title, body, created_at)
		 VALUES ($1, 'order_status_changed', true, 'j', 'b', $2) RETURNING id`,
		h.acc, baseTime).Scan(&notifID)
	if err != nil {
		t.Fatalf("seed notif: %v", err)
	}
	_, err = h.pool.Exec(ctx,
		`INSERT INTO notification_channel (notification_id, channel, status, attempts)
		 VALUES ($1, 'email', 'pending', 0)`, notifID)
	if err != nil {
		t.Fatalf("seed channel: %v", err)
	}
}

// runDeliver acquires a pooled connection (the job expects the advisory-locked
// connection) and runs one delivery pass.
func (h *harness) runDeliver(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer conn.Release()
	if err := h.svc.deliver(ctx, conn); err != nil {
		t.Fatalf("deliver: %v", err)
	}
}
