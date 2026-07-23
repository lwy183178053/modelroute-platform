ALTER TABLE offer_versions DROP CONSTRAINT offer_versions_status_check;
ALTER TABLE offer_versions ADD CONSTRAINT offer_versions_status_check
    CHECK (status IN ('syncing', 'active', 'inactive', 'sync_failed', 'deleted'));
