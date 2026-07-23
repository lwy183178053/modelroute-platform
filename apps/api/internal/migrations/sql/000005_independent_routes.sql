DROP TABLE routing_decisions;
DROP TABLE routing_policy_merchants;
DROP TABLE routing_policies;

DELETE FROM token_bindings;

CREATE TABLE routing_policies (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    newapi_user_id bigint NOT NULL,
    name varchar(50) NOT NULL,
    priority text NOT NULL CHECK (priority IN ('balanced', 'price', 'success', 'latency', 'cache')),
    minimum_multiplier_bps integer NOT NULL CHECK (minimum_multiplier_bps BETWEEN 0 AND 50000),
    maximum_multiplier_bps integer NOT NULL CHECK (maximum_multiplier_bps BETWEEN 0 AND 50000),
    minimum_cache_hit_ppm integer,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (minimum_multiplier_bps <= maximum_multiplier_bps),
    CHECK (minimum_cache_hit_ppm IS NULL OR minimum_cache_hit_ppm BETWEEN 0 AND 1000000)
);
CREATE UNIQUE INDEX routing_policies_user_name_idx
    ON routing_policies (newapi_user_id, lower(name));
CREATE INDEX routing_policies_user_updated_idx
    ON routing_policies (newapi_user_id, updated_at DESC, id DESC);

ALTER TABLE token_bindings
    ADD COLUMN routing_policy_id bigint NOT NULL REFERENCES routing_policies(id);
CREATE INDEX token_bindings_policy_status_idx
    ON token_bindings (routing_policy_id, status);

CREATE TABLE routing_decisions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_id varchar(128) NOT NULL UNIQUE,
    token_binding_id bigint NOT NULL REFERENCES token_bindings(id),
    routing_policy_id bigint NOT NULL REFERENCES routing_policies(id),
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    offer_channel_alias_id bigint REFERENCES offer_channel_aliases(id),
    priority text NOT NULL,
    algorithm_version varchar(32) NOT NULL,
    candidate_count integer NOT NULL DEFAULT 0 CHECK (candidate_count >= 0),
    selected_metrics jsonb NOT NULL DEFAULT '{}'::jsonb,
    attempt_count integer NOT NULL DEFAULT 0,
    upstream_status_code integer,
    first_byte_ms integer,
    result text NOT NULL DEFAULT 'selected',
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz
);
