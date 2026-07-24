package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestEmbeddedMigrationIsV2Baseline(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatalf("Files() 返回错误：%v", err)
	}
	if len(files) != 12 {
		t.Fatalf("V2 必须包含十二份连续迁移，实际 = %d", len(files))
	}
	file := files[0]
	if file.Version != 1 || file.Name != "000001_v2_baseline.sql" {
		t.Fatalf("V2 基线迁移不符合预期：%+v", file)
	}
	if file.Checksum == "" || file.SQL == "" {
		t.Fatal("迁移内容和校验和必须存在")
	}

	for _, table := range []string{
		"merchant_profiles", "merchant_channel_tests", "merchant_channel_reviews", "merchant_channels",
		"canonical_models", "official_price_versions", "text_price_tiers", "image_price_items",
		"offer_versions", "offer_channel_aliases", "token_bindings", "routing_policies",
		"channel_health", "routing_decisions", "payment_setting_versions", "payment_orders",
		"settlement_sources", "merchant_balances", "merchant_ledger_entries", "merchant_withdrawals",
		"announcements", "announcement_reads",
	} {
		if !strings.Contains(file.SQL, "CREATE TABLE "+table) {
			t.Fatalf("V2 基线缺少表 %s", table)
		}
	}

	for _, fragment := range []string{
		"canonical_model_id bigint NOT NULL REFERENCES canonical_models(id)",
		"adapter_type text NOT NULL",
		"expires_at timestamptz NOT NULL",
		"multiplier_bps BETWEEN 0 AND 50000",
		"offer_versions_one_active_per_merchant_model_idx",
		"UNIQUE (offer_version_id, merchant_channel_id)",
		"input_price_micros", "cache_write_price_micros",
		"quality varchar(32)", "size varchar(32)", "price_micros",
		"minimum_multiplier_bps BETWEEN 0 AND 50000",
		"maximum_multiplier_bps BETWEEN 0 AND 50000",
		"next_probe_at timestamptz",
		"minimum_cny_cents", "maximum_cny_cents", "amount_cny_cents", "newapi_quota",
		"amount_quota >= 25000000", "amount_quota % 500000 = 0",
		"tax_quota bigint GENERATED ALWAYS AS", "net_payout_quota bigint GENERATED ALWAYS AS",
		"recipient_type = 'alipay'",
	} {
		if !strings.Contains(file.SQL, fragment) {
			t.Fatalf("V2 基线缺少约束或字段 %q", fragment)
		}
	}

	for _, removed := range []string{
		"requested_models", "lifecycle", "original_currency",
		"usd_exchange_rate_micros", "source_url", "checked_at", "usd_cents",
		"exchange_rate_cny_cents_per_usd", "embeddings", "protocol text",
	} {
		if strings.Contains(file.SQL, removed) {
			t.Fatalf("V2 基线仍包含已删除字段或能力 %q", removed)
		}
	}
	if !strings.Contains(files[1].SQL, "merchant_review_model_bindings") || !strings.Contains(files[1].SQL, "merchant_channel_model_bindings") {
		t.Fatal("多模型渠道迁移缺少模型绑定表")
	}
	if !strings.Contains(files[1].SQL, "channel_type integer NOT NULL DEFAULT 1") || !strings.Contains(files[1].SQL, "test_status") {
		t.Fatal("多模型渠道迁移缺少渠道类型或逐模型检查字段")
	}
	if !strings.Contains(files[1].SQL, "ALTER COLUMN channel_test_id DROP NOT NULL") {
		t.Fatal("多模型渠道迁移未允许没有旧版检测凭证的审核记录")
	}
	if !strings.Contains(files[2].SQL, "'deleted'") {
		t.Fatal("模型生命周期迁移缺少 deleted 状态")
	}
	for _, fragment := range []string{
		"display_name_key", "CREATE TABLE merchant_name_reservations",
		"merchant_name_reservations_pkey", "status IN ('pending', 'active')",
	} {
		if !strings.Contains(files[3].SQL, fragment) {
			t.Fatalf("商户名称迁移缺少 %q", fragment)
		}
	}
	for _, fragment := range []string{
		"newapi_user_id bigint NOT NULL",
		"name varchar(50) NOT NULL",
		"priority IN ('balanced', 'price', 'success', 'latency', 'cache')",
		"minimum_cache_hit_ppm",
		"routing_policy_id bigint NOT NULL REFERENCES routing_policies(id)",
		"algorithm_version varchar(32)",
		"candidate_count integer",
		"selected_metrics jsonb",
	} {
		if !strings.Contains(files[4].SQL, fragment) {
			t.Fatalf("独立路由迁移缺少 %q", fragment)
		}
	}
	for _, removed := range []string{
		"mode text NOT NULL CHECK (mode IN ('smart', 'fixed'))",
		"CREATE TABLE routing_policy_merchants",
		"merchant_profile_id bigint NOT NULL REFERENCES merchant_profiles(id)",
	} {
		if strings.Contains(files[4].SQL, removed) {
			t.Fatalf("独立路由迁移仍包含已删除的固定路由结构 %q", removed)
		}
	}
	for _, fragment := range []string{
		"CREATE TABLE route_metric_buckets",
		"CREATE TABLE route_metric_snapshots",
		"prompt_tokens bigint",
		"cache_tokens bigint",
		"cache_hit_ppm integer",
		"cache_weight_ready boolean",
		"metrics_updated_at timestamptz",
	} {
		if !strings.Contains(files[5].SQL, fragment) {
			t.Fatalf("缓存指标迁移缺少 %q", fragment)
		}
	}
	for _, fragment := range []string{
		"DROP CONSTRAINT IF EXISTS merchant_withdrawals_amount_quota_check",
		"CHECK (amount_quota > 0)",
	} {
		if !strings.Contains(files[6].SQL, fragment) {
			t.Fatalf("动态额度单位迁移缺少 %q", fragment)
		}
	}
	for _, fragment := range []string{
		"ADD COLUMN withdrawal_tax_bps integer NOT NULL DEFAULT 300",
		"withdrawal_tax_bps BETWEEN 0 AND 10000",
		"amount_quota * withdrawal_tax_bps / 10000",
	} {
		if !strings.Contains(files[7].SQL, fragment) {
			t.Fatalf("提现税率迁移缺少 %q", fragment)
		}
	}
	if !strings.Contains(files[9].SQL, "ADD COLUMN provider varchar(32)") {
		t.Fatal("渠道厂商迁移缺少 provider 字段")
	}
	for _, provider := range []string{"OpenAI", "Anthropic", "Gemini", "DeepSeek", "Qwen", "Moonshot"} {
		if !strings.Contains(files[9].SQL, provider) {
			t.Fatalf("渠道厂商迁移缺少厂商 %q", provider)
		}
	}
	if !strings.Contains(files[8].SQL, "business_description") {
		t.Fatal("商户资料迁移缺少商户介绍字段")
	}
	if !strings.Contains(files[9].SQL, "TRUNCATE TABLE merchant_channel_reviews, merchant_channels") {
		t.Fatal("渠道厂商迁移未按开发约定清理旧渠道数据")
	}
	for _, fragment := range []string{
		"canonical_model_id bigint NOT NULL REFERENCES canonical_models(id)",
		"provider varchar(32) NOT NULL",
		"CREATE TABLE token_model_routes",
		"UNIQUE (token_binding_id, canonical_model_id)",
		"FOREIGN KEY (routing_policy_id, canonical_model_id)",
		"ALTER COLUMN token_binding_id DROP NOT NULL",
		"ALTER COLUMN routing_policy_id DROP NOT NULL",
		"ON DELETE SET NULL",
		"DELETE FROM token_bindings",
		"DELETE FROM routing_policies",
	} {
		if !strings.Contains(files[10].SQL, fragment) {
			t.Fatalf("单厂商密钥迁移缺少 %q", fragment)
		}
	}
	for _, provider := range []string{"OpenAI", "Anthropic", "Gemini", "DeepSeek", "Qwen", "Moonshot"} {
		if !strings.Contains(files[10].SQL, provider) {
			t.Fatalf("单厂商密钥迁移缺少厂商 %q", provider)
		}
	}
	if strings.Contains(files[10].SQL, "DELETE FROM routing_decisions") {
		t.Fatal("单厂商密钥迁移不得删除历史调用路由记录")
	}
	for _, fragment := range []string{
		"health_state text NOT NULL DEFAULT 'pending'",
		"consecutive_request_failures integer NOT NULL DEFAULT 0",
		"automatic_check_failures integer NOT NULL DEFAULT 0",
		"next_check_at timestamptz",
		"deleted_at timestamptz",
		"DROP TABLE merchant_channel_reviews",
		"DROP TABLE merchant_review_model_bindings",
		"DROP TABLE channel_health",
	} {
		if !strings.Contains(files[11].SQL, fragment) {
			t.Fatalf("逐模型健康迁移缺少 %q", fragment)
		}
	}
	if strings.Contains(files[11].SQL, "merchant_channel_test_receipts") {
		t.Fatal("逐模型检查迁移不应保留批量检测凭证表")
	}
}

