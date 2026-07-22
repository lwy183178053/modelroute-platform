package newapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthenticateCapturesUpstreamCookieWithoutReturningItAsUserData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/login" {
			t.Fatalf("请求路径 = %s", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("解析请求失败：%v", err)
		}
		if body["username"] != "alice" || body["password"] != "secret123" {
			t.Fatalf("登录参数不符合预期：%v", body)
		}
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "upstream-secret", HttpOnly: true})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"id": 7, "username": "alice", "display_name": "Alice",
				"role": 1, "status": 1, "group": "default",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	result, err := client.Authenticate(context.Background(), "/api/user/login", map[string]string{
		"username": "alice",
		"password": "secret123",
	}, "")
	if err != nil {
		t.Fatalf("登录失败：%v", err)
	}
	if result.Cookie != "session=upstream-secret" {
		t.Fatalf("上游 Cookie = %q", result.Cookie)
	}
	if result.User.ID != 7 || result.User.Role != 1 || result.User.Username != "alice" {
		t.Fatalf("用户信息不符合预期：%+v", result.User)
	}
}

func TestDoSendsSessionCookieAndNewAPIUserHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Cookie"); got != "session=upstream-secret" {
			t.Fatalf("Cookie = %q", got)
		}
		if got := r.Header.Get("New-Api-User"); got != "7" {
			t.Fatalf("New-Api-User = %q", got)
		}
		if got := r.Header.Get("X-Forwarded-For"); got != "203.0.113.9" {
			t.Fatalf("X-Forwarded-For = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"id": 7},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	ctx := WithClientIP(context.Background(), "203.0.113.9")
	data, err := client.Do(ctx, http.MethodGet, "/api/user/self", nil, &Auth{
		Cookie: "session=upstream-secret",
		UserID: 7,
	})
	if err != nil {
		t.Fatalf("请求失败：%v", err)
	}
	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		t.Fatalf("解析用户失败：%v", err)
	}
	if user.ID != 7 {
		t.Fatalf("用户 ID = %d", user.ID)
	}
}

func TestDoReturnsTypedUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"message": "账号已封禁",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	_, err := client.Do(context.Background(), http.MethodGet, "/api/user/self", nil, nil)
	upstreamError, ok := err.(*APIError)
	if !ok || upstreamError.Message != "账号已封禁" {
		t.Fatalf("错误 = %#v", err)
	}
}

func TestProxyPreservesAuthorizationStatusHeadersAndBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" || r.Header.Get("Authorization") != "Bearer sk-user" {
			t.Fatalf("代理请求不符合预期：%s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte(`"model":"mkt-ov-9"`)) {
			t.Fatalf("代理请求正文 = %s", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Upstream-Request-Id", "upstream-1")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("data: first\n\ndata: second\n\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	response, err := client.Proxy(context.Background(), http.MethodPost, "/v1/responses", http.Header{
		"Authorization": []string{"Bearer sk-user"},
		"Content-Type":  []string{"application/json"},
	}, []byte(`{"model":"mkt-ov-9"}`))
	if err != nil {
		t.Fatalf("原始代理失败：%v", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusAccepted || response.Header.Get("X-Upstream-Request-Id") != "upstream-1" {
		t.Fatalf("代理响应元数据不符合预期：%d %v", response.StatusCode, response.Header)
	}
	if string(body) != "data: first\n\ndata: second\n\n" {
		t.Fatalf("代理响应正文 = %q", body)
	}
}
