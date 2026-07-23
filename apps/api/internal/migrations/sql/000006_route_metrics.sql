CREATE TABLE route_metric_buckets (
    merchant_channel_id bigint NOT NULL REFERENCES merchant_channels(id) ON DELETE CASCADE,
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    bucket_start timestamptz NOT NULL,
    request_count bigint NOT NULL DEFAULT 0 CHECK (request_count >= 0),
    success_count bigint NOT NULL DEFAULT 0 CHECK (success_count >= 0),
    ttft_total_ms bigint NOT NULL DEFAULT 0 CHECK (ttft_total_ms >= 0),
    ttft_count bigint NOT NULL DEFAULT 0 CHECK (ttft_count >= 0),
    prompt_tokens bigint NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),
    cache_tokens bigint NOT NULL DEFAULT 0 CHECK (cache_tokens >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (merchant_channel_id, canonical_model_id, bucket_start),
    CHECK (success_count <= request_count)
);
CREATE INDEX route_metric_buckets_window_idx
    ON route_metric_buckets (bucket_start, canonical_model_id, merchant_channel_id);

CREATE TABLE route_metric_snapshots (
    merchant_channel_id bigint NOT NULL REFERENCES merchant_channels(id) ON DELETE CASCADE,
    canonical_model_id bigint NOT NULL REFERENCES canonical_models(id),
    request_count bigint NOT NULL DEFAULT 0 CHECK (request_count >= 0),
    success_count bigint NOT NULL DEFAULT 0 CHECK (success_count >= 0),
    availability_ppm integer NOT NULL DEFAULT 0 CHECK (availability_ppm BETWEEN 0 AND 1000000),
    average_ttft_ms integer NOT NULL DEFAULT 0 CHECK (average_ttft_ms >= 0),
    prompt_tokens bigint NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),
    cache_tokens bigint NOT NULL DEFAULT 0 CHECK (cache_tokens >= 0),
    cache_hit_ppm integer NOT NULL DEFAULT 0 CHECK (cache_hit_ppm BETWEEN 0 AND 1000000),
    cache_weight_ready boolean NOT NULL DEFAULT false,
    metrics_updated_at timestamptz NOT NULL,
    PRIMARY KEY (merchant_channel_id, canonical_model_id),
    CHECK (success_count <= request_count)
);
CREATE INDEX route_metric_snapshots_model_idx
    ON route_metric_snapshots (canonical_model_id, cache_weight_ready, cache_hit_ppm);
