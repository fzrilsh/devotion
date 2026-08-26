-- confirm_warn_sent_at records the moment the FR-069 "auto-confirm deadline
-- approaching" notice was sent to the buyer for a shipped order. It is the dedup
-- marker for the in-process ticker (research.md R-07 layer 2): the scan for
-- orders needing the warning filters on confirm_warn_sent_at IS NULL, so the
-- 5-minute ticker warns each order once across the 2-day lead rather than on
-- every tick. It stays NULL for orders that reach the auto-confirm deadline
-- inside one tick interval without ever entering the warning window.
ALTER TABLE work_order ADD COLUMN confirm_warn_sent_at timestamptz;
