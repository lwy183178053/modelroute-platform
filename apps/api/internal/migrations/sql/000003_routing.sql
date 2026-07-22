CREATE TABLE canonical_models (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_name varchar(255) NOT NULL UNIQUE,
    provider varchar(80) NOT NULL,
    protocol text NOT NULL CHECK (protocol IN ('chat_completions', 'responses', 'embeddings')),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    current_official_price_version_id bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE official_price_versions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    version integer NOT NULL CHECK (version > 0),
    billing_mode text NOT NULL CHECK (billing_mode IN ('token', 'fixed')),
    model_ratio_micros bigint,
    model_price_microusd bigint,
    completion_ratio_micros bigint NOT NULL CHECK (completion_ratio_micros >= 0),
    cache_ratio_micros bigint NOT NULL CHECK (cache_ratio_micros >= 0),
    create_cache_ratio_micros bigint NOT NULL CHECK (create_cache_ratio_micros >= 0),
    image_ratio_micros bigint NOT NULL CHECK (image_ratio_micros >= 0),
    audio_ratio_micros bigint NOT NULL CHECK (audio_ratio_micros >= 0),
    audio_completion_ratio_micros bigint NOT NULL CHECK (audio_completion_ratio_micros >= 0),
    published_by_newapi_user_id bigint NOT NULL,
    published_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (canonical_model_id, version),
    CHECK (
        (billing_mode = 'token' AND model_ratio_micros IS NOT NULL AND model_ratio_micros > 0 AND model_price_microusd IS NULL)
        OR
        (billing_mode = 'fixed' AND model_price_microusd IS NOT NULL AND model_price_microusd > 0 AND model_ratio_micros IS NULL)
    )
);

ALTER TABLE canonical_models
    ADD CONSTRAINT canonical_models_current_price_fk
    FOREIGN KEY (current_official_price_version_id) REFERENCES official_price_versions(id);

CREATE TABLE offer_versions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    merchant_channel_id bigint NOT NULL REFERENCES merchant_channels(id),
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    official_price_version_id bigint NOT NULL REFERENCES official_price_versions(id),
    previous_offer_version_id bigint REFERENCES offer_versions(id),
    multiplier_bps integer NOT NULL CHECK (multiplier_bps BETWEEN 1 AND 10000),
    internal_alias varchar(80) NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'syncing' CHECK (status IN ('syncing', 'active', 'inactive', 'sync_failed')),
    sync_error text,
    created_by_newapi_user_id bigint NOT NULL,
    activated_at timestamptz,
    retired_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX offer_versions_one_active_per_channel_model_idx
    ON offer_versions (merchant_channel_id, canonical_model_id)
    WHERE status = 'active';

CREATE UNIQUE INDEX offer_versions_one_syncing_per_channel_model_idx
    ON offer_versions (merchant_channel_id, canonical_model_id)
    WHERE status = 'syncing';

CREATE TABLE token_bindings (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    newapi_token_id bigint NOT NULL UNIQUE,
    newapi_user_id bigint NOT NULL,
    token_fingerprint char(64) NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    synced_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE routing_policies (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    token_binding_id bigint NOT NULL UNIQUE REFERENCES token_bindings(id) ON DELETE CASCADE,
    mode text NOT NULL DEFAULT 'smart' CHECK (mode IN ('smart', 'fixed')),
    priority text NOT NULL DEFAULT 'stability' CHECK (priority IN ('price', 'stability')),
    minimum_multiplier_bps integer NOT NULL DEFAULT 1 CHECK (minimum_multiplier_bps BETWEEN 1 AND 10000),
    maximum_multiplier_bps integer NOT NULL DEFAULT 10000 CHECK (maximum_multiplier_bps BETWEEN 1 AND 10000),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (minimum_multiplier_bps <= maximum_multiplier_bps)
);

CREATE TABLE routing_policy_merchants (
    routing_policy_id bigint NOT NULL REFERENCES routing_policies(id) ON DELETE CASCADE,
    merchant_profile_id bigint NOT NULL REFERENCES merchant_profiles(id),
    PRIMARY KEY (routing_policy_id, merchant_profile_id)
);

CREATE TABLE channel_model_health (
    merchant_channel_id bigint NOT NULL REFERENCES merchant_channels(id),
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'healthy', 'degraded', 'cooling', 'recovering', 'disabled')),
    consecutive_failures integer NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    cooldown_level integer NOT NULL DEFAULT 0 CHECK (cooldown_level >= 0),
    recovery_successes integer NOT NULL DEFAULT 0 CHECK (recovery_successes >= 0),
    observations bigint NOT NULL DEFAULT 0 CHECK (observations >= 0),
    successes bigint NOT NULL DEFAULT 0 CHECK (successes >= 0),
    availability_ppm integer NOT NULL DEFAULT 0 CHECK (availability_ppm BETWEEN 0 AND 1000000),
    ttft_p95_ms integer NOT NULL DEFAULT 0 CHECK (ttft_p95_ms >= 0),
    cooldown_until timestamptz,
    next_probe_at timestamptz NOT NULL DEFAULT now(),
    last_success_at timestamptz,
    last_failure_at timestamptz,
    last_failure_class text,
    last_status_code integer,
    healthy_since timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (merchant_channel_id, canonical_model_id)
);

CREATE TABLE routing_decisions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_id varchar(128) NOT NULL UNIQUE,
    token_binding_id bigint NOT NULL REFERENCES token_bindings(id),
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    offer_version_id bigint REFERENCES offer_versions(id),
    mode text NOT NULL,
    priority text NOT NULL,
    attempt_count integer NOT NULL DEFAULT 0,
    upstream_status_code integer,
    first_byte_ms integer,
    result text NOT NULL DEFAULT 'selected',
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz
);

CREATE TABLE channel_health_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    merchant_channel_id bigint NOT NULL REFERENCES merchant_channels(id),
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    offer_version_id bigint REFERENCES offer_versions(id),
    source text NOT NULL CHECK (source IN ('probe', 'request')),
    outcome text NOT NULL CHECK (outcome IN ('success', 'ignored', 'temporary_failure', 'hard_failure')),
    status_code integer,
    first_byte_ms integer,
    request_id varchar(128),
    failure_summary varchar(255),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE performance_rollups (
    merchant_channel_id bigint NOT NULL REFERENCES merchant_channels(id),
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    bucket_start timestamptz NOT NULL,
    bucket_minutes integer NOT NULL CHECK (bucket_minutes > 0),
    request_count bigint NOT NULL DEFAULT 0,
    success_count bigint NOT NULL DEFAULT 0,
    temporary_failure_count bigint NOT NULL DEFAULT 0,
    hard_failure_count bigint NOT NULL DEFAULT 0,
    ttft_p50_ms integer NOT NULL DEFAULT 0,
    ttft_p95_ms integer NOT NULL DEFAULT 0,
    PRIMARY KEY (merchant_channel_id, canonical_model_id, bucket_start, bucket_minutes)
);

CREATE INDEX offer_versions_model_status_idx ON offer_versions (canonical_model_id, status, multiplier_bps);
CREATE INDEX token_bindings_fingerprint_status_idx ON token_bindings (token_fingerprint, status);
CREATE INDEX channel_model_health_probe_idx ON channel_model_health (next_probe_at) WHERE state <> 'disabled';
CREATE INDEX channel_health_events_lookup_idx ON channel_health_events (merchant_channel_id, canonical_model_id, created_at DESC);
CREATE INDEX routing_decisions_created_idx ON routing_decisions (created_at DESC);
