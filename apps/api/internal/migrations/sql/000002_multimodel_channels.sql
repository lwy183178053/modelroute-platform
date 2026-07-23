ALTER TABLE merchant_channel_reviews
    ADD COLUMN channel_type integer NOT NULL DEFAULT 1,
    ADD COLUMN last_synced_at timestamptz;

ALTER TABLE merchant_channel_reviews
    ALTER COLUMN channel_test_id DROP NOT NULL;

ALTER TABLE merchant_channels
    ADD COLUMN channel_type integer NOT NULL DEFAULT 1,
    ADD COLUMN last_synced_at timestamptz;

ALTER TABLE merchant_channels
    DROP CONSTRAINT merchant_channels_status_check,
    ADD CONSTRAINT merchant_channels_status_check
        CHECK (status IN ('active', 'disabled', 'archived'));

CREATE TABLE merchant_review_model_bindings (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    review_id bigint NOT NULL REFERENCES merchant_channel_reviews(id) ON DELETE CASCADE,
    upstream_model_name varchar(255) NOT NULL,
    canonical_model_id bigint REFERENCES canonical_models(id),
    match_status text NOT NULL CHECK (match_status IN ('matched', 'unmatched')),
    test_status text NOT NULL DEFAULT 'untested'
        CHECK (test_status IN ('untested', 'testing', 'passed', 'failed')),
    tested_by_newapi_user_id bigint,
    tested_at timestamptz,
    test_latency_ms integer,
    test_error varchar(500),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (review_id, upstream_model_name)
);
CREATE INDEX merchant_review_model_bindings_review_idx
    ON merchant_review_model_bindings (review_id, match_status, upstream_model_name);

CREATE TABLE merchant_channel_model_bindings (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    merchant_channel_id bigint NOT NULL REFERENCES merchant_channels(id) ON DELETE CASCADE,
    upstream_model_name varchar(255) NOT NULL,
    canonical_model_id bigint REFERENCES canonical_models(id),
    match_status text NOT NULL CHECK (match_status IN ('matched', 'unmatched')),
    last_synced_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (merchant_channel_id, upstream_model_name)
);
CREATE INDEX merchant_channel_model_bindings_channel_idx
    ON merchant_channel_model_bindings (merchant_channel_id, match_status, upstream_model_name);
CREATE UNIQUE INDEX merchant_channel_model_bindings_one_model_idx
    ON merchant_channel_model_bindings (merchant_channel_id, canonical_model_id)
    WHERE canonical_model_id IS NOT NULL AND match_status = 'matched';

UPDATE canonical_models
SET status = 'archived', updated_at = now()
WHERE lower(provider) = 'minimax' AND status <> 'archived';

UPDATE offer_versions
SET status = 'inactive', retired_at = now()
WHERE canonical_model_id IN (SELECT id FROM canonical_models WHERE lower(provider) = 'minimax')
  AND status IN ('active', 'syncing');
