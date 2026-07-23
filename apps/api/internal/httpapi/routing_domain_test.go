package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestScaledDecimalUsesIntegerArithmetic(t *testing.T) {
	tests := []struct {
		value int64
		scale int64
		want  string
	}{
		{value: 1_500_000, scale: 1_000_000, want: "1.5"},
		{value: 250_000, scale: 1_000_000, want: "0.25"},
		{value: 1, scale: 1_000_000, want: "0.000001"},
		{value: 2_000_000, scale: 1_000_000, want: "2"},
	}
	for _, test := range tests {
		if got := scaledDecimal(test.value, test.scale); got != test.want {
			t.Fatalf("scaledDecimal(%d, %d) = %q，期望 %q", test.value, test.scale, got, test.want)
		}
	}
}

func TestNextProbeStateAppliesThreeFailuresAndSingleRecoveryProbe(t *testing.T) {
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	probe := HealthProbe{State: "healthy", ConsecutiveFailures: 2, CooldownLevel: 0}
	failure := nextProbeState(probe, healthFailureTemporary, now, 0)
	if failure.State != "cooling" || failure.CooldownLevel != 1 || !failure.NextProbeAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("第三次失败状态=%+v", failure)
	}
	recovered := nextProbeState(HealthProbe{State: "cooling", CooldownLevel: 1}, healthFailureNone, now.Add(time.Minute), 0)
	if recovered.State != "healthy" || recovered.CooldownLevel != 0 || !recovered.NextProbeAt.IsZero() {
		t.Fatalf("一次恢复探测后的状态=%+v", recovered)
	}
}

func TestNextProbeStateUsesEscalatingCooldownAndRetryAfter(t *testing.T) {
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	state := nextProbeState(HealthProbe{State: "recovering", CooldownLevel: 2}, healthFailureTemporary, now, 20*time.Minute)
	if state.CooldownLevel != 3 || !state.NextProbeAt.Equal(now.Add(20*time.Minute)) {
		t.Fatalf("升级冷却状态=%+v", state)
	}
}

func TestCooldownCapsAtTwoHoursAndHealthyRequestsDoNotScheduleProbe(t *testing.T) {
	if got := cooldownForLevel(6); got != 120*time.Minute {
		t.Fatalf("第六档冷却=%s", got)
	}
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	healthy := nextProbeState(HealthProbe{State: "healthy"}, healthFailureNone, now, 0)
	if !healthy.NextProbeAt.IsZero() {
		t.Fatalf("健康请求安排了主动探活：%s", healthy.NextProbeAt)
	}
}

type fixedRouteRandom struct{ values []float64 }

func (r *fixedRouteRandom) Float64() float64 {
	value := r.values[0]
	r.values = append(r.values[1:], value)
	return value
}

func TestMerchantRouteWeightsAreBoundedAtFourToOne(t *testing.T) {
	candidates := []RouteCandidate{
		{MerchantID: 1, MerchantChannelID: 11, MultiplierBPS: 1_000, AvailabilityPPM: 1_000_000, TTFTP95MS: 100, MetricRequestCount: 100, CacheHitPPM: 1_000_000, CacheSampleInputTokens: 1_000_000, CacheWeightReady: true},
		{MerchantID: 2, MerchantChannelID: 21, MultiplierBPS: 5_000, AvailabilityPPM: 0, TTFTP95MS: 5_000, MetricRequestCount: 100, CacheHitPPM: 0, CacheSampleInputTokens: 1_000_000, CacheWeightReady: true},
	}
	weights := merchantCandidateWeights(candidates, "cache")
	if weights[1] != 4 || weights[2] != 1 {
		t.Fatalf("商户权重=%v", weights)
	}
}

func TestWeightedCandidateOrderStartsFromMerchantNotChannelCount(t *testing.T) {
	candidates := []RouteCandidate{{MerchantID: 2, MerchantChannelID: 21, OfferChannelAliasID: 21, MultiplierBPS: 1_000}}
	for index := int64(1); index <= 10; index++ {
		candidates = append(candidates, RouteCandidate{MerchantID: 1, MerchantChannelID: index, OfferChannelAliasID: index, MultiplierBPS: 1_000})
	}
	ordered := weightedCandidateOrder(candidates, "balanced", &fixedRouteRandom{values: []float64{0.75, 0}})
	if len(ordered) != 11 || ordered[0].MerchantID != 2 {
		t.Fatalf("首选候选=%+v，总数=%d", ordered[0], len(ordered))
	}
	seen := map[int64]bool{}
	for _, candidate := range ordered {
		if seen[candidate.OfferChannelAliasID] {
			t.Fatalf("候选重复=%d", candidate.OfferChannelAliasID)
		}
		seen[candidate.OfferChannelAliasID] = true
	}
}

