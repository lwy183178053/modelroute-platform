package httpapi

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMerchantStoreMapsReservedNameConflict(t *testing.T) {
	err := merchantStoreError(&pgconn.PgError{Code: "23505", ConstraintName: "merchant_name_reservations_pkey"})
	if !errors.Is(err, ErrMerchantNameConflict) {
		t.Fatalf("商户名称冲突映射=%v", err)
	}
}

func TestSQLMerchantStoreCreatesApplicationWithoutChannelReview(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLMerchantStore(database)
	createdAt := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	command := CreateMerchantApplication{
		UserID: 7, DisplayName: "北辰云智", DisplayNameKey: "北辰云智", WebsiteURL: "https://merchant.example", BusinessDescription: "企业模型 API 服务",
	}
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_applications")).
		WithArgs(7, "北辰云智", "北辰云智", "https://merchant.example", "企业模型 API 服务").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(11, createdAt))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO merchant_name_reservations")).
		WithArgs("北辰云智", "北辰云智", 7, int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	application, err := store.CreateApplication(context.Background(), command)
	if err != nil {
		t.Fatalf("创建申请失败：%v", err)
	}
	if application.ID != 11 || application.Status != "pending" || application.DisplayName != "北辰云智" {
		t.Fatalf("申请结果=%+v", application)
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
	store := NewSQLMerchantStore(database)
	approvedAt := time.Date(2026, 7, 22, 15, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, newapi_user_id, display_name, website_url, business_description")).
		WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "display_name", "website_url", "business_description"}).
			AddRow(11, 7, "北辰云智", "https://merchant.example", "企业模型 API 服务"))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_profiles")).
		WithArgs(7, int64(11), "北辰云智", "https://merchant.example", "企业模型 API 服务", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "display_name", "website_url", "business_description", "status", "status_reason", "approved_at", "is_platform_owned"}).
			AddRow(31, 7, "北辰云智", "https://merchant.example", "企业模型 API 服务", "active", "", approvedAt, false))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_applications")).
		WithArgs(1, int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_name_reservations")).
		WithArgs(int64(31), int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	profile, err := store.ApproveApplication(context.Background(), 11, 1)
	if err != nil {
		t.Fatalf("通过商户申请失败：%v", err)
	}
	if profile.ID != 31 || profile.UserID != 7 || profile.BusinessDescription != "企业模型 API 服务" || profile.IsPlatformOwned {
		t.Fatalf("商户资料=%+v", profile)
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
	store := NewSQLMerchantStore(database)
	createdAt := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_profiles")).
		WithArgs(1, "ModelRoute 官方", "https://aggregate.lwylink.xyz").
		WillReturnRows(sqlmock.NewRows([]string{"id", "newapi_user_id", "display_name", "status", "is_platform_owned", "created_at"}).
			AddRow(31, 1, "ModelRoute 官方", "active", true, createdAt))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO merchant_name_reservations")).
		WithArgs("modelroute 官方", "ModelRoute 官方", 1, int64(31)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	identity, err := store.EnsurePlatformMerchant(context.Background(), 1, "https://aggregate.lwylink.xyz")
	if err != nil {
		t.Fatalf("创建平台商户失败：%v", err)
	}
	if identity.MerchantID != 31 || identity.UserID != 1 || identity.DisplayName != "ModelRoute 官方" || !identity.IsMerchant || !identity.IsPlatformOwned {
		t.Fatalf("平台商户身份=%+v", identity)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLMerchantStoreDeletesRejectedReviewAndWithdrawsPendingReview(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLMerchantStore(database)

	for _, testCase := range []struct {
		name     string
		status   string
		expected string
		testID   any
	}{
		{name: "撤销等待审核", status: "pending", expected: "withdrawn", testID: int64(81)},
		{name: "删除驳回记录", status: "rejected", expected: "deleted", testID: nil},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mock.ExpectBegin()
			mock.ExpectQuery(regexp.QuoteMeta("SELECT r.status, r.application_id, r.channel_test_id")).
				WithArgs(int64(21), int64(31)).
				WillReturnRows(sqlmock.NewRows([]string{"status", "application_id", "channel_test_id"}).AddRow(testCase.status, nil, testCase.testID))
			mock.ExpectExec(regexp.QuoteMeta("DELETE FROM merchant_channel_reviews WHERE id = $1")).
				WithArgs(int64(21)).WillReturnResult(sqlmock.NewResult(0, 1))
			if testCase.testID != nil {
				mock.ExpectExec(regexp.QuoteMeta("DELETE FROM merchant_channel_tests")).
					WithArgs(testCase.testID).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			mock.ExpectCommit()

			action, deleteErr := store.DeleteChannelReview(context.Background(), 21, 31)
			if deleteErr != nil || action != testCase.expected {
				t.Fatalf("删除审核记录 action=%q err=%v", action, deleteErr)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSQLMerchantStoreRejectsReviewDeletionWhileAdministratorProcessesIt(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT r.status, r.application_id, r.channel_test_id")).
		WithArgs(int64(21), int64(31)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "application_id", "channel_test_id"}).AddRow("approving", nil, int64(81)))
	mock.ExpectRollback()

	_, deleteErr := NewSQLMerchantStore(database).DeleteChannelReview(context.Background(), 21, 31)
	if !errors.Is(deleteErr, ErrMerchantInvalidState) {
		t.Fatalf("审批处理中删除错误=%v", deleteErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLMerchantStoreListsChannelHealthForAdministrator(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLMerchantStore(database)
	createdAt := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	cooldownUntil := createdAt.Add(5 * time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM merchant_channels")).
		WithArgs("").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT c.id, c.merchant_profile_id")).
		WithArgs("", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "merchant_profile_id", "merchant_user_id", "merchant_name", "review_id", "newapi_channel_id", "channel_name", "canonical_model_id",
			"public_name", "provider", "adapter_type", "base_url", "credential_fingerprint", "status", "created_at",
			"health_state", "availability_ppm", "ttft_p95_ms", "consecutive_failures", "cooldown_until",
		}).AddRow(51, 31, 7, "北辰云智", 21, 41, "主线路", 61, "gpt-5.6", "OpenAI", "openai_compatible",
			"https://api.merchant.example", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "active", createdAt,
			"cooling", 975000, 680, 3, cooldownUntil))

	channels, total, err := store.ListChannels(context.Background(), 1, 20, "")
	if err != nil {
		t.Fatalf("查询管理员渠道失败：%v", err)
	}
	if total != 1 || len(channels) != 1 || channels[0].UserID != 7 || channels[0].MerchantName != "北辰云智" || channels[0].HealthState != "cooling" || channels[0].AvailabilityPPM != 975000 || channels[0].TTFTP95MS != 680 || channels[0].ConsecutiveFailures != 3 || channels[0].CooldownUntil == nil {
		t.Fatalf("渠道健康字段=%+v total=%d", channels, total)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLMerchantStoreFindsPlatformChannelByID(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLMerchantStore(database)
	createdAt := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE c.id = $1")).WithArgs(int64(51)).WillReturnRows(sqlmock.NewRows([]string{
		"id", "merchant_profile_id", "merchant_user_id", "merchant_name", "review_id", "newapi_channel_id", "channel_name", "canonical_model_id",
		"public_name", "provider", "adapter_type", "base_url", "credential_fingerprint", "status", "created_at",
		"health_state", "availability_ppm", "ttft_p95_ms", "consecutive_failures", "cooldown_until",
	}).AddRow(51, 31, 7, "北辰云智", 21, 41, "主线路", 61, "gpt-5.6", "OpenAI", "openai_compatible",
		"https://api.merchant.example", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "active", createdAt,
		"healthy", 999000, 520, 0, nil))
	channel, err := store.Channel(context.Background(), 51)
	if err != nil || channel.ID != 51 || channel.UserID != 7 || channel.NewAPIChannelID != 41 {
		t.Fatalf("渠道=%+v err=%v", channel, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLMerchantStoreClaimsReviewWithAtomicStateTransition(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLMerchantStore(database)
	createdAt := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows(reviewColumns()).AddRow(
		21, 7, 11, 0, "主线路", 61, "gpt-5.6", "OpenAI", "text", 81, "openai_compatible",
		"https://api.merchant.example", []byte("ciphertext"), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "verifying", "", "", nil, nil, createdAt,
	)
	mock.ExpectQuery(regexp.QuoteMeta("WITH claimed AS")).WithArgs(int64(21)).WillReturnRows(rows)
	review, err := store.ClaimReview(context.Background(), 21, "verifying")
	if err != nil {
		t.Fatalf("认领审核失败：%v", err)
	}
	if review.Status != "verifying" || review.UserID != 7 || string(review.EncryptedCredential) != "ciphertext" {
		t.Fatalf("审核记录=%+v", review)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLMerchantStoreFinalApprovalClearsCredentialAndCreatesOwnership(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLMerchantStore(database)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT r.id, r.merchant_newapi_user_id")).WithArgs(int64(21)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "application_id", "canonical_model_id", "channel_name", "provider", "base_url", "adapter_type", "fingerprint", "display_name", "website_url"}).
			AddRow(21, 7, 11, 61, "主线路", "OpenAI", "https://api.merchant.example", "openai_compatible", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "北辰云智", "https://merchant.example"))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_profiles")).WithArgs(7, int64(11), "北辰云智", "https://merchant.example", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(31))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_channels")).WithArgs(int64(31), int64(21), int64(61), 41, "OpenAI", "主线路", "https://api.merchant.example", "openai_compatible", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(51))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO channel_health")).WithArgs(int64(51)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_channel_reviews")).WithArgs(1, int64(21), int64(31)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_applications")).WithArgs(1, int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_name_reservations")).WithArgs(int64(31), int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	approval, err := store.FinishApproval(context.Background(), 21, 1, 41)
	if err != nil {
		t.Fatalf("完成审批失败：%v", err)
	}
	if approval.MerchantID != 31 || approval.ChannelID != 51 || approval.NewAPIChannelID != 41 || approval.UserID != 7 {
		t.Fatalf("审批结果=%+v", approval)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLMerchantStoreRejectsApplicationAndReleasesReservedName(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLMerchantStore(database)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_applications")).
		WithArgs("资料不完整", 1, int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM merchant_name_reservations")).
		WithArgs(int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := store.RejectApplication(context.Background(), 11, 1, "资料不完整"); err != nil {
		t.Fatalf("驳回商户申请失败：%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func reviewColumns() []string {
	return []string{"id", "merchant_newapi_user_id", "application_id", "replaces_channel_id", "channel_name", "canonical_model_id", "public_name", "provider", "price_kind", "channel_test_id", "adapter_type", "base_url", "encrypted_credential", "credential_fingerprint", "status", "verification_error", "review_reason", "verified_at", "reviewed_at", "created_at"}
}
