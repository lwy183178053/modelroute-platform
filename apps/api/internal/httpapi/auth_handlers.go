package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if !s.authReady(w) {
		return
	}
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Username) == "" || request.Password == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "请输入账号和密码")
		return
	}
	result, err := s.services.NewAPI.Authenticate(r.Context(), "/api/user/login", map[string]string{
		"username": strings.TrimSpace(request.Username),
		"password": request.Password,
	}, "")
	if err != nil {
		writeUpstreamError(w, err, http.StatusUnauthorized)
		return
	}
	result.User.Role = platformRole(result.User.Username, result.User.Role, s.services.AdminUsername)
	session := platformauth.Session{
		UserID:           result.User.ID,
		Username:         result.User.Username,
		DisplayName:      result.User.DisplayName,
		Role:             result.User.Role,
		Status:           result.User.Status,
		Group:            result.User.Group,
		UpstreamCookie:   result.Cookie,
		PendingTwoFactor: result.RequireTwoFactor,
	}
	platformToken, err := s.services.Sessions.Create(r.Context(), session)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_error", "创建登录会话失败")
		return
	}
	s.setSessionCookie(w, platformToken)
	if result.RequireTwoFactor {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"data":    map[string]bool{"requires_two_factor": true},
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": result.User})
}

func (s *server) loginTwoFactor(w http.ResponseWriter, r *http.Request) {
	token, session, ok := s.requirePendingSession(w, r)
	if !ok {
		return
	}
	if !s.requireSameOrigin(w, r) {
		return
	}
	var request struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Code) == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "请输入二次验证码")
		return
	}
	result, err := s.services.NewAPI.Authenticate(r.Context(), "/api/user/login/2fa", map[string]string{
		"code": strings.TrimSpace(request.Code),
	}, session.UpstreamCookie)
	if err != nil {
		writeUpstreamError(w, err, http.StatusUnauthorized)
		return
	}
	result.User.Role = platformRole(result.User.Username, result.User.Role, s.services.AdminUsername)
	updated := platformauth.Session{
		UserID:         result.User.ID,
		Username:       result.User.Username,
		DisplayName:    result.User.DisplayName,
		Role:           result.User.Role,
		Status:         result.User.Status,
		Group:          result.User.Group,
		UpstreamCookie: result.Cookie,
	}
	if updated.UserID <= 0 || updated.Status != 1 {
		writeAPIError(w, http.StatusUnauthorized, "authentication_failed", "二次验证登录失败")
		return
	}
	if err := s.services.Sessions.Save(r.Context(), token, updated); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_error", "更新登录会话失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": result.User})
}

func (s *server) register(w http.ResponseWriter, r *http.Request) {
	if !s.authReady(w) {
		return
	}
	var request struct {
		Username         string `json:"username"`
		Password         string `json:"password"`
		Email            string `json:"email"`
		VerificationCode string `json:"verification_code"`
	}
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Username) == "" || len(request.Password) < 8 || !validEmail(request.Email) || strings.TrimSpace(request.VerificationCode) == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "注册信息不完整")
		return
	}
	_, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/user/register", map[string]string{
		"username":          strings.TrimSpace(request.Username),
		"password":          request.Password,
		"email":             strings.ToLower(strings.TrimSpace(request.Email)),
		"verification_code": strings.TrimSpace(request.VerificationCode),
	}, nil)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nil})
}

func (s *server) sendEmailVerification(w http.ResponseWriter, r *http.Request) {
	s.forwardEmailRequest(w, r, "/api/verification")
}

func (s *server) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	s.forwardEmailRequest(w, r, "/api/reset_password")
}

func (s *server) forwardEmailRequest(w http.ResponseWriter, r *http.Request, upstreamPath string) {
	if !s.authReady(w) {
		return
	}
	var request struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &request); err != nil || !validEmail(request.Email) {
		writeAPIError(w, http.StatusBadRequest, "invalid_email", "邮箱地址无效")
		return
	}
	email := strings.ToLower(strings.TrimSpace(request.Email))
	_, err := s.services.NewAPI.Do(r.Context(), http.MethodGet, upstreamPath+"?"+url.Values{"email": []string{email}}.Encode(), nil, nil)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nil})
}

func (s *server) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	if !s.authReady(w) {
		return
	}
	var request struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}
	if err := decodeJSON(r, &request); err != nil || !validEmail(request.Email) || strings.TrimSpace(request.Token) == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "重置链接无效")
		return
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/user/reset", map[string]string{
		"email": strings.ToLower(strings.TrimSpace(request.Email)),
		"token": strings.TrimSpace(request.Token),
	}, nil)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	writeRawData(w, data)
}

