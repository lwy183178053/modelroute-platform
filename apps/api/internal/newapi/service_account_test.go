package newapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServiceAccountAuthenticatesAndReusesServerSession(t *testing.T) {
	loginCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
			loginCount++
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "root-cookie"})
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"id": 1, "role": 100, "status": 1}})
		case "/api/option/":
			if r.Header.Get("Cookie") != "session=root-cookie" || r.Header.Get("New-Api-User") != "1" {
				t.Fatalf("服务账号鉴权头不符合预期")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	account := NewServiceAccount(NewClient(server.URL, time.Second), "modelroute", "secret")
	for range 2 {
		if _, err := account.Do(context.Background(), http.MethodGet, "/api/option/", nil); err != nil {
			t.Fatalf("服务账号请求失败：%v", err)
		}
	}
	if loginCount != 1 {
		t.Fatalf("登录次数 = %d", loginCount)
	}
}

func TestServiceAccountFindsAndRevealsDedicatedProbeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "root-cookie"})
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"id": 1, "role": 100, "status": 1}})
		case "/api/token/":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"items": []map[string]any{{"id": 77, "name": probeTokenName}}, "total": 1}})
		case "/api/token/77/key":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"key": "sk-probe-secret"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	account := NewServiceAccount(NewClient(server.URL, time.Second), "modelroute", "secret")
	token, err := account.ProbeToken(context.Background())
	if err != nil {
		t.Fatalf("获取探活令牌失败：%v", err)
	}
	if token.ID != 77 || token.Key != "sk-probe-secret" {
		t.Fatalf("探活令牌=%+v", token)
	}
}
