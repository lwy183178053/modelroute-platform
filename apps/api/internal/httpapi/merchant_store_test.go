package httpapi

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSQLMerchantStoreCreatesApplicationAndInitialReviewAtomically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := NewSQLMerchantStore(database)
	createdAt := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	command := CreateMerchantApplication{
		UserID: 7, DisplayName: "北辰云智", WebsiteURL: "https://merchant.example", BusinessDescription: "企业模型 API 服务",
		Channel: ChannelSubmission{ChannelName: "主线路", ChannelType: 1, BaseURL: "https://api.merchant.example", Models: []string{"gpt-4o-mini"}, EncryptedCredential: []byte("ciphertext"), CredentialFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_applications")).
		WithArgs(7, "北辰云智", "https://merchant.example", "企业模型 API 服务").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(11, createdAt))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_channel_reviews")).
		WithArgs(7, int64(11), "主线路", 1, "https://api.merchant.example", []byte(`["gpt-4o-mini"]`), []byte("ciphertext"), command.Channel.CredentialFingerprint).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(21, createdAt))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_applications SET initial_channel_review_id = $1 WHERE id = $2")).
		WithArgs(int64(21), int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	application, err := store.CreateApplication(context.Background(), command)
	if err != nil {
		t.Fatalf("创建申请失败：%v", err)
	}
	if application.ID != 11 || application.InitialReview.ID != 21 || !application.InitialReview.CredentialConfigured || application.Status != "pending" {
		t.Fatalf("申请结果=%+v", application)
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
		21, 7, 11, 0, "主线路", 1, "https://api.merchant.example", []byte(`["gpt-4o-mini"]`),
		[]byte("ciphertext"), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "verifying", []byte(`[]`), "", "", nil, nil, createdAt,
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
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "application_id", "channel_name", "channel_type", "base_url", "requested_models", "fingerprint", "display_name", "website_url"}).
			AddRow(21, 7, 11, "主线路", 1, "https://api.merchant.example", []byte(`["gpt-4o-mini"]`), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "北辰云智", "https://merchant.example"))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_profiles")).WithArgs(7, int64(11), "北辰云智", "https://merchant.example", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(31))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO merchant_channels")).WithArgs(int64(31), int64(21), 41, "主线路", 1, "https://api.merchant.example", []byte(`["gpt-4o-mini"]`), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(51))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_channel_reviews")).WithArgs(1, int64(21)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE merchant_applications")).WithArgs(1, int64(11)).WillReturnResult(sqlmock.NewResult(0, 1))
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

func reviewColumns() []string {
	return []string{"id", "merchant_newapi_user_id", "application_id", "replaces_channel_id", "channel_name", "channel_type", "base_url", "requested_models", "encrypted_credential", "credential_fingerprint", "status", "verified_models", "verification_error", "review_reason", "verified_at", "reviewed_at", "created_at"}
}
