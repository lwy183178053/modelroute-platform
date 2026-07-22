package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
	"github.com/redis/go-redis/v9"
)

type auditRecorder struct {
	mu      sync.Mutex
	entries []AuditEntry
}

func (a *auditRecorder) Record(_ context.Context, entry AuditEntry) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return nil
}

func (a *auditRecorder) List(_ context.Context, page int, pageSize int) ([]AuditEntry, int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	start := (page - 1) * pageSize
	if start >= len(a.entries) {
		return []AuditEntry{}, len(a.entries), nil
	}
	end := start + pageSize
	if end > len(a.entries) {
		end = len(a.entries)
	}
	return append([]AuditEntry(nil), a.entries[start:end]...), len(a.entries), nil
}

func newHTTPTestServices(t *testing.T, upstream http.Handler) (Services, *platformauth.Store, *auditRecorder) {
	t.Helper()
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	upstreamServer := httptest.NewServer(upstream)
	t.Cleanup(upstreamServer.Close)
	store := platformauth.NewStore(redisClient, time.Hour)
	audit := &auditRecorder{}
	return Services{
		Sessions: store,
		NewAPI:   newapi.NewClient(upstreamServer.URL, time.Second),
		Audit:    audit,
		Cookie: CookieConfig{
			Name:         "modelroute_session",
			Secure:       true,
			PublicOrigin: "https://aggregate.lwylink.xyz",
			MaxAge:       time.Hour,
		},
	}, store, audit
}

func TestLoginReturnsPlatformCookieAndHidesUpstreamCookie(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/login" {
			t.Fatalf("上游路径 = %s", r.URL.Path)
		}
		if r.Header.Get("X-Forwarded-For") != "203.0.113.9" {
			t.Fatalf("上游来源地址 = %q", r.Header.Get("X-Forwarded-For"))
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
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"alice","password":"secret123"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.2")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "upstream-secret") || strings.Contains(response.Header().Get("Set-Cookie"), "upstream-secret") {
		t.Fatalf("响应泄露上游 Cookie：%s %s", response.Header().Get("Set-Cookie"), response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("平台 Cookie 数量 = %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "modelroute_session" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("平台 Cookie 属性不符合预期：%+v", cookie)
	}
}

func TestClientIPRejectsInvalidForwardedValue(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("X-Forwarded-For", "not-an-ip")
	request.RemoteAddr = "192.0.2.8:43120"

	if got := clientIP(request); got != "192.0.2.8" {
		t.Fatalf("客户端地址 = %q", got)
	}
}

func TestRegularUserCannotCallAdminUsers(t *testing.T) {
	upstreamCalls := 0
	services, store, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamCalls++
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": []any{}})
	}))
	token, err := store.Create(context.Background(), platformauth.Session{
		UserID: 7, Username: "alice", Role: 1, Status: 1, UpstreamCookie: "session=user",
	})
	if err != nil {
		t.Fatalf("创建会话失败：%v", err)
	}
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("状态码 = %d", response.Code)
	}
	if upstreamCalls != 0 {
		t.Fatalf("越权请求触发了 %d 次上游调用", upstreamCalls)
	}
}

