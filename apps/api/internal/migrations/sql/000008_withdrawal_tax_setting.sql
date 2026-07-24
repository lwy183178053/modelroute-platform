ALTER TABLE payment_setting_versions
    ADD COLUMN withdrawal_tax_bps integer NOT NULL DEFAULT 300
        CHECK (withdrawal_tax_bps BETWEEN 0 AND 10000);

ALTER TABLE merchant_withdrawals
    ADD COLUMN withdrawal_tax_bps integer NOT NULL DEFAULT 300
        CHECK (withdrawal_tax_bps BETWEEN 0 AND 10000);

ALTER TABLE merchant_withdrawals
    DROP COLUMN tax_quota,
    DROP COLUMN net_payout_quota;

ALTER TABLE merchant_withdrawals
    ADD COLUMN tax_quota bigint GENERATED ALWAYS AS
        (amount_quota * withdrawal_tax_bps / 10000) STORED,
    ADD COLUMN net_payout_quota bigint GENERATED ALWAYS AS
        (amount_quota - amount_quota * withdrawal_tax_bps / 10000) STORED;
