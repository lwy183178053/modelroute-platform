package httpapi

import (
	"net/http"
	"testing"
	"time"
)

func TestThirdTemporaryRequestFailureCoolsOnlyTextModel(t *testing.T) {
	now := time.Date(2026, 7, 25, 3, 0, 0, 0, time.UTC)
	text := nextModelHealthAfterRequest(ModelHealthState{
		State: "healthy", ConsecutiveRequestFailures: 2,
	}, healthFailureTemporary, "text", http.StatusServiceUnavailable, now)
	if text.State != "cooling" || text.ConsecutiveRequestFailures != 3 || !text.NextCheckAt.Equal(now.Add(3*time.Minute)) {
		t.Fatalf("文本模型第三次失败状态=%+v", text)
	}
	image := nextModelHealthAfterRequest(ModelHealthState{
		State: "healthy", ConsecutiveRequestFailures: 2,
	}, healthFailureTemporary, "image", http.StatusServiceUnavailable, now)
	if image.State != "error" || !image.NextCheckAt.IsZero() {
		t.Fatalf("图片模型第三次失败状态=%+v", image)
	}
}

func TestAutomaticCheckRunsEveryThreeMinutesAndErrorsOnTenthFailure(t *testing.T) {
	now := time.Date(2026, 7, 25, 3, 0, 0, 0, time.UTC)
	ninth := nextModelHealthAfterAutomaticCheck(ModelHealthState{
		State: "cooling", AutomaticCheckFailures: 8,
	}, healthFailureTemporary, now)
	if ninth.State != "cooling" || ninth.AutomaticCheckFailures != 9 || !ninth.NextCheckAt.Equal(now.Add(3*time.Minute)) {
		t.Fatalf("第九次自动检查失败状态=%+v", ninth)
	}
	tenth := nextModelHealthAfterAutomaticCheck(ModelHealthState{
		State: "cooling", AutomaticCheckFailures: 9,
	}, healthFailureTemporary, now)
	if tenth.State != "error" || tenth.AutomaticCheckFailures != 10 || !tenth.NextCheckAt.IsZero() {
		t.Fatalf("第十次自动检查失败状态=%+v", tenth)
	}
	recovered := nextModelHealthAfterAutomaticCheck(ninth, healthFailureNone, now)
	if recovered.State != "healthy" || recovered.AutomaticCheckFailures != 0 || recovered.ConsecutiveRequestFailures != 0 || !recovered.NextCheckAt.IsZero() {
		t.Fatalf("自动检查恢复状态=%+v", recovered)
	}
}

func TestManualCheckImmediatelyAppliesSuccessOrError(t *testing.T) {
	now := time.Date(2026, 7, 25, 3, 0, 0, 0, time.UTC)
	success := nextModelHealthAfterManualCheck(ModelHealthState{
		State: "error", ConsecutiveRequestFailures: 3, AutomaticCheckFailures: 10,
	}, healthFailureNone, now)
	if success.State != "healthy" || success.ConsecutiveRequestFailures != 0 || success.AutomaticCheckFailures != 0 || !success.NextCheckAt.IsZero() {
		t.Fatalf("手动检查成功状态=%+v", success)
	}
	failure := nextModelHealthAfterManualCheck(ModelHealthState{State: "cooling"}, healthFailureTemporary, now)
	if failure.State != "error" || !failure.NextCheckAt.IsZero() {
		t.Fatalf("手动检查失败状态=%+v", failure)
	}
}

func TestSuccessfulRealRequestClearsOnlyRequestFailureCounter(t *testing.T) {
	now := time.Date(2026, 7, 25, 3, 0, 0, 0, time.UTC)
	state := nextModelHealthAfterRequest(ModelHealthState{
		State: "healthy", ConsecutiveRequestFailures: 2, AutomaticCheckFailures: 4,
	}, healthFailureNone, "text", http.StatusOK, now)
	if state.State != "healthy" || state.ConsecutiveRequestFailures != 0 || state.AutomaticCheckFailures != 4 || !state.NextCheckAt.IsZero() {
		t.Fatalf("真实请求成功状态=%+v", state)
	}
}
