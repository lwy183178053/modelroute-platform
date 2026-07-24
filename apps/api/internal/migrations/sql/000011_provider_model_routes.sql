ALTER TABLE routing_decisions
    DROP CONSTRAINT routing_decisions_token_binding_id_fkey,
    DROP CONSTRAINT routing_decisions_routing_policy_id_fkey,
    ALTER COLUMN token_binding_id DROP NOT NULL,
    ALTER COLUMN routing_policy_id DROP NOT NULL,
    ADD CONSTRAINT routing_decisions_token_binding_id_fkey
        FOREIGN KEY (token_binding_id) REFERENCES token_bindings(id) ON DELETE SET NULL,
    ADD CONSTRAINT routing_decisions_routing_policy_id_fkey
        FOREIGN KEY (routing_policy_id) REFERENCES routing_policies(id) ON DELETE SET NULL;

DELETE FROM token_bindings;
DELETE FROM routing_policies;

DROP INDEX IF EXISTS token_bindings_policy_status_idx;

ALTER TABLE token_bindings
    DROP COLUMN routing_policy_id,
    ADD COLUMN provider varchar(32) NOT NULL
        CHECK (provider IN ('OpenAI', 'Anthropic', 'Gemini', 'DeepSeek', 'Qwen', 'Moonshot'));

ALTER TABLE routing_policies
    ADD COLUMN canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    ADD CONSTRAINT routing_policies_id_model_unique UNIQUE (id, canonical_model_id);

CREATE INDEX routing_policies_user_model_idx
    ON routing_policies (newapi_user_id, canonical_model_id, updated_at DESC);

CREATE TABLE token_model_routes (
    token_binding_id bigint NOT NULL REFERENCES token_bindings(id) ON DELETE CASCADE,
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    routing_policy_id bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (token_binding_id, routing_policy_id),
    UNIQUE (token_binding_id, canonical_model_id),
    FOREIGN KEY (routing_policy_id, canonical_model_id)
        REFERENCES routing_policies(id, canonical_model_id) ON DELETE CASCADE
);

CREATE INDEX token_model_routes_policy_idx
    ON token_model_routes (routing_policy_id, token_binding_id);
