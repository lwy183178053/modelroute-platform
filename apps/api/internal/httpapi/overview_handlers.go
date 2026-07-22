package httpapi

import (
	"context"
	"net/http"
	"time"
)

type overviewDependencyStatus struct {
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
}

func (s *server) adminOverview(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok {
		return
	}
	if s.services.Overview == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "overview_unavailable", "管理员总览服务尚未就绪")
		return
	}
	now := time.Now().UTC()
	snapshot, err := s.services.Overview.Snapshot(r.Context(), now)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "overview_query_failed", "读取管理员总览失败")
		return
	}
	syncDelay := int64(0)
	if snapshot.LastSyncedThrough > 0 && now.Unix() > snapshot.LastSyncedThrough {
		syncDelay = now.Unix() - snapshot.LastSyncedThrough
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"generated_at":       now,
			"summary":            snapshot,
			"sync_delay_seconds": syncDelay,
			"services": map[string]overviewDependencyStatus{
				"database": dependencyStatus(r.Context(), s.dependencies.Database),
				"redis":    dependencyStatus(r.Context(), s.dependencies.Redis),
				"new_api":  dependencyStatus(r.Context(), s.dependencies.NewAPI),
			},
		},
	})
}

func dependencyStatus(ctx context.Context, dependency Pinger) overviewDependencyStatus {
	if dependency == nil {
		return overviewDependencyStatus{Status: "unavailable"}
	}
	started := time.Now()
	err := dependency.Ping(ctx)
	latency := time.Since(started).Milliseconds()
	if err != nil {
		return overviewDependencyStatus{Status: "unhealthy", LatencyMS: latency}
	}
	return overviewDependencyStatus{Status: "healthy", LatencyMS: latency}
}
