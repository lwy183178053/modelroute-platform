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

CREATE INDEX merchant_withdrawals_profile_created_idx
    ON merchant_withdrawals (merchant_profile_id, created_at DESC);
CREATE INDEX merchant_withdrawals_status_created_idx
    ON merchant_withdrawals (status, created_at ASC);

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
    merchant_quota bigint NOT NULL DEFAULT 0,
    platform_quota bigint NOT NULL DEFAULT 0,
    processed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX settlement_sources_task_lookup_idx
    ON settlement_sources (task_id, newapi_channel_id, model_name, source_created_at)
    WHERE source_type = 2 AND status = 'settled';
CREATE INDEX settlement_sources_created_idx
    ON settlement_sources (source_created_at DESC, newapi_log_id DESC);

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

CREATE INDEX settlement_reconciliation_status_created_idx
    ON settlement_reconciliation_items (status, created_at ASC);

CREATE TABLE merchant_ledger_entries (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    merchant_profile_id bigint NOT NULL REFERENCES merchant_profiles(id),
    settlement_source_id bigint REFERENCES settlement_sources(id),
    withdrawal_id bigint REFERENCES merchant_withdrawals(id),
    entry_type text NOT NULL CHECK (entry_type IN (
        'earning', 'refund', 'withdrawal_freeze', 'withdrawal_release', 'withdrawal_paid'
    )),
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
CREATE INDEX merchant_ledger_profile_created_idx
    ON merchant_ledger_entries (merchant_profile_id, created_at DESC, id DESC);

CREATE FUNCTION prevent_merchant_ledger_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'merchant ledger entries are append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER merchant_ledger_entries_append_only
BEFORE UPDATE OR DELETE ON merchant_ledger_entries
FOR EACH ROW EXECUTE FUNCTION prevent_merchant_ledger_mutation();
