package httpapi

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

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
