package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type routeRepositoryStub struct {
	RoutingRepository
	selection    RouteSelection
	resolveError error
	started      []RouteDecision
	completed    []RouteDecisionResult
	observations []RouteObservation
}

func (s *routeRepositoryStub) ResolveRoute(context.Context, string, string) (RouteSelection, error) {
	return s.selection, s.resolveError
}

func (s *routeRepositoryStub) StartRouteDecision(_ context.Context, decision RouteDecision) error {
	s.started = append(s.started, decision)
	return nil
}

func (s *routeRepositoryStub) CompleteRouteDecision(_ context.Context, result RouteDecisionResult) error {
	s.completed = append(s.completed, result)
	return nil
}

func (s *routeRepositoryStub) RecordRouteObservation(_ context.Context, observation RouteObservation) error {
	s.observations = append(s.observations, observation)
	return nil
}

func TestModelProxyRetriesBeforeFirstByteAndPreservesStream(t *testing.T) {
	aliases := make([]string, 0)
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-user-route" || r.URL.Path != "/v1/responses" {
			t.Fatalf("模型代理请求=%s %s，鉴权=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		alias := body["model"].(string)
		aliases = append(aliases, alias)
		if alias == "mkt-ov-1" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"message":"first failed"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Upstream-Request-Id", "req-upstream")
		_, _ = w.Write([]byte("data: first\n\ndata: [DONE]\n\n"))
	}))
	box := testCredentialBox(t)
	repository := &routeRepositoryStub{selection: RouteSelection{
		TokenBindingID: 81, PolicyID: 91, Mode: "smart", Priority: "price", Protocol: "responses",
		Candidates: []RouteCandidate{
			{OfferID: 1, MerchantChannelID: 51, CanonicalModelID: 61, InternalAlias: "mkt-ov-1", MultiplierBPS: 3000, HealthState: "healthy"},
			{OfferID: 2, MerchantChannelID: 52, CanonicalModelID: 61, InternalAlias: "mkt-ov-2", MultiplierBPS: 4000, HealthState: "healthy"},
		},
	}}
	services.Routing = repository
	services.Credentials = box
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.5","input":"hello","stream":true}`))
	request.Header.Set("Authorization", "Bearer sk-user-route")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "public-request-1")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("X-Upstream-Request-Id") != "req-upstream" || response.Body.String() != "data: first\n\ndata: [DONE]\n\n" {
		t.Fatalf("代理响应=%d %v %q", response.Code, response.Header(), response.Body.String())
	}
	if strings.Join(aliases, ",") != "mkt-ov-1,mkt-ov-2" {
		t.Fatalf("候选尝试顺序=%v", aliases)
	}
	if len(repository.started) != 1 || len(repository.completed) != 1 || repository.completed[0].OfferID != 2 || repository.completed[0].Attempts != 2 {
		t.Fatalf("路由决策记录不完整：started=%+v completed=%+v", repository.started, repository.completed)
	}
	if len(repository.observations) != 2 || repository.observations[0].Outcome != "temporary_failure" || repository.observations[1].Outcome != "success" {
		t.Fatalf("被动健康记录=%+v", repository.observations)
	}
}

func TestModelProxyReturnsOpenAI503WithoutUpstreamWhenNoCandidate(t *testing.T) {
	upstreamCalls := 0
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { upstreamCalls++ }))
	services.Credentials = testCredentialBox(t)
	services.Routing = &routeRepositoryStub{selection: RouteSelection{TokenBindingID: 81, PolicyID: 91, Mode: "fixed", Priority: "price", Protocol: "chat_completions"}}
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-5.5","messages":[]}`))
	request.Header.Set("Authorization", "Bearer sk-user-route")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || upstreamCalls != 0 || !strings.Contains(response.Body.String(), `"code":"no_available_channel"`) {
		t.Fatalf("无候选响应=%d %s，上游调用=%d", response.Code, response.Body.String(), upstreamCalls)
	}
}
