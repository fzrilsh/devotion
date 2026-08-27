-- A dispute's status tracks where it sits in the mediation flow (reported ->
-- in_mediation -> resolved); its result records what the admin decided when they
-- closed it. The two are distinct: a resolved dispute always has a status of
-- 'resolved', but the outcome may be that the order was cancelled, that mediation
-- continued the order along its production chain, or that the admin forced the
-- order confirmed. The result cannot be derived from the order's later status,
-- because a 'continued' order proceeds and can itself reach 'confirmed', becoming
-- indistinguishable from an admin-forced 'confirmed' close. So the outcome is
-- stored explicitly, the same way allocation_reversed and liable_party_id already
-- are, rather than inferred from side effects (data-model.md, contract Dispute).
CREATE TYPE dispute_result AS ENUM ('cancelled', 'continued', 'confirmed');

-- NULL until the dispute is resolved, mirroring allocation_reversed and
-- liable_party_id which are also NULL before resolution. No DEFAULT: the value is
-- written by the resolve transaction from the injected Clock's decision, together
-- with resolved_at, handled_by, and the confirm_warn_sent_at reset.
ALTER TABLE dispute ADD COLUMN result dispute_result;

-- A resolved dispute must carry its result, extending the resolution_complete
-- invariant so a dispute can never close with a contract field left null. The
-- other four conditions are unchanged from 000012.
ALTER TABLE dispute DROP CONSTRAINT resolution_complete;
ALTER TABLE dispute ADD CONSTRAINT resolution_complete CHECK (
    status <> 'resolved'
    OR (handled_by IS NOT NULL AND resolved_at IS NOT NULL
        AND allocation_reversed IS NOT NULL AND admin_note IS NOT NULL
        AND result IS NOT NULL)
);
