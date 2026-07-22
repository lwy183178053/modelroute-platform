package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

func TestAPIKeyLifecycleUsesAuthenticatedNewAPIEndpoints(t *testing.T) {
	var calls []string
	services, store, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("New-Api-User") != "7" || r.Header.Get("Cookie") != "session=user" {
			t.Fatalf("上游鉴权头不符合预期")
		}
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		data := any(nil)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/token/":
			data = map[string]any{"page": 1, "page_size": 20, "total": 1, "items": []any{map[string]any{"id": 9, "name": "主密钥", "key": "abcd**********wxyz"}}}
		case r.Method == http.MethodPost && r.URL.Path == "/api/token/":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("解析创建密钥请求失败：%v", err)
			}
			if body["name"] != "主密钥" || body["unlimited_quota"] != true {
				t.Fatalf("创建密钥参数=%v", body)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/api/token/" && r.URL.Query().Get("status_only") == "true":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["id"] != float64(9) || body["status"] != float64(2) {
				t.Fatalf("密钥状态参数=%v，错误=%v", body, err)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/api/token/":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["id"] != float64(9) || body["name"] != "新名称" {
				t.Fatalf("更新密钥参数=%v，错误=%v", body, err)
			}
			data = map[string]any{"id": 9, "name": "新名称"}
		case r.Method == http.MethodPost && r.URL.Path == "/api/token/9/key":
			data = map[string]any{"key": "sk-full-secret"}
		case r.Method == http.MethodDelete && r.URL.Path == "/api/token/9":
		default:
			t.Fatalf("未知上游请求：%s %s", r.Method, r.URL.RequestURI())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
	}))
	token, err := store.Create(context.Background(), platformauth.Session{
		UserID: 7, Username: "alice", Role: 1, Status: 1, UpstreamCookie: "session=user",
	})
	if err != nil {
		t.Fatalf("创建会话失败：%v", err)
	}
	handler := NewServerWithServices("test", Dependencies{}, services)
	tests := []struct {
		method string
		path   string
		body   string
		origin bool
	}{
		{method: http.MethodGet, path: "/api/keys?p=1&page_size=20"},
		{method: http.MethodPost, path: "/api/keys", body: `{"name":"主密钥","expired_time":-1,"unlimited_quota":true}`, origin: true},
		{method: http.MethodPut, path: "/api/keys/9", body: `{"name":"新名称","expired_time":-1,"unlimited_quota":true}`, origin: true},
		{method: http.MethodPost, path: "/api/keys/9/status", body: `{"enabled":false}`, origin: true},
		{method: http.MethodPost, path: "/api/keys/9/reveal", origin: true},
		{method: http.MethodDelete, path: "/api/keys/9", origin: true},
	}
	for _, test := range tests {
		request := httptest.NewRequest(test.method, test.path, bytes.NewBufferString(test.body))
		request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
		if test.origin {
			request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s %s 状态码=%d，响应=%s", test.method, test.path, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "session=user") {
			t.Fatalf("响应泄露上游会话")
		}
	}
	want := []string{
		"GET /api/token/?p=1&page_size=20",
		"POST /api/token/",
		"PUT /api/token/",
		"PUT /api/token/?status_only=true",
		"POST /api/token/9/key",
		"DELETE /api/token/9",
	}
	if strings.Join(calls, "|") != strings.Join(want, "|") {
		t.Fatalf("上游调用=%v", calls)
	}
}

func TestAdminQuotaAndPasswordOperationsUseNewAPIAndAudit(t *testing.T) {
	var calls []string
	services, store, audit := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		data := any(nil)
		switch r.Method + " " + r.URL.Path {
		case "POST /api/user/manage":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["id"] != float64(7) || body["action"] != "add_quota" || body["mode"] != "add" || body["value"] != float64(500000) {
				t.Fatalf("额度参数=%v，错误=%v", body, err)
			}
		case "GET /api/user/7":
			data = map[string]any{"id": 7, "username": "alice", "display_name": "Alice", "role": 1, "status": 1, "group": "default", "remark": "重要客户"}
		case "PUT /api/user/":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["id"] != float64(7) || body["username"] != "alice" || body["password"] != "new-secret123" || body["remark"] != "重要客户" {
				t.Fatalf("密码重置参数=%v，错误=%v", body, err)
			}
		default:
			t.Fatalf("未知上游请求：%s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
	}))
	adminToken, err := store.Create(context.Background(), platformauth.Session{
		UserID: 1, Username: "modelroute", Role: 100, Status: 1, UpstreamCookie: "session=root",
	})
	if err != nil {
		t.Fatalf("创建管理员会话失败：%v", err)
	}
	targetToken, err := store.Create(context.Background(), platformauth.Session{
		UserID: 7, Username: "alice", Role: 1, Status: 1, UpstreamCookie: "session=user",
	})
	if err != nil {
		t.Fatalf("创建目标会话失败：%v", err)
	}
	handler := NewServerWithServices("test", Dependencies{}, services)
	requests := []*http.Request{
		httptest.NewRequest(http.MethodPost, "/api/admin/users/7/quota", bytes.NewBufferString(`{"mode":"add","value":500000}`)),
		httptest.NewRequest(http.MethodPost, "/api/admin/users/7/password", bytes.NewBufferString(`{"password":"new-secret123"}`)),
	}
	for _, request := range requests {
		request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
		request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: adminToken})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s 状态码=%d，响应=%s", request.URL.Path, response.Code, response.Body.String())
		}
	}
	if _, err := store.Get(context.Background(), targetToken); !errors.Is(err, platformauth.ErrSessionNotFound) {
		t.Fatalf("密码变更后目标会话仍有效：%v", err)
	}
	if strings.Join(calls, "|") != "POST /api/user/manage|GET /api/user/7|PUT /api/user/" {
		t.Fatalf("上游调用=%v", calls)
	}
	if len(audit.entries) != 2 || audit.entries[0].Action != "user.quota.add" || audit.entries[1].Action != "user.password.reset" {
		t.Fatalf("审计记录=%+v", audit.entries)
	}
}

func TestAdminCanListPlatformAuditLogs(t *testing.T) {
	services, store, audit := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatalf("审计列表不应调用 New API")
	}))
	audit.entries = append(audit.entries, AuditEntry{
		ID: 3, ActorUserID: 1, ActorRole: "100", Action: "user.disable",
		ObjectType: "user", ObjectID: "7", Result: "success",
	})
	adminToken, err := store.Create(context.Background(), platformauth.Session{
		UserID: 1, Username: "modelroute", Role: 100, Status: 1, UpstreamCookie: "session=root",
	})
	if err != nil {
		t.Fatalf("创建管理员会话失败：%v", err)
	}
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs?p=1&page_size=20", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: adminToken})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"action":"user.disable"`) || !strings.Contains(response.Body.String(), `"actor_user_id":1`) {
		t.Fatalf("审计响应=%d %s", response.Code, response.Body.String())
	}
}
