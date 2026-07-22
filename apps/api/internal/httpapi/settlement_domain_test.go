package httpapi

import (
	"encoding/json"
	"testing"
	"time"
)

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
	raw := json.RawMessage(`{"page":1,"page_size":100,"total":1,"items":[{"id":98,"user_id":7,"created_at":1784692800,"type":2,"model_name":"mkt-ov-42","quota":37,"channel":41,"token_id":9,"request_id":"req-cache","other":"{\"cache_tokens\":1200,\"cache_ratio\":0.5}"}]}`)
	page, err := decodeSettlementLogPage(raw)
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("日志页解析结果=%+v err=%v", page, err)
	}
	item := page.Items[0]
	if item.Quota != 37 || item.CacheTokens != 1200 || item.ModelName != "mkt-ov-42" || item.ChannelID != 41 {
		t.Fatalf("最终日志字段被错误处理：%+v", item)
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
