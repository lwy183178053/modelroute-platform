ALTER TABLE canonical_models
    ADD COLUMN lifecycle text NOT NULL DEFAULT 'stable'
        CHECK (lifecycle IN ('stable', 'preview', 'limited', 'retiring'));

ALTER TABLE official_price_versions
    ADD COLUMN pricing_scheme text NOT NULL DEFAULT 'legacy_fixed'
        CHECK (pricing_scheme IN ('absolute', 'legacy_fixed')),
    ADD COLUMN original_currency char(3) NOT NULL DEFAULT 'USD'
        CHECK (original_currency IN ('USD', 'CNY')),
    ADD COLUMN usd_exchange_rate_micros bigint NOT NULL DEFAULT 1000000
        CHECK (usd_exchange_rate_micros > 0),
    ADD COLUMN source_url text NOT NULL DEFAULT '旧版本兼容记录',
    ADD COLUMN checked_at timestamptz NOT NULL DEFAULT now();

CREATE TABLE price_tiers (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    official_price_version_id bigint NOT NULL REFERENCES official_price_versions(id) ON DELETE CASCADE,
    tier_order integer NOT NULL CHECK (tier_order > 0),
    minimum_input_tokens bigint NOT NULL CHECK (minimum_input_tokens >= 0),
    maximum_input_tokens bigint,
    input_price_micros bigint NOT NULL CHECK (input_price_micros > 0),
    output_price_micros bigint CHECK (output_price_micros > 0),
    cache_read_price_micros bigint CHECK (cache_read_price_micros >= 0),
    cache_write_5m_price_micros bigint CHECK (cache_write_5m_price_micros >= 0),
    cache_write_1h_price_micros bigint CHECK (cache_write_1h_price_micros >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (official_price_version_id, tier_order),
    UNIQUE (official_price_version_id, minimum_input_tokens),
    CHECK (maximum_input_tokens IS NULL OR maximum_input_tokens >= minimum_input_tokens)
);

INSERT INTO price_tiers (
    official_price_version_id, tier_order, minimum_input_tokens, maximum_input_tokens,
    input_price_micros, output_price_micros, cache_read_price_micros,
    cache_write_5m_price_micros, cache_write_1h_price_micros
)
SELECT
    id, 1, 0, NULL,
    (model_ratio_micros::numeric * 2)::bigint,
    NULLIF(round(model_ratio_micros::numeric * 2 * completion_ratio_micros / 1000000)::bigint, 0),
    round(model_ratio_micros::numeric * 2 * cache_ratio_micros / 1000000)::bigint,
    round(model_ratio_micros::numeric * 2 * create_cache_ratio_micros / 1000000)::bigint,
    NULL
FROM official_price_versions
WHERE billing_mode = 'token';

UPDATE official_price_versions
SET pricing_scheme = 'absolute',
    original_currency = 'USD',
    usd_exchange_rate_micros = 1000000,
    source_url = '旧倍率兼容迁移',
    checked_at = published_at
WHERE billing_mode = 'token';

UPDATE canonical_models cm
SET status = 'inactive', updated_at = now()
FROM official_price_versions price
WHERE price.id = cm.current_official_price_version_id
  AND price.billing_mode = 'fixed';

CREATE INDEX price_tiers_version_order_idx
    ON price_tiers (official_price_version_id, tier_order);
