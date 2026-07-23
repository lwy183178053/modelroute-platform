package httpapi

import (
	"encoding/json"
	"testing"
	"time"
)

const quotaPerCNY int64 = 500_000

func TestSplitSettlementQuotaKeepsIntegerIdentity(t *testing.T) {
	tests := []struct {
		quota    int64
		merchant int64
		platform int64
	}{
		{quota: 1, merchant: 1, platform: 0},
		{quota: 9, merchant: 9, platform: 0},
		{quota: 10, merchant: 9, platform: 1},
		{quota: 11, merchant: 10, platform: 1},
		{quota: 101, merchant: 91, platform: 10},
	}
	for _, test := range tests {
		merchant, platform := splitSettlementQuota(test.quota)
		if merchant != test.merchant || platform != test.platform || merchant+platform != test.quota {
			t.Fatalf("quota=%d 拆分为 merchant=%d platform=%d", test.quota, merchant, platform)
		}
	}
}

func TestWithdrawalBreakdownRequiresWholeYuanAndMinimumAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		wantTax int64
		wantNet int64
		wantOK  bool
	}{
		{name: "四十九元", amount: 49 * quotaPerCNY, wantOK: false},
		{name: "五十元", amount: 50 * quotaPerCNY, wantTax: 750_000, wantNet: 24_250_000, wantOK: true},
		{name: "五十一元", amount: 51 * quotaPerCNY, wantTax: 765_000, wantNet: 24_735_000, wantOK: true},
		{name: "一百元", amount: 100 * quotaPerCNY, wantTax: 1_500_000, wantNet: 48_500_000, wantOK: true},
		{name: "五十点五元", amount: 50*quotaPerCNY + quotaPerCNY/2, wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tax, net, ok := withdrawalBreakdown(test.amount, quotaPerCNY)
			if tax != test.wantTax || net != test.wantNet || ok != test.wantOK {
				t.Fatalf("withdrawalBreakdown(%d)=(%d,%d,%t)，期望=(%d,%d,%t)", test.amount, tax, net, ok, test.wantTax, test.wantNet, test.wantOK)
			}
		})
	}
}

func TestWithdrawalBreakdownUsesRuntimeCoreQuotaUnit(t *testing.T) {
	tax, net, ok := withdrawalBreakdown(50*750_000, 750_000)
	if !ok || tax != 1_125_000 || net != 36_375_000 {
		t.Fatalf("非默认额度单位提现拆分 tax=%d net=%d ok=%t", tax, net, ok)
	}
	if _, _, ok := withdrawalBreakdown(50*750_000+1, 750_000); ok {
		t.Fatal("应拒绝非人民币整数金额")
	}
}

func TestRefundSplitUsesOriginalCumulativeAllocation(t *testing.T) {
	merchant, platform, err := refundSplitDelta(101, 91, 10, 0, 34)
	if err != nil || merchant != 31 || platform != 3 {
		t.Fatalf("首次部分退款拆分 merchant=%d platform=%d err=%v", merchant, platform, err)
	}
	merchant, platform, err = refundSplitDelta(101, 91, 10, 34, 67)
	if err != nil || merchant != 60 || platform != 7 {
		t.Fatalf("剩余退款拆分 merchant=%d platform=%d err=%v", merchant, platform, err)
	}
	if _, _, err := refundSplitDelta(101, 91, 10, 100, 2); err == nil {
		t.Fatal("超出原消费的退款必须失败")
	}
}

func TestDecodeSettlementLogPageKeepsNewAPIFinalQuotaAndCacheMetadata(t *testing.T) {
	raw := json.RawMessage(`{"page":1,"page_size":100,"total":1,"items":[{"id":98,"user_id":7,"created_at":1784692800,"type":2,"model_name":"mkt-ov-42","quota":37,"channel":41,"token_id":9,"request_id":"req-cache","prompt_tokens":2400,"completion_tokens":680,"other":"{\"cache_tokens\":1200,\"cache_ratio\":0.5}"}]}`)
	page, err := decodeSettlementLogPage(raw)
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("日志页解析结果=%+v err=%v", page, err)
	}
	item := page.Items[0]
	if item.Quota != 37 || item.CacheTokens != 1200 || item.PromptTokens != 2400 || item.CompletionTokens != 680 || item.ModelName != "mkt-ov-42" || item.ChannelID != 41 {
		t.Fatalf("最终日志字段被错误处理：%+v", item)
	}
	var metadata map[string]any
	if err := json.Unmarshal(item.Other, &metadata); err != nil || metadata["prompt_tokens"] != "2400" || metadata["completion_tokens"] != "680" {
		t.Fatalf("Token 元数据没有合并到结算来源：%s err=%v", item.Other, err)
	}
}

func TestSettlementWindowOverlapsPreviousWatermark(t *testing.T) {
	now := time.Unix(1784696400, 0).UTC()
	start, end := settlementWindow(1784692800, now)
	if start != 1784692500 || end != 1784696395 {
		t.Fatalf("同步窗口 start=%d end=%d", start, end)
	}
	firstStart, firstEnd := settlementWindow(0, now)
	if firstStart != now.Add(-24*time.Hour).Unix() || firstEnd != end {
		t.Fatalf("首次同步窗口 start=%d end=%d", firstStart, firstEnd)
	}
}
