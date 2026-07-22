CREATE TABLE platform_runtime_metadata (
    key text PRIMARY KEY,
    value text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE platform_audit_logs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_newapi_user_id bigint,
    actor_role text NOT NULL,
    action text NOT NULL,
    object_type text NOT NULL,
    object_id text,
    result text NOT NULL,
    reason text,
    request_id text,
    source_ip inet,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX platform_audit_logs_created_at_idx ON platform_audit_logs (created_at DESC);

CREATE TABLE merchant_applications (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    newapi_user_id bigint NOT NULL,
    display_name varchar(80) NOT NULL,
    website_url text NOT NULL,
    business_description text NOT NULL,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'verifying', 'verified', 'approving', 'approved', 'rejected')),
    initial_channel_review_id bigint,
    review_reason text,
    reviewed_by_newapi_user_id bigint,
    reviewed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX merchant_applications_one_open_per_user_idx
    ON merchant_applications (newapi_user_id)
    WHERE status IN ('pending', 'verifying', 'verified', 'approving');

CREATE TABLE merchant_profiles (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    newapi_user_id bigint NOT NULL UNIQUE,
    approved_application_id bigint UNIQUE REFERENCES merchant_applications(id),
    display_name varchar(80) NOT NULL,
    website_url text NOT NULL,
    is_platform_owned boolean NOT NULL DEFAULT false,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    approved_by_newapi_user_id bigint NOT NULL,
    approved_at timestamptz NOT NULL DEFAULT now(),
    status_reason text,
    status_changed_by_newapi_user_id bigint,
    status_changed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((is_platform_owned AND approved_application_id IS NULL)
        OR (NOT is_platform_owned AND approved_application_id IS NOT NULL))
);
CREATE UNIQUE INDEX merchant_profiles_one_platform_owned_idx
    ON merchant_profiles (is_platform_owned) WHERE is_platform_owned;

CREATE TABLE canonical_models (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_name varchar(255) NOT NULL UNIQUE,
    provider varchar(80) NOT NULL,
    protocol text NOT NULL CHECK (protocol IN ('chat_completions', 'responses', 'image_generation')),
    adapter_hint text NOT NULL DEFAULT 'auto',
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'archived')),
    current_official_price_version_id bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE official_price_versions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    version integer NOT NULL CHECK (version > 0),
    price_kind text NOT NULL CHECK (price_kind IN ('text', 'image')),
    published_by_newapi_user_id bigint NOT NULL,
    published_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (canonical_model_id, version)
);
ALTER TABLE canonical_models ADD CONSTRAINT canonical_models_current_price_fk
    FOREIGN KEY (current_official_price_version_id) REFERENCES official_price_versions(id);

