package main

import (
	"context"
	"testing"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/bootstrap"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/httpapi"
)

type cycleRunnerStub struct {
	calls int
}

func (s *cycleRunnerStub) RunCycle(context.Context, time.Time) error {
	s.calls++
	return nil
}

func TestRunRouteMetricCycleInvokesConfiguredRunner(t *testing.T) {
	runner := &cycleRunnerStub{}
	resources := &bootstrap.Resources{Services: httpapi.Services{RouteMetrics: runner}}
	runRouteMetricCycle(context.Background(), resources)
	if runner.calls != 1 {
		t.Fatalf("缓存指标刷新次数=%d", runner.calls)
	}
}
