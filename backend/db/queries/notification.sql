-- InsertNotification writes the in-platform notification row. It is always
-- called inside the triggering event's transaction (Enqueue takes a pgx.Tx),
-- so the notification is committed with the event or not at all (FR-086): a
-- rolled-back order change leaves no orphan notification. created_at comes from
-- the Clock, never a DB default (Rule 5).
-- name: InsertNotification :one
INSERT INTO notification (id, account_id, event, transactional, title, body, link, created_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- InsertNotificationChannel queues one external delivery channel for a
-- notification. The row starts pending with zero attempts; the delivery job
-- claims it later. The in-platform notification itself needs no channel row,
-- it is always visible (FR-054); these rows track only email and WhatsApp fan
-- out.
-- name: InsertNotificationChannel :exec
INSERT INTO notification_channel (id, notification_id, channel, status, attempts)
VALUES (gen_random_uuid(), $1, $2, 'pending', 0);

-- ListNotifications returns one account's notifications newest first, keyset
-- paginated on (created_at, id) so the order is stable across pages even as new
-- rows arrive. unread_only filters to still-unread rows (FR-051 list). A null
-- before_created is the first page; later pages pass the last row's cursor.
-- name: ListNotifications :many
SELECT * FROM notification
WHERE account_id = $1
  AND (NOT sqlc.arg(unread_only)::boolean OR read_at IS NULL)
  AND (
    sqlc.narg(before_created)::timestamptz IS NULL
    OR (created_at, id) < (sqlc.narg(before_created)::timestamptz, sqlc.narg(before_id)::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit)::int;

-- CountUnreadNotifications backs the unread_count field of NotificationList. It
-- rides the partial idx_notification_unread index.
-- name: CountUnreadNotifications :one
SELECT count(*) FROM notification WHERE account_id = $1 AND read_at IS NULL;

-- MarkNotificationRead stamps read_at on one notification the caller owns. The
-- account_id predicate is the ownership check: a caller marking another
-- account's id affects zero rows, which the handler turns into a 404, so the
-- endpoint never confirms a notification the caller cannot see. COALESCE keeps
-- the first read time, so re-marking an already-read notification still matches
-- one row (idempotent 204) instead of a false 404.
-- name: MarkNotificationRead :execrows
UPDATE notification
SET read_at = COALESCE(read_at, $3)
WHERE id = $1 AND account_id = $2;

-- ClaimPendingChannels returns channels still awaiting delivery, oldest first
-- (attempted_at NULLS FIRST puts never-tried rows ahead of retried ones), with
-- the recipient address and message text joined in so the delivery job needs no
-- second query. The attempts < 3 guard skips rows already exhausted; the
-- delivery job runs under an advisory lock, so no row lock is needed to keep two
-- instances off the same channel.
-- name: ClaimPendingChannels :many
SELECT
    nc.id            AS channel_id,
    nc.channel       AS channel,
    nc.attempts      AS attempts,
    n.title          AS title,
    n.body           AS body,
    ua.email         AS email,
    ua.phone         AS phone
FROM notification_channel nc
JOIN notification n  ON n.id = nc.notification_id
JOIN user_account ua ON ua.id = n.account_id
WHERE nc.status = 'pending' AND nc.attempts < 3
ORDER BY nc.attempted_at NULLS FIRST, nc.id
LIMIT $1;

-- MarkChannelSent records a successful delivery: status sent, attempts bumped to
-- reflect the try that worked, sent_at and attempted_at stamped from the Clock.
-- A claimed row had attempts <= 2, so the +1 stays within attempts_max_three.
-- name: MarkChannelSent :exec
UPDATE notification_channel
SET status = 'sent', attempts = attempts + 1, attempted_at = $2, sent_at = $2
WHERE id = $1;

-- MarkChannelFailed records a failed attempt: attempts bumped, last_error and
-- attempted_at stamped. The third failure (attempts reaching 3) flips status to
-- failed_permanent (FR-085); earlier failures stay pending for the next tick.
-- The CASE keeps status and attempts consistent with failed_after_three_attempts.
-- name: MarkChannelFailed :exec
UPDATE notification_channel
SET attempts     = attempts + 1,
    last_error   = $2,
    attempted_at = $3,
    status       = CASE WHEN attempts + 1 >= 3
                        THEN 'failed_permanent'::delivery_status
                        ELSE 'pending'::delivery_status END
WHERE id = $1;

-- GetNotifPreferences reads the two non-transactional channel toggles for one
-- account, backing GET /notifications/preferences (FR-054).
-- name: GetNotifPreferences :one
SELECT notif_nontx_email, notif_nontx_whatsapp
FROM user_account
WHERE id = $1;

-- UpdateNotifPreferences writes the two non-transactional channel toggles and
-- returns them, backing PUT /notifications/preferences. Transactional
-- notifications ignore these flags, so only the non-transactional channels are
-- affected (FR-091).
-- name: UpdateNotifPreferences :one
UPDATE user_account
SET notif_nontx_email = $2, notif_nontx_whatsapp = $3, updated_at = $4
WHERE id = $1
RETURNING notif_nontx_email, notif_nontx_whatsapp;
