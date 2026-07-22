ALTER TABLE merchant_profiles
    ALTER COLUMN approved_application_id DROP NOT NULL;

ALTER TABLE merchant_profiles
    ADD COLUMN is_platform_owned boolean NOT NULL DEFAULT false;

ALTER TABLE merchant_profiles
    ADD CONSTRAINT merchant_profiles_platform_owned_check CHECK (
        (is_platform_owned AND approved_application_id IS NULL)
        OR
        (NOT is_platform_owned AND approved_application_id IS NOT NULL)
    );

CREATE UNIQUE INDEX merchant_profiles_one_platform_owned_idx
    ON merchant_profiles (is_platform_owned)
    WHERE is_platform_owned;

ALTER TABLE merchant_channel_reviews
    ADD COLUMN merchant_profile_id bigint REFERENCES merchant_profiles(id);

UPDATE merchant_channel_reviews r
SET merchant_profile_id = p.id
FROM merchant_profiles p
WHERE p.newapi_user_id = r.merchant_newapi_user_id;

CREATE INDEX merchant_channel_reviews_profile_created_idx
    ON merchant_channel_reviews (merchant_profile_id, created_at DESC);
