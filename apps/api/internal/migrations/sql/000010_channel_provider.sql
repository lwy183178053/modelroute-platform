TRUNCATE TABLE merchant_channel_reviews, merchant_channels RESTART IDENTITY CASCADE;

ALTER TABLE merchant_channel_reviews
    ADD COLUMN provider varchar(32) NOT NULL,
    ADD CONSTRAINT merchant_channel_reviews_provider_check
        CHECK (provider IN ('OpenAI', 'Anthropic', 'Gemini', 'DeepSeek', 'Qwen', 'Moonshot'));

ALTER TABLE merchant_channels
    ADD COLUMN provider varchar(32) NOT NULL,
    ADD CONSTRAINT merchant_channels_provider_check
        CHECK (provider IN ('OpenAI', 'Anthropic', 'Gemini', 'DeepSeek', 'Qwen', 'Moonshot'));