func (s *server) me(w http.ResponseWriter, r *http.Request) {
	token, session, ok := s.requireSession(w, r, roleCommon)
	if !ok {
		return
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodGet, "/api/user/self", nil, upstreamAuth(session))
	if err != nil {
		_ = s.services.Sessions.Delete(r.Context(), token)
		s.clearSessionCookie(w)
		writeUpstreamError(w, err, http.StatusUnauthorized)
		return
	}
	var user newapi.User
	if err := json.Unmarshal(data, &user); err != nil || user.ID != session.UserID || user.Status != 1 {
		_ = s.services.Sessions.Delete(r.Context(), token)
		s.clearSessionCookie(w)
		writeAPIError(w, http.StatusUnauthorized, "session_invalid", "账号状态已变化，请重新登录")
		return
	}
	session.Username = user.Username
	session.DisplayName = user.DisplayName
	user.Role = platformRole(user.Username, user.Role, s.services.AdminUsername)
	session.Role = user.Role
	session.Status = user.Status
	session.Group = user.Group
	if err := s.services.Sessions.Save(r.Context(), token, session); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_error", "刷新登录状态失败")
		return
	}
	var response map[string]any
	if err := json.Unmarshal(data, &response); err != nil {
		writeAPIError(w, http.StatusBadGateway, "upstream_response_invalid", "账号资料响应格式无效")
		return
	}
	response["role"] = user.Role
	response["is_merchant"] = false
	response["is_platform_merchant"] = false
	if s.services.Merchants != nil {
		identity, err := s.services.Merchants.Identity(r.Context(), session.UserID)
		switch {
		case err == nil:
			response["is_merchant"] = identity.IsMerchant && identity.Status == "active"
			response["merchant_id"] = identity.MerchantID
			response["merchant_name"] = identity.DisplayName
			response["merchant_status"] = identity.Status
			response["is_platform_merchant"] = identity.IsPlatformOwned
		case errors.Is(err, ErrMerchantNotFound):
			response["merchant_status"] = "none"
		default:
			writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取商户身份失败")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": response})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	token, session, ok := s.requireSession(w, r, roleCommon)
	if !ok {
		return
	}
	if !s.requireSameOrigin(w, r) {
		return
	}
	_, upstreamErr := s.services.NewAPI.Do(r.Context(), http.MethodGet, "/api/user/logout", nil, upstreamAuth(session))
	deleteErr := s.services.Sessions.Delete(r.Context(), token)
	s.clearSessionCookie(w)
	if deleteErr != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_error", "退出登录失败")
		return
	}
	if upstreamErr != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]bool{"upstream_session_cleared": false}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nil})
}

func (s *server) adminUsers(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok {
		return
	}
	query := url.Values{}
	for _, key := range []string{"p", "page_size", "keyword", "group", "role", "status"} {
		if value := r.URL.Query().Get(key); value != "" {
			query.Set(key, value)
		}
	}
	path := "/api/user/?" + query.Encode()
	if query.Get("keyword") != "" || query.Get("group") != "" || query.Get("role") != "" || query.Get("status") != "" {
		path = "/api/user/search?" + query.Encode()
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodGet, path, nil, upstreamAuth(session))
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	writeRawData(w, data)
}

func (s *server) adminUserStatus(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok {
		return
	}
	if !s.requireSameOrigin(w, r) {
		return
	}
	targetUserID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || targetUserID <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_user_id", "用户编号无效")
		return
	}
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeJSON(r, &request); err != nil || request.Enabled == nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "状态参数无效")
		return
	}
	action := "disable"
	if *request.Enabled {
		action = "enable"
	}
	_, err = s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/user/manage", map[string]any{
		"id": targetUserID, "action": action,
	}, upstreamAuth(session))
	if err != nil {
		s.recordAudit(r, session, "user."+action, targetUserID, "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	if action == "disable" {
		if err := s.services.Sessions.DeleteUserSessions(r.Context(), targetUserID); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "session_cleanup_failed", "用户已封禁，但会话清理失败")
			return
		}
	}
	if err := s.recordAudit(r, session, "user."+action, targetUserID, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "操作已生效，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]string{"status": action}})
}

func (s *server) authReady(w http.ResponseWriter) bool {
	if s.services.Sessions == nil || s.services.NewAPI == nil || s.services.Cookie.Name == "" {
		writeAPIError(w, http.StatusServiceUnavailable, "service_not_ready", "账号服务正在初始化")
		return false
	}
	return true
}