CREATE TABLE text_price_tiers (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    official_price_version_id bigint NOT NULL REFERENCES official_price_versions(id) ON DELETE CASCADE,
    tier_order integer NOT NULL CHECK (tier_order > 0),
    minimum_input_tokens bigint NOT NULL CHECK (minimum_input_tokens >= 0),
    maximum_input_tokens bigint,
    input_price_micros bigint NOT NULL CHECK (input_price_micros > 0),
    output_price_micros bigint NOT NULL CHECK (output_price_micros > 0),
    cache_read_price_micros bigint CHECK (cache_read_price_micros >= 0),
    cache_write_price_micros bigint CHECK (cache_write_price_micros >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (official_price_version_id, tier_order),
    UNIQUE (official_price_version_id, minimum_input_tokens),
    CHECK (maximum_input_tokens IS NULL OR maximum_input_tokens >= minimum_input_tokens)
);

CREATE TABLE image_price_items (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    official_price_version_id bigint NOT NULL REFERENCES official_price_versions(id) ON DELETE CASCADE,
    quality varchar(32) NOT NULL,
    size varchar(32) NOT NULL,
    price_micros bigint NOT NULL CHECK (price_micros > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (official_price_version_id, quality, size)
);

CREATE TABLE merchant_channel_tests (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    newapi_user_id bigint NOT NULL,
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    base_url text NOT NULL,
    credential_fingerprint char(64) NOT NULL,
    adapter_type text NOT NULL,
    capabilities jsonb NOT NULL CHECK (jsonb_typeof(capabilities) = 'array'),
    latency_ms integer NOT NULL CHECK (latency_ms >= 0),
    receipt_token_hash char(64) NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX merchant_channel_tests_owner_expiry_idx
    ON merchant_channel_tests (newapi_user_id, expires_at DESC);

CREATE TABLE merchant_channel_reviews (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    merchant_newapi_user_id bigint NOT NULL,
    merchant_profile_id bigint REFERENCES merchant_profiles(id),
    application_id bigint REFERENCES merchant_applications(id),
    replaces_channel_id bigint,
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    channel_test_id bigint NOT NULL REFERENCES merchant_channel_tests(id),
    channel_name varchar(100) NOT NULL,
    base_url text NOT NULL,
    adapter_type text NOT NULL,
    encrypted_credential bytea,
    credential_fingerprint char(64) NOT NULL,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'verifying', 'verified', 'approving', 'approved', 'rejected')),
    verification_error text,
    verified_by_newapi_user_id bigint,
    verified_at timestamptz,
    review_reason text,
    reviewed_by_newapi_user_id bigint,
    reviewed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((status IN ('approved', 'rejected') AND encrypted_credential IS NULL)
        OR (status NOT IN ('approved', 'rejected') AND encrypted_credential IS NOT NULL))
);
ALTER TABLE merchant_applications ADD CONSTRAINT merchant_applications_initial_channel_review_fk
    FOREIGN KEY (initial_channel_review_id) REFERENCES merchant_channel_reviews(id);

CREATE TABLE merchant_channels (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    merchant_profile_id bigint NOT NULL REFERENCES merchant_profiles(id),
    review_id bigint NOT NULL UNIQUE REFERENCES merchant_channel_reviews(id),
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    newapi_channel_id bigint NOT NULL UNIQUE,
    channel_name varchar(100) NOT NULL,
    base_url text NOT NULL,
    adapter_type text NOT NULL,
    credential_fingerprint char(64) NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE merchant_channel_reviews ADD CONSTRAINT merchant_channel_reviews_replaces_channel_fk
    FOREIGN KEY (replaces_channel_id) REFERENCES merchant_channels(id);
CREATE INDEX merchant_channel_reviews_profile_created_idx
    ON merchant_channel_reviews (merchant_profile_id, created_at DESC);
CREATE INDEX merchant_channel_reviews_status_created_idx
    ON merchant_channel_reviews (status, created_at ASC);
CREATE INDEX merchant_channels_profile_model_idx
    ON merchant_channels (merchant_profile_id, canonical_model_id, created_at DESC);

CREATE TABLE offer_versions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    merchant_profile_id bigint NOT NULL REFERENCES merchant_profiles(id),
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    official_price_version_id bigint NOT NULL REFERENCES official_price_versions(id),
    previous_offer_version_id bigint REFERENCES offer_versions(id),
    multiplier_bps integer NOT NULL CHECK (multiplier_bps BETWEEN 0 AND 50000),
    version integer NOT NULL CHECK (version > 0),
    status text NOT NULL DEFAULT 'syncing' CHECK (status IN ('syncing', 'active', 'inactive', 'sync_failed')),
    sync_error text,
    created_by_newapi_user_id bigint NOT NULL,
    activated_at timestamptz,
    retired_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (merchant_profile_id, canonical_model_id, version)
);
CREATE UNIQUE INDEX offer_versions_one_active_per_merchant_model_idx
    ON offer_versions (merchant_profile_id, canonical_model_id) WHERE status = 'active';
CREATE UNIQUE INDEX offer_versions_one_syncing_per_merchant_model_idx
    ON offer_versions (merchant_profile_id, canonical_model_id) WHERE status = 'syncing';
CREATE INDEX offer_versions_model_status_idx
    ON offer_versions (canonical_model_id, status, multiplier_bps);

CREATE TABLE offer_channel_aliases (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    offer_version_id bigint NOT NULL REFERENCES offer_versions(id) ON DELETE CASCADE,
    merchant_channel_id bigint NOT NULL REFERENCES merchant_channels(id),
    internal_alias varchar(96) NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (offer_version_id, merchant_channel_id)
);

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
    minimum_multiplier_bps integer NOT NULL DEFAULT 0 CHECK (minimum_multiplier_bps BETWEEN 0 AND 50000),
    maximum_multiplier_bps integer NOT NULL DEFAULT 50000 CHECK (maximum_multiplier_bps BETWEEN 0 AND 50000),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (minimum_multiplier_bps <= maximum_multiplier_bps)
);

