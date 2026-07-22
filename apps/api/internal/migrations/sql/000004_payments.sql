CREATE TABLE payment_setting_versions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    version integer NOT NULL UNIQUE CHECK (version > 0),
    gateway_url varchar(512) NOT NULL,
    merchant_pid varchar(128) NOT NULL,
    encrypted_key bytea NOT NULL,
    key_fingerprint char(64) NOT NULL,
    payment_methods jsonb NOT NULL CHECK (jsonb_typeof(payment_methods) = 'array'),
    minimum_usd_cents bigint NOT NULL CHECK (minimum_usd_cents > 0),
    maximum_usd_cents bigint NOT NULL CHECK (maximum_usd_cents >= minimum_usd_cents),
    exchange_rate_cny_cents_per_usd bigint NOT NULL CHECK (exchange_rate_cny_cents_per_usd > 0),
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
    usd_cents bigint NOT NULL CHECK (usd_cents > 0),
    cny_cents bigint NOT NULL CHECK (cny_cents > 0),
    exchange_rate_cny_cents_per_usd bigint NOT NULL CHECK (exchange_rate_cny_cents_per_usd > 0),
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
CREATE INDEX payment_orders_user_created_idx
    ON payment_orders (newapi_user_id, created_at DESC);
CREATE INDEX payment_orders_admin_created_idx
    ON payment_orders (created_at DESC);

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
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error varchar(500),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    finished_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX newapi_credit_operations_claim_idx
    ON newapi_credit_operations (next_attempt_at, id)
    WHERE status IN ('pending', 'failed');
