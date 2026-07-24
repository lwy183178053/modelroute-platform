package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
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

type createKeyRequest struct {
	keyRequest
	Provider         string  `json:"provider"`
	RoutingPolicyIDs []int64 `json:"routing_policy_ids"`
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
	if s.services.Routing == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "routing_unavailable", "路由服务正在初始化")
		return
	}
	bindings, err := s.services.Routing.ListTokenRoutingBindings(r.Context(), session.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "key_routing_query_failed", "读取密钥路由失败")
		return
	}
	joined, err := joinKeyRoutingBindings(data, bindings)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "key_list_decode_failed", "密钥列表格式无效")
		return
	}
	writeRawData(w, joined)
}

func (s *server) createKey(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	request, ok := decodeCreateKeyRequest(w, r)
	if !ok {
		return
	}
	if s.services.Routing == nil || s.services.Credentials == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "routing_unavailable", "路由服务正在初始化")
		return
	}
	tokenIDsBefore, err := s.userTokenIDs(r, session)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/token/", request.keyRequest, upstreamAuth(session))
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	tokenIDsAfter, err := s.userTokenIDs(r, session)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	createdTokenID := 0
	for tokenID := range tokenIDsAfter {
		if _, existed := tokenIDsBefore[tokenID]; !existed {
			createdTokenID = tokenID
			break
		}
	}
	if createdTokenID == 0 {
		writeAPIError(w, http.StatusBadGateway, "token_bind_failed", "密钥已创建，但没有返回新增密钥编号")
		return
	}
	if _, err := s.bindTokenRoutes(r, session.UserID, session.UpstreamCookie, createdTokenID, request.Provider, request.RoutingPolicyIDs); err != nil {
		_, cleanupErr := s.services.NewAPI.Do(r.Context(), http.MethodDelete, "/api/token/"+strconv.Itoa(createdTokenID), nil, upstreamAuth(session))
		if cleanupErr != nil {
			writeAPIError(w, http.StatusBadGateway, "token_bind_cleanup_failed", "路由绑定失败，且新密钥清理失败")
			return
		}
		writeAPIError(w, http.StatusConflict, "token_bind_failed", "所选路由不存在或不属于当前账号")
		return
	}
	writeRawData(w, data)
}

func (s *server) replaceKeyRoutingPolicies(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	tokenID, ok := pathID(w, r)
	if !ok {
		return
	}
	var request struct {
		RoutingPolicyIDs []int64 `json:"routing_policy_ids"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_routing_policy", "模型路由参数无效")
		return
	}
	if s.services.Routing == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "routing_unavailable", "路由服务正在初始化")
		return
	}
	binding, err := s.services.Routing.ReplaceTokenRoutes(r.Context(), session.UserID, tokenID, request.RoutingPolicyIDs)
	if errors.Is(err, ErrRoutingNotFound) {
		writeAPIError(w, http.StatusNotFound, "routing_binding_not_found", "密钥或模型路由不存在")
		return
	}
	if errors.Is(err, ErrRoutingInvalidState) {
		writeAPIError(w, http.StatusConflict, "routing_binding_conflict", "只能选择当前密钥厂商的路由，且同一模型只能选择一条")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "routing_binding_failed", "重新绑定路由失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": binding})
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
	if s.services.Routing != nil {
		if err := s.services.Routing.SetTokenBindingStatus(r.Context(), session.UserID, id, *request.Enabled); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "token_status_sync_failed", "密钥状态已更新，但路由状态同步失败")
			return
		}
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
	if s.services.Routing != nil {
		if err := s.services.Routing.RevokeTokenBinding(r.Context(), session.UserID, id); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "token_revoke_failed", "密钥已删除，但路由绑定清理失败")
			return
		}
	}
	writeRawData(w, data)
}

func (s *server) userTokenIDs(r *http.Request, session platformauth.Session) (map[int]struct{}, error) {
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodGet, "/api/token/?p=1&page_size=100", nil, upstreamAuth(session))
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []struct {
			ID int `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	ids := make(map[int]struct{}, len(result.Items))
	for _, item := range result.Items {
		if item.ID > 0 {
			ids[item.ID] = struct{}{}
		}
	}
	return ids, nil
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
		writeUpstreamError(w, err, http.StatusBadGateway)
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
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	if err := s.services.Sessions.DeleteUserSessions(r.Context(), id); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_cleanup_failed", "密码已修改，但会话清理失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nil})
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

func decodeCreateKeyRequest(w http.ResponseWriter, r *http.Request) (createKeyRequest, bool) {
	var request createKeyRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Name) == "" || len(request.Name) > 50 || request.RemainQuota < 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_key", "密钥参数无效")
		return createKeyRequest{}, false
	}
	provider, err := normalizeChannelProvider(request.Provider)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_key_provider", "请选择有效的密钥厂商")
		return createKeyRequest{}, false
	}
	request.Provider = provider
	request.Name = strings.TrimSpace(request.Name)
	return request, true
}

func joinKeyRoutingBindings(data json.RawMessage, bindings []TokenRoutingBinding) (json.RawMessage, error) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	byToken := make(map[int]TokenRoutingBinding, len(bindings))
	for _, binding := range bindings {
		byToken[binding.TokenID] = binding
	}
	items, _ := payload["items"].([]any)
	filtered := make([]any, 0, len(items))
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		tokenID, _ := item["id"].(float64)
		binding, exists := byToken[int(tokenID)]
		if !exists {
			continue
		}
		item["provider"] = binding.Provider
		item["routing_policies"] = binding.Routes
		item["model_count"] = binding.ModelCount
		filtered = append(filtered, item)
	}
	payload["items"] = filtered
	payload["total"] = len(bindings)
	return json.Marshal(payload)
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
