package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

type HealthProbeResult struct {
	Probe       HealthProbe
	State       HealthProbe
	Class       healthFailureClass
	StatusCode  int
	FirstByteMS int
	Summary     string
}

type HealthRepository interface {
	ClaimHealthProbes(context.Context, time.Time, int) ([]HealthProbe, error)
	SaveProbeResult(context.Context, HealthProbeResult) error
	RegisterProbeToken(context.Context, int) error
}

type ProbeTokenSource interface {
	ProbeToken(context.Context) (newapi.ProbeToken, error)
}

type RawModelClient interface {
	Proxy(context.Context, string, string, http.Header, []byte) (*http.Response, error)
}

type HealthRunner struct {
	repository HealthRepository
	tokens     ProbeTokenSource
	client     RawModelClient
	admin      AdminNewAPI
}

func NewHealthRunner(repository HealthRepository, tokens ProbeTokenSource, client RawModelClient, admin AdminNewAPI) *HealthRunner {
	return &HealthRunner{repository: repository, tokens: tokens, client: client, admin: admin}
}

func (r *HealthRunner) RunCycle(ctx context.Context, now time.Time) error {
	token, err := r.tokens.ProbeToken(ctx)
	if err != nil {
		return err
	}
	if err := r.repository.RegisterProbeToken(ctx, token.ID); err != nil {
		return err
	}
	probes, err := r.repository.ClaimHealthProbes(ctx, now, 20)
	if err != nil {
		return err
	}
	var cycleErrors []error
	for _, probe := range probes {
		result := r.executeProbe(ctx, probe, token.Key, now)
		if result.Class == healthFailureHard && r.admin != nil {
			if _, err := r.admin.Do(ctx, http.MethodPost, fmt.Sprintf("/api/channel/%d/status", probe.NewAPIChannelID), map[string]int{"status": 2}); err != nil {
				cycleErrors = append(cycleErrors, fmt.Errorf("关闭 New API 渠道 %d：%w", probe.NewAPIChannelID, err))
			}
		}
		if err := r.repository.SaveProbeResult(ctx, result); err != nil {
			cycleErrors = append(cycleErrors, err)
		}
	}
	return errors.Join(cycleErrors...)
}

func (r *HealthRunner) executeProbe(ctx context.Context, probe HealthProbe, key string, now time.Time) HealthProbeResult {
	path, body := probeRequest(probe)
	started := time.Now()
	response, err := r.client.Proxy(ctx, http.MethodPost, path, http.Header{
		"Authorization": []string{"Bearer " + key},
		"Content-Type":  []string{"application/json"},
	}, body)
	if err != nil {
		state := nextProbeState(probe, healthFailureTemporary, now, 0)
		return HealthProbeResult{Probe: probe, State: state, Class: healthFailureTemporary, FirstByteMS: int(time.Since(started).Milliseconds()), Summary: "New API 连接失败"}
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	class := classifyProbeStatus(response.StatusCode)
	retryAfter := parseRetryAfter(response.Header.Get("Retry-After"), now)
	state := nextProbeState(probe, class, now, retryAfter)
	return HealthProbeResult{
		Probe: probe, State: state, Class: class, StatusCode: response.StatusCode,
		FirstByteMS: int(time.Since(started).Milliseconds()),
	}
}

func probeRequest(probe HealthProbe) (string, []byte) {
	model := strings.ReplaceAll(probe.InternalAlias, `"`, "")
	switch probe.Protocol {
	case "chat_completions":
		return "/v1/chat/completions", []byte(fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"ping"}],"max_tokens":1,"stream":false}`, model))
	case "embeddings":
		return "/v1/embeddings", []byte(fmt.Sprintf(`{"model":"%s","input":"ping"}`, model))
	default:
		return "/v1/responses", []byte(fmt.Sprintf(`{"model":"%s","input":"ping","max_output_tokens":1}`, model))
	}
}

func parseRetryAfter(raw string, now time.Time) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if deadline, err := http.ParseTime(raw); err == nil && deadline.After(now) {
		return deadline.Sub(now)
	}
	return 0
}
