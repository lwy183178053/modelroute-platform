ALTER TABLE merchant_withdrawals
    DROP CONSTRAINT IF EXISTS merchant_withdrawals_amount_quota_check;

ALTER TABLE merchant_withdrawals
    ADD CONSTRAINT merchant_withdrawals_amount_quota_check CHECK (amount_quota > 0);
