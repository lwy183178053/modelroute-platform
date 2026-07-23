package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCreateRoutingPolicyStoresIndependentSmartRoute(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	createdAt := time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO routing_policies")).
		WithArgs(7, "智能主路由", "balanced", 2_000, 15_000, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "updated_at"}).AddRow(91, createdAt))

	policy, err := NewSQLRoutingStore(database).CreateRoutingPolicy(context.Background(), 7, RoutingPolicyInput{
		Name: " 智能主路由 ", Priority: "balanced",
		MinimumMultiplierBPS: 2_000, MaximumMultiplierBPS: 15_000,
	})
	if err != nil || policy.ID != 91 || policy.Name != "智能主路由" || policy.UserID != 7 {
		t.Fatalf("创建路由=%+v err=%v", policy, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBindAndRebindTokenToOwnedRoutingPolicy(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLRoutingStore(database)

	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO token_bindings")).
		WithArgs(9, 7, "fingerprint", int64(91)).
		WillReturnRows(sqlmock.NewRows([]string{"newapi_token_id", "routing_policy_id", "name"}).AddRow(9, 91, "主路由"))
	binding, err := store.BindTokenToPolicy(context.Background(), 7, 9, "fingerprint", 91)
	if err != nil || binding.TokenID != 9 || binding.PolicyID != 91 || binding.PolicyName != "主路由" {
		t.Fatalf("绑定结果=%+v err=%v", binding, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE token_bindings binding")).
		WithArgs(7, 9, int64(92)).
		WillReturnRows(sqlmock.NewRows([]string{"newapi_token_id", "routing_policy_id", "name"}).AddRow(9, 92, "备用路由"))
	binding, err = store.RebindTokenPolicy(context.Background(), 7, 9, 92)
	if err != nil || binding.PolicyID != 92 || binding.PolicyName != "备用路由" {
		t.Fatalf("重新绑定结果=%+v err=%v", binding, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteRoutingPolicyRejectsBoundKeys(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery(regexp.QuoteMeta("WITH target AS")).
		WithArgs(7, int64(91)).
		WillReturnRows(sqlmock.NewRows([]string{"bound_key_count", "deleted_id"}).AddRow(2, nil))

	err = NewSQLRoutingStore(database).DeleteRoutingPolicy(context.Background(), 7, 91)
	if !errors.Is(err, ErrRoutingInvalidState) {
		t.Fatalf("删除已绑定路由错误=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRoutingStoreErrorMapsUniquePolicyNameToConflict(t *testing.T) {
	err := routingStoreError(&pgconn.PgError{Code: "23505", ConstraintName: "routing_policies_user_name_idx"})
	if !errors.Is(err, ErrRoutingInvalidState) {
		t.Fatalf("路由名称重复错误=%v", err)
	}
}

func TestRoutingPolicyCacheFloorScansNullableValue(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery(regexp.QuoteMeta("FROM routing_policies policy")).
		WithArgs(7, int64(91)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "priority", "minimum_multiplier_bps", "maximum_multiplier_bps", "minimum_cache_hit_ppm", "bound_key_count", "updated_at"}).
			AddRow(91, 7, "缓存路由", "cache", 0, 50_000, sql.NullInt64{Int64: 650_000, Valid: true}, 1, time.Now()))
	policy, err := NewSQLRoutingStore(database).RoutingPolicy(context.Background(), 7, 91)
	if err != nil || policy.MinimumCacheHitPPM == nil || *policy.MinimumCacheHitPPM != 650_000 {
		t.Fatalf("缓存阈值路由=%+v err=%v", policy, err)
	}
}

func TestListAndUpdateIndependentRoutingPolicies(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLRoutingStore(database)
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("FROM routing_policies policy")).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "name", "priority", "minimum_multiplier_bps", "maximum_multiplier_bps", "minimum_cache_hit_ppm", "bound_key_count", "updated_at"}).
			AddRow(91, 7, "主路由", "price", 0, 50_000, nil, 2, now).
			AddRow(92, 7, "备用路由", "success", 5_000, 10_000, nil, 0, now))
	items, err := store.ListRoutingPolicies(context.Background(), 7)
	if err != nil || len(items) != 2 || items[0].BoundKeyCount != 2 || items[1].Priority != "success" {
		t.Fatalf("路由列表=%+v err=%v", items, err)
	}

	cacheFloor := 600_000
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE routing_policies")).
		WithArgs("缓存优先", "cache", 0, 50_000, &cacheFloor, 7, int64(91)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "bound_key_count", "updated_at"}).AddRow(91, 2, now))
	updated, err := store.UpdateRoutingPolicy(context.Background(), 7, 91, RoutingPolicyInput{
		Name: "缓存优先", Priority: "cache",
		MinimumMultiplierBPS: 0, MaximumMultiplierBPS: 50_000,
		MinimumCacheHitPPM: &cacheFloor,
	})
	if err != nil || updated.Name != "缓存优先" || updated.BoundKeyCount != 2 || updated.MinimumCacheHitPPM == nil {
		t.Fatalf("更新路由=%+v err=%v", updated, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListTokenRoutingBindingsReturnsPlatformRouteIdentity(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery(regexp.QuoteMeta("FROM token_bindings binding")).
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"token_id", "policy_id", "policy_name"}).AddRow(9, 91, "主路由"))
	items, err := NewSQLRoutingStore(database).ListTokenRoutingBindings(context.Background(), 7)
	if err != nil || len(items) != 1 || items[0].TokenID != 9 || items[0].PolicyName != "主路由" {
		t.Fatalf("密钥路由摘要=%+v err=%v", items, err)
	}
}

func TestResolveRouteKeepsInsufficientCacheSampleAndFiltersKnownLowValue(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	fingerprint := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	mock.ExpectQuery(regexp.QuoteMeta("JOIN routing_policies p ON p.id = b.routing_policy_id")).
		WithArgs(fingerprint, "gpt-5.6").
		WillReturnRows(sqlmock.NewRows([]string{"binding_id", "policy_id", "model_id", "priority", "protocol", "minimum_cache_hit_ppm"}).
			AddRow(81, 91, 61, "cache", "responses", 600_000))
	mock.ExpectQuery(regexp.QuoteMeta("WITH merchant_metrics AS")).
		WithArgs(int64(91), int64(61)).
		WillReturnRows(sqlmock.NewRows([]string{
			"offer_id", "alias_id", "merchant_id", "channel_id", "upstream_channel_id", "model_id", "internal_alias",
			"public_name", "protocol", "multiplier_bps", "health_state", "availability_ppm", "average_ttft_ms",
			"request_count", "cache_hit_ppm", "prompt_tokens", "cache_weight_ready", "merchant_cache_hit_ppm", "merchant_cache_weight_ready",
		}).
			AddRow(41, 81, 31, 51, 71, 61, "mkt-o41-c51", "gpt-5.6", "responses", 8000, "healthy", 990000, 800, 100, 900000, 999999, false, 900000, false).
			AddRow(42, 82, 32, 52, 72, 61, "mkt-o42-c52", "gpt-5.6", "responses", 8500, "healthy", 980000, 900, 100, 500000, 2000000, true, 500000, true))

	selection, err := NewSQLRoutingStore(database).ResolveRoute(context.Background(), fingerprint, "gpt-5.6")
	if err != nil || len(selection.Candidates) != 1 || selection.Candidates[0].MerchantID != 31 {
		t.Fatalf("缓存阈值候选=%+v err=%v", selection.Candidates, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStartRouteDecisionPersistsWeightedRoutingContext(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO routing_decisions")).
		WithArgs(
			"request-1", int64(81), int64(91), int64(61), int64(101),
			"cache", routeAlgorithmVersion, 2,
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = NewSQLRoutingStore(database).StartRouteDecision(context.Background(), RouteDecision{
		RequestID: "request-1", TokenBindingID: 81, PolicyID: 91,
		CanonicalModelID: 61, OfferChannelAliasID: 101,
		Priority: "cache", AlgorithmVersion: routeAlgorithmVersion, CandidateCount: 2,
		SelectedMetrics: map[string]any{
			"merchant_id": 31, "multiplier_bps": 8000, "availability_ppm": 990000,
			"ttft_p95_ms": 420, "cache_hit_ppm": 720000, "cache_weight_ready": true,
		},
	})
	if err != nil {
		t.Fatalf("创建路由决策失败：%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateCanonicalModelStoresAbsolutePriceAndTiersAtomically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	createdAt := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	maximum := int64(200_000)
	output := int64(15_000_000)
	cacheRead := int64(300_000)
	cacheWrite := int64(3_750_000)
	price := normalizeOfficialPrice(OfficialPrice{Tiers: []PriceTier{{MinimumInputTokens: 0, MaximumInputTokens: &maximum, InputPriceMicros: 3_000_000, OutputPriceMicros: &output, CacheReadPriceMicros: &cacheRead, CacheWritePriceMicros: &cacheWrite}, {MinimumInputTokens: 200_001, InputPriceMicros: 6_000_000, OutputPriceMicros: &output}}})
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO canonical_models (public_name, provider, protocol, adapter_hint, status)")).
		WithArgs("claude-sonnet", "Anthropic", "responses", "auto", "active").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "created_at"}).AddRow(61, "active", createdAt))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO official_price_versions")).
		WithArgs(int64(61), 1, "text", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "version", "published_at"}).AddRow(71, 1, createdAt))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO text_price_tiers")).
		WithArgs(int64(71), 1, int64(0), maximum, int64(3_000_000), output, cacheRead, cacheWrite).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO text_price_tiers")).
		WithArgs(int64(71), 2, int64(200_001), nil, int64(6_000_000), output, nil, nil).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE canonical_models SET current_official_price_version_id")).
		WithArgs(int64(71), int64(61)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	store := NewSQLRoutingStore(database)
	model, err := store.CreateCanonicalModel(context.Background(), CreateCanonicalModelCommand{
		PublicName: "claude-sonnet", Provider: "Anthropic", Protocol: "responses", AdminID: 1, Price: price,
	})
	if err != nil || model.ID != 61 || model.Price.ID != 71 || len(model.Price.Tiers) != 2 {
		t.Fatalf("创建模型=%+v，错误=%v", model, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateOfferRequiresOwnedActiveChannelAndCurrentOfficialPrice(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT mp.id, cm.current_official_price_version_id")).
		WithArgs(7, int64(61)).WillReturnRows(sqlmock.NewRows([]string{"merchant_id", "price_id", "public_name", "protocol", "previous_id", "version"}).
		AddRow(31, 71, "gpt-5.6", "responses", 40, 2))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, newapi_channel_id, channel_name FROM merchant_channels")).
		WithArgs(int64(31), int64(61)).WillReturnRows(sqlmock.NewRows([]string{"id", "newapi_channel_id", "channel_name"}).
		AddRow(51, 41, "主线路").AddRow(52, 42, "备用线路"))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO offer_versions")).
		WithArgs(int64(31), int64(61), int64(71), int64(40), 4000, 2, 7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "created_at"}).AddRow(42, "syncing", time.Now()))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO offer_channel_aliases")).
		WithArgs(int64(42), int64(51), "mkt-o42-c51").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(81))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO offer_channel_aliases")).
		WithArgs(int64(42), int64(52), "mkt-o42-c52").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(82))
	mock.ExpectCommit()

	store := NewSQLRoutingStore(database)
	offer, err := store.CreateOffer(context.Background(), CreateOfferCommand{
		UserID: 7, CanonicalModelID: 61, MultiplierBPS: 4000,
	})
	if err != nil {
		t.Fatalf("创建报价失败：%v", err)
	}
	if offer.ID != 42 || offer.PreviousOfferID != 40 || offer.Status != "syncing" || len(offer.Aliases) != 2 || offer.Aliases[1].InternalAlias != "mkt-o42-c52" {
		t.Fatalf("报价结果=%+v", offer)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOfferLifecycleKeepsInactiveVisibleAndDeletedHidden(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLRoutingStore(database)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE offer_versions offer")).
		WithArgs(7, int64(61)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.RetireOffer(context.Background(), 7, 61); err != nil {
		t.Fatalf("下架模型失败：%v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT offer.multiplier_bps, offer.status")).
		WithArgs(7, int64(61)).WillReturnRows(sqlmock.NewRows([]string{"multiplier_bps", "status"}).AddRow(8500, "inactive"))
	state, err := store.MerchantOfferState(context.Background(), 7, 61)
	if err != nil || state.MultiplierBPS != 8500 || state.Status != "inactive" {
		t.Fatalf("读取下架模型状态=%+v err=%v", state, err)
	}

	mock.ExpectExec(regexp.QuoteMeta("SET status = 'deleted'")).
		WithArgs(7, int64(61)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.DeleteMerchantOffer(context.Background(), 7, 61); err != nil {
		t.Fatalf("删除模型失败：%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestActivateOfferRetiresPreviousVersionWithoutSchedulingHealthyProbe(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT previous_offer_version_id, merchant_profile_id, canonical_model_id")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"previous_offer_version_id", "merchant_channel_id", "canonical_model_id"}).AddRow(40, 51, 61))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE offer_versions SET status = 'inactive'")).
		WithArgs(int64(40)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE offer_versions")).
		WithArgs(int64(42), "active").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	store := NewSQLRoutingStore(database)
	if err := store.ActivateOffer(context.Background(), 42); err != nil {
		t.Fatalf("激活报价失败：%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAliasPricesReturnsEverySyncedOrPendingImmutableAlias(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery(regexp.QuoteMeta("FROM offer_channel_aliases aliases")).
		WillReturnRows(sqlmock.NewRows([]string{
			"internal_alias", "protocol", "multiplier_bps", "price_id",
		}).AddRow("mkt-o42-c51", "responses", 4000, 71))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, version, price_kind, published_at")).
		WithArgs(int64(71)).WillReturnRows(sqlmock.NewRows([]string{"id", "version", "price_kind", "published_at"}).
		AddRow(71, 2, "text", time.Now()))
	mock.ExpectQuery(regexp.QuoteMeta("FROM text_price_tiers")).
		WithArgs(int64(71)).WillReturnRows(sqlmock.NewRows([]string{
		"id", "tier_order", "minimum", "maximum", "input", "output", "cache_read", "cache_write",
	}).AddRow(81, 1, 0, nil, 1250000, 3000000, 100000, 1250000))
	store := NewSQLRoutingStore(database)
	prices, err := store.AliasPrices(context.Background())
	if err != nil || len(prices) != 1 || prices[0].Protocol != "responses" || len(prices[0].Price.Tiers) != 1 || prices[0].Price.Tiers[0].InputPriceMicros != 1_250_000 {
		t.Fatalf("别名价格=%+v，错误=%v", prices, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOfferChannelRoutesKeepsAllActiveAndSyncingAliases(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery(regexp.QuoteMeta("FROM offer_channel_aliases target")).
		WithArgs(int64(42)).WillReturnRows(sqlmock.NewRows([]string{"merchant_channel_id", "newapi_channel_id", "public_name", "internal_alias"}).
		AddRow(51, 41, "gpt-5.5", "mkt-ov-41").AddRow(51, 41, "gpt-5.5", "mkt-ov-42"))
	store := NewSQLRoutingStore(database)
	routes, err := store.OfferChannelRoutes(context.Background(), 42)
	if err != nil || len(routes) != 1 || routes[0].NewAPIChannelID != 41 || len(routes[0].PublicModels) != 1 || routes[0].Aliases["mkt-ov-42"] != "gpt-5.5" {
		t.Fatalf("渠道映射=%+v，错误=%v", routes, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveProbeHardFailureDisablesPlatformChannelInSameTransaction(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT state, consecutive_failures, cooldown_level")).
		WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"state", "consecutive_failures", "cooldown_level"}).AddRow("healthy", 0, 0))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO channel_health_events")).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE channel_health")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO route_metric_buckets")).
		WithArgs(int64(51), int64(61), sqlmock.AnyArg(), int64(1), int64(0), int64(0), int64(0)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_channels SET status = 'disabled'")).
		WithArgs(int64(51)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	store := NewSQLRoutingStore(database)
	candidate := RouteCandidate{OfferID: 42, OfferChannelAliasID: 81, MerchantChannelID: 51, CanonicalModelID: 61}
	if err := store.RecordRouteObservation(context.Background(), RouteObservation{
		Candidate: candidate, Outcome: "hard_failure", StatusCode: 401, FirstByteMS: 25,
	}); err != nil {
		t.Fatalf("保存硬故障失败：%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshRouteMetricSnapshotsUsesSevenDayWindowAndEightDayRetention(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, 7, 23, 10, 42, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM route_metric_snapshots snapshot")).
		WithArgs(now).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO route_metric_snapshots")).
		WithArgs(now).WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM route_metric_buckets")).
		WithArgs(now).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	if err := NewSQLRoutingStore(database).RefreshRouteMetricSnapshots(context.Background(), now); err != nil {
		t.Fatalf("刷新路由指标失败：%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishOfficialPriceRetiresOldOffersButKeepsImmutableAliases(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT protocol FROM canonical_models WHERE id = $1 FOR UPDATE")).
		WithArgs(int64(61)).WillReturnRows(sqlmock.NewRows([]string{"protocol"}).AddRow("responses"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(max(version), 0) + 1 FROM official_price_versions")).
		WithArgs(int64(61)).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO official_price_versions")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "version", "published_at"}).AddRow(72, 2, time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO text_price_tiers")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE canonical_models SET current_official_price_version_id")).
		WithArgs(int64(72), int64(61)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE offer_versions SET status = 'inactive'")).
		WithArgs(int64(61)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	store := NewSQLRoutingStore(database)
	output := int64(2_000_000)
	price, err := store.PublishOfficialPrice(context.Background(), 61, 1, normalizeOfficialPrice(OfficialPrice{
		Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 1_000_000, OutputPriceMicros: &output}},
	}))
	if err != nil || price.ID != 72 || price.Version != 2 {
		t.Fatalf("发布价格=%+v，错误=%v", price, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteCanonicalModelPhysicallyDeletesUnreferencedModel(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS (")).WithArgs(int64(61)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE canonical_models SET current_official_price_version_id = NULL")).
		WithArgs(int64(61)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM official_price_versions WHERE canonical_model_id = $1")).
		WithArgs(int64(61)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM canonical_models WHERE id = $1")).
		WithArgs(int64(61)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	result, err := NewSQLRoutingStore(database).DeleteCanonicalModel(context.Background(), 61)
	if err != nil || result != "deleted" {
		t.Fatalf("删除结果=%q，错误=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteCanonicalModelArchivesReferencedModelAndOffers(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS (")).WithArgs(int64(61)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE canonical_models SET status = 'archived'")).
		WithArgs(int64(61)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE offer_versions SET status = 'inactive'")).
		WithArgs(int64(61)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	result, err := NewSQLRoutingStore(database).DeleteCanonicalModel(context.Background(), 61)
	if err != nil || result != "archived" {
		t.Fatalf("归档结果=%q，错误=%v", result, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
