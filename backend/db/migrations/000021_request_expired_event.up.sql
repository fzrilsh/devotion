-- FR-037: a quota request no candidate replied to before its 72-hour window
-- lapses is marked expired, and the buyer is notified. The notice is its own
-- event so the buyer sees why the request closed, distinct from a candidate's
-- explicit rejection. ADD VALUE cannot run inside a transaction block that then
-- uses the new value, but golang-migrate runs each migration in its own
-- transaction and does not use the value here, so this is safe on its own.
ALTER TYPE event_type ADD VALUE IF NOT EXISTS 'request_expired';
