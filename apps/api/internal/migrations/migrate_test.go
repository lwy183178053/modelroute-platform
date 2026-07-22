package migrations

import (
	"strings"
	"testing"
)

func TestEmbeddedMigrationsAreOrdered(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatalf("Files() 返回错误：%v", err)
	}
	if len(files) != 5 {
		t.Fatalf("迁移文件数量 = %d", len(files))
	}
	if files[0].Version != 1 || files[0].Name != "000001_foundation.sql" {
		t.Fatalf("迁移文件不符合预期：%+v", files[0])
	}
	if files[0].Checksum == "" || files[0].SQL == "" {
		t.Fatal("迁移内容和校验和必须存在")
	}
	if files[1].Version != 2 || files[1].Name != "000002_merchants.sql" {
		t.Fatalf("商户迁移不符合预期：%+v", files[1])
	}
	if files[2].Version != 3 || files[2].Name != "000003_routing.sql" {
		t.Fatalf("路由迁移不符合预期：%+v", files[2])
	}
	if files[3].Version != 4 || files[3].Name != "000004_payments.sql" {
		t.Fatalf("支付迁移不符合预期：%+v", files[3])
	}
	if files[4].Version != 5 || files[4].Name != "000005_settlement.sql" {
		t.Fatalf("结算迁移不符合预期：%+v", files[4])
	}
	for _, table := range []string{"merchant_applications", "merchant_profiles", "merchant_channel_reviews", "merchant_channels"} {
		if !strings.Contains(files[1].SQL, "CREATE TABLE "+table) {
			t.Fatalf("商户迁移缺少表 %s", table)
		}
	}
	for _, table := range []string{"canonical_models", "official_price_versions", "offer_versions", "token_bindings", "routing_policies", "channel_model_health", "routing_decisions", "channel_health_events"} {
		if !strings.Contains(files[2].SQL, "CREATE TABLE "+table) {
			t.Fatalf("路由迁移缺少表 %s", table)
		}
	}
	for _, table := range []string{"payment_setting_versions", "payment_orders", "payment_callback_events", "newapi_credit_operations"} {
		if !strings.Contains(files[3].SQL, "CREATE TABLE "+table) {
			t.Fatalf("支付迁移缺少表 %s", table)
		}
	}
	for _, table := range []string{
		"settlement_sync_state", "settlement_sources", "settlement_refund_allocations",
		"settlement_reconciliation_items", "merchant_balances", "merchant_ledger_entries",
		"merchant_withdrawals",
	} {
		if !strings.Contains(files[4].SQL, "CREATE TABLE "+table) {
			t.Fatalf("结算迁移缺少表 %s", table)
		}
	}
}
