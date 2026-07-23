ALTER TABLE merchant_applications
    ADD COLUMN display_name_key varchar(80);

UPDATE merchant_applications
SET display_name_key = lower(display_name);

ALTER TABLE merchant_applications
    ALTER COLUMN display_name_key SET NOT NULL;

CREATE TABLE merchant_name_reservations (
    display_name_key varchar(80) CONSTRAINT merchant_name_reservations_pkey PRIMARY KEY,
    display_name varchar(80) NOT NULL,
    newapi_user_id bigint NOT NULL,
    application_id bigint UNIQUE REFERENCES merchant_applications(id),
    merchant_profile_id bigint UNIQUE REFERENCES merchant_profiles(id),
    status text NOT NULL CHECK (status IN ('pending', 'active')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'pending' AND application_id IS NOT NULL AND merchant_profile_id IS NULL)
        OR (status = 'active' AND merchant_profile_id IS NOT NULL))
);

INSERT INTO merchant_name_reservations
    (display_name_key, display_name, newapi_user_id, application_id, merchant_profile_id, status)
SELECT lower(profile.display_name), profile.display_name, profile.newapi_user_id,
       profile.approved_application_id, profile.id, 'active'
FROM merchant_profiles profile;

INSERT INTO merchant_name_reservations
    (display_name_key, display_name, newapi_user_id, application_id, status)
SELECT application.display_name_key, application.display_name, application.newapi_user_id,
       application.id, 'pending'
FROM merchant_applications application
WHERE application.status IN ('pending', 'verifying', 'verified', 'approving')
ON CONFLICT (display_name_key) DO NOTHING;
