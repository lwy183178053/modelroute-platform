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
    approved_application_id bigint NOT NULL UNIQUE REFERENCES merchant_applications(id),
    display_name varchar(80) NOT NULL,
    website_url text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    approved_by_newapi_user_id bigint NOT NULL,
    approved_at timestamptz NOT NULL DEFAULT now(),
    status_reason text,
    status_changed_by_newapi_user_id bigint,
    status_changed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE merchant_channel_reviews (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    merchant_newapi_user_id bigint NOT NULL,
    application_id bigint REFERENCES merchant_applications(id),
    replaces_channel_id bigint,
    channel_name varchar(100) NOT NULL,
    channel_type integer NOT NULL CHECK (channel_type > 0),
    base_url text NOT NULL,
    requested_models jsonb NOT NULL,
    encrypted_credential bytea,
    credential_fingerprint char(64) NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'verifying', 'verified', 'approving', 'approved', 'rejected')),
    verified_models jsonb,
    verification_error text,
    verified_by_newapi_user_id bigint,
    verified_at timestamptz,
    review_reason text,
    reviewed_by_newapi_user_id bigint,
    reviewed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (status IN ('approved', 'rejected') AND encrypted_credential IS NULL)
        OR
        (status NOT IN ('approved', 'rejected') AND encrypted_credential IS NOT NULL)
    )
);

ALTER TABLE merchant_applications
    ADD CONSTRAINT merchant_applications_initial_channel_review_fk
    FOREIGN KEY (initial_channel_review_id) REFERENCES merchant_channel_reviews(id);

CREATE TABLE merchant_channels (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    merchant_profile_id bigint NOT NULL REFERENCES merchant_profiles(id),
    review_id bigint NOT NULL UNIQUE REFERENCES merchant_channel_reviews(id),
    newapi_channel_id bigint NOT NULL UNIQUE,
    channel_name varchar(100) NOT NULL,
    channel_type integer NOT NULL,
    base_url text NOT NULL,
    models jsonb NOT NULL,
    credential_fingerprint char(64) NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE merchant_channel_reviews
    ADD CONSTRAINT merchant_channel_reviews_replaces_channel_fk
    FOREIGN KEY (replaces_channel_id) REFERENCES merchant_channels(id);

CREATE INDEX merchant_channel_reviews_user_created_idx
    ON merchant_channel_reviews (merchant_newapi_user_id, created_at DESC);

CREATE INDEX merchant_channel_reviews_status_created_idx
    ON merchant_channel_reviews (status, created_at ASC);

CREATE INDEX merchant_channels_profile_created_idx
    ON merchant_channels (merchant_profile_id, created_at DESC);
