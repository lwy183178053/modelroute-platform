package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

type keyRequest struct {
	Name               string  `json:"name"`
	ExpiredTime        int64   `json:"expired_time"`
	RemainQuota        int     `json:"remain_quota"`
	UnlimitedQuota     bool    `json:"unlimited_quota"`
	ModelLimitsEnabled bool    `json:"model_limits_enabled"`
	ModelLimits        string  `json:"model_limits"`
	AllowIPs           *string `json:"allow_ips"`
	Group              string  `json:"group"`
	CrossGroupRetry    bool    `json:"cross_group_retry"`
}

func (s *server) keys(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok {
		return
	}
	query := url.Values{}
	for _, key := range []string{"p", "page_size"} {
		if value := r.URL.Query().Get(key); value != "" {
			query.Set(key, value)
		}
	}
	path := "/api/token/"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodGet, path, nil, upstreamAuth(session))
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	writeRawData(w, data)
}

func (s *server) createKey(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	request, ok := decodeKeyRequest(w, r)
	if !ok {
		return
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/token/", request, upstreamAuth(session))
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	writeRawData(w, data)
}

func (s *server) updateKey(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	request, ok := decodeKeyRequest(w, r)
	if !ok {
		return
	}
	body := map[string]any{
		"id":                   id,
		"name":                 request.Name,
		"expired_time":         request.ExpiredTime,
		"remain_quota":         request.RemainQuota,
		"unlimited_quota":      request.UnlimitedQuota,
		"model_limits_enabled": request.ModelLimitsEnabled,
		"model_limits":         request.ModelLimits,
		"allow_ips":            request.AllowIPs,
		"group":                request.Group,
		"cross_group_retry":    request.CrossGroupRetry,
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodPut, "/api/token/", body, upstreamAuth(session))
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	writeRawData(w, data)
}

func (s *server) updateKeyStatus(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeJSON(r, &request); err != nil || request.Enabled == nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_key_status", "密钥状态参数无效")
		return
	}
	status := 2
	if *request.Enabled {
		status = 1
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodPut, "/api/token/?status_only=true", map[string]int{
		"id": id, "status": status,
	}, upstreamAuth(session))
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	writeRawData(w, data)
}

func (s *server) revealKey(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/token/"+strconv.Itoa(id)+"/key", nil, upstreamAuth(session))
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	writeRawData(w, data)
}

func (s *server) deleteKey(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodDelete, "/api/token/"+strconv.Itoa(id), nil, upstreamAuth(session))
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	writeRawData(w, data)
}

func (s *server) adminUser(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodGet, "/api/user/"+strconv.Itoa(id), nil, upstreamAuth(session))
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	writeRawData(w, data)
}

func (s *server) adminUserQuota(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var request struct {
		Mode  string `json:"mode"`
		Value int    `json:"value"`
	}
	if err := decodeJSON(r, &request); err != nil || !validQuotaChange(request.Mode, request.Value) {
		writeAPIError(w, http.StatusBadRequest, "invalid_quota_change", "额度调整参数无效")
		return
	}
	_, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/user/manage", map[string]any{
		"id": id, "action": "add_quota", "mode": request.Mode, "value": request.Value,
	}, upstreamAuth(session))
	if err != nil {
		s.recordAudit(r, session, "user.quota."+request.Mode, id, "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	if err := s.recordAudit(r, session, "user.quota."+request.Mode, id, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "额度已调整，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nil})
}

func (s *server) adminUserPassword(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var request struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &request); err != nil || len(request.Password) < 8 || len(request.Password) > 20 {
		writeAPIError(w, http.StatusBadRequest, "invalid_password", "密码长度必须为 8 到 20 位")
		return
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodGet, "/api/user/"+strconv.Itoa(id), nil, upstreamAuth(session))
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	var target newapi.User
	if err := json.Unmarshal(data, &target); err != nil || target.ID != id || strings.TrimSpace(target.Username) == "" {
		writeAPIError(w, http.StatusBadGateway, "new_api_contract_error", "读取用户资料失败")
		return
	}
	_, err = s.services.NewAPI.Do(r.Context(), http.MethodPut, "/api/user/", map[string]any{
		"id": id, "username": target.Username, "display_name": target.DisplayName,
		"group": target.Group, "remark": target.Remark, "password": request.Password,
	}, upstreamAuth(session))
	if err != nil {
		s.recordAudit(r, session, "user.password.reset", id, "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	if err := s.services.Sessions.DeleteUserSessions(r.Context(), id); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_cleanup_failed", "密码已修改，但会话清理失败")
		return
	}
	if err := s.recordAudit(r, session, "user.password.reset", id, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "密码已修改，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nil})
}

func (s *server) adminAuditLogs(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok {
		return
	}
	if s.services.Audit == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "audit_not_ready", "审计服务正在初始化")
		return
	}
	page := positiveQueryInt(r, "p", 1)
	pageSize := positiveQueryInt(r, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	entries, total, err := s.services.Audit.List(r.Context(), page, pageSize)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_query_failed", "读取审计日志失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"page": page, "page_size": pageSize, "total": total, "items": entries,
		},
	})
}

func decodeKeyRequest(w http.ResponseWriter, r *http.Request) (keyRequest, bool) {
	var request keyRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Name) == "" || len(request.Name) > 50 || request.RemainQuota < 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_key", "密钥参数无效")
		return keyRequest{}, false
	}
	request.Name = strings.TrimSpace(request.Name)
	return request, true
}

func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_id", "编号无效")
		return 0, false
	}
	return id, true
}

func validQuotaChange(mode string, value int) bool {
	switch mode {
	case "add", "subtract":
		return value > 0
	case "override":
		return value >= 0
	default:
		return false
	}
}

func positiveQueryInt(r *http.Request, name string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
