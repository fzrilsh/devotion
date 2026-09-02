CREATE TYPE rate_limit_target AS ENUM ('login_account', 'otp_phone', 'otp_address', 'quota_request');

CREATE TABLE rate_limit (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target       rate_limit_target NOT NULL,
    key          text NOT NULL,
    window_start timestamptz NOT NULL,
    count        integer NOT NULL DEFAULT 1,

    CONSTRAINT one_row_per_key_window UNIQUE (target, key, window_start),
    CONSTRAINT count_positive CHECK (count > 0)
);

CREATE INDEX idx_rate_limit_cleanup ON rate_limit (window_start);
