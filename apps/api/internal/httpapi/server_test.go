package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWriteUpstreamErrorHidesInternalTransportDetails(t *testing.T) {
	response := httptest.NewRecorder()
	writeUpstreamError(response, errors.New(`请求 New API：Get "http://new-api:3000/api/user/self": connection refused`), http.StatusBadGateway)
	if response.Code != http.StatusBadGateway {
		t.Fatalf("状态码=%d", response.Code)
	}
	if strings.Contains(response.Body.String(), "new-api:3000") || !strings.Contains(response.Body.String(), "内部服务连接异常") {
		t.Fatalf("响应未脱敏：%s", response.Body.String())
	}
}

type stubDependency struct {
	err error
}

func (s stubDependency) Ping(context.Context) error { return s.err }

type countingDependency struct {
	calls atomic.Int32
}

func (s *countingDependency) Ping(context.Context) error {
	s.calls.Add(1)
	return nil
}

func TestLiveDoesNotRequireDependencies(t *testing.T) {
	handler := NewServer("test-version", Dependencies{
		Database: stubDependency{err: errors.New("database down")},
		Redis:    stubDependency{err: errors.New("redis down")},
		NewAPI:   stubDependency{err: errors.New("new api down")},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/health/live", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", response.Code)
	}
}

func TestReadyReportsFailedDependency(t *testing.T) {
	handler := NewServer("test-version", Dependencies{
		Database: stubDependency{},
		Redis:    stubDependency{err: errors.New("redis down")},
		NewAPI:   stubDependency{},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/health/ready", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d", response.Code)
	}
	var body struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("解析响应失败：%v", err)
	}
	if body.Success || body.Error.Code != "dependency_unavailable" {
		t.Fatalf("响应内容不符合预期：%+v", body)
	}
}

func TestReadyCoalescesConcurrentDependencyChecks(t *testing.T) {
	database := &countingDependency{}
	redis := &countingDependency{}
	newAPI := &countingDependency{}
	handler := NewServer("test-version", Dependencies{
		Database: database,
		Redis:    redis,
		NewAPI:   newAPI,
	})

	const requests = 50
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(requests)
	for range requests {
		go func() {
			defer wait.Done()
			<-start
			request := httptest.NewRequest(http.MethodGet, "/api/health/ready", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Errorf("状态码 = %d", response.Code)
			}
		}()
	}
	close(start)
	wait.Wait()

	if database.calls.Load() != 1 || redis.calls.Load() != 1 || newAPI.calls.Load() != 1 {
		t.Fatalf("依赖探测次数 database=%d redis=%d new_api=%d", database.calls.Load(), redis.calls.Load(), newAPI.calls.Load())
	}
}

func TestModelRouteIsClosedDuringFoundationPhase(t *testing.T) {
	handler := NewServer("test-version", Dependencies{})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d", response.Code)
	}
	var body struct {
		Error struct {
			Type string `json:"type"`
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("解析响应失败：%v", err)
	}
	if body.Error.Type != "server_error" || body.Error.Code != "service_not_ready" {
		t.Fatalf("OpenAI 错误格式不符合预期：%+v", body)
	}
}
