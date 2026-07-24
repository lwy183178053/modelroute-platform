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
TRUNCATE route_metric_snapshots, route_metric_buckets, merchant_name_reservations,
         channel_health_events, performance_rollups, routing_decisions,
		 token_model_routes, routing_policies, token_bindings,
         channel_health, offer_channel_aliases, offer_versions,
         merchant_channels, merchant_channel_reviews, merchant_channel_tests,
         merchant_profiles, merchant_applications, text_price_tiers,
         image_price_items, official_price_versions, canonical_models
RESTART IDENTITY CASCADE
`); err != nil {
		t.Fatalf("清理路由测试表：%v", err)
	}

	store := NewSQLRoutingStore(database)
	outputPrice := int64(4_800_000)
	cacheReadPrice := int64(600_000)
	model, err := store.CreateCanonicalModel(ctx, CreateCanonicalModelCommand{
		PublicName: "gpt-4o-mini", Provider: "OpenAI", AdminID: 1,
		Price: OfficialPrice{Tiers: []PriceTier{{
			MinimumInputTokens: 0, InputPriceMicros: 1_200_000,
			OutputPriceMicros: &outputPrice, CacheReadPriceMicros: &cacheReadPrice,
		}}},
	})
	if err != nil {
		t.Fatalf("创建标准模型：%v", err)
	}

	var applicationID, merchantID, testID, reviewID, channelID int64
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_applications
    (newapi_user_id, display_name, display_name_key, website_url, business_description, status)
VALUES (7, '集成测试商户', '集成测试商户', 'https://merchant.test', 'PostgreSQL 集成测试', 'approved')
RETURNING id
`).Scan(&applicationID); err != nil {
		t.Fatalf("创建商户申请：%v", err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_profiles
    (newapi_user_id, approved_application_id, display_name, website_url, approved_by_newapi_user_id)
VALUES (7, $1, '集成测试商户', 'https://merchant.test', 1)
RETURNING id
`, applicationID).Scan(&merchantID); err != nil {
		t.Fatalf("创建商户资料：%v", err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_channel_tests
    (newapi_user_id, canonical_model_id, base_url, credential_fingerprint,
     adapter_type, capabilities, latency_ms, receipt_token_hash, expires_at, consumed_at)
VALUES (7, $1, 'https://upstream.test/v1', $2, 'openai_compatible',
        '["text"]'::jsonb, 10, $3, now() + interval '15 minutes', now())
RETURNING id
`, model.ID,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc").Scan(&testID); err != nil {
		t.Fatalf("创建渠道检测：%v", err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_channel_reviews
    (merchant_newapi_user_id, merchant_profile_id, application_id, canonical_model_id,
     channel_test_id, channel_name, base_url, provider, adapter_type, credential_fingerprint, status)
VALUES (7, $1, $2, $3, $4, '集成测试渠道', 'https://upstream.test/v1',
        'OpenAI', 'openai_compatible', $5, 'approved')
RETURNING id
`, merchantID, applicationID, model.ID, testID,
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Scan(&reviewID); err != nil {
		t.Fatalf("创建渠道审核：%v", err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO merchant_channels
    (merchant_profile_id, review_id, canonical_model_id, newapi_channel_id,
     channel_name, base_url, provider, adapter_type, credential_fingerprint)
VALUES ($1, $2, $3, 41, '集成测试渠道', 'https://upstream.test/v1',
        'OpenAI', 'openai_compatible', $4)
RETURNING id
`, merchantID, reviewID, model.ID,
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb").Scan(&channelID); err != nil {
		t.Fatalf("创建商户渠道：%v", err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO channel_health (merchant_channel_id) VALUES ($1)`, channelID); err != nil {
		t.Fatalf("创建渠道健康状态：%v", err)
	}

	offer, err := store.CreateOffer(ctx, CreateOfferCommand{UserID: 7, CanonicalModelID: model.ID, MultiplierBPS: 7200})
	if err != nil {
		t.Fatalf("创建报价：%v", err)
	}
	if err := store.ActivateOffer(ctx, offer.ID); err != nil {
		t.Fatalf("激活报价：%v", err)
	}
	models, err := store.ListCanonicalModels(ctx)
	if err != nil || len(models) != 1 || len(models[0].Price.Tiers) != 1 || models[0].Price.Tiers[0].InputPriceMicros != 1_200_000 {
		t.Fatalf("标准模型目录=%+v，错误=%v", models, err)
	}
	market, err := store.ListMarketOffers(ctx, 0, "gpt-4o-mini")
	if err != nil || len(market) != 1 || market[0].MultiplierBPS != 7200 || market[0].ChannelCount != 1 || len(market[0].PriceTiers) != 1 || market[0].PriceTiers[0].InputPriceMicros != 864_000 {
		t.Fatalf("市场折后价格=%+v，错误=%v", market, err)
	}
	if _, err := database.ExecContext(ctx, `
UPDATE channel_health
SET state = 'cooling', cooldown_until = now() + interval '5 minutes', next_probe_at = now() + interval '5 minutes'
WHERE merchant_channel_id = $1
`, channelID); err != nil {
		t.Fatalf("设置渠道冷却状态：%v", err)
	}
	market, err = store.ListMarketOffers(ctx, 0, "gpt-4o-mini")
	if err != nil || len(market) != 0 {
		t.Fatalf("冷却渠道不应出现在市场：报价=%+v，错误=%v", market, err)
	}
	if _, err := database.ExecContext(ctx, `
UPDATE channel_health
SET state = 'degraded', cooldown_until = NULL, next_probe_at = NULL
WHERE merchant_channel_id = $1
`, channelID); err != nil {
		t.Fatalf("恢复渠道降级状态：%v", err)
	}
	market, err = store.ListMarketOffers(ctx, 0, "gpt-4o-mini")
	if err != nil || len(market) != 1 || market[0].ChannelCount != 1 {
		t.Fatalf("降级渠道应作为可用渠道展示：报价=%+v，错误=%v", market, err)
	}
	lastCalledAt := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Second)
	if _, err := database.ExecContext(ctx, `
INSERT INTO settlement_sources
    (newapi_log_id, newapi_user_id, source_created_at, source_type, model_name,
     newapi_channel_id, newapi_token_id, request_id, gross_quota, other, status,
     merchant_profile_id, offer_version_id, offer_channel_alias_id, merchant_quota,
     platform_quota, processed_at)
VALUES (9001, 21, $1, 2, $2, 41, 9, 'req-merchant-stats', 1000,
        '{"prompt_tokens":"2000000000","completion_tokens":"600000000"}'::jsonb,
        'settled', $3, $4, $5, 900, 100, now())
`, lastCalledAt, offer.Aliases[0].InternalAlias, merchantID, offer.ID, offer.Aliases[0].ID); err != nil {
		t.Fatalf("创建商户调用统计来源：%v", err)
	}
	market, err = store.ListMarketOffers(ctx, 0, "gpt-4o-mini")
	if err != nil || len(market) != 1 || market[0].CallCount != 1 || market[0].TokenUsage != 2_600_000_000 || market[0].LastCalledAt == nil || !market[0].LastCalledAt.Equal(lastCalledAt) {
		t.Fatalf("市场调用统计=%+v，错误=%v", market, err)
	}
	merchantOffers, err := store.ListMerchantOffers(ctx, 7)
	if err != nil || len(merchantOffers) != 1 || merchantOffers[0].ChannelCount != 1 || merchantOffers[0].CallCount != 1 || merchantOffers[0].TokenUsage != 2_600_000_000 || merchantOffers[0].LastCalledAt == nil || !merchantOffers[0].LastCalledAt.Equal(lastCalledAt) || len(merchantOffers[0].Channels) != 1 || len(merchantOffers[0].PriceTiers) != 1 || merchantOffers[0].PriceTiers[0].OutputPriceMicros == nil || *merchantOffers[0].PriceTiers[0].OutputPriceMicros != 3_456_000 {
		t.Fatalf("商户折后价格=%+v，错误=%v", merchantOffers, err)
	}

	policy, err := store.CreateRoutingPolicy(ctx, 7, RoutingPolicyInput{
		CanonicalModelID: model.ID, Name: "主路由", Priority: "price", MinimumMultiplierBPS: 7300, MaximumMultiplierBPS: 8000,
	})
	if err != nil {
		t.Fatalf("创建路由：%v", err)
	}
	if _, err := store.CreateTokenBinding(ctx, 7, 9, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "OpenAI", []int64{policy.ID}); err != nil {
		t.Fatalf("绑定密钥路由：%v", err)
	}
	if _, err := store.UpdateRoutingPolicy(ctx, 7, policy.ID, RoutingPolicyInput{
		Name: "主路由", Priority: "price", MinimumMultiplierBPS: 7300, MaximumMultiplierBPS: 8000,
	}); err != nil {
		t.Fatalf("保存排除倍率策略：%v", err)
	}
	selection, err := store.ResolveRoute(ctx, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "gpt-4o-mini")
	if err != nil || len(selection.Candidates) != 0 {
		t.Fatalf("倍率区间应排除候选：候选=%d 错误=%v", len(selection.Candidates), err)
	}
	if _, err := store.UpdateRoutingPolicy(ctx, 7, policy.ID, RoutingPolicyInput{
		Name: "主路由", Priority: "success", MinimumMultiplierBPS: 0, MaximumMultiplierBPS: 50_000,
	}); err != nil {
		t.Fatalf("保存智能路由策略：%v", err)
	}
	selection, err = store.ResolveRoute(ctx, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "gpt-4o-mini")
	if err != nil || len(selection.Candidates) != 1 || selection.Candidates[0].OfferID != offer.ID || selection.Candidates[0].OfferChannelAliasID == 0 {
		t.Fatalf("智能路由候选不正确：%+v，错误=%v", selection.Candidates, err)
	}
	if err := store.StartRouteDecision(ctx, RouteDecision{
		RequestID: "postgres-routing-decision", TokenBindingID: selection.TokenBindingID,
		PolicyID: selection.PolicyID, CanonicalModelID: selection.CanonicalModelID,
		OfferChannelAliasID: selection.Candidates[0].OfferChannelAliasID,
		Priority:            selection.Priority, AlgorithmVersion: routeAlgorithmVersion,
		CandidateCount: len(selection.Candidates), SelectedMetrics: routeDecisionMetrics(selection.Candidates[0]),
	}); err != nil {
		t.Fatalf("写入新版路由决策：%v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	probes, err := store.ClaimHealthProbes(ctx, now, 20)
	if err != nil || len(probes) != 0 {
		t.Fatalf("健康渠道不应主动探活：数量=%d 错误=%v", len(probes), err)
	}
	for range 3 {
		if err := store.RecordRouteObservation(ctx, RouteObservation{
			Candidate: selection.Candidates[0], Outcome: "temporary_failure",
			StatusCode: 503, FirstByteMS: 240, RequestID: time.Now().String(),
		}); err != nil {
			t.Fatalf("保存真实失败：%v", err)
		}
	}
	probes, err = store.ClaimHealthProbes(ctx, now.Add(2*time.Minute), 20)
	if err != nil || len(probes) != 1 {
		t.Fatalf("领取冷却探活任务：数量=%d 错误=%v", len(probes), err)
	}
	next := nextProbeState(probes[0], healthFailureNone, now.Add(2*time.Minute), 0)
	if err := store.SaveProbeResult(ctx, HealthProbeResult{
		Probe: probes[0], State: next, Class: healthFailureNone,
		StatusCode: 200, FirstByteMS: 120,
	}); err != nil {
		t.Fatalf("保存恢复探活结果：%v", err)
	}
	var state string
	var nextProbe sql.NullTime
	if err := database.QueryRowContext(ctx, `
SELECT state, next_probe_at FROM channel_health WHERE merchant_channel_id = $1
`, channelID).Scan(&state, &nextProbe); err != nil {
		t.Fatalf("读取探活结果：%v", err)
	}
	if state != "healthy" || nextProbe.Valid {
		t.Fatalf("探活恢复不正确：状态=%s 下次探活=%v", state, nextProbe)
	}
	repricedOffer, err := store.CreateOffer(ctx, CreateOfferCommand{UserID: 7, CanonicalModelID: model.ID, MultiplierBPS: 8500})
	if err != nil {
		t.Fatalf("创建调价报价：%v", err)
	}
	if err := store.ActivateOffer(ctx, repricedOffer.ID); err != nil {
		t.Fatalf("激活调价报价：%v", err)
	}
	merchantOffers, err = store.ListMerchantOffers(ctx, 7)
	if err != nil || len(merchantOffers) != 1 || merchantOffers[0].OfferID != repricedOffer.ID || merchantOffers[0].MultiplierBPS != 8500 || merchantOffers[0].CallCount != 1 || merchantOffers[0].TokenUsage != 2_600_000_000 || merchantOffers[0].LastCalledAt == nil || !merchantOffers[0].LastCalledAt.Equal(lastCalledAt) {
		t.Fatalf("调价后商户调用统计丢失：报价=%+v，错误=%v", merchantOffers, err)
	}
	affectedKeys, err := store.DeleteRoutingPolicy(ctx, 7, policy.ID)
	if err != nil || affectedKeys != 1 {
		t.Fatalf("删除已使用路由：受影响密钥=%d，错误=%v", affectedKeys, err)
	}
	var historicalTokenBindingID, historicalPolicyID sql.NullInt64
	if err := database.QueryRowContext(ctx, `
SELECT token_binding_id, routing_policy_id
FROM routing_decisions
WHERE request_id = 'postgres-routing-decision'
`).Scan(&historicalTokenBindingID, &historicalPolicyID); err != nil {
		t.Fatalf("删除路由后读取历史调用记录：%v", err)
	}
	if !historicalTokenBindingID.Valid || historicalPolicyID.Valid {
		t.Fatalf("删除路由后的历史关联不正确：密钥=%v 路由=%v", historicalTokenBindingID, historicalPolicyID)
	}
}
