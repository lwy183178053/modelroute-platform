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

func TestSQLRoutingStoreAgainstPostgres(t *testing.T) {
	databaseURL := os.Getenv("MODELROUTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("未设置 PostgreSQL 集成测试地址")
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("打开测试数据库：%v", err)
	}
	defer database.Close()
	ctx := context.Background()
	if err := migrations.Up(ctx, database); err != nil {
		t.Fatalf("执行迁移：%v", err)
	}
	if _, err := database.ExecContext(ctx, `
TRUNCATE channel_health_events, performance_rollups, routing_decisions,
         routing_policy_merchants, routing_policies, token_bindings,
         channel_model_health, offer_versions, official_price_versions,
         canonical_models, merchant_channels, merchant_channel_reviews,
         merchant_profiles, merchant_applications RESTART IDENTITY CASCADE
`); err != nil {
		t.Fatalf("清理路由测试表：%v", err)
	}

	var applicationID, merchantID, reviewID, channelID int64
	err = database.QueryRowContext(ctx, `
INSERT INTO merchant_applications
    (newapi_user_id, display_name, website_url, business_description, status)
VALUES (7, '集成测试商户', 'https://merchant.test', 'PostgreSQL 集成测试', 'approved')
RETURNING id
`).Scan(&applicationID)
	if err != nil {
		t.Fatalf("创建商户申请：%v", err)
	}
	err = database.QueryRowContext(ctx, `
INSERT INTO merchant_profiles
    (newapi_user_id, approved_application_id, display_name, website_url, approved_by_newapi_user_id)
VALUES (7, $1, '集成测试商户', 'https://merchant.test', 1)
RETURNING id
`, applicationID).Scan(&merchantID)
	if err != nil {
		t.Fatalf("创建商户资料：%v", err)
	}
	err = database.QueryRowContext(ctx, `
INSERT INTO merchant_channel_reviews
    (merchant_newapi_user_id, application_id, channel_name, channel_type, base_url,
     requested_models, credential_fingerprint, status, verified_models)
VALUES (7, $1, '集成测试渠道', 1, 'https://upstream.test/v1',
        '["gpt-4o-mini"]'::jsonb, $2, 'approved', '["gpt-4o-mini"]'::jsonb)
RETURNING id
`, applicationID, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").Scan(&reviewID)
	if err != nil {
		t.Fatalf("创建渠道审核：%v", err)
	}
	err = database.QueryRowContext(ctx, `
INSERT INTO merchant_channels
    (merchant_profile_id, review_id, newapi_channel_id, channel_name, channel_type,
     base_url, models, credential_fingerprint)
VALUES ($1, $2, 41, '集成测试渠道', 1, 'https://upstream.test/v1',
        '["gpt-4o-mini"]'::jsonb, $3)
RETURNING id
`, merchantID, reviewID, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Scan(&channelID)
	if err != nil {
		t.Fatalf("创建商户渠道：%v", err)
	}

	store := NewSQLRoutingStore(database)
	model, err := store.CreateCanonicalModel(ctx, CreateCanonicalModelCommand{
		PublicName: "gpt-4o-mini", Provider: "OpenAI", Protocol: "chat_completions", AdminID: 1,
		Price: OfficialPrice{BillingMode: "token", ModelRatioMicros: 600_000, CompletionRatioMicros: 4_000_000, CacheRatioMicros: 500_000},
	})
	if err != nil {
		t.Fatalf("创建标准模型：%v", err)
	}
	offer, err := store.CreateOffer(ctx, CreateOfferCommand{
		UserID: 7, MerchantChannelID: channelID, CanonicalModelID: model.ID,
		MultiplierBPS: 7200, InternalAlias: "mkt-ov-postgres-integration",
	})
	if err != nil {
		t.Fatalf("创建报价：%v", err)
	}
	if err := store.ActivateOffer(ctx, offer.ID); err != nil {
		t.Fatalf("激活报价：%v", err)
	}
	if _, err := store.BindTokenAndDefaultPolicy(ctx, 7, 9, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"); err != nil {
		t.Fatalf("绑定令牌策略：%v", err)
	}
	if _, err := store.UpdatePolicy(ctx, 7, 9, RoutingPolicyInput{
		Mode: "smart", Priority: "price", MinimumMultiplierBPS: 7300, MaximumMultiplierBPS: 8000,
	}); err != nil {
		t.Fatalf("保存排除倍率策略：%v", err)
	}
	if _, err := database.ExecContext(ctx, `
UPDATE channel_model_health
SET state = 'healthy', availability_ppm = 999000, ttft_p95_ms = 1200, next_probe_at = now() - interval '1 second'
WHERE merchant_channel_id = $1 AND canonical_model_id = $2
`, channelID, model.ID); err != nil {
		t.Fatalf("准备健康状态：%v", err)
	}
	selection, err := store.ResolveRoute(ctx, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "gpt-4o-mini")
	if err != nil || len(selection.Candidates) != 0 {
		t.Fatalf("倍率区间应排除候选：候选=%d 错误=%v", len(selection.Candidates), err)
	}
	if _, err := store.UpdatePolicy(ctx, 7, 9, RoutingPolicyInput{
		Mode: "fixed", Priority: "stability", MinimumMultiplierBPS: 7000, MaximumMultiplierBPS: 8000,
		MerchantIDs: []int64{merchantID},
	}); err != nil {
		t.Fatalf("保存固定商户策略：%v", err)
	}
	selection, err = store.ResolveRoute(ctx, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "gpt-4o-mini")
	if err != nil || len(selection.Candidates) != 1 || selection.Candidates[0].OfferID != offer.ID {
		t.Fatalf("固定商户候选不正确：%+v，错误=%v", selection.Candidates, err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	probes, err := store.ClaimHealthProbes(ctx, now, 20)
	if err != nil || len(probes) != 1 {
		t.Fatalf("领取探活任务：数量=%d 错误=%v", len(probes), err)
	}
	next := nextProbeState(probes[0], healthFailureTemporary, now, 0)
	if err := store.SaveProbeResult(ctx, HealthProbeResult{
		Probe: probes[0], State: next, Class: healthFailureTemporary,
		StatusCode: 503, FirstByteMS: 240, Summary: "集成测试临时故障",
	}); err != nil {
		t.Fatalf("保存探活结果：%v", err)
	}
	var state string
	var observations, successes int64
	var lastStatus int
	if err := database.QueryRowContext(ctx, `
SELECT state, observations, successes, last_status_code
FROM channel_model_health
WHERE merchant_channel_id = $1 AND canonical_model_id = $2
`, channelID, model.ID).Scan(&state, &observations, &successes, &lastStatus); err != nil {
		t.Fatalf("读取探活结果：%v", err)
	}
	if state != "degraded" || observations != 1 || successes != 0 || lastStatus != 503 {
		t.Fatalf("探活结果不正确：状态=%s 观测=%d 成功=%d 状态码=%d", state, observations, successes, lastStatus)
	}
}
