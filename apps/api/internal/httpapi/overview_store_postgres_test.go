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
TRUNCATE route_metric_snapshots, route_metric_buckets, merchant_name_reservations,
         merchant_applications, canonical_models, token_bindings,
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

	var modelID, priceID int64
	if err := database.QueryRowContext(ctx, `
INSERT INTO canonical_models (public_name, provider)
VALUES ('gpt-4o-mini', 'OpenAI') RETURNING id
`).Scan(&modelID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO official_price_versions
    (canonical_model_id, version, price_kind, published_by_newapi_user_id)
VALUES ($1, 1, 'text', 1) RETURNING id
`, modelID).Scan(&priceID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO text_price_tiers
    (official_price_version_id, tier_order, minimum_input_tokens, input_price_micros,
     output_price_micros, cache_read_price_micros)
VALUES ($1, 1, 0, 600000, 4000000, 500000)
`, priceID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `UPDATE canonical_models SET current_official_price_version_id = $1 WHERE id = $2`, priceID, modelID); err != nil {
		t.Fatal(err)
	}

	var pendingApplicationID, approvedApplicationID, merchantID, pendingTestID, approvedTestID, approvedReviewID, channelID int64
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_applications
    (newapi_user_id, display_name, display_name_key, website_url, business_description, status)
VALUES (7, '待审商户', '待审商户', 'https://pending.example', '待审核商户业务说明', 'pending') RETURNING id
`).Scan(&pendingApplicationID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_applications
    (newapi_user_id, display_name, display_name_key, website_url, business_description, status)
VALUES (8, '已审商户', '已审商户', 'https://approved.example', '已审核商户业务说明', 'approved') RETURNING id
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
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_channel_tests
    (newapi_user_id, canonical_model_id, base_url, credential_fingerprint, adapter_type,
     capabilities, latency_ms, receipt_token_hash, expires_at, consumed_at)
VALUES (7, $1, 'https://pending-upstream.example', $2, 'openai_compatible',
        '["chat_completions"]'::jsonb, 20, $3, now() + interval '15 minutes', now())
RETURNING id
`, modelID,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"1111111111111111111111111111111111111111111111111111111111111111").Scan(&pendingTestID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_channel_tests
    (newapi_user_id, canonical_model_id, base_url, credential_fingerprint, adapter_type,
     capabilities, latency_ms, receipt_token_hash, expires_at, consumed_at)
VALUES (8, $1, 'https://approved-upstream.example', $2, 'openai_compatible',
        '["chat_completions"]'::jsonb, 18, $3, now() + interval '15 minutes', now())
RETURNING id
`, modelID,
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"2222222222222222222222222222222222222222222222222222222222222222").Scan(&approvedTestID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO merchant_channel_reviews
	(merchant_newapi_user_id, application_id, canonical_model_id, channel_test_id, channel_name,
	 base_url, adapter_type, encrypted_credential, credential_fingerprint, status)
VALUES (7, $1, $2, $3, '待审渠道', 'https://pending-upstream.example',
	'openai_compatible', 'encrypted', $4, 'pending')
`, pendingApplicationID, modelID, pendingTestID, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_channel_reviews
	(merchant_newapi_user_id, merchant_profile_id, application_id, canonical_model_id, channel_test_id,
	 channel_name, base_url, adapter_type, credential_fingerprint, status)
VALUES (8, $1, $2, $3, $4, '已审渠道', 'https://approved-upstream.example',
	'openai_compatible', $5, 'approved') RETURNING id
`, merchantID, approvedApplicationID, modelID, approvedTestID,
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Scan(&approvedReviewID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_channels
	(merchant_profile_id, review_id, canonical_model_id, newapi_channel_id, channel_name,
	 base_url, adapter_type, credential_fingerprint)
VALUES ($1, $2, $3, 41, '已审渠道', 'https://approved-upstream.example',
	'openai_compatible', $4) RETURNING id
`, merchantID, approvedReviewID, modelID,
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc").Scan(&channelID); err != nil {
		t.Fatal(err)
	}
	const alias = "mkt-ov-0123456789abcdef01234567"
	var offerID int64
	if err := database.QueryRowContext(ctx, `
INSERT INTO offer_versions
	(merchant_profile_id, canonical_model_id, official_price_version_id, multiplier_bps,
	 version, status, created_by_newapi_user_id, activated_at)
VALUES ($1, $2, $3, 9000, 1, 'active', 8, $4) RETURNING id
`, merchantID, modelID, priceID, now).Scan(&offerID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO offer_channel_aliases (offer_version_id, merchant_channel_id, internal_alias)
VALUES ($1, $2, $3)
`, offerID, channelID, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO channel_health (merchant_channel_id, state, next_probe_at, cooldown_until)
VALUES ($1, 'cooling', $2, $2)
`, channelID, now); err != nil {
		t.Fatal(err)
	}
	var routingPolicyID int64
	if err := database.QueryRowContext(ctx, `
INSERT INTO routing_policies
	(newapi_user_id, name, priority, minimum_multiplier_bps, maximum_multiplier_bps)
VALUES (7, '运营测试路由', 'success', 0, 50000)
RETURNING id
`).Scan(&routingPolicyID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO token_bindings (newapi_token_id, newapi_user_id, token_fingerprint, routing_policy_id)
VALUES (9, 7, $1, $2)
`, "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", routingPolicyID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO routing_decisions
	(request_id, token_binding_id, routing_policy_id, canonical_model_id, priority,
	 algorithm_version, candidate_count, selected_metrics, result, upstream_status_code, created_at)
SELECT 'req-ok', id, $1, $2, 'success', 'weighted-v1', 1, '{}'::jsonb,
       'completed', 200, $3 FROM token_bindings WHERE newapi_token_id = 9
`, routingPolicyID, modelID, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO routing_decisions
	(request_id, token_binding_id, routing_policy_id, canonical_model_id, priority,
	 algorithm_version, candidate_count, selected_metrics, result, upstream_status_code, created_at)
SELECT 'req-error', id, $1, $2, 'success', 'weighted-v1', 1, '{}'::jsonb,
       'completed', 500, $3 FROM token_bindings WHERE newapi_token_id = 9
`, routingPolicyID, modelID, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	if _, err := database.ExecContext(ctx, `
INSERT INTO payment_setting_versions
    (version, gateway_url, merchant_pid, encrypted_key, key_fingerprint, payment_methods,
     minimum_cny_cents, maximum_cny_cents, enabled, created_by_newapi_user_id)
VALUES (1, 'https://pay.example', 'pid', 'encrypted', $1, '["alipay"]'::jsonb, 100, 10000, true, 1)
`, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO merchant_balances (merchant_profile_id, available_quota) VALUES ($1, 50000000)`, merchantID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO merchant_withdrawals
    (withdrawal_no, merchant_profile_id, amount_quota, available_before_quota,
     recipient_type, encrypted_recipient_account, recipient_account_masked)
VALUES ('MW-OVERVIEW', $1, 25000000, 50000000, 'alipay', 'encrypted', '138****2621')
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
