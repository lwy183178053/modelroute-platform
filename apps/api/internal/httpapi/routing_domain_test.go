package httpapi

import (
	"encoding/json"
	"net/http"
	"slices"
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

func TestNextProbeStateAppliesThreeFailuresAndRecoveryGate(t *testing.T) {
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	probe := HealthProbe{State: "healthy", ConsecutiveFailures: 2, CooldownLevel: 0}
	failure := nextProbeState(probe, healthFailureTemporary, now, 0)
	if failure.State != "cooling" || failure.CooldownLevel != 1 || !failure.NextProbeAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("第三次失败状态=%+v", failure)
	}
	firstRecovery := nextProbeState(HealthProbe{State: "cooling", CooldownLevel: 1}, healthFailureNone, now.Add(time.Minute), 0)
	if firstRecovery.State != "recovering" || firstRecovery.RecoverySuccesses != 1 || !firstRecovery.NextProbeAt.Equal(now.Add(time.Minute+10*time.Second)) {
		t.Fatalf("第一次恢复状态=%+v", firstRecovery)
	}
	secondRecovery := nextProbeState(HealthProbe{State: "recovering", CooldownLevel: 1, RecoverySuccesses: 1}, healthFailureNone, now.Add(time.Minute+10*time.Second), 0)
	if secondRecovery.State != "healthy" || secondRecovery.RecoverySuccesses != 0 {
		t.Fatalf("第二次恢复状态=%+v", secondRecovery)
	}
}

func TestNextProbeStateUsesEscalatingCooldownAndRetryAfter(t *testing.T) {
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	state := nextProbeState(HealthProbe{State: "recovering", CooldownLevel: 2}, healthFailureTemporary, now, 20*time.Minute)
	if state.CooldownLevel != 3 || !state.NextProbeAt.Equal(now.Add(20*time.Minute)) {
		t.Fatalf("升级冷却状态=%+v", state)
	}
}

func TestSortCandidatesImplementsPriceAndStabilityPriority(t *testing.T) {
	base := []RouteCandidate{
		{OfferID: 1, MultiplierBPS: 4_000, HealthState: "healthy", AvailabilityPPM: 990_000, TTFTP95MS: 500},
		{OfferID: 2, MultiplierBPS: 3_000, HealthState: "degraded", AvailabilityPPM: 999_000, TTFTP95MS: 100},
		{OfferID: 3, MultiplierBPS: 4_000, HealthState: "healthy", AvailabilityPPM: 995_000, TTFTP95MS: 300},
	}
	price := append([]RouteCandidate(nil), base...)
	sortCandidates(price, "price")
	if got := []int64{price[0].OfferID, price[1].OfferID, price[2].OfferID}; !slices.Equal(got, []int64{2, 3, 1}) {
		t.Fatalf("价格优先顺序 = %v", got)
	}
	stability := append([]RouteCandidate(nil), base...)
	sortCandidates(stability, "stability")
	if got := []int64{stability[0].OfferID, stability[1].OfferID, stability[2].OfferID}; !slices.Equal(got, []int64{3, 1, 2}) {
		t.Fatalf("稳定性优先顺序 = %v", got)
	}
}

func TestApplyBasisPointsRoundsDownWithoutFloat(t *testing.T) {
	if got := applyBasisPoints(1_234_567, 4_000); got != 493_826 {
		t.Fatalf("缩放结果 = %d", got)
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