func TestAdminDisableClearsTargetSessionsAndWritesAudit(t *testing.T) {
	services, store, audit := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/manage" || r.Method != http.MethodPost {
			t.Fatalf("上游请求 = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("New-Api-User") != "1" || r.Header.Get("Cookie") != "session=root" {
			t.Fatalf("上游鉴权头不符合预期")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("解析上游请求失败：%v", err)
		}
		if body["id"] != float64(7) || body["action"] != "disable" {
			t.Fatalf("上游管理参数 = %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": nil})
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
	request := httptest.NewRequest(http.MethodPost, "/api/admin/users/7/status", bytes.NewBufferString(`{"enabled":false}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: adminToken})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，响应=%s", response.Code, response.Body.String())
	}
	if _, err := store.Get(context.Background(), targetToken); !errors.Is(err, platformauth.ErrSessionNotFound) {
		t.Fatalf("目标用户会话仍有效：%v", err)
	}
	if len(audit.entries) != 1 || audit.entries[0].ActorUserID != 1 || audit.entries[0].ObjectID != "7" || audit.entries[0].Action != "user.disable" {
		t.Fatalf("审计记录不符合预期：%+v", audit.entries)
	}
}

func TestAuthenticatedWriteRequiresSameOrigin(t *testing.T) {
	upstreamCalls := 0
	services, store, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamCalls++
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	adminToken, err := store.Create(context.Background(), platformauth.Session{
		UserID: 1, Username: "modelroute", Role: 100, Status: 1, UpstreamCookie: "session=root",
	})
	if err != nil {
		t.Fatalf("创建管理员会话失败：%v", err)
	}
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/users/7/status", bytes.NewBufferString(`{"enabled":false}`))
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: adminToken})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("状态码 = %d", response.Code)
	}
	if upstreamCalls != 0 {
		t.Fatalf("跨站请求触发了 %d 次上游调用", upstreamCalls)
	}
}

func TestPublicAuthEndpointsTranslateToNewAPI(t *testing.T) {
	var requests []string
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("解析上游请求失败：%v", err)
			}
		}
		data := any(nil)
		if r.URL.Path == "/api/user/reset" {
			data = "new-password"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
	}))
	handler := NewServerWithServices("test", Dependencies{}, services)
	tests := []struct {
		path string
		body string
	}{
		{path: "/api/auth/register", body: `{"username":"alice","password":"secret123","email":"alice@example.com","verification_code":"123456"}`},
		{path: "/api/auth/email-verification", body: `{"email":"alice@example.com"}`},
		{path: "/api/auth/password-reset/request", body: `{"email":"alice@example.com"}`},
		{path: "/api/auth/password-reset/confirm", body: `{"email":"alice@example.com","token":"reset-token"}`},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodPost, test.path, bytes.NewBufferString(test.body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s 状态码=%d，响应=%s", test.path, response.Code, response.Body.String())
		}
	}
	want := []string{
		"POST /api/user/register",
		"GET /api/verification?email=alice%40example.com",
		"GET /api/reset_password?email=alice%40example.com",
		"POST /api/user/reset",
	}
	if strings.Join(requests, "|") != strings.Join(want, "|") {
		t.Fatalf("上游请求=%v", requests)
	}
}

func TestTwoFactorLoginPromotesSessionAndMeRefreshesUser(t *testing.T) {
	services, store, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "pending-upstream", HttpOnly: true})
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data":    map[string]any{"require_2fa": true},
			})
		case "/api/user/login/2fa":
			if r.Header.Get("Cookie") != "session=pending-upstream" {
				t.Fatalf("待验证 Cookie=%q", r.Header.Get("Cookie"))
			}
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "ready-upstream", HttpOnly: true})
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"id": 7, "username": "alice", "display_name": "Alice",
					"role": 1, "status": 1, "group": "default",
				},
			})
		case "/api/user/self":
			if r.Header.Get("Cookie") != "session=ready-upstream" || r.Header.Get("New-Api-User") != "7" {
				t.Fatalf("用户鉴权头不符合预期")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"id": 7, "username": "alice", "display_name": "Alice",
					"role": 1, "status": 1, "group": "default", "quota": 500000,
				},
			})
		default:
			t.Fatalf("未知上游路径：%s", r.URL.Path)
		}
	}))
	handler := NewServerWithServices("test", Dependencies{}, services)
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"alice","password":"secret123"}`))
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("登录状态码=%d", loginResponse.Code)
	}
	platformCookie := loginResponse.Result().Cookies()[0]
	pending, err := store.Get(context.Background(), platformCookie.Value)
	if err != nil || !pending.PendingTwoFactor {
		t.Fatalf("待验证会话=%+v，错误=%v", pending, err)
	}

	verifyRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login/2fa", bytes.NewBufferString(`{"code":"123456"}`))
	verifyRequest.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	verifyRequest.AddCookie(platformCookie)
	verifyResponse := httptest.NewRecorder()
	handler.ServeHTTP(verifyResponse, verifyRequest)
	if verifyResponse.Code != http.StatusOK {
		t.Fatalf("二次验证状态码=%d，响应=%s", verifyResponse.Code, verifyResponse.Body.String())
	}
	ready, err := store.Get(context.Background(), platformCookie.Value)
	if err != nil || ready.PendingTwoFactor || ready.UserID != 7 {
		t.Fatalf("完成后的会话=%+v，错误=%v", ready, err)
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meRequest.AddCookie(platformCookie)
	meResponse := httptest.NewRecorder()
	handler.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK || !strings.Contains(meResponse.Body.String(), `"quota":500000`) {
		t.Fatalf("me 响应=%d %s", meResponse.Code, meResponse.Body.String())
	}
}

func TestLogoutDeletesPlatformSession(t *testing.T) {
	services, store, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/logout" || r.Method != http.MethodGet {
			t.Fatalf("上游请求=%s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": nil})
	}))
	token, err := store.Create(context.Background(), platformauth.Session{
		UserID: 7, Username: "alice", Role: 1, Status: 1, UpstreamCookie: "session=user",
	})
	if err != nil {
		t.Fatalf("创建会话失败：%v", err)
	}
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("状态码=%d", response.Code)
	}
	if _, err := store.Get(context.Background(), token); !errors.Is(err, platformauth.ErrSessionNotFound) {
		t.Fatalf("退出后会话仍有效：%v", err)
	}
}
