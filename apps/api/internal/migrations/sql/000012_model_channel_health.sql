ALTER TABLE merchant_channels
    ADD COLUMN deleted_at timestamptz;

INSERT INTO merchant_channel_model_bindings
    (merchant_channel_id, upstream_model_name, canonical_model_id, match_status, last_synced_at)
SELECT channel.id, model.public_name, model.id, 'matched', COALESCE(channel.last_synced_at, now())
FROM merchant_channels channel
JOIN canonical_models model ON model.id = channel.canonical_model_id
WHERE NOT EXISTS (
    SELECT 1 FROM merchant_channel_model_bindings binding
    WHERE binding.merchant_channel_id = channel.id
      AND binding.canonical_model_id = model.id
);

ALTER TABLE merchant_channel_model_bindings
    ADD COLUMN health_state text NOT NULL DEFAULT 'pending'
        CHECK (health_state IN ('pending', 'healthy', 'cooling', 'error')),
    ADD COLUMN consecutive_request_failures integer NOT NULL DEFAULT 0
        CHECK (consecutive_request_failures >= 0),
    ADD COLUMN automatic_check_failures integer NOT NULL DEFAULT 0
        CHECK (automatic_check_failures BETWEEN 0 AND 10),
    ADD COLUMN next_check_at timestamptz,
    ADD COLUMN check_lease_until timestamptz,
    ADD COLUMN last_checked_at timestamptz,
    ADD COLUMN last_check_latency_ms integer,
    ADD COLUMN last_check_error varchar(500),
    ADD COLUMN last_status_code integer,
    ADD CONSTRAINT merchant_channel_model_bindings_next_check_ck
        CHECK ((health_state = 'cooling' AND next_check_at IS NOT NULL)
            OR (health_state <> 'cooling' AND next_check_at IS NULL));

UPDATE merchant_channel_model_bindings
SET health_state = 'pending'
WHERE match_status <> 'matched' OR canonical_model_id IS NULL;

CREATE INDEX merchant_channel_model_bindings_due_check_idx
    ON merchant_channel_model_bindings (next_check_at)
    WHERE health_state = 'cooling';

CREATE TABLE channel_model_health_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    channel_model_binding_id bigint NOT NULL
        REFERENCES merchant_channel_model_bindings(id) ON DELETE CASCADE,
    offer_channel_alias_id bigint REFERENCES offer_channel_aliases(id),
    source text NOT NULL CHECK (source IN ('request', 'automatic_check', 'manual_check')),
    outcome text NOT NULL CHECK (outcome IN ('success', 'ignored', 'temporary_failure', 'hard_failure')),
    status_code integer,
    first_byte_ms integer,
    request_id varchar(128),
    failure_summary varchar(500),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX channel_model_health_events_lookup_idx
    ON channel_model_health_events (channel_model_binding_id, created_at DESC);

ALTER TABLE merchant_applications
    DROP COLUMN initial_channel_review_id;

ALTER TABLE merchant_channels
    DROP COLUMN review_id,
    DROP COLUMN canonical_model_id,
    DROP COLUMN status;

DROP TABLE merchant_review_model_bindings;
DROP TABLE merchant_channel_reviews;
DROP TABLE merchant_channel_tests;
DROP TABLE channel_health_events;
DROP TABLE performance_rollups;
DROP TABLE channel_health;