CREATE TABLE routing_policy_merchants (
    routing_policy_id bigint NOT NULL REFERENCES routing_policies(id) ON DELETE CASCADE,
    merchant_profile_id bigint NOT NULL REFERENCES merchant_profiles(id),
    PRIMARY KEY (routing_policy_id, merchant_profile_id)
);

CREATE TABLE channel_health (
    merchant_channel_id bigint PRIMARY KEY REFERENCES merchant_channels(id),
    state text NOT NULL DEFAULT 'healthy' CHECK (state IN ('healthy', 'degraded', 'cooling', 'recovering', 'disabled')),
    consecutive_failures integer NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    cooldown_level integer NOT NULL DEFAULT 0 CHECK (cooldown_level BETWEEN 0 AND 6),
    observations bigint NOT NULL DEFAULT 0 CHECK (observations >= 0),
    successes bigint NOT NULL DEFAULT 0 CHECK (successes >= 0),
    availability_ppm integer NOT NULL DEFAULT 0 CHECK (availability_ppm BETWEEN 0 AND 1000000),
    ttft_p95_ms integer NOT NULL DEFAULT 0 CHECK (ttft_p95_ms >= 0),
    cooldown_until timestamptz,
    next_probe_at timestamptz,
    last_success_at timestamptz,
    last_failure_at timestamptz,
    last_failure_class text,
    last_status_code integer,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((state IN ('cooling', 'recovering') AND next_probe_at IS NOT NULL)
        OR (state NOT IN ('cooling', 'recovering') AND next_probe_at IS NULL))
);
CREATE INDEX channel_health_probe_idx ON channel_health (next_probe_at)
    WHERE state IN ('cooling', 'recovering');

CREATE TABLE routing_decisions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_id varchar(128) NOT NULL UNIQUE,
    token_binding_id bigint NOT NULL REFERENCES token_bindings(id),
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    offer_channel_alias_id bigint REFERENCES offer_channel_aliases(id),
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
    offer_channel_alias_id bigint REFERENCES offer_channel_aliases(id),
    source text NOT NULL CHECK (source IN ('probe', 'request')),
    outcome text NOT NULL CHECK (outcome IN ('success', 'ignored', 'temporary_failure', 'hard_failure')),
    status_code integer,
    first_byte_ms integer,
    request_id varchar(128),
    failure_summary varchar(255),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX channel_health_events_lookup_idx
    ON channel_health_events (merchant_channel_id, created_at DESC);

CREATE TABLE performance_rollups (
    merchant_channel_id bigint NOT NULL REFERENCES merchant_channels(id),
    bucket_start timestamptz NOT NULL,
    bucket_minutes integer NOT NULL CHECK (bucket_minutes > 0),
    request_count bigint NOT NULL DEFAULT 0,
    success_count bigint NOT NULL DEFAULT 0,
    temporary_failure_count bigint NOT NULL DEFAULT 0,
    hard_failure_count bigint NOT NULL DEFAULT 0,
    ttft_p50_ms integer NOT NULL DEFAULT 0,
    ttft_p95_ms integer NOT NULL DEFAULT 0,
    PRIMARY KEY (merchant_channel_id, bucket_start, bucket_minutes)
);

