package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

type Pinger interface {
	Ping(context.Context) error
}

type Dependencies struct {
	Database Pinger
	Redis    Pinger
	NewAPI   Pinger
}

type server struct {
	version      string
	dependencies Dependencies
	services     Services
}

func NewServer(version string, dependencies Dependencies) http.Handler {
	return NewServerWithServices(version, dependencies, Services{})
}

func NewServerWithServices(version string, dependencies Dependencies, services Services) http.Handler {
	s := &server{version: version, dependencies: dependencies, services: services}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health/live", s.live)
	mux.HandleFunc("GET /api/health/ready", s.ready)
	mux.HandleFunc("GET /api/health", s.ready)
	mux.HandleFunc("GET /api/version", s.versionInfo)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/login/2fa", s.loginTwoFactor)
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/email-verification", s.sendEmailVerification)
	mux.HandleFunc("POST /api/auth/password-reset/request", s.requestPasswordReset)
	mux.HandleFunc("POST /api/auth/password-reset/confirm", s.confirmPasswordReset)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/me", s.me)
	mux.HandleFunc("GET /api/keys", s.keys)
	mux.HandleFunc("POST /api/keys", s.createKey)
	mux.HandleFunc("PUT /api/keys/{id}", s.updateKey)
	mux.HandleFunc("POST /api/keys/{id}/status", s.updateKeyStatus)
	mux.HandleFunc("POST /api/keys/{id}/reveal", s.revealKey)
	mux.HandleFunc("DELETE /api/keys/{id}", s.deleteKey)
	mux.HandleFunc("GET /api/admin/users", s.adminUsers)
	mux.HandleFunc("GET /api/admin/users/{id}", s.adminUser)
	mux.HandleFunc("POST /api/admin/users/{id}/status", s.adminUserStatus)
	mux.HandleFunc("POST /api/admin/users/{id}/quota", s.adminUserQuota)
	mux.HandleFunc("POST /api/admin/users/{id}/password", s.adminUserPassword)
	mux.HandleFunc("GET /api/admin/audit-logs", s.adminAuditLogs)
	mux.HandleFunc("POST /v1/chat/completions", modelRouteDisabled)
	mux.HandleFunc("POST /v1/responses", modelRouteDisabled)
	mux.HandleFunc("POST /v1/embeddings", modelRouteDisabled)
	return requestID(clientMetadata(mux))
}

func clientMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := newapi.WithClientIP(r.Context(), clientIP(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *server) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]string{
			"service": "modelroute-platform",
			"status":  "ok",
			"version": s.version,
		},
	})
}

func (s *server) ready(w http.ResponseWriter, r *http.Request) {
	checks := []struct {
		name   string
		pinger Pinger
	}{
		{name: "database", pinger: s.dependencies.Database},
		{name: "redis", pinger: s.dependencies.Redis},
		{name: "new_api", pinger: s.dependencies.NewAPI},
	}

	for _, check := range checks {
		if check.pinger == nil || check.pinger.Ping(r.Context()) != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"success": false,
				"error": map[string]string{
					"code":    "dependency_unavailable",
					"message": check.name + " 连接异常",
				},
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    map[string]string{"status": "ready"},
	})
}

func (s *server) versionInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    map[string]string{"version": s.version},
	})
}

func modelRouteDisabled(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]any{
		"error": map[string]any{
			"message": "模型路由正在部署中",
			"type":    "server_error",
			"param":   nil,
			"code":    "service_not_ready",
		},
	})
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			var value [12]byte
			if _, err := rand.Read(value[:]); err == nil {
				requestID = hex.EncodeToString(value[:])
			}
		}
		if requestID != "" {
			w.Header().Set("X-Request-ID", requestID)
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
