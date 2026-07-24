package httpapi

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/migrations"
)

func TestSQLPaymentStoreAgainstPostgres(t *testing.T) {
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
TRUNCATE payment_callback_events, newapi_credit_operations, payment_orders, payment_setting_versions RESTART IDENTITY CASCADE
`); err != nil {
		t.Fatalf("清理支付测试表：%v", err)
	}

	store := NewSQLPaymentStore(database)
	setting, err := store.CreatePaymentSetting(ctx, CreatePaymentSettingCommand{
		AdminUserID: 1, GatewayURL: "https://zpayz.cn", MerchantPID: "postgres-pid",
		EncryptedKey: []byte("postgres-encrypted-key"), KeyFingerprint: strings64("a"),
		PaymentMethods: []string{"alipay"}, MinimumCNYCents: 100, MaximumCNYCents: 100000, Enabled: true,
	})
	if err != nil {
		t.Fatalf("创建支付配置：%v", err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	order, err := store.CreatePaymentOrder(ctx, CreatePaymentOrderCommand{
		OrderNo: "MR-PG-CONCURRENT", UserID: 7, SettingVersionID: setting.ID, PaymentType: "alipay",
		AmountCNYCents: 10_000, NewAPIQuota: 50_000_000, ExpiresAt: now.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("创建支付订单：%v", err)
	}
	callbackContext, err := store.PaymentCallbackContext(ctx, order.OrderNo)
	if err != nil || callbackContext.Setting.ID != setting.ID || callbackContext.Order.ID != order.ID {
		t.Fatalf("读取回调上下文=%+v，错误=%v", callbackContext, err)
	}

	var accepted atomic.Int32
	errorsChannel := make(chan error, 8)
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, first, callbackErr := store.AcceptPaymentCallback(ctx, AcceptPaymentCallbackCommand{
				OrderNo: order.OrderNo, TradeNo: "EP-PG-CONCURRENT", CallbackKey: strings64("b"),
				Payload: map[string]string{"money": "720.00"}, PaidAt: now,
			})
			if callbackErr != nil {
				errorsChannel <- callbackErr
				return
			}
			if first {
				accepted.Add(1)
			}
		}()
	}
	wait.Wait()
	close(errorsChannel)
	for callbackErr := range errorsChannel {
		t.Fatalf("并发回调失败：%v", callbackErr)
	}
	if accepted.Load() != 1 {
		t.Fatalf("首次认账次数=%d", accepted.Load())
	}
	var callbackCount, creditCount int
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM payment_callback_events WHERE payment_order_id = $1", order.ID).Scan(&callbackCount); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM newapi_credit_operations WHERE payment_order_id = $1", order.ID).Scan(&creditCount); err != nil {
		t.Fatal(err)
	}
	if callbackCount != 1 || creditCount != 1 {
		t.Fatalf("回调事件=%d，入账操作=%d", callbackCount, creditCount)
	}

	claimed := make(chan []PaymentCreditOperation, 2)
	claimErrors := make(chan error, 2)
	claimAt := now.Add(time.Minute)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			operations, claimErr := store.ClaimPaymentCredits(ctx, claimAt, 10)
			if claimErr != nil {
				claimErrors <- claimErr
				return
			}
			claimed <- operations
		}()
	}
	wait.Wait()
	close(claimed)
	close(claimErrors)
	for claimErr := range claimErrors {
		t.Fatalf("并发领取失败：%v", claimErr)
	}
	var operations []PaymentCreditOperation
	for group := range claimed {
		operations = append(operations, group...)
	}
	if len(operations) != 1 || operations[0].OrderID != order.ID || operations[0].Attempts != 1 {
		var creditStatus, paymentStatus string
		var attempts int
		var nextAttempt time.Time
		if inspectErr := database.QueryRowContext(ctx, `
SELECT c.status, c.attempt_count, c.next_attempt_at, o.payment_status
FROM newapi_credit_operations c JOIN payment_orders o ON o.id = c.payment_order_id
WHERE c.payment_order_id = $1
`, order.ID).Scan(&creditStatus, &attempts, &nextAttempt, &paymentStatus); inspectErr != nil {
			t.Fatalf("并发领取结果=%+v，读取任务状态失败：%v", operations, inspectErr)
		}
		t.Fatalf("并发领取结果=%+v，任务状态=%s，尝试=%d，下次执行=%s，领取时间=%s，支付状态=%s", operations, creditStatus, attempts, nextAttempt.Format(time.RFC3339Nano), claimAt.Format(time.RFC3339Nano), paymentStatus)
	}
	if err := store.CompletePaymentCredit(ctx, operations[0].ID, now.Add(2*time.Second)); err != nil {
		t.Fatalf("完成入账：%v", err)
	}
	completed, err := store.PaymentOrderForUser(ctx, 7, order.OrderNo)
	if err != nil || completed.CreditStatus != "succeeded" || completed.CreditedAt == nil {
		t.Fatalf("完成订单=%+v，错误=%v", completed, err)
	}

	unknownOrder, err := store.CreatePaymentOrder(ctx, CreatePaymentOrderCommand{
		OrderNo: "MR-PG-UNKNOWN", UserID: 8, SettingVersionID: setting.ID, PaymentType: "alipay",
		AmountCNYCents: 5000, NewAPIQuota: 25_000_000, ExpiresAt: now.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.AcceptPaymentCallback(ctx, AcceptPaymentCallbackCommand{
		OrderNo: unknownOrder.OrderNo, TradeNo: "EP-PG-UNKNOWN", CallbackKey: strings64("c"), PaidAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	unknownOperations, err := store.ClaimPaymentCredits(ctx, claimAt, 10)
	if err != nil || len(unknownOperations) != 1 {
		t.Fatalf("领取结果未知任务=%+v，错误=%v", unknownOperations, err)
	}
	if count, err := store.MarkStalePaymentCreditsUnknown(ctx, now.Add(10*time.Minute)); err != nil || count != 1 {
		t.Fatalf("标记结果未知数量=%d，错误=%v", count, err)
	}
	unknown, err := store.PaymentOrderForUser(ctx, 8, unknownOrder.OrderNo)
	if err != nil || unknown.CreditStatus != "unknown" {
		t.Fatalf("结果未知订单=%+v，错误=%v", unknown, err)
	}
	if err := store.RetryPaymentCredit(ctx, unknownOrder.ID, true); err != nil {
		t.Fatalf("人工核对后重试：%v", err)
	}
	retried, err := store.PaymentOrderForUser(ctx, 8, unknownOrder.OrderNo)
	if err != nil || retried.CreditStatus != "pending" {
		t.Fatalf("重新排队订单=%+v，错误=%v", retried, err)
	}
}