func TestProviderModelRouteMigrationKeepsHistoricalDecisions(t *testing.T) {
	databaseURL := os.Getenv("MODELROUTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("未设置 PostgreSQL 集成测试地址")
	}
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("modelroute_migration_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsedURL.Query()
	query.Set("search_path", schema)
	parsedURL.RawQuery = query.Encode()
	database, err := sql.Open("pgx", parsedURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `
CREATE TABLE platform_schema_migrations (
    version integer PRIMARY KEY,
    name text NOT NULL,
    checksum text NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now()
)
`); err != nil {
		t.Fatal(err)
	}
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files[:10] {
		if _, err := database.ExecContext(ctx, file.SQL); err != nil {
			t.Fatalf("执行旧迁移 %s：%v", file.Name, err)
		}
		if _, err := database.ExecContext(ctx, `
INSERT INTO platform_schema_migrations (version, name, checksum) VALUES ($1, $2, $3)
`, file.Version, file.Name, file.Checksum); err != nil {
			t.Fatal(err)
		}
	}

	var modelID, policyID, bindingID int64
	if err := database.QueryRowContext(ctx, `
INSERT INTO canonical_models (public_name, provider, status)
VALUES ('migration-history-model', 'OpenAI', 'active') RETURNING id
`).Scan(&modelID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO routing_policies
    (newapi_user_id, name, priority, minimum_multiplier_bps, maximum_multiplier_bps)
VALUES (7, '旧路由', 'balanced', 0, 50000) RETURNING id
`).Scan(&policyID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `
INSERT INTO token_bindings
    (newapi_token_id, newapi_user_id, token_fingerprint, routing_policy_id)
VALUES (9, 7, $1, $2) RETURNING id
`, strings.Repeat("a", 64), policyID).Scan(&bindingID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
INSERT INTO routing_decisions
    (request_id, token_binding_id, routing_policy_id, canonical_model_id,
     priority, algorithm_version, candidate_count, result)
VALUES ('migration-history-request', $1, $2, $3, 'balanced', 'weighted-v2', 1, 'completed')
`, bindingID, policyID, modelID); err != nil {
		t.Fatal(err)
	}

	if err := Up(ctx, database); err != nil {
		t.Fatalf("执行第 11 号迁移：%v", err)
	}
	var historicalBindingID, historicalPolicyID sql.NullInt64
	if err := database.QueryRowContext(ctx, `
SELECT token_binding_id, routing_policy_id
FROM routing_decisions
WHERE request_id = 'migration-history-request'
`).Scan(&historicalBindingID, &historicalPolicyID); err != nil {
		t.Fatalf("读取保留的调用记录：%v", err)
	}
	if historicalBindingID.Valid || historicalPolicyID.Valid {
		t.Fatalf("旧密钥或路由引用未解除：密钥=%v 路由=%v", historicalBindingID, historicalPolicyID)
	}
	var tokenCount, policyCount int
	if err := database.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM token_bindings), (SELECT count(*) FROM routing_policies)`).Scan(&tokenCount, &policyCount); err != nil {
		t.Fatal(err)
	}
	if tokenCount != 0 || policyCount != 0 {
		t.Fatalf("旧密钥或路由未清空：密钥=%d 路由=%d", tokenCount, policyCount)
	}
}
