package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/mail"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type smtpSettings struct {
	Server             string `json:"server"`
	Port               int    `json:"port"`
	Account            string `json:"account"`
	From               string `json:"from"`
	Password           string `json:"password,omitempty"`
	SSLEnabled         bool   `json:"ssl_enabled"`
	StartTLSEnabled    bool   `json:"starttls_enabled"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	ForceAuthLogin     bool   `json:"force_auth_login"`
}

type registrationSettings struct {
	EmailVerificationEnabled bool `json:"email_verification_enabled"`
	RegisterEnabled          bool `json:"register_enabled"`
	PasswordLoginEnabled     bool `json:"password_login_enabled"`
	PasswordRegisterEnabled  bool `json:"password_register_enabled"`
}

type platformSettingsRequest struct {
	SMTP         smtpSettings         `json:"smtp"`
	Registration registrationSettings `json:"registration"`
}

func (s *server) adminPlatformSettings(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok || !s.adminOperationsReady(w) {
		return
	}
	options, err := s.readAdminOptions(r)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": platformSettingsFromOptions(options)})
}

func (s *server) updateAdminPlatformSettings(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.adminOperationsReady(w) {
		return
	}
	var request platformSettingsRequest
	if err := decodeJSON(r, &request); err != nil || !validPlatformSettings(request) {
		writeAPIError(w, http.StatusBadRequest, "invalid_platform_settings", "邮件或注册设置参数无效")
		return
	}
	current, err := s.readAdminOptions(r)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	updates := []struct {
		key   string
		value string
	}{
		{key: "SMTPServer", value: strings.TrimSpace(request.SMTP.Server)},
		{key: "SMTPPort", value: strconv.Itoa(request.SMTP.Port)},
		{key: "SMTPAccount", value: strings.TrimSpace(request.SMTP.Account)},
		{key: "SMTPFrom", value: strings.TrimSpace(request.SMTP.From)},
		{key: "SMTPSSLEnabled", value: strconv.FormatBool(request.SMTP.SSLEnabled)},
		{key: "SMTPStartTLSEnabled", value: strconv.FormatBool(request.SMTP.StartTLSEnabled)},
		{key: "SMTPInsecureSkipVerify", value: strconv.FormatBool(request.SMTP.InsecureSkipVerify)},
		{key: "SMTPForceAuthLogin", value: strconv.FormatBool(request.SMTP.ForceAuthLogin)},
		{key: "EmailVerificationEnabled", value: strconv.FormatBool(request.Registration.EmailVerificationEnabled)},
		{key: "RegisterEnabled", value: strconv.FormatBool(request.Registration.RegisterEnabled)},
		{key: "PasswordLoginEnabled", value: strconv.FormatBool(request.Registration.PasswordLoginEnabled)},
		{key: "PasswordRegisterEnabled", value: strconv.FormatBool(request.Registration.PasswordRegisterEnabled)},
	}
	if password := strings.TrimSpace(request.SMTP.Password); password != "" {
		updates = append(updates, struct {
			key   string
			value string
		}{key: "SMTPToken", value: password})
	}
	for _, update := range updates {
		if update.key != "SMTPToken" && current[update.key] == update.value {
			continue
		}
		if _, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodPut, "/api/option/", map[string]any{"key": update.key, "value": update.value}); err != nil {
			s.recordAudit(r, session, "platform_settings.update", 0, "failed", err.Error())
			writeUpstreamError(w, err, http.StatusBadGateway)
			return
		}
	}
	if err := s.recordAudit(r, session, "platform_settings.update", 0, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "平台设置已保存，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": platformSettingsFromRequest(request)})
}

func validPlatformSettings(request platformSettingsRequest) bool {
	if request.SMTP.Port < 1 || request.SMTP.Port > 65535 || len(strings.TrimSpace(request.SMTP.Server)) > 255 || strings.ContainsAny(request.SMTP.Server, "\r\n") {
		return false
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(request.SMTP.Account)); err != nil {
		return false
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(request.SMTP.From)); err != nil {
		return false
	}
	if request.Registration.EmailVerificationEnabled {
		return strings.TrimSpace(request.SMTP.Server) != "" && strings.TrimSpace(request.SMTP.Account) != "" && strings.TrimSpace(request.SMTP.From) != ""
	}
	return true
}

func platformSettingsFromOptions(options map[string]string) map[string]any {
	port, _ := strconv.Atoi(options["SMTPPort"])
	return platformSettingsFromRequest(platformSettingsRequest{
		SMTP: smtpSettings{
			Server: strings.TrimSpace(options["SMTPServer"]), Port: port,
			Account: strings.TrimSpace(options["SMTPAccount"]), From: strings.TrimSpace(options["SMTPFrom"]),
			SSLEnabled: optionBool(options, "SMTPSSLEnabled"), StartTLSEnabled: optionBool(options, "SMTPStartTLSEnabled"),
			InsecureSkipVerify: optionBool(options, "SMTPInsecureSkipVerify"), ForceAuthLogin: optionBool(options, "SMTPForceAuthLogin"),
		},
		Registration: registrationSettings{
			EmailVerificationEnabled: optionBool(options, "EmailVerificationEnabled"), RegisterEnabled: optionBool(options, "RegisterEnabled"),
			PasswordLoginEnabled: optionBool(options, "PasswordLoginEnabled"), PasswordRegisterEnabled: optionBool(options, "PasswordRegisterEnabled"),
		},
	})
}

func platformSettingsFromRequest(request platformSettingsRequest) map[string]any {
	request.SMTP.Password = ""
	return map[string]any{
		"smtp": map[string]any{
			"server": request.SMTP.Server, "port": request.SMTP.Port, "account": request.SMTP.Account, "from": request.SMTP.From,
			"ssl_enabled": request.SMTP.SSLEnabled, "starttls_enabled": request.SMTP.StartTLSEnabled,
			"insecure_skip_verify": request.SMTP.InsecureSkipVerify, "force_auth_login": request.SMTP.ForceAuthLogin,
			"password_write_only": true,
		},
		"registration": request.Registration,
	}
}

func optionBool(options map[string]string, key string) bool {
	value, _ := strconv.ParseBool(strings.TrimSpace(options[key]))
	return value
}

func (s *server) readAdminOptions(r *http.Request) (map[string]string, error) {
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodGet, "/api/option/", nil)
	if err != nil {
		return nil, err
	}
	var items []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("解析 New API 设置：%w", err)
	}
	options := make(map[string]string, len(items))
	for _, item := range items {
		options[item.Key] = item.Value
	}
	return options, nil
}

type groupRatioItem struct {
	Name     string `json:"name"`
	RatioBPS int64  `json:"ratio_bps"`
}

func (s *server) adminGroups(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok || !s.adminOperationsReady(w) {
		return
	}
	options, err := s.readAdminOptions(r)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	groups, err := decodeGroupRatios(options["GroupRatio"])
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "new_api_contract_error", "New API 分组倍率格式无效")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": groups})
}

func (s *server) updateAdminGroups(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.adminOperationsReady(w) {
		return
	}
	var request struct {
		Groups []groupRatioItem `json:"groups"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_groups", "分组倍率参数无效")
		return
	}
	serialized, groups, ok := encodeGroupRatios(request.Groups)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "invalid_groups", "分组名称、倍率或默认分组无效")
		return
	}
	if _, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodPut, "/api/option/", map[string]any{"key": "GroupRatio", "value": serialized}); err != nil {
		s.recordAudit(r, session, "groups.update", 0, "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	if err := s.recordAudit(r, session, "groups.update", 0, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "分组倍率已保存，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": groups})
}

func decodeGroupRatios(raw string) ([]groupRatioItem, error) {
	if strings.TrimSpace(raw) == "" {
		return []groupRatioItem{}, nil
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	values := map[string]json.Number{}
	if err := decoder.Decode(&values); err != nil {
		return nil, err
	}
	groups := make([]groupRatioItem, 0, len(values))
	for name, number := range values {
		ratio, err := strconv.ParseFloat(number.String(), 64)
		if err != nil || ratio < 0 || ratio > float64(math.MaxInt32)/10_000 {
			return nil, fmt.Errorf("分组倍率无效")
		}
		groups = append(groups, groupRatioItem{Name: name, RatioBPS: int64(math.Round(ratio * 10_000))})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups, nil
}

func encodeGroupRatios(input []groupRatioItem) (string, []groupRatioItem, bool) {
	if len(input) == 0 || len(input) > 100 {
		return "", nil, false
	}
	seen := make(map[string]struct{}, len(input))
	groups := make([]groupRatioItem, 0, len(input))
	hasDefault := false
	for _, group := range input {
		group.Name = strings.TrimSpace(group.Name)
		if !validGroupName(group.Name) || group.RatioBPS < 0 || group.RatioBPS > math.MaxInt32 {
			return "", nil, false
		}
		if _, exists := seen[group.Name]; exists {
			return "", nil, false
		}
		seen[group.Name] = struct{}{}
		hasDefault = hasDefault || group.Name == "default"
		groups = append(groups, group)
	}
	if !hasDefault {
		return "", nil, false
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	var result bytes.Buffer
	result.WriteByte('{')
	for index, group := range groups {
		if index > 0 {
			result.WriteByte(',')
		}
		name, _ := json.Marshal(group.Name)
		result.Write(name)
		result.WriteByte(':')
		result.WriteString(formatBasisPoints(group.RatioBPS))
	}
	result.WriteByte('}')
	return result.String(), groups, true
}

func validGroupName(value string) bool {
	if value == "" || utf8.RuneCountInString(value) > 64 {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return false
		}
	}
	return true
}

func formatBasisPoints(value int64) string {
	whole, fraction := value/10_000, value%10_000
	if fraction == 0 {
		return strconv.FormatInt(whole, 10)
	}
	return strconv.FormatInt(whole, 10) + "." + strings.TrimRight(fmt.Sprintf("%04d", fraction), "0")
}

func (s *server) adminRedemptions(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok || !s.adminOperationsReady(w) {
		return
	}
	page, pageSize := pagination(r)
	query := url.Values{"p": []string{strconv.Itoa(page)}, "page_size": []string{strconv.Itoa(pageSize)}}
	path := "/api/redemption/?" + query.Encode()
	if keyword := strings.TrimSpace(r.URL.Query().Get("keyword")); keyword != "" || strings.TrimSpace(r.URL.Query().Get("status")) != "" {
		query.Set("keyword", keyword)
		query.Set("status", strings.TrimSpace(r.URL.Query().Get("status")))
		path = "/api/redemption/search?" + query.Encode()
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodGet, path, nil)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	writeRawData(w, data)
}

func (s *server) createAdminRedemption(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.adminOperationsReady(w) {
		return
	}
	request, ok := decodeRedemptionRequest(w, r)
	if !ok {
		return
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodPost, "/api/redemption/", request)
	if err != nil {
		s.recordAudit(r, session, "redemption.create", 0, "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	if err := s.recordAudit(r, session, "redemption.create", 0, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "兑换码已创建，但审计记录失败")
		return
	}
	writeRawDataStatus(w, http.StatusCreated, data)
}

func (s *server) updateAdminRedemption(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.adminOperationsReady(w) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	request, ok := decodeRedemptionUpdateRequest(w, r)
	if !ok {
		return
	}
	request["id"] = id
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodPut, "/api/redemption/", request)
	if err != nil {
		s.recordAudit(r, session, "redemption.update", id, "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	if err := s.recordAudit(r, session, "redemption.update", id, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "兑换码已更新，但审计记录失败")
		return
	}
	writeRawData(w, data)
}

func (s *server) updateAdminRedemptionStatus(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.adminOperationsReady(w) {
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
		writeAPIError(w, http.StatusBadRequest, "invalid_redemption_status", "兑换码状态参数无效")
		return
	}
	status := 2
	if *request.Enabled {
		status = 1
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodPut, "/api/redemption/?status_only=true", map[string]any{"id": id, "status": status})
	if err != nil {
		s.recordAudit(r, session, "redemption.status.update", id, "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	if err := s.recordAudit(r, session, "redemption.status.update", id, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "兑换码状态已更新，但审计记录失败")
		return
	}
	writeRawData(w, data)
}

func (s *server) deleteAdminRedemption(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.adminOperationsReady(w) {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodDelete, "/api/redemption/"+strconv.Itoa(id), nil)
	if err != nil {
		s.recordAudit(r, session, "redemption.delete", id, "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	if err := s.recordAudit(r, session, "redemption.delete", id, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "兑换码已删除，但审计记录失败")
		return
	}
	writeRawData(w, data)
}

func (s *server) deleteInvalidAdminRedemptions(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.adminOperationsReady(w) {
		return
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodDelete, "/api/redemption/invalid", nil)
	if err != nil {
		s.recordAudit(r, session, "redemption.invalid.delete", 0, "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	if err := s.recordAudit(r, session, "redemption.invalid.delete", 0, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "失效兑换码已清理，但审计记录失败")
		return
	}
	writeRawData(w, data)
}

func decodeRedemptionRequest(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	var request struct {
		Name        string `json:"name"`
		Quota       int    `json:"quota"`
		Count       int    `json:"count"`
		ExpiredTime int64  `json:"expired_time"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_redemption", "兑换码参数无效")
		return nil, false
	}
	request.Name = strings.TrimSpace(request.Name)
	if utf8.RuneCountInString(request.Name) < 1 || utf8.RuneCountInString(request.Name) > 20 || request.Quota <= 0 || request.Count < 1 || request.Count > 100 || (request.ExpiredTime != 0 && request.ExpiredTime < time.Now().Unix()) {
		writeAPIError(w, http.StatusBadRequest, "invalid_redemption", "兑换码名称、额度、数量或过期时间无效")
		return nil, false
	}
	return map[string]any{"name": request.Name, "quota": request.Quota, "count": request.Count, "expired_time": request.ExpiredTime}, true
}

func decodeRedemptionUpdateRequest(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	var request struct {
		Name        string `json:"name"`
		Quota       int    `json:"quota"`
		ExpiredTime int64  `json:"expired_time"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_redemption", "兑换码参数无效")
		return nil, false
	}
	request.Name = strings.TrimSpace(request.Name)
	if utf8.RuneCountInString(request.Name) < 1 || utf8.RuneCountInString(request.Name) > 20 || request.Quota <= 0 || (request.ExpiredTime != 0 && request.ExpiredTime < time.Now().Unix()) {
		writeAPIError(w, http.StatusBadRequest, "invalid_redemption", "兑换码名称、额度或过期时间无效")
		return nil, false
	}
	return map[string]any{"name": request.Name, "quota": request.Quota, "expired_time": request.ExpiredTime}, true
}

func (s *server) redeemCode(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	var request struct {
		Key string `json:"key"`
	}
	if err := decodeJSON(r, &request); err != nil || len(strings.TrimSpace(request.Key)) < 1 || len(strings.TrimSpace(request.Key)) > 128 {
		writeAPIError(w, http.StatusBadRequest, "invalid_redemption", "请输入有效兑换码")
		return
	}
	request.Key = strings.TrimSpace(request.Key)
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/user/topup", map[string]string{"key": request.Key}, upstreamAuth(session))
	if err != nil {
		s.recordAudit(r, session, "redemption.redeem", 0, "failed", "兑换失败")
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	if err := s.recordAudit(r, session, "redemption.redeem", 0, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "兑换成功，但审计记录失败")
		return
	}
	writeRawData(w, data)
}

func (s *server) adminSystemTasks(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok || !s.adminOperationsReady(w) {
		return
	}
	limit := positiveQueryInt(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodGet, "/api/system-task/list?limit="+strconv.Itoa(limit), nil)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	writeRawData(w, data)
}

func (s *server) adminSystemTask(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok || !s.adminOperationsReady(w) {
		return
	}
	taskID := strings.TrimSpace(r.PathValue("task_id"))
	if len(taskID) < 1 || len(taskID) > 100 || strings.ContainsAny(taskID, "/\\") {
		writeAPIError(w, http.StatusBadRequest, "invalid_system_task", "系统任务编号无效")
		return
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodGet, "/api/system-task/"+url.PathEscape(taskID), nil)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	writeRawData(w, data)
}

func (s *server) createAdminLogCleanupTask(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.adminOperationsReady(w) {
		return
	}
	var request struct {
		BeforeTimestamp int64 `json:"before_timestamp"`
	}
	if err := decodeJSON(r, &request); err != nil || request.BeforeTimestamp <= 0 || request.BeforeTimestamp >= time.Now().Unix() {
		writeAPIError(w, http.StatusBadRequest, "invalid_log_cleanup", "日志清理截止时间必须早于当前时间")
		return
	}
	query := url.Values{"target_timestamp": []string{strconv.FormatInt(request.BeforeTimestamp, 10)}}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodPost, "/api/system-task/log-cleanup?"+query.Encode(), nil)
	if err != nil {
		s.recordAudit(r, session, "system_task.log_cleanup.create", 0, "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadRequest)
		return
	}
	if err := s.recordAudit(r, session, "system_task.log_cleanup.create", 0, "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "日志清理任务已创建，但审计记录失败")
		return
	}
	writeRawDataStatus(w, http.StatusAccepted, data)
}

func (s *server) adminChannelRuntime(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok || !s.channelOperationsReady(w) {
		return
	}
	channelID, ok := pathInt64(w, r, "id", "渠道编号无效")
	if !ok {
		return
	}
	newAPIChannelID, err := s.services.Routing.ChannelForHealthReset(r.Context(), channelID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "channel_not_found", "渠道不存在")
		return
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodGet, "/api/channel/"+strconv.Itoa(newAPIChannelID), nil)
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	var runtime struct {
		ID                 int     `json:"id"`
		Status             int     `json:"status"`
		Balance            float64 `json:"balance"`
		BalanceUpdatedTime int64   `json:"balance_updated_time"`
		ResponseTime       int     `json:"response_time"`
		TestTime           int64   `json:"test_time"`
	}
	if err := json.Unmarshal(data, &runtime); err != nil || runtime.ID != newAPIChannelID {
		writeAPIError(w, http.StatusBadGateway, "new_api_contract_error", "New API 渠道状态格式无效")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"newapi_channel_id": runtime.ID, "status": runtime.Status,
		"balance_usd_micros": int64(math.Round(runtime.Balance * 1_000_000)), "balance_updated_time": runtime.BalanceUpdatedTime,
		"response_time_ms": runtime.ResponseTime, "test_time": runtime.TestTime,
	}})
}

func (s *server) adminTestChannel(w http.ResponseWriter, r *http.Request) {
	s.adminChannelOperation(w, r, "test")
}

func (s *server) adminRefreshChannelBalance(w http.ResponseWriter, r *http.Request) {
	s.adminChannelOperation(w, r, "balance.refresh")
}

func (s *server) adminChannelOperation(w http.ResponseWriter, r *http.Request, operation string) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.channelOperationsReady(w) {
		return
	}
	channelID, ok := pathInt64(w, r, "id", "渠道编号无效")
	if !ok {
		return
	}
	newAPIChannelID, err := s.services.Routing.ChannelForHealthReset(r.Context(), channelID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "channel_not_found", "渠道不存在")
		return
	}
	path := "/api/channel/test/" + strconv.Itoa(newAPIChannelID)
	action := "channel.test"
	if operation == "balance.refresh" {
		path = "/api/channel/update_balance/" + strconv.Itoa(newAPIChannelID)
		action = "channel.balance.refresh"
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodGet, path, nil)
	if err != nil {
		s.recordAudit(r, session, action, int(channelID), "failed", err.Error())
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	if err := s.recordAudit(r, session, action, int(channelID), "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "渠道操作已完成，但审计记录失败")
		return
	}
	value, err := sanitizedJSONValue(data)
	if err != nil {
		value = nil
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"status": "succeeded", "result": value}})
}

func (s *server) adminOperationsReady(w http.ResponseWriter) bool {
	if s.services.AdminNewAPI == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "admin_operations_unavailable", "New API 管理服务尚未就绪")
		return false
	}
	return true
}

func (s *server) channelOperationsReady(w http.ResponseWriter) bool {
	return s.adminOperationsReady(w) && s.routingReady(w, false)
}

func writeRawDataStatus(w http.ResponseWriter, status int, data []byte) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		writeAPIError(w, http.StatusBadGateway, "new_api_contract_error", "New API 响应格式无效")
		return
	}
	writeJSON(w, status, map[string]any{"success": true, "data": value})
}