func (s *server) requireSession(w http.ResponseWriter, r *http.Request, minimumRole int) (string, platformauth.Session, bool) {
	if !s.authReady(w) {
		return "", platformauth.Session{}, false
	}
	cookie, err := r.Cookie(s.services.Cookie.Name)
	if err != nil || cookie.Value == "" {
		writeAPIError(w, http.StatusUnauthorized, "authentication_required", "请先登录")
		return "", platformauth.Session{}, false
	}
	session, err := s.services.Sessions.Get(r.Context(), cookie.Value)
	if errors.Is(err, platformauth.ErrSessionNotFound) {
		s.clearSessionCookie(w)
		writeAPIError(w, http.StatusUnauthorized, "session_expired", "登录状态已过期")
		return "", platformauth.Session{}, false
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "session_error", "读取登录状态失败")
		return "", platformauth.Session{}, false
	}
	if session.PendingTwoFactor || session.UserID <= 0 || session.Status != 1 {
		_ = s.services.Sessions.Delete(r.Context(), cookie.Value)
		s.clearSessionCookie(w)
		writeAPIError(w, http.StatusUnauthorized, "authentication_required", "登录状态无效")
		return "", platformauth.Session{}, false
	}
	session.Role = platformRole(session.Username, session.Role, s.services.AdminUsername)
	if session.Role < minimumRole {
		writeAPIError(w, http.StatusForbidden, "forbidden", "账号权限不足")
		return "", platformauth.Session{}, false
	}
	return cookie.Value, session, true
}

func platformRole(username string, upstreamRole int, administratorUsername string) int {
	administratorUsername = strings.TrimSpace(administratorUsername)
	if upstreamRole < roleAdmin || administratorUsername == "" {
		return upstreamRole
	}
	if strings.EqualFold(strings.TrimSpace(username), administratorUsername) {
		return upstreamRole
	}
	return roleCommon
}

func (s *server) requirePendingSession(w http.ResponseWriter, r *http.Request) (string, platformauth.Session, bool) {
	if !s.authReady(w) {
		return "", platformauth.Session{}, false
	}
	cookie, err := r.Cookie(s.services.Cookie.Name)
	if err != nil || cookie.Value == "" {
		writeAPIError(w, http.StatusUnauthorized, "two_factor_session_required", "二次验证会话已过期")
		return "", platformauth.Session{}, false
	}
	session, err := s.services.Sessions.Get(r.Context(), cookie.Value)
	if err != nil || !session.PendingTwoFactor || session.UpstreamCookie == "" {
		writeAPIError(w, http.StatusUnauthorized, "two_factor_session_required", "二次验证会话已过期")
		return "", platformauth.Session{}, false
	}
	return cookie.Value, session, true
}

func (s *server) requireSameOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
	expected := strings.TrimRight(strings.TrimSpace(s.services.Cookie.PublicOrigin), "/")
	if origin == "" || expected == "" || origin != expected {
		writeAPIError(w, http.StatusForbidden, "csrf_rejected", "请求来源校验失败")
		return false
	}
	return true
}

func (s *server) setSessionCookie(w http.ResponseWriter, token string) {
	maxAge := int(s.services.Cookie.MaxAge / time.Second)
	http.SetCookie(w, &http.Cookie{
		Name:     s.services.Cookie.Name,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.services.Cookie.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (s *server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.services.Cookie.Name,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.services.Cookie.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func upstreamAuth(session platformauth.Session) *newapi.Auth {
	return &newapi.Auth{Cookie: session.UpstreamCookie, UserID: session.UserID}
}

func (s *server) recordAudit(r *http.Request, actor platformauth.Session, action string, objectID int, result string, reason string) error {
	if s.services.Audit == nil {
		return nil
	}
	return s.services.Audit.Record(r.Context(), AuditEntry{
		ActorUserID: actor.UserID,
		ActorRole:   strconv.Itoa(actor.Role),
		Action:      action,
		ObjectType:  "user",
		ObjectID:    strconv.Itoa(objectID),
		Result:      result,
		Reason:      reason,
		RequestID:   r.Header.Get("X-Request-ID"),
		SourceIP:    clientIP(r),
	})
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		if parsed := net.ParseIP(forwarded); parsed != nil {
			return parsed.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && net.ParseIP(host) != nil {
		return net.ParseIP(host).String()
	}
	if parsed := net.ParseIP(strings.TrimSpace(r.RemoteAddr)); parsed != nil {
		return parsed.String()
	}
	return ""
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeRawData(w http.ResponseWriter, data json.RawMessage) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"success":true,"data":%s}`+"\n", data)
}

func writeAPIError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"success": false,
		"error":   map[string]string{"code": code, "message": message},
	})
}

func writeUpstreamError(w http.ResponseWriter, err error, fallbackStatus int) {
	status := fallbackStatus
	message := "内部服务连接异常"
	var upstreamError *newapi.APIError
	if errors.As(err, &upstreamError) {
		message = upstreamError.Message
		if upstreamError.StatusCode >= 400 {
			status = upstreamError.StatusCode
		}
	}
	writeAPIError(w, status, "new_api_error", message)
}

func validEmail(value string) bool {
	address, err := mail.ParseAddress(strings.TrimSpace(value))
	return err == nil && address.Address == strings.TrimSpace(value) && strings.Contains(address.Address, "@")
}
