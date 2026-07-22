package httpapi

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/migrations"
)

func TestSQLAdminOverviewStoreUsesSettledIntegerLedgerData(t *testing.T) {
	databaseURL := os.Getenv("MODELROUTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("未设置 PostgreSQL 集成测试地址")
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	ctx := context.Background()
	if err := migrations.Up(ctx, database); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
TRUNCATE merchant_applications, canonical_models, token_bindings,
         settlement_sync_state, payment_setting_versions
RESTART IDENTITY CASCADE
`); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	lastSuccess := now.Add(-10 * time.Second)
	if _, err := database.ExecContext(ctx, `
INSERT INTO settlement_sync_state (singleton, last_synced_through, last_success_at)
VALUES (true, $1, $2)
`, now.Add(-5*time.Second).Unix(), lastSuccess); err != nil {
		t.Fatal(err)
	}

	var pendingApplicationID, approvedApplicationID, merchantID, approvedReviewID, channelID int64
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_applications
    (newapi_user_id, display_name, website_url, business_description, status)
VALUES (7, '待审商户', 'https://pending.example', '待审核商户业务说明', 'pending') RETURNING id
`).Scan(&pendingApplicationID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_applications
    (newapi_user_id, display_name, website_url, business_description, status)
VALUES (8, '已审商户', 'https://approved.example', '已审核商户业务说明', 'approved') RETURNING id
`).Scan(&approvedApplicationID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_profiles
    (newapi_user_id, approved_application_id, display_name, website_url, approved_by_newapi_user_id)
VALUES (8, $1, '已审商户', 'https://approved.example', 1) RETURNING id
`, approvedApplicationID).Scan(&merchantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO merchant_channel_reviews
    (merchant_newapi_user_id, application_id, channel_name, channel_type, base_url,
     requested_models, encrypted_credential, credential_fingerprint, status)
VALUES (7, $1, '待审渠道', 1, 'https://pending-upstream.example',
        '["gpt-4o-mini"]'::jsonb, 'encrypted', $2, 'pending')
`, pendingApplicationID, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_channel_reviews
    (merchant_newapi_user_id, application_id, channel_name, channel_type, base_url,
     requested_models, credential_fingerprint, status, verified_models)
VALUES (8, $1, '已审渠道', 1, 'https://approved-upstream.example',
        '["gpt-4o-mini"]'::jsonb, $2, 'approved', '["gpt-4o-mini"]'::jsonb) RETURNING id
`, approvedApplicationID, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Scan(&approvedReviewID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_channels
    (merchant_profile_id, review_id, newapi_channel_id, channel_name, channel_type, base_url, models, credential_fingerprint)
VALUES ($1, $2, 41, '已审渠道', 1, 'https://approved-upstream.example', '["gpt-4o-mini"]'::jsonb, $3) RETURNING id
`, merchantID, approvedReviewID, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc").Scan(&channelID); err != nil {
		t.Fatal(err)
	}

	var modelID, priceID int64
	if err := database.QueryRowContext(ctx, `
INSERT INTO canonical_models (public_name, provider, protocol)
VALUES ('gpt-4o-mini', 'OpenAI', 'chat_completions') RETURNING id
`).Scan(&modelID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO official_price_versions
    (canonical_model_id, version, billing_mode, model_ratio_micros, completion_ratio_micros,
     cache_ratio_micros, create_cache_ratio_micros, image_ratio_micros, audio_ratio_micros,
     audio_completion_ratio_micros, published_by_newapi_user_id)
VALUES ($1, 1, 'token', 600000, 4000000, 500000, 0, 0, 0, 0, 1) RETURNING id
`, modelID).Scan(&priceID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `UPDATE canonical_models SET current_official_price_version_id = $1 WHERE id = $2`, priceID, modelID); err != nil {
		t.Fatal(err)
	}
	const alias = "mkt-ov-0123456789abcdef01234567"
	if _, err := database.ExecContext(ctx, `
INSERT INTO offer_versions
    (merchant_channel_id, canonical_model_id, official_price_version_id, multiplier_bps,
     internal_alias, status, created_by_newapi_user_id, activated_at)
VALUES ($1, $2, $3, 9000, $4, 'active', 8, $5)
`, channelID, modelID, priceID, alias, now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO channel_model_health (merchant_channel_id, canonical_model_id, state, next_probe_at)
VALUES ($1, $2, 'cooling', $3)
`, channelID, modelID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO token_bindings (newapi_token_id, newapi_user_id, token_fingerprint)
VALUES (9, 7, $1)
`, "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO routing_decisions
    (request_id, token_binding_id, canonical_model_id, mode, priority, result, upstream_status_code, created_at)
SELECT 'req-ok', id, $1, 'smart', 'stability', 'completed', 200, $2 FROM token_bindings WHERE newapi_token_id = 9
`, modelID, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO routing_decisions
    (request_id, token_binding_id, canonical_model_id, mode, priority, result, upstream_status_code, created_at)
SELECT 'req-error', id, $1, 'smart', 'stability', 'completed', 500, $2 FROM token_bindings WHERE newapi_token_id = 9
`, modelID, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	if _, err := database.ExecContext(ctx, `
INSERT INTO payment_setting_versions
    (version, gateway_url, merchant_pid, encrypted_key, key_fingerprint, payment_methods,
     minimum_usd_cents, maximum_usd_cents, exchange_rate_cny_cents_per_usd, enabled, created_by_newapi_user_id)
VALUES (1, 'https://pay.example', 'pid', 'encrypted', $1, '["alipay"]'::jsonb, 100, 10000, 720, true, 1)
`, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO merchant_balances (merchant_profile_id, available_quota) VALUES ($1, 1000)`, merchantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO merchant_withdrawals
    (withdrawal_no, merchant_profile_id, amount_quota, available_before_quota,
     recipient_type, encrypted_recipient_account, recipient_account_masked)
VALUES ('MW-OVERVIEW', $1, 100, 1000, 'alipay', 'encrypted', '138****2621')
`, merchantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO settlement_sources
    (newapi_log_id, newapi_user_id, source_created_at, source_type, model_name,
     newapi_channel_id, gross_quota, other, status, merchant_profile_id, merchant_quota, platform_quota)
VALUES
    (1001, 7, $1, 2, $2, 41, 1000, '{}'::jsonb, 'settled', $3, 900, 100),
    (1002, 7, $1, 6, $2, 41, 100, '{}'::jsonb, 'settled', $3, 90, 10),
    (1003, 7, $1, 2, 'missing-alias', 99, 50, '{}'::jsonb, 'reconciliation', NULL, 0, 0)
`, now.Add(-time.Minute), alias, merchantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO settlement_reconciliation_items (settlement_source_id, reason_code, details)
SELECT id, 'offer_mapping_missing', '{}'::jsonb FROM settlement_sources WHERE newapi_log_id = 1003
`); err != nil {
		t.Fatal(err)
	}

	snapshot, err := NewSQLAdminOverviewStore(database).Snapshot(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.TodayGrossQuota != 900 || snapshot.TodayMerchantQuota != 810 || snapshot.TodayPlatformQuota != 90 || snapshot.TodayRefundQuota != 100 {
		t.Fatalf("总览整数账务=%+v", snapshot)
	}
	if snapshot.TodayRequestCount != 2 || snapshot.TodaySuccessfulRequests != 1 || snapshot.PendingMerchantApplications != 1 || snapshot.PendingChannelReviews != 1 || snapshot.PendingWithdrawals != 1 || snapshot.OpenReconciliation != 1 {
		t.Fatalf("总览请求与待办=%+v", snapshot)
	}
	if snapshot.TotalChannels != 1 || snapshot.ActiveChannels != 1 || snapshot.CoolingChannels != 1 || snapshot.DisabledChannels != 0 || !snapshot.PaymentEnabled || snapshot.LastSyncedThrough == 0 || snapshot.LastSyncSuccessAt == nil {
		t.Fatalf("总览运行状态=%+v", snapshot)
	}

	mappings, err := NewSQLRoutingStore(database).PublicModelNames(ctx, []string{alias, "missing-alias", alias})
	if err != nil || len(mappings) != 1 || mappings[alias] != "gpt-4o-mini" {
		t.Fatalf("别名映射=%v，错误=%v", mappings, err)
	}
}