CREATE TABLE payment_setting_versions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    version integer NOT NULL UNIQUE CHECK (version > 0),
    gateway_url varchar(512) NOT NULL,
    merchant_pid varchar(128) NOT NULL,
    encrypted_key bytea NOT NULL,
    key_fingerprint char(64) NOT NULL,
    payment_methods jsonb NOT NULL CHECK (jsonb_typeof(payment_methods) = 'array'),
    minimum_cny_cents bigint NOT NULL CHECK (minimum_cny_cents > 0),
    maximum_cny_cents bigint NOT NULL CHECK (maximum_cny_cents >= minimum_cny_cents),
    enabled boolean NOT NULL DEFAULT false,
    is_current boolean NOT NULL DEFAULT true,
    created_by_newapi_user_id integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX payment_setting_versions_current_idx
    ON payment_setting_versions (is_current) WHERE is_current;

CREATE TABLE payment_orders (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_no varchar(64) NOT NULL UNIQUE,
    newapi_user_id integer NOT NULL,
    setting_version_id bigint NOT NULL REFERENCES payment_setting_versions(id),
    payment_type varchar(32) NOT NULL,
    amount_cny_cents bigint NOT NULL CHECK (amount_cny_cents > 0),
    newapi_quota bigint NOT NULL CHECK (newapi_quota > 0),
    payment_status text NOT NULL DEFAULT 'pending' CHECK (payment_status IN ('pending', 'paid', 'closed')),
    credit_status text NOT NULL DEFAULT 'pending' CHECK (credit_status IN ('pending', 'processing', 'succeeded', 'failed', 'unknown')),
    third_party_trade_no varchar(128),
    paid_at timestamptz,
    credited_at timestamptz,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX payment_orders_trade_no_idx
    ON payment_orders (third_party_trade_no) WHERE third_party_trade_no IS NOT NULL;
CREATE INDEX payment_orders_user_created_idx ON payment_orders (newapi_user_id, created_at DESC);

CREATE TABLE payment_callback_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    callback_key char(64) NOT NULL UNIQUE,
    payment_order_id bigint NOT NULL REFERENCES payment_orders(id),
    third_party_trade_no varchar(128) NOT NULL UNIQUE,
    provider_payload jsonb NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE newapi_credit_operations (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    payment_order_id bigint NOT NULL UNIQUE REFERENCES payment_orders(id),
    idempotency_key varchar(96) NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'succeeded', 'failed', 'unknown')),
    last_error varchar(500),
    attempt_count integer NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE settlement_sync_state (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    last_synced_through bigint NOT NULL DEFAULT 0 CHECK (last_synced_through >= 0),
    last_success_at timestamptz,
    last_error varchar(500),
    updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO settlement_sync_state (singleton) VALUES (true);

CREATE TABLE merchant_balances (
    merchant_profile_id bigint PRIMARY KEY REFERENCES merchant_profiles(id),
    available_quota bigint NOT NULL DEFAULT 0,
    frozen_quota bigint NOT NULL DEFAULT 0 CHECK (frozen_quota >= 0),
    lifetime_earned_quota bigint NOT NULL DEFAULT 0 CHECK (lifetime_earned_quota >= 0),
    lifetime_refunded_quota bigint NOT NULL DEFAULT 0 CHECK (lifetime_refunded_quota >= 0),
    lifetime_paid_quota bigint NOT NULL DEFAULT 0 CHECK (lifetime_paid_quota >= 0),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE merchant_withdrawals (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    withdrawal_no varchar(64) NOT NULL UNIQUE,
    merchant_profile_id bigint NOT NULL REFERENCES merchant_profiles(id),
    amount_quota bigint NOT NULL CHECK (amount_quota > 0),
    available_before_quota bigint NOT NULL,
    recipient_type varchar(32) NOT NULL CHECK (recipient_type IN ('alipay', 'bank', 'other')),
    encrypted_recipient_account bytea NOT NULL,
    recipient_account_masked varchar(160) NOT NULL,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'paid', 'rejected')),
    review_reason varchar(500),
    reviewed_by_newapi_user_id bigint,
    reviewed_at timestamptz,
    payout_reference varchar(200),
    payout_proof text,
    paid_by_newapi_user_id bigint,
    paid_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE settlement_sources (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    newapi_log_id bigint NOT NULL UNIQUE,
    newapi_user_id bigint NOT NULL,
    source_created_at timestamptz NOT NULL,
    source_type integer NOT NULL CHECK (source_type IN (2, 6)),
    model_name varchar(255) NOT NULL,
    newapi_channel_id bigint NOT NULL,
    newapi_token_id bigint NOT NULL DEFAULT 0,
    request_id varchar(128),
    task_id varchar(128),
    gross_quota bigint NOT NULL CHECK (gross_quota > 0),
    other jsonb NOT NULL,
    status text NOT NULL DEFAULT 'reconciliation' CHECK (status IN ('settled', 'reconciliation')),
    merchant_profile_id bigint REFERENCES merchant_profiles(id),
    offer_version_id bigint REFERENCES offer_versions(id),
    offer_channel_alias_id bigint REFERENCES offer_channel_aliases(id),
    merchant_quota bigint NOT NULL DEFAULT 0,
    platform_quota bigint NOT NULL DEFAULT 0,
    processed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE settlement_refund_allocations (
    refund_source_id bigint NOT NULL REFERENCES settlement_sources(id),
    consume_source_id bigint NOT NULL REFERENCES settlement_sources(id),
    gross_quota bigint NOT NULL CHECK (gross_quota > 0),
    merchant_quota bigint NOT NULL CHECK (merchant_quota >= 0),
    platform_quota bigint NOT NULL CHECK (platform_quota >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (refund_source_id, consume_source_id),
    CHECK (merchant_quota + platform_quota = gross_quota)
);

CREATE TABLE settlement_reconciliation_items (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    settlement_source_id bigint NOT NULL UNIQUE REFERENCES settlement_sources(id),
    reason_code varchar(64) NOT NULL,
    details jsonb NOT NULL,
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved', 'ignored')),
    resolution_note varchar(500),
    resolved_by_newapi_user_id bigint,
    resolved_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE merchant_ledger_entries (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    merchant_profile_id bigint NOT NULL REFERENCES merchant_profiles(id),
    settlement_source_id bigint REFERENCES settlement_sources(id),
    withdrawal_id bigint REFERENCES merchant_withdrawals(id),
    entry_type text NOT NULL CHECK (entry_type IN ('earning', 'refund', 'withdrawal_freeze', 'withdrawal_release', 'withdrawal_paid')),
    available_delta bigint NOT NULL,
    frozen_delta bigint NOT NULL,
    available_after bigint NOT NULL,
    frozen_after bigint NOT NULL CHECK (frozen_after >= 0),
    description varchar(255) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (settlement_source_id IS NOT NULL OR withdrawal_id IS NOT NULL)
);
CREATE UNIQUE INDEX merchant_ledger_source_idx
    ON merchant_ledger_entries (settlement_source_id) WHERE settlement_source_id IS NOT NULL;
CREATE UNIQUE INDEX merchant_ledger_withdrawal_action_idx
    ON merchant_ledger_entries (withdrawal_id, entry_type) WHERE withdrawal_id IS NOT NULL;

CREATE FUNCTION prevent_merchant_ledger_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'merchant ledger entries are append-only';
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER merchant_ledger_entries_append_only
BEFORE UPDATE OR DELETE ON merchant_ledger_entries
FOR EACH ROW EXECUTE FUNCTION prevent_merchant_ledger_mutation();

CREATE TABLE announcements (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title varchar(120) NOT NULL CHECK (char_length(btrim(title)) BETWEEN 1 AND 120),
    body text NOT NULL CHECK (char_length(btrim(body)) BETWEEN 1 AND 10000),
    audience_mask smallint NOT NULL CHECK (audience_mask BETWEEN 1 AND 7),
    pinned boolean NOT NULL DEFAULT false,
    publish_at timestamptz,
    expires_at timestamptz,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'scheduled', 'published', 'withdrawn')),
    created_by_newapi_user_id bigint NOT NULL,
    updated_by_newapi_user_id bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((status IN ('scheduled', 'published') AND publish_at IS NOT NULL) OR status IN ('draft', 'withdrawn')),
    CHECK (expires_at IS NULL OR publish_at IS NULL OR expires_at > publish_at)
);

CREATE TABLE announcement_reads (
    announcement_id bigint NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    newapi_user_id bigint NOT NULL,
    read_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (announcement_id, newapi_user_id)
);
