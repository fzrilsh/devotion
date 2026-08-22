CREATE TYPE event_type AS ENUM (
    'request_received', 'offer_received', 'counter_offer', 'agreement_formed',
    'order_status_changed', 'payment_record', 'deadline_approaching', 'deadline_passed',
    'verification_decision', 'rating_request', 'order_cancelled',
    'confirmation_due_approaching', 'order_auto_closed', 'item_proposal_decision',
    'calendar_stale'
);

CREATE TABLE notification (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    uuid NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    event         event_type NOT NULL,
    transactional boolean NOT NULL,
    title         text NOT NULL,
    body          text NOT NULL,
    link          text,
    read_at       timestamptz,
    created_at    timestamptz NOT NULL
);

CREATE INDEX idx_notification_account ON notification (account_id, created_at DESC);
CREATE INDEX idx_notification_unread ON notification (account_id) WHERE read_at IS NULL;

CREATE TYPE notification_channel_type AS ENUM ('email', 'whatsapp');
CREATE TYPE delivery_status AS ENUM ('pending', 'sent', 'failed_permanent');

CREATE TABLE notification_channel (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id uuid NOT NULL REFERENCES notification(id) ON DELETE CASCADE,
    channel         notification_channel_type NOT NULL,
    status          delivery_status NOT NULL DEFAULT 'pending',
    attempts        smallint NOT NULL DEFAULT 0,
    last_error      text,
    attempted_at    timestamptz,
    sent_at         timestamptz,

    CONSTRAINT one_channel_per_notification UNIQUE (notification_id, channel),
    CONSTRAINT attempts_max_three CHECK (attempts >= 0 AND attempts <= 3),
    CONSTRAINT failed_after_three_attempts CHECK (
        status <> 'failed_permanent' OR attempts = 3
    )
);

CREATE INDEX idx_channel_queue ON notification_channel (attempted_at NULLS FIRST)
    WHERE status = 'pending';
