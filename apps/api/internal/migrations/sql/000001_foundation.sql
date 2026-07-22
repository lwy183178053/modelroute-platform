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

CREATE INDEX platform_audit_logs_created_at_idx
    ON platform_audit_logs (created_at DESC);
