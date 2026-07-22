package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubDependency struct {
	err error
}

func (s stubDependency) Ping(context.Context) error { return s.err }

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
