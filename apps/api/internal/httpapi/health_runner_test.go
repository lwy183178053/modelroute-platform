package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

type healthRepositoryStub struct {
	probes       []HealthProbe
	result       HealthProbeResult
	probeTokenID int
}

func (s *healthRepositoryStub) ClaimHealthProbes(context.Context, time.Time, int) ([]HealthProbe, error) {
	return s.probes, nil
}

func (s *healthRepositoryStub) SaveProbeResult(_ context.Context, result HealthProbeResult) error {
	s.result = result
	return nil
}

func (s *healthRepositoryStub) RegisterProbeToken(_ context.Context, tokenID int) error {
	s.probeTokenID = tokenID
	return nil
}

type probeTokenSourceStub struct{ token newapi.ProbeToken }

func (s probeTokenSourceStub) ProbeToken(context.Context) (newapi.ProbeToken, error) {
	return s.token, nil
}

func TestHealthRunnerMovesThirdTemporaryFailureIntoCooldown(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer sk-probe" {
			t.Fatalf("探活请求=%s，鉴权=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		messages, _ := body["messages"].([]any)
		if body["model"] != "mkt-ov-42" || len(messages) != 1 {
			t.Fatalf("探活正文=%v", body)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"temporary"}}`))
	}))
	defer upstream.Close()
	repository := &healthRepositoryStub{probes: []HealthProbe{{
		OfferChannelAliasID: 81, OfferID: 42, MerchantChannelID: 51, CanonicalModelID: 61, NewAPIChannelID: 41,
		InternalAlias: "mkt-ov-42", PriceKind: "text", State: "healthy", ConsecutiveFailures: 2,
	}}}
	runner := NewHealthRunner(repository, probeTokenSourceStub{token: newapi.ProbeToken{ID: 77, Key: "sk-probe"}}, newapi.NewClient(upstream.URL, time.Second), &adminNewAPIStub{})
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	if err := runner.RunCycle(context.Background(), now); err != nil {
		t.Fatalf("执行探活失败：%v", err)
	}
	if repository.probeTokenID != 77 || repository.result.State.State != "cooling" || repository.result.Class != healthFailureTemporary || repository.result.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("探活结果=%+v，探活令牌=%d", repository.result, repository.probeTokenID)
	}
}
