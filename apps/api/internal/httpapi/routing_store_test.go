package httpapi

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateOfferRequiresOwnedActiveChannelAndCurrentOfficialPrice(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("mc.models ? cm.public_name")).
		WithArgs(7, int64(51), int64(61), 4000, "mkt-ov-test").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "merchant_channel_id", "canonical_model_id", "official_price_version_id", "previous_offer_version_id",
			"multiplier_bps", "internal_alias", "status", "public_name", "protocol", "newapi_channel_id",
		}).AddRow(42, 51, 61, 71, 40, 4000, "mkt-ov-test", "syncing", "gpt-5.5", "responses", 41))
	mock.ExpectCommit()

	store := NewSQLRoutingStore(database)
	offer, err := store.CreateOffer(context.Background(), CreateOfferCommand{
		UserID: 7, MerchantChannelID: 51, CanonicalModelID: 61, MultiplierBPS: 4000, InternalAlias: "mkt-ov-test",
	})
	if err != nil {
		t.Fatalf("创建报价失败：%v", err)
	}
	if offer.ID != 42 || offer.PreviousOfferID != 40 || offer.Status != "syncing" || offer.InternalAlias != "mkt-ov-test" {
		t.Fatalf("报价结果=%+v", offer)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestActivateOfferRetiresPreviousVersionAndStartsPendingHealth(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT previous_offer_version_id, merchant_channel_id, canonical_model_id")).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"previous_offer_version_id", "merchant_channel_id", "canonical_model_id"}).AddRow(40, 51, 61))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE offer_versions SET status = 'inactive'")).
		WithArgs(int64(40)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE offer_versions SET status = 'active'")).
		WithArgs(int64(42)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO channel_model_health")).
		WithArgs(int64(51), int64(61)).WillReturnResult(sqlmock.NewResult(1, 1))
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
	mock.ExpectQuery(regexp.QuoteMeta("WHERE o.status <> 'sync_failed'")).
		WillReturnRows(sqlmock.NewRows([]string{
			"internal_alias", "billing_mode", "multiplier_bps", "model_ratio_micros", "model_price_microusd",
			"completion_ratio_micros", "cache_ratio_micros", "create_cache_ratio_micros", "image_ratio_micros",
			"audio_ratio_micros", "audio_completion_ratio_micros",
		}).AddRow("mkt-ov-42", "token", 4000, 1250000, nil, 3000000, 100000, 1250000, 1000000, 1000000, 1000000))
	store := NewSQLRoutingStore(database)
	prices, err := store.AliasPrices(context.Background())
	if err != nil || len(prices) != 1 || prices[0].ModelRatioMicros != 1_250_000 {
		t.Fatalf("别名价格=%+v，错误=%v", prices, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestChannelPriceRouteKeepsOriginalModelsAndAllNonFailedAliases(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT newapi_channel_id, models FROM merchant_channels")).
		WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"newapi_channel_id", "models"}).AddRow(41, `["gpt-5.5"]`))
	mock.ExpectQuery(regexp.QuoteMeta("WHERE o.merchant_channel_id = $1 AND o.status <> 'sync_failed'")).
		WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{"internal_alias", "public_name"}).AddRow("mkt-ov-42", "gpt-5.5"))
	store := NewSQLRoutingStore(database)
	route, err := store.ChannelPriceRoute(context.Background(), 51)
	if err != nil || route.NewAPIChannelID != 41 || len(route.PublicModels) != 1 || route.Aliases["mkt-ov-42"] != "gpt-5.5" {
		t.Fatalf("渠道映射=%+v，错误=%v", route, err)
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
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO channel_health_events")).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE channel_model_health")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_channels SET status = 'disabled'")).
		WithArgs(int64(51)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	store := NewSQLRoutingStore(database)
	probe := HealthProbe{OfferID: 42, MerchantChannelID: 51, CanonicalModelID: 61, State: "healthy"}
	state := probe
	state.State = "disabled"
	if err := store.SaveProbeResult(context.Background(), HealthProbeResult{
		Probe: probe, State: state, Class: healthFailureHard, StatusCode: 401, FirstByteMS: 25,
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
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(max(version), 0) + 1 FROM official_price_versions")).
		WithArgs(int64(61)).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO official_price_versions")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "published_at"}).AddRow(72, time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE canonical_models SET current_official_price_version_id")).
		WithArgs(int64(72), int64(61)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE offer_versions SET status = 'inactive'")).
		WithArgs(int64(61)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()
	store := NewSQLRoutingStore(database)
	price, err := store.PublishOfficialPrice(context.Background(), 61, 1, OfficialPrice{
		BillingMode: "token", ModelRatioMicros: 1_000_000, CompletionRatioMicros: 2_000_000,
		CacheRatioMicros: 100_000, CreateCacheRatioMicros: 1_000_000,
		ImageRatioMicros: 1_000_000, AudioRatioMicros: 1_000_000, AudioCompletionRatioMicros: 1_000_000,
	})
	if err != nil || price.ID != 72 || price.Version != 2 {
		t.Fatalf("发布价格=%+v，错误=%v", price, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
