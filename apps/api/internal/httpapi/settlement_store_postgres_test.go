package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/migrations"
)

func TestSQLSettlementStoreFreezesWithdrawalsAtomicallyAndReleasesOrPays(t *testing.T) {
	databaseURL := os.Getenv("MODELROUTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("未设置 PostgreSQL 集成测试地址")
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	ctx := context.Background()
	if err := migrations.Up(ctx, database); err != nil {
		t.Fatal(err)
	}
	truncateSettlementTables(t, database)
	merchantID := seedSettlementOffer(t, database, 41, "mkt-ov-withdrawal")
	store := NewSQLSettlementStore(database)
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := store.ApplySettlementLog(ctx, SettlementLog{
		ID: 2001, UserID: 17, CreatedAt: now.Unix(), Type: 2, ModelName: "mkt-ov-withdrawal",
		Quota: 100 * quotaPerCNY, ChannelID: 41, Other: json.RawMessage(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	var successes atomic.Int32
	var accepted MerchantWithdrawal
	var acceptedMu sync.Mutex
	errorsChannel := make(chan error, 2)
	var wait sync.WaitGroup
	for index := range 2 {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			withdrawal, createErr := store.CreateMerchantWithdrawal(ctx, CreateMerchantWithdrawalCommand{
				WithdrawalNo: "MW-CONCURRENT-" + string(rune('A'+index)), UserID: 17, AmountQuota: 50 * quotaPerCNY, QuotaPerUnit: quotaPerCNY,
				RecipientType: "alipay", EncryptedRecipientAccount: []byte("encrypted-account"),
				RecipientAccountMasked: "138****2621", CreatedAt: now,
			})
			if createErr == nil {
				successes.Add(1)
				acceptedMu.Lock()
				accepted = withdrawal
				acceptedMu.Unlock()
				return
			}
			if !errors.Is(createErr, ErrInsufficientMerchantBalance) {
				errorsChannel <- createErr
			}
		}(index)
	}
	wait.Wait()
	close(errorsChannel)
	for createErr := range errorsChannel {
		t.Fatalf("并发提现错误：%v", createErr)
	}
	if successes.Load() != 1 || accepted.ID == 0 {
		t.Fatalf("成功冻结次数=%d，申请=%+v", successes.Load(), accepted)
	}
	if accepted.TaxQuota != 750_000 || accepted.NetPayoutQuota != 24_250_000 || accepted.RecipientType != "alipay" {
		t.Fatalf("提现税费或支付宝类型错误：%+v", accepted)
	}
	adminPending, pendingTotal, err := store.ListMerchantWithdrawals(ctx, 1, 20, "pending")
	if err != nil || pendingTotal != 1 || len(adminPending) != 1 || adminPending[0].ID != accepted.ID {
		t.Fatalf("管理员待审核列表未读取到新申请：total=%d items=%+v err=%v", pendingTotal, adminPending, err)
	}
	assertSettlementBalance(t, database, merchantID, 40*quotaPerCNY, 50*quotaPerCNY, 90*quotaPerCNY, 0, 0)

	approved, err := store.ApproveMerchantWithdrawal(ctx, accepted.ID, 1, "资料与账本核对通过", now.Add(time.Minute))
	if err != nil || approved.Status != "approved" {
		t.Fatalf("审核通过=%+v err=%v", approved, err)
	}
	if _, err := store.ApproveMerchantWithdrawal(ctx, accepted.ID, 1, "重复审核", now.Add(time.Minute)); !errors.Is(err, ErrSettlementConflict) {
		t.Fatalf("重复审核错误=%v", err)
	}
	rejected, err := store.RejectMerchantWithdrawal(ctx, accepted.ID, 1, "收款信息需要重新提交", now.Add(2*time.Minute))
	if err != nil || rejected.Status != "rejected" {
		t.Fatalf("驳回申请=%+v err=%v", rejected, err)
	}
	assertSettlementBalance(t, database, merchantID, 90*quotaPerCNY, 0, 90*quotaPerCNY, 0, 0)
	if _, err := store.RejectMerchantWithdrawal(ctx, accepted.ID, 1, "重复驳回", now.Add(2*time.Minute)); !errors.Is(err, ErrSettlementConflict) {
		t.Fatalf("重复驳回错误=%v", err)
	}

	second, err := store.CreateMerchantWithdrawal(ctx, CreateMerchantWithdrawalCommand{
		WithdrawalNo: "MW-PAID", UserID: 17, AmountQuota: 50 * quotaPerCNY, QuotaPerUnit: quotaPerCNY, RecipientType: "alipay",
		EncryptedRecipientAccount: []byte("encrypted-alipay"), RecipientAccountMasked: "138****2621", CreatedAt: now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ApproveMerchantWithdrawal(ctx, second.ID, 1, "已核对", now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	paid, err := store.PayMerchantWithdrawal(ctx, second.ID, 1, "ALIPAY-20260722-01", "支付宝转账凭证 20260722-01", now.Add(5*time.Minute))
	if err != nil || paid.Status != "paid" || paid.PayoutReference != "ALIPAY-20260722-01" {
		t.Fatalf("确认打款=%+v err=%v", paid, err)
	}
	assertSettlementBalance(t, database, merchantID, 40*quotaPerCNY, 0, 90*quotaPerCNY, 0, 50*quotaPerCNY)
	if _, err := store.PayMerchantWithdrawal(ctx, second.ID, 1, "duplicate", "重复", now.Add(6*time.Minute)); !errors.Is(err, ErrSettlementConflict) {
		t.Fatalf("重复确认打款错误=%v", err)
	}

	entries, total, err := store.MerchantLedger(ctx, 17, 1, 20)
	if err != nil || total != 5 || len(entries) != 5 {
		t.Fatalf("商户账本 total=%d len=%d err=%v", total, len(entries), err)
	}
	withdrawals, total, err := store.MerchantWithdrawals(ctx, 17, 1, 20)
	if err != nil || total != 2 || len(withdrawals) != 2 {
		t.Fatalf("商户提现 total=%d len=%d err=%v", total, len(withdrawals), err)
	}
}

func TestSQLSettlementStoreDeduplicatesConcurrentOverlap(t *testing.T) {
	database := openSettlementTestDatabase(t)
	defer database.Close()
	truncateSettlementTables(t, database)
	merchantID := seedSettlementOffer(t, database, 41, "mkt-ov-overlap")
	store := NewSQLSettlementStore(database)
	item := SettlementLog{
		ID: 3001, UserID: 17, CreatedAt: time.Now().UTC().Unix(), Type: 2,
		ModelName: "mkt-ov-overlap", Quota: 100, ChannelID: 41, Other: json.RawMessage(`{}`),
	}
	var wait sync.WaitGroup
	errorsChannel := make(chan error, 8)
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := store.ApplySettlementLog(context.Background(), item)
			if err != nil {
				errorsChannel <- err
			}
		}()
	}
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Fatalf("并发重叠同步失败：%v", err)
	}
	assertSettlementCounts(t, database, 1, 1, 0)
	assertSettlementBalance(t, database, merchantID, 90, 0, 90, 0, 0)
}

func TestSQLSettlementStoreRetriesReconciliationAfterMappingAppears(t *testing.T) {
	database := openSettlementTestDatabase(t)
	defer database.Close()
	truncateSettlementTables(t, database)
	store := NewSQLSettlementStore(database)
	item := SettlementLog{
		ID: 3002, UserID: 17, CreatedAt: time.Now().UTC().Unix(), Type: 2,
		ModelName: "mkt-ov-recovered", Quota: 50, ChannelID: 99, Other: json.RawMessage(`{}`),
	}
	result, err := store.ApplySettlementLog(context.Background(), item)
	if err != nil || result.Status != "reconciliation" {
		t.Fatalf("首次对账状态=%+v err=%v", result, err)
	}
	merchantID := seedSettlementOffer(t, database, 99, "mkt-ov-recovered")
	settled, err := store.RetrySettlementReconciliation(context.Background(), 10)
	if err != nil || settled != 1 {
		t.Fatalf("映射恢复重试 settled=%d err=%v", settled, err)
	}
	assertSettlementBalance(t, database, merchantID, 45, 0, 45, 0, 0)
	var status string
	if err := database.QueryRow("SELECT status FROM settlement_reconciliation_items WHERE settlement_source_id = $1", result.SourceID).Scan(&status); err != nil || status != "resolved" {
		t.Fatalf("对账项状态=%q err=%v", status, err)
	}
}

func TestSQLSettlementStoreMerchantLedgerUsesPublicModelName(t *testing.T) {
	database := openSettlementTestDatabase(t)
	defer database.Close()
	truncateSettlementTables(t, database)
	seedSettlementOffer(t, database, 41, "mkt-ov-ledger-hidden")
	store := NewSQLSettlementStore(database)
	result, err := store.ApplySettlementLog(context.Background(), SettlementLog{
		ID: 3003, UserID: 17, CreatedAt: time.Now().UTC().Unix(), Type: 2,
		ModelName: "mkt-ov-ledger-hidden", Quota: 100, ChannelID: 41, Other: json.RawMessage(`{}`),
	})
	if err != nil || result.Status != "settled" {
		t.Fatalf("消费结算=%+v err=%v", result, err)
	}

	entries, total, err := store.MerchantLedger(context.Background(), 17, 1, 20)
	if err != nil || total != 1 || len(entries) != 1 {
		t.Fatalf("商户账本 total=%d len=%d err=%v", total, len(entries), err)
	}
	if entries[0].ModelName != "gpt-test" {
		t.Fatalf("商户账本模型=%q，预期标准模型名", entries[0].ModelName)
	}
}

func TestSQLSettlementStoreFallsBackToUniqueUnidentifiedConsumption(t *testing.T) {
	database := openSettlementTestDatabase(t)
	defer database.Close()
	truncateSettlementTables(t, database)
	merchantID := seedSettlementOffer(t, database, 41, "mkt-ov-fallback")
	store := NewSQLSettlementStore(database)
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()

	consume := SettlementLog{
		ID: 3101, UserID: 17, CreatedAt: now.Unix(), Type: 2,
		ModelName: "mkt-ov-fallback", Quota: 101, ChannelID: 41, Other: json.RawMessage(`{}`),
	}
	if result, err := store.ApplySettlementLog(ctx, consume); err != nil || result.Status != "settled" {
		t.Fatalf("消费结算=%+v err=%v", result, err)
	}
	refund := SettlementLog{
		ID: 3102, UserID: 17, CreatedAt: now.Add(time.Minute).Unix(), Type: 6,
		ModelName: "mkt-ov-fallback", Quota: 101, ChannelID: 41, TaskID: "async-task-3102",
		Other: json.RawMessage(`{"task_id":"async-task-3102"}`),
	}
	if result, err := store.ApplySettlementLog(ctx, refund); err != nil || result.Status != "settled" {
		t.Fatalf("唯一候选退款=%+v err=%v", result, err)
	}

	assertSettlementBalance(t, database, merchantID, 0, 0, 91, 91, 0)
	assertSettlementCounts(t, database, 2, 2, 0)
	var allocationCount int
	if err := database.QueryRow("SELECT count(*) FROM settlement_refund_allocations").Scan(&allocationCount); err != nil || allocationCount != 1 {
		t.Fatalf("退款分配数量=%d err=%v", allocationCount, err)
	}
}

func TestSQLSettlementStoreLeavesAmbiguousFallbackForReconciliation(t *testing.T) {
	database := openSettlementTestDatabase(t)
	defer database.Close()
	truncateSettlementTables(t, database)
	merchantID := seedSettlementOffer(t, database, 41, "mkt-ov-ambiguous")
	store := NewSQLSettlementStore(database)
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()

	for index, id := range []int64{3201, 3202} {
		result, err := store.ApplySettlementLog(ctx, SettlementLog{
			ID: id, UserID: 17, CreatedAt: now.Add(time.Duration(index) * time.Second).Unix(), Type: 2,
			ModelName: "mkt-ov-ambiguous", Quota: 100, ChannelID: 41, Other: json.RawMessage(`{}`),
		})
		if err != nil || result.Status != "settled" {
			t.Fatalf("候选消费 %d=%+v err=%v", id, result, err)
		}
	}
	refund := SettlementLog{
		ID: 3203, UserID: 17, CreatedAt: now.Add(time.Minute).Unix(), Type: 6,
		ModelName: "mkt-ov-ambiguous", Quota: 100, ChannelID: 41, TaskID: "async-task-3203",
		Other: json.RawMessage(`{"task_id":"async-task-3203"}`),
	}
	result, err := store.ApplySettlementLog(ctx, refund)
	if err != nil || result.Status != "reconciliation" {
		t.Fatalf("歧义候选退款=%+v err=%v", result, err)
	}

	assertSettlementBalance(t, database, merchantID, 180, 0, 180, 0, 0)
	assertSettlementCounts(t, database, 3, 2, 1)
	var reason string
	if err := database.QueryRow(`
SELECT reason_code FROM settlement_reconciliation_items WHERE settlement_source_id = $1
`, result.SourceID).Scan(&reason); err != nil || reason != "refund_origin_ambiguous" {
		t.Fatalf("对账原因=%q err=%v", reason, err)
	}
}

func TestSQLSettlementStoreLeavesInsufficientFallbackForReconciliation(t *testing.T) {
	database := openSettlementTestDatabase(t)
	defer database.Close()
	truncateSettlementTables(t, database)
	merchantID := seedSettlementOffer(t, database, 41, "mkt-ov-insufficient")
	store := NewSQLSettlementStore(database)
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()

	if result, err := store.ApplySettlementLog(ctx, SettlementLog{
		ID: 3301, UserID: 17, CreatedAt: now.Unix(), Type: 2,
		ModelName: "mkt-ov-insufficient", Quota: 50, ChannelID: 41, Other: json.RawMessage(`{}`),
	}); err != nil || result.Status != "settled" {
		t.Fatalf("消费结算=%+v err=%v", result, err)
	}
	refund := SettlementLog{
		ID: 3302, UserID: 17, CreatedAt: now.Add(time.Minute).Unix(), Type: 6,
		ModelName: "mkt-ov-insufficient", Quota: 60, ChannelID: 41, TaskID: "async-task-3302",
		Other: json.RawMessage(`{"task_id":"async-task-3302"}`),
	}
	result, err := store.ApplySettlementLog(ctx, refund)
	if err != nil || result.Status != "reconciliation" {
		t.Fatalf("额度不足候选退款=%+v err=%v", result, err)
	}

	assertSettlementBalance(t, database, merchantID, 45, 0, 45, 0, 0)
	assertSettlementCounts(t, database, 2, 1, 1)
	var reason string
	if err := database.QueryRow(`
SELECT reason_code FROM settlement_reconciliation_items WHERE settlement_source_id = $1
`, result.SourceID).Scan(&reason); err != nil || reason != "refund_origin_insufficient" {
		t.Fatalf("对账原因=%q err=%v", reason, err)
	}
}

func openSettlementTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("MODELROUTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("未设置 PostgreSQL 集成测试地址")
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrations.Up(context.Background(), database); err != nil {
		database.Close()
		t.Fatal(err)
	}
	return database
}

func TestSQLSettlementStoreAppliesLogsIdempotentlyAndReversesOriginalSplit(t *testing.T) {
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
	truncateSettlementTables(t, database)
	merchantID := seedSettlementOffer(t, database, 41, "mkt-ov-settlement")
	store := NewSQLSettlementStore(database)
	now := time.Now().UTC().Truncate(time.Second)

	consume := SettlementLog{
		ID: 1001, UserID: 17, CreatedAt: now.Unix(), Type: 2, ModelName: "mkt-ov-settlement",
		Quota: 101, ChannelID: 41, TokenID: 9, RequestID: "req-cache", TaskID: "task-one",
		Other: json.RawMessage(`{"task_id":"task-one","cache_tokens":1200}`), CacheTokens: 1200,
	}
	first, err := store.ApplySettlementLog(ctx, consume)
	if err != nil || first.Status != "settled" || first.SourceID == 0 {
		t.Fatalf("首次消费结算=%+v err=%v", first, err)
	}
	duplicate, err := store.ApplySettlementLog(ctx, consume)
	if err != nil || duplicate.SourceID != first.SourceID || duplicate.Status != "settled" {
		t.Fatalf("重复消费结算=%+v err=%v", duplicate, err)
	}
	assertSettlementBalance(t, database, merchantID, 91, 0, 91, 0, 0)
	assertSettlementCounts(t, database, 1, 1, 0)

	var merchantQuota, platformQuota int64
	var other string
	if err := database.QueryRowContext(ctx, `
SELECT merchant_quota, platform_quota, other::text FROM settlement_sources WHERE newapi_log_id = 1001
`).Scan(&merchantQuota, &platformQuota, &other); err != nil {
		t.Fatal(err)
	}
	if merchantQuota != 91 || platformQuota != 10 || !json.Valid([]byte(other)) {
		t.Fatalf("缓存消费拆分 merchant=%d platform=%d other=%q", merchantQuota, platformQuota, other)
	}

	refundOne := SettlementLog{
		ID: 1002, UserID: 17, CreatedAt: now.Add(time.Minute).Unix(), Type: 6,
		ModelName: "mkt-ov-settlement", Quota: 34, ChannelID: 41, TaskID: "task-one",
		Other: json.RawMessage(`{"task_id":"task-one","reason":"partial"}`),
	}
	if result, err := store.ApplySettlementLog(ctx, refundOne); err != nil || result.Status != "settled" {
		t.Fatalf("首次退款=%+v err=%v", result, err)
	}
	assertSettlementBalance(t, database, merchantID, 60, 0, 91, 31, 0)

	refundTwo := refundOne
	refundTwo.ID = 1003
	refundTwo.Quota = 67
	refundTwo.CreatedAt = now.Add(2 * time.Minute).Unix()
	if result, err := store.ApplySettlementLog(ctx, refundTwo); err != nil || result.Status != "settled" {
		t.Fatalf("剩余退款=%+v err=%v", result, err)
	}
	assertSettlementBalance(t, database, merchantID, 0, 0, 91, 91, 0)
	assertSettlementCounts(t, database, 3, 3, 0)

	var refundedMerchant, refundedPlatform int64
	if err := database.QueryRowContext(ctx, `
SELECT -sum(merchant_quota), -sum(platform_quota)
FROM settlement_sources WHERE source_type = 6 AND status = 'settled'
`).Scan(&refundedMerchant, &refundedPlatform); err != nil {
		t.Fatal(err)
	}
	if refundedMerchant != 91 || refundedPlatform != 10 {
		t.Fatalf("累计退款 merchant=%d platform=%d", refundedMerchant, refundedPlatform)
	}

	unmapped := SettlementLog{
		ID: 1004, UserID: 18, CreatedAt: now.Unix(), Type: 2, ModelName: "mkt-ov-missing",
		Quota: 50, ChannelID: 99, Other: json.RawMessage(`{}`),
	}
	result, err := store.ApplySettlementLog(ctx, unmapped)
	if err != nil || result.Status != "reconciliation" {
		t.Fatalf("无映射消费=%+v err=%v", result, err)
	}
	assertSettlementCounts(t, database, 4, 3, 1)

	if _, err := database.ExecContext(ctx, "UPDATE merchant_ledger_entries SET description = 'tampered' WHERE id = 1"); err == nil {
		t.Fatal("只追加账本接受了更新")
	}
	if err := store.CompleteSettlementSync(ctx, now.Unix(), now); err != nil {
		t.Fatalf("推进同步水位：%v", err)
	}
	if watermark, err := store.LastSettlementSync(ctx); err != nil || watermark != now.Unix() {
		t.Fatalf("同步水位=%d err=%v", watermark, err)
	}
}

func truncateSettlementTables(t *testing.T, database *sql.DB) {
	t.Helper()
	_, err := database.Exec(`
TRUNCATE merchant_ledger_entries, settlement_refund_allocations,
         settlement_reconciliation_items, settlement_sources, merchant_withdrawals,
         merchant_balances, channel_health_events, performance_rollups, routing_decisions,
         routing_policy_merchants, routing_policies, token_bindings, channel_health,
         offer_channel_aliases, offer_versions, text_price_tiers, image_price_items,
         official_price_versions, merchant_channels, merchant_channel_reviews,
         merchant_channel_tests, merchant_profiles, merchant_applications, canonical_models
RESTART IDENTITY CASCADE
`)
	if err != nil {
		t.Fatalf("清理结算测试表：%v", err)
	}
	_, err = database.Exec(`
UPDATE settlement_sync_state
SET last_synced_through = 0, last_success_at = NULL, last_error = NULL, updated_at = now()
`)
	if err != nil {
		t.Fatalf("重置结算水位：%v", err)
	}
}

func seedSettlementOffer(t *testing.T, database *sql.DB, newAPIChannelID int, alias string) int64 {
	t.Helper()
	var modelID, priceID int64
	if err := database.QueryRow(`
INSERT INTO canonical_models (public_name, provider, protocol)
VALUES ('gpt-test', 'OpenAI', 'chat_completions') RETURNING id
`).Scan(&modelID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`
INSERT INTO official_price_versions
    (canonical_model_id, version, price_kind, published_by_newapi_user_id)
VALUES ($1, 1, 'text', 1) RETURNING id
`, modelID).Scan(&priceID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
INSERT INTO text_price_tiers
    (official_price_version_id, tier_order, minimum_input_tokens, input_price_micros, output_price_micros)
VALUES ($1, 1, 0, 1000000, 1000000)
`, priceID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("UPDATE canonical_models SET current_official_price_version_id = $1 WHERE id = $2", priceID, modelID); err != nil {
		t.Fatal(err)
	}

	var applicationID, merchantID, testID, reviewID, channelID int64
	if err := database.QueryRow(`
INSERT INTO merchant_applications
    (newapi_user_id, display_name, display_name_key, website_url, business_description, status)
VALUES (17, '结算测试商户', '结算测试商户', 'https://merchant.test', '结算测试', 'approved') RETURNING id
`).Scan(&applicationID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`
INSERT INTO merchant_profiles
    (newapi_user_id, approved_application_id, display_name, website_url, approved_by_newapi_user_id)
VALUES (17, $1, '结算测试商户', 'https://merchant.test', 1) RETURNING id
	`, applicationID).Scan(&merchantID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`
INSERT INTO merchant_channel_tests
    (newapi_user_id, canonical_model_id, base_url, credential_fingerprint, adapter_type,
     capabilities, latency_ms, receipt_token_hash, expires_at, consumed_at)
VALUES (17, $1, 'https://upstream.test', $2, 'openai_compatible',
        '["chat_completions"]'::jsonb, 10, $3, now() + interval '15 minutes', now())
RETURNING id
`, modelID, strings64("d"), strings64("f")).Scan(&testID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`
INSERT INTO merchant_channel_reviews
	(merchant_newapi_user_id, merchant_profile_id, application_id, canonical_model_id,
	 channel_test_id, channel_name, base_url, adapter_type, credential_fingerprint, status)
VALUES (17, $1, $2, $3, $4, '结算测试渠道', 'https://upstream.test',
	'openai_compatible', $5, 'approved') RETURNING id
`, merchantID, applicationID, modelID, testID, strings64("d")).Scan(&reviewID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(`
INSERT INTO merchant_channels
	(merchant_profile_id, review_id, canonical_model_id, newapi_channel_id, channel_name,
	 base_url, adapter_type, credential_fingerprint)
VALUES ($1, $2, $3, $4, '结算测试渠道', 'https://upstream.test', 'openai_compatible', $5)
RETURNING id
`, merchantID, reviewID, modelID, newAPIChannelID, strings64("e")).Scan(&channelID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO channel_health (merchant_channel_id) VALUES ($1)`, channelID); err != nil {
		t.Fatal(err)
	}
	var offerID int64
	if err := database.QueryRow(`
INSERT INTO offer_versions
	(merchant_profile_id, canonical_model_id, official_price_version_id, multiplier_bps,
	 version, status, created_by_newapi_user_id, activated_at)
VALUES ($1, $2, $3, 10000, 1, 'active', 17, now()) RETURNING id
`, merchantID, modelID, priceID).Scan(&offerID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
INSERT INTO offer_channel_aliases (offer_version_id, merchant_channel_id, internal_alias)
VALUES ($1, $2, $3)
`, offerID, channelID, alias); err != nil {
		t.Fatal(err)
	}
	return merchantID
}

func assertSettlementBalance(t *testing.T, database *sql.DB, merchantID, available, frozen, earned, refunded, paid int64) {
	t.Helper()
	var actualAvailable, actualFrozen, actualEarned, actualRefunded, actualPaid int64
	if err := database.QueryRow(`
SELECT available_quota, frozen_quota, lifetime_earned_quota, lifetime_refunded_quota, lifetime_paid_quota
FROM merchant_balances WHERE merchant_profile_id = $1
`, merchantID).Scan(&actualAvailable, &actualFrozen, &actualEarned, &actualRefunded, &actualPaid); err != nil {
		t.Fatal(err)
	}
	if actualAvailable != available || actualFrozen != frozen || actualEarned != earned || actualRefunded != refunded || actualPaid != paid {
		t.Fatalf("余额 actual=[%d %d %d %d %d] want=[%d %d %d %d %d]", actualAvailable, actualFrozen, actualEarned, actualRefunded, actualPaid, available, frozen, earned, refunded, paid)
	}
}

func assertSettlementCounts(t *testing.T, database *sql.DB, sources, ledger, reconciliation int) {
	t.Helper()
	var sourceCount, ledgerCount, reconciliationCount int
	if err := database.QueryRow("SELECT count(*) FROM settlement_sources").Scan(&sourceCount); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT count(*) FROM merchant_ledger_entries").Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow("SELECT count(*) FROM settlement_reconciliation_items WHERE status = 'open'").Scan(&reconciliationCount); err != nil {
		t.Fatal(err)
	}
	if sourceCount != sources || ledgerCount != ledger || reconciliationCount != reconciliation {
		t.Fatalf("sources=%d ledger=%d reconciliation=%d", sourceCount, ledgerCount, reconciliationCount)
	}
}
