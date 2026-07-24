package httpapi

import (
	"context"
	"errors"
	"testing"
	"time"
)

type healthRepositoryStub struct {
	probes []HealthProbe
	result HealthProbeResult
}

func (s *healthRepositoryStub) ClaimHealthProbes(context.Context, time.Time, int) ([]HealthProbe, error) {
	return s.probes, nil
}

func (s *healthRepositoryStub) SaveProbeResult(_ context.Context, result HealthProbeResult) error {
	s.result = result
	return nil
}

type existingChannelTesterStub struct {
	channelID int
	model     CanonicalModel
	outcome   ChannelTestOutcome
	err       error
}

func (s *existingChannelTesterStub) TestExisting(_ context.Context, channelID int, model CanonicalModel) (ChannelTestOutcome, error) {
	s.channelID = channelID
	s.model = model
	return s.outcome, s.err
}

func TestHealthRunnerChecksDueModelThroughFormalChannel(t *testing.T) {
	repository := &healthRepositoryStub{probes: []HealthProbe{{
		BindingID: 91, MerchantChannelID: 51, CanonicalModelID: 61, NewAPIChannelID: 41,
		Model: CanonicalModel{ID: 61, PublicName: "gpt-5.5", Price: OfficialPrice{PriceKind: "text"}},
		State: "cooling", AutomaticCheckFailures: 4,
	}}}
	tester := &existingChannelTesterStub{outcome: ChannelTestOutcome{LatencyMS: 35}}
	runner := NewHealthRunner(repository, tester)
	now := time.Date(2026, 7, 25, 4, 0, 0, 0, time.UTC)
	if err := runner.RunCycle(context.Background(), now); err != nil {
		t.Fatalf("执行自动检查失败：%v", err)
	}
	if tester.channelID != 41 || tester.model.PublicName != "gpt-5.5" {
		t.Fatalf("正式渠道检查参数=%d %+v", tester.channelID, tester.model)
	}
	if repository.result.Class != healthFailureNone || repository.result.State.State != "healthy" || repository.result.FirstByteMS != 35 {
		t.Fatalf("自动检查结果=%+v", repository.result)
	}
}

func TestHealthRunnerKeepsThreeMinuteCooldownAfterFailedAutomaticCheck(t *testing.T) {
	now := time.Date(2026, 7, 25, 4, 0, 0, 0, time.UTC)
	repository := &healthRepositoryStub{probes: []HealthProbe{{
		BindingID: 91, MerchantChannelID: 51, CanonicalModelID: 61, NewAPIChannelID: 41,
		Model: CanonicalModel{ID: 61, PublicName: "gpt-5.5", Price: OfficialPrice{PriceKind: "text"}},
		State: "cooling", AutomaticCheckFailures: 8,
	}}}
	runner := NewHealthRunner(repository, &existingChannelTesterStub{outcome: ChannelTestOutcome{LatencyMS: 50}, err: errors.New("上游失败")})
	if err := runner.RunCycle(context.Background(), now); err != nil {
		t.Fatalf("自动检查周期失败：%v", err)
	}
	if repository.result.Class != healthFailureTemporary || repository.result.State.AutomaticCheckFailures != 9 || !repository.result.State.NextCheckAt.Equal(now.Add(3*time.Minute)) {
		t.Fatalf("自动检查失败结果=%+v", repository.result)
	}
}
