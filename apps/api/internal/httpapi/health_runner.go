package httpapi

import (
	"context"
	"errors"
	"time"
)

type HealthProbeResult struct {
	Probe       HealthProbe
	State       ModelHealthState
	Class       healthFailureClass
	StatusCode  int
	FirstByteMS int
	Summary     string
}

type HealthRepository interface {
	ClaimHealthProbes(context.Context, time.Time, int) ([]HealthProbe, error)
	SaveProbeResult(context.Context, HealthProbeResult) error
}

type HealthRunner struct {
	repository HealthRepository
	tester     ExistingChannelTester
}

func NewHealthRunner(repository HealthRepository, tester ExistingChannelTester) *HealthRunner {
	return &HealthRunner{repository: repository, tester: tester}
}

func (r *HealthRunner) RunCycle(ctx context.Context, now time.Time) error {
	probes, err := r.repository.ClaimHealthProbes(ctx, now, 20)
	if err != nil {
		return err
	}
	var cycleErrors []error
	for _, probe := range probes {
		result := r.executeProbe(ctx, probe, now)
		if err := r.repository.SaveProbeResult(ctx, result); err != nil {
			cycleErrors = append(cycleErrors, err)
		}
	}
	return errors.Join(cycleErrors...)
}

func (r *HealthRunner) executeProbe(ctx context.Context, probe HealthProbe, now time.Time) HealthProbeResult {
	outcome, err := r.tester.TestExisting(ctx, probe.NewAPIChannelID, probe.Model)
	class := healthFailureNone
	summary := ""
	if err != nil {
		class = healthFailureTemporary
		summary = sanitizedChannelError(err, "")
	}
	current := ModelHealthState{
		State: probe.State, ConsecutiveRequestFailures: probe.ConsecutiveFailures,
		AutomaticCheckFailures: probe.AutomaticCheckFailures, NextCheckAt: probe.NextProbeAt,
	}
	return HealthProbeResult{
		Probe: probe, State: nextModelHealthAfterAutomaticCheck(current, class, now),
		Class: class, FirstByteMS: outcome.LatencyMS, Summary: summary,
	}
}
