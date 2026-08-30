-- Postgres cannot remove a value from an enum type. The down migration is a
-- no-op: request_expired stays in event_type. This is safe because a value the
-- code no longer emits simply goes unused; dropping and recreating the type
-- would require rewriting every event_type column and is never worth it for a
-- reversal. Documented so the asymmetry is not mistaken for an omission.
SELECT 1;
