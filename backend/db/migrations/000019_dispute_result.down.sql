-- Reverse 000019: restore resolution_complete to its 000012 form (without the
-- result condition), drop the result column, then drop the enum type.
ALTER TABLE dispute DROP CONSTRAINT resolution_complete;
ALTER TABLE dispute ADD CONSTRAINT resolution_complete CHECK (
    status <> 'resolved'
    OR (handled_by IS NOT NULL AND resolved_at IS NOT NULL
        AND allocation_reversed IS NOT NULL AND admin_note IS NOT NULL)
);
ALTER TABLE dispute DROP COLUMN result;
DROP TYPE dispute_result;
