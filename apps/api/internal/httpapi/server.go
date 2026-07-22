package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
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
}

func NewServer(version string, dependencies Dependencies) http.Handler {
	s := &server{version: version, dependencies: dependencies}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health/live", s.live)
	mux.HandleFunc("GET /api/health/ready", s.ready)
	mux.HandleFunc("GET /api/health", s.ready)
	mux.HandleFunc("GET /api/version", s.versionInfo)
	mux.HandleFunc("POST /v1/chat/completions", modelRouteDisabled)
	mux.HandleFunc("POST /v1/responses", modelRouteDisabled)
	mux.HandleFunc("POST /v1/embeddings", modelRouteDisabled)
	return requestID(mux)
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
