CREATE TABLE quota_request (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id       uuid NOT NULL REFERENCES business_profile(id) ON DELETE CASCADE,
    product_item_id uuid NOT NULL REFERENCES catalog_item(id) ON DELETE RESTRICT,
    quantity       integer NOT NULL,
    material       text NOT NULL,
    deadline       date NOT NULL,
    note           text,
    reply_due_at   timestamptz NOT NULL,
    created_at     timestamptz NOT NULL,

    CONSTRAINT request_quantity_positive CHECK (quantity > 0),
    CONSTRAINT reply_due_after_created CHECK (reply_due_at > created_at)
);

CREATE INDEX idx_request_buyer ON quota_request (buyer_id, created_at DESC);
CREATE INDEX idx_request_due ON quota_request (reply_due_at);

CREATE TYPE candidate_status AS ENUM (
    'awaiting_reply', 'offered', 'rejected', 'expired', 'not_continued', 'agreed'
);

CREATE TABLE request_candidate (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id        uuid NOT NULL REFERENCES quota_request(id) ON DELETE CASCADE,
    listing_id        uuid NOT NULL REFERENCES capacity_listing(id) ON DELETE RESTRICT,
    subcontractor_id  uuid NOT NULL REFERENCES business_profile(id) ON DELETE RESTRICT,
    status            candidate_status NOT NULL DEFAULT 'awaiting_reply',
    rejection_reason  text,
    updated_at        timestamptz NOT NULL,

    CONSTRAINT one_candidate_per_request UNIQUE (request_id, listing_id)
);

CREATE INDEX idx_candidate_subcon ON request_candidate (subcontractor_id, status);
CREATE INDEX idx_candidate_request ON request_candidate (request_id);
CREATE UNIQUE INDEX idx_one_agreement_per_request
    ON request_candidate (request_id) WHERE status = 'agreed';

CREATE FUNCTION reject_self_request() RETURNS trigger AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM quota_request r
        WHERE r.id = NEW.request_id AND r.buyer_id = NEW.subcontractor_id
    ) THEN
        RAISE EXCEPTION 'FR-083: quota request cannot be sent to your own listing';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_reject_self_request
    BEFORE INSERT OR UPDATE ON request_candidate
    FOR EACH ROW EXECUTE FUNCTION reject_self_request();

CREATE TYPE offer_party AS ENUM ('subcontractor', 'buyer');

CREATE TABLE offer (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id        uuid NOT NULL REFERENCES request_candidate(id) ON DELETE CASCADE,
    sequence            integer NOT NULL,
    proposed_by         offer_party NOT NULL,
    total_price         bigint NOT NULL,
    readiness_lead_days integer NOT NULL,
    note                text,
    created_at          timestamptz NOT NULL,

    CONSTRAINT sequence_unique_per_candidate UNIQUE (candidate_id, sequence),
    CONSTRAINT price_positive CHECK (total_price > 0),
    CONSTRAINT offer_lead_reasonable CHECK (readiness_lead_days >= 0 AND readiness_lead_days <= 365)
);

CREATE INDEX idx_offer_candidate ON offer (candidate_id, sequence);
