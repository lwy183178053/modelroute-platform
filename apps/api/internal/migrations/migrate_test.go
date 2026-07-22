package migrations

import (
	"strings"
	"testing"
)

func TestEmbeddedMigrationIsV2Baseline(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatalf("Files() 返回错误：%v", err)
	}
	if len(files) != 1 {
		t.Fatalf("V2 必须只有一个空库基线迁移，实际 = %d", len(files))
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
		"protocol IN ('chat_completions', 'responses', 'image_generation')",
		"input_price_micros", "cache_write_price_micros",
		"quality varchar(32)", "size varchar(32)", "price_micros",
		"minimum_multiplier_bps BETWEEN 0 AND 50000",
		"maximum_multiplier_bps BETWEEN 0 AND 50000",
		"next_probe_at timestamptz",
		"minimum_cny_cents", "maximum_cny_cents", "amount_cny_cents", "newapi_quota",
	} {
		if !strings.Contains(file.SQL, fragment) {
			t.Fatalf("V2 基线缺少约束或字段 %q", fragment)
		}
	}

	for _, removed := range []string{
		"channel_type", "requested_models", "lifecycle", "original_currency",
		"usd_exchange_rate_micros", "source_url", "checked_at", "usd_cents",
		"exchange_rate_cny_cents_per_usd", "embeddings",
	} {
		if strings.Contains(file.SQL, removed) {
			t.Fatalf("V2 基线仍包含已删除字段或能力 %q", removed)
		}
	}
}
