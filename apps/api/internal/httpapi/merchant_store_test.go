package httpapi

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMerchantStoreMapsReservedNameConflict(t *testing.T) {
	err := merchantStoreError(&pgconn.PgError{Code: "23505", ConstraintName: "merchant_name_reservations_pkey"})
	if !errors.Is(err, ErrMerchantNameConflict) {
		t.Fatalf("重名错误=%v", err)
	}
}

func TestSQLMerchantStoreCreatesApplicationWithoutChannel(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	createdAt := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_applications")).
		WithArgs(7, "北辰云智", "北辰云智", "https://merchant.example", "企业模型 API 服务").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(11, createdAt))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO merchant_name_reservations")).
		WithArgs("北辰云智", "北辰云智", 7, int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	application, err := NewSQLMerchantStore(database).CreateApplication(context.Background(), CreateMerchantApplication{
		UserID: 7, DisplayName: "北辰云智", DisplayNameKey: "北辰云智",
		WebsiteURL: "https://merchant.example", BusinessDescription: "企业模型 API 服务",
	})
	if err != nil || application.ID != 11 || application.Status != "pending" {
		t.Fatalf("商户申请=%+v err=%v", application, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLMerchantStoreApprovesApplicationWithoutCreatingChannel(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	approvedAt := time.Date(2026, 7, 22, 15, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, newapi_user_id, display_name, website_url, business_description")).
		WithArgs(int64(11)).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "display_name", "website_url", "business_description"}).
		AddRow(11, 7, "北辰云智", "https://merchant.example", "企业模型 API 服务"))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_profiles")).
		WithArgs(7, int64(11), "北辰云智", "https://merchant.example", "企业模型 API 服务", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "display_name", "website_url", "business_description", "status", "status_reason", "approved_at", "is_platform_owned"}).
			AddRow(31, 7, "北辰云智", "https://merchant.example", "企业模型 API 服务", "active", "", approvedAt, false))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_applications")).WithArgs(1, int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_name_reservations")).WithArgs(int64(31), int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	profile, err := NewSQLMerchantStore(database).ApproveApplication(context.Background(), 11, 1)
	if err != nil || profile.ID != 31 || profile.UserID != 7 {
		t.Fatalf("商户资料=%+v err=%v", profile, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLMerchantStoreEnsuresPlatformMerchantForAdministrator(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	createdAt := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_profiles")).
		WithArgs(1, "ModelRoute 官方", "https://aggregate.lwylink.xyz").
		WillReturnRows(sqlmock.NewRows([]string{"id", "newapi_user_id", "display_name", "status", "is_platform_owned", "created_at"}).
			AddRow(31, 1, "ModelRoute 官方", "active", true, createdAt))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO merchant_name_reservations")).
		WithArgs("modelroute 官方", "ModelRoute 官方", 1, int64(31)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	identity, err := NewSQLMerchantStore(database).EnsurePlatformMerchant(context.Background(), 1, "https://aggregate.lwylink.xyz")
	if err != nil || !identity.IsPlatformOwned || !identity.IsMerchant {
		t.Fatalf("平台商户=%+v err=%v", identity, err)
	}
}

func TestSQLMerchantStoreListsChannelsWithoutChannelLevelStatus(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	createdAt := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*)")).WithArgs("", 0).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT c.id, c.merchant_profile_id")).
		WithArgs("", 0, 20, 0).WillReturnRows(sqlmock.NewRows(channelColumns()).
		AddRow(51, 31, 7, "北辰云智", 41, "主线路", 1, "OpenAI", "openai_compatible",
			"https://api.merchant.example", stringsOf(64, "a"), createdAt, createdAt))
	channels, total, err := NewSQLMerchantStore(database).ListChannels(context.Background(), 1, 20)
	if err != nil || total != 1 || len(channels) != 1 || channels[0].ID != 51 || channels[0].CredentialFingerprint != "aaaaaa...aaaaaa" {
		t.Fatalf("渠道=%+v total=%d err=%v", channels, total, err)
	}
}

func TestSQLMerchantStoreArchivesChannelWithDeletedAt(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_channels SET deleted_at = now()")).
		WithArgs(int64(51), int64(31)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := NewSQLMerchantStore(database).DeleteChannel(context.Background(), 51, 31); err != nil {
		t.Fatalf("归档渠道失败：%v", err)
	}
}

func TestSQLMerchantStoreRejectsApplicationAndReleasesReservedName(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_applications")).
		WithArgs("资料不完整", 1, int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM merchant_name_reservations")).
		WithArgs(int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := NewSQLMerchantStore(database).RejectApplication(context.Background(), 11, 1, "资料不完整"); err != nil {
		t.Fatalf("驳回商户申请失败：%v", err)
	}
}

func channelColumns() []string {
	return []string{"id", "merchant_profile_id", "merchant_user_id", "merchant_name", "newapi_channel_id", "channel_name", "channel_type", "provider", "adapter_type", "base_url", "credential_fingerprint", "last_synced_at", "created_at"}
}

func stringsOf(count int, value string) string {
	return strings.Repeat(value, count)
}
