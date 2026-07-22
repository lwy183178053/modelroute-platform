CREATE TABLE announcements (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title varchar(120) NOT NULL CHECK (char_length(btrim(title)) BETWEEN 1 AND 120),
    body text NOT NULL CHECK (char_length(btrim(body)) BETWEEN 1 AND 10000),
    audience_mask smallint NOT NULL CHECK (audience_mask BETWEEN 1 AND 7),
    pinned boolean NOT NULL DEFAULT false,
    publish_at timestamptz,
    expires_at timestamptz,
    status text NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'scheduled', 'published', 'withdrawn')),
    created_by_newapi_user_id bigint NOT NULL,
    updated_by_newapi_user_id bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (status IN ('scheduled', 'published') AND publish_at IS NOT NULL)
        OR status IN ('draft', 'withdrawn')
    ),
    CHECK (expires_at IS NULL OR publish_at IS NULL OR expires_at > publish_at)
);

CREATE INDEX announcements_visibility_idx
    ON announcements (status, publish_at, expires_at);

CREATE TABLE announcement_reads (
    announcement_id bigint NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    newapi_user_id bigint NOT NULL,
    read_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (announcement_id, newapi_user_id)
);

CREATE INDEX announcement_reads_user_idx
    ON announcement_reads (newapi_user_id, read_at DESC);
