DROP INDEX IF EXISTS idx_order_deadline_warn;
ALTER TABLE work_order DROP COLUMN IF EXISTS deadline_warn_sent_at;
