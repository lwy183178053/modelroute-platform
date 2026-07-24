package httpapi

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreatePaymentSettingPublishesOneCurrentVersion(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	createdAt := time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(782403222)")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE payment_setting_versions SET is_current = false WHERE is_current")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO payment_setting_versions")).
		WithArgs("https://zpayz.cn", "1001", []byte("encrypted"), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			jsonBytes(`[`+`"alipay"`+`]`), int64(100), int64(2_000_000), int64(750), true, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "version", "gateway_url", "merchant_pid", "encrypted_key", "key_fingerprint", "payment_methods",
			"minimum_cny_cents", "maximum_cny_cents", "withdrawal_tax_bps", "enabled", "created_at",
		}).AddRow(11, 3, "https://zpayz.cn", "1001", []byte("encrypted"),
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", `["alipay"]`, 100, 2_000_000, 750, true, createdAt))
	mock.ExpectCommit()

	store := NewSQLPaymentStore(database)
	setting, err := store.CreatePaymentSetting(context.Background(), CreatePaymentSettingCommand{
		AdminUserID: 1, GatewayURL: "https://zpayz.cn", MerchantPID: "1001", EncryptedKey: []byte("encrypted"),
		KeyFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PaymentMethods: []string{"alipay"}, MinimumCNYCents: 100, MaximumCNYCents: 2_000_000,
		WithdrawalTaxBPS: 750, Enabled: true,
	})
	if err != nil || setting.ID != 11 || setting.Version != 3 || len(setting.PaymentMethods) != 1 || setting.WithdrawalTaxBPS != 750 {
		t.Fatalf("支付配置=%+v，错误=%v", setting, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreatePaymentOrderPersistsLockedAmounts(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	expiresAt := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)
	createdAt := time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO payment_orders")).
		WithArgs("MR202607220001", 7, int64(11), "alipay", int64(10_000), int64(50_000_000), expiresAt).
		WillReturnRows(paymentOrderRows().AddRow(21, "MR202607220001", 7, 11, "alipay", 10_000, 50_000_000,
			"pending", "pending", nil, nil, nil, expiresAt, createdAt, createdAt))
	store := NewSQLPaymentStore(database)
	order, err := store.CreatePaymentOrder(context.Background(), CreatePaymentOrderCommand{
		OrderNo: "MR202607220001", UserID: 7, SettingVersionID: 11, PaymentType: "alipay",
		AmountCNYCents: 10_000, NewAPIQuota: 50_000_000, ExpiresAt: expiresAt,
	})
	if err != nil || order.ID != 21 || order.NewAPIQuota != 50_000_000 {
		t.Fatalf("支付订单=%+v，错误=%v", order, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptPaymentCallbackCreatesOneCreditOperation(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	paidAt := time.Date(2026, 7, 22, 9, 5, 0, 0, time.UTC)
	expiresAt := paidAt.Add(25 * time.Minute)
	createdAt := paidAt.Add(-5 * time.Minute)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("FROM payment_orders WHERE order_no = $1 FOR UPDATE")).WithArgs("MR202607220001").
		WillReturnRows(paymentOrderRows().AddRow(21, "MR202607220001", 7, 11, "alipay", 72_000, 50_000_000,
			"pending", "pending", nil, nil, nil, expiresAt, createdAt, createdAt))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO payment_callback_events")).
		WithArgs("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", int64(21), "EP202607220009", jsonBytes(`{"money":"720.00"}`), paidAt).
		WillReturnResult(sqlmock.NewResult(31, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE payment_orders")).
		WithArgs("EP202607220009", paidAt, int64(21)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO newapi_credit_operations")).
		WithArgs(int64(21), "payment:MR202607220001:credit").WillReturnResult(sqlmock.NewResult(41, 1))
	mock.ExpectCommit()

	store := NewSQLPaymentStore(database)
	order, accepted, err := store.AcceptPaymentCallback(context.Background(), AcceptPaymentCallbackCommand{
		OrderNo: "MR202607220001", TradeNo: "EP202607220009",
		CallbackKey: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Payload:     map[string]string{"money": "720.00"}, PaidAt: paidAt,
	})
	if err != nil || !accepted || order.PaymentStatus != "paid" || order.ThirdPartyTradeNo != "EP202607220009" {
		t.Fatalf("认账结果=%+v，首次=%v，错误=%v", order, accepted, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptPaymentCallbackTreatsSameTradeAsDuplicate(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, 7, 22, 9, 5, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("FROM payment_orders WHERE order_no = $1 FOR UPDATE")).WithArgs("MR202607220001").
		WillReturnRows(paymentOrderRows().AddRow(21, "MR202607220001", 7, 11, "alipay", 72_000, 50_000_000,
			"paid", "succeeded", "EP202607220009", now, now, now.Add(time.Hour), now.Add(-time.Minute), now))
	mock.ExpectCommit()
	store := NewSQLPaymentStore(database)
	order, accepted, err := store.AcceptPaymentCallback(context.Background(), AcceptPaymentCallbackCommand{
		OrderNo: "MR202607220001", TradeNo: "EP202607220009", CallbackKey: strings64("b"), PaidAt: now,
	})
	if err != nil || accepted || order.PaymentStatus != "paid" {
		t.Fatalf("重复认账结果=%+v，首次=%v，错误=%v", order, accepted, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type jsonBytes string

func (j jsonBytes) Match(value driver.Value) bool {
	data, ok := value.([]byte)
	return ok && string(data) == string(j)
}

func paymentOrderRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "order_no", "newapi_user_id", "setting_version_id", "payment_type", "amount_cny_cents",
		"newapi_quota", "payment_status", "credit_status", "third_party_trade_no",
		"paid_at", "credited_at", "expires_at", "created_at", "updated_at",
	})
}

func strings64(value string) string {
	result := ""
	for len(result) < 64 {
		result += value
	}
	return result[:64]
}