func TestWeightedCandidateOrderUsesOneChannelPerMerchantInEachRound(t *testing.T) {
	candidates := []RouteCandidate{
		{MerchantID: 1, MerchantChannelID: 11, OfferChannelAliasID: 11, MultiplierBPS: 1_000},
		{MerchantID: 1, MerchantChannelID: 12, OfferChannelAliasID: 12, MultiplierBPS: 1_000},
		{MerchantID: 1, MerchantChannelID: 13, OfferChannelAliasID: 13, MultiplierBPS: 1_000},
		{MerchantID: 2, MerchantChannelID: 21, OfferChannelAliasID: 21, MultiplierBPS: 1_000},
		{MerchantID: 2, MerchantChannelID: 22, OfferChannelAliasID: 22, MultiplierBPS: 1_000},
		{MerchantID: 3, MerchantChannelID: 31, OfferChannelAliasID: 31, MultiplierBPS: 1_000},
	}
	ordered := weightedCandidateOrder(candidates, "balanced", &fixedRouteRandom{values: []float64{0}})
	wantMerchants := []int64{1, 2, 3, 1, 2, 1}
	if len(ordered) != len(wantMerchants) {
		t.Fatalf("候选数量=%d，期望=%d", len(ordered), len(wantMerchants))
	}
	for index, wantMerchant := range wantMerchants {
		if ordered[index].MerchantID != wantMerchant {
			t.Fatalf("第 %d 个候选商户=%d，期望=%d；完整顺序=%+v", index, ordered[index].MerchantID, wantMerchant, ordered)
		}
	}
}

func TestCachePriorityTreatsLowSampleAsNeutral(t *testing.T) {
	candidates := []RouteCandidate{
		{MerchantID: 1, MerchantChannelID: 11, MultiplierBPS: 1_000, CacheHitPPM: 1_000_000, CacheSampleInputTokens: 999_999},
		{MerchantID: 2, MerchantChannelID: 21, MultiplierBPS: 1_000, CacheHitPPM: 0, CacheSampleInputTokens: 999_999},
	}
	weights := merchantCandidateWeights(candidates, "cache")
	if weights[1] != weights[2] {
		t.Fatalf("低样本缓存影响了权重=%v", weights)
	}
}

func TestSmartCacheFloorTreatsInsufficientSampleAsNeutral(t *testing.T) {
	floor := 600_000
	selection := RouteSelection{MinimumCacheHitPPM: &floor}
	for _, test := range []struct {
		name      string
		candidate RouteCandidate
		want      bool
	}{
		{name: "样本不足", candidate: RouteCandidate{MerchantCacheHitPPM: 0, MerchantCacheWeightReady: false}, want: true},
		{name: "低于阈值", candidate: RouteCandidate{MerchantCacheHitPPM: 599_999, MerchantCacheWeightReady: true}, want: false},
		{name: "达到阈值", candidate: RouteCandidate{MerchantCacheHitPPM: 600_000, MerchantCacheWeightReady: true}, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := routeCandidateMeetsCacheFloor(selection, test.candidate); got != test.want {
				t.Fatalf("过滤结果=%v want=%v", got, test.want)
			}
		})
	}
	if !routeCandidateMeetsCacheFloor(selection, RouteCandidate{Protocol: "image_generation"}) {
		t.Fatal("图片模型不应应用缓存命中率阈值")
	}
}

func TestRewritePublicModelOnlyChangesModelField(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":"保留内容"}],"stream":true}`)
	rewritten, model, err := rewritePublicModel(body, "mkt-ov-42")
	if err != nil {
		t.Fatalf("改写模型失败：%v", err)
	}
	if model != "gpt-5.5" {
		t.Fatalf("公开模型 = %q", model)
	}
	var decoded map[string]any
	if err := json.Unmarshal(rewritten, &decoded); err != nil {
		t.Fatalf("解析改写请求失败：%v", err)
	}
	if decoded["model"] != "mkt-ov-42" || decoded["stream"] != true {
		t.Fatalf("改写结果不符合预期：%v", decoded)
	}
	messages := decoded["messages"].([]any)
	if messages[0].(map[string]any)["content"] != "保留内容" {
		t.Fatalf("请求内容被改动：%v", messages)
	}
}

func TestClassifyProbeResultDistinguishesHardAndTemporaryFailures(t *testing.T) {
	for _, test := range []struct {
		status int
		want   healthFailureClass
	}{
		{status: http.StatusUnauthorized, want: healthFailureHard},
		{status: http.StatusPaymentRequired, want: healthFailureHard},
		{status: http.StatusTooManyRequests, want: healthFailureTemporary},
		{status: http.StatusBadGateway, want: healthFailureTemporary},
		{status: http.StatusBadRequest, want: healthFailureIgnored},
		{status: http.StatusOK, want: healthFailureNone},
	} {
		if got := classifyProbeStatus(test.status); got != test.want {
			t.Fatalf("状态 %d 分类 = %q，期望 %q", test.status, got, test.want)
		}
	}
}
