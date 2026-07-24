package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

func (s *server) createMerchantApplication(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	identity, err := s.services.Merchants.Identity(r.Context(), session.UserID)
	switch {
	case err == nil && identity.Status == "active":
		s.writeMerchantStoreError(w, ErrMerchantConflict)
		return
	case err == nil, errors.Is(err, ErrMerchantNotFound):
	case err != nil:
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取商户身份失败")
		return
	}
	var request merchantSubmissionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "商户申请格式无效")
		return
	}
	command, err := s.buildApplicationCommand(session.UserID, request)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	application, err := s.services.Merchants.CreateApplication(r.Context(), command)
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": application})
}

func (s *server) merchantApplication(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.merchantReady(w) {
		return
	}
	application, err := s.services.Merchants.ApplicationForUser(r.Context(), session.UserID)
	if errors.Is(err, ErrMerchantNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nil})
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取商户申请失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": application})
}

func (s *server) merchantProfile(w http.ResponseWriter, r *http.Request) {
	_, _, identity, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	profile, err := s.services.Merchants.Merchant(r.Context(), identity.MerchantID)
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": profile})
}

func (s *server) updateMerchantProfile(w http.ResponseWriter, r *http.Request) {
	_, _, identity, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	var request struct {
		BusinessDescription string `json:"business_description"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "商户资料格式无效")
		return
	}
	description := strings.TrimSpace(request.BusinessDescription)
	if len([]rune(description)) < 10 || len([]rune(description)) > 2000 {
		writeAPIError(w, http.StatusBadRequest, "invalid_description", "商户介绍需为 10 至 2000 字")
		return
	}
	profile, err := s.services.Merchants.UpdateMerchantDescription(r.Context(), identity.MerchantID, description)
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": profile})
}

func (s *server) merchantChannels(w http.ResponseWriter, r *http.Request) {
	_, session, _, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	channels, err := s.services.Merchants.ChannelsForUser(r.Context(), session.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取商户渠道失败")
		return
	}
	s.attachChannelModels(r.Context(), channels)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"items": channels}})
}

func (s *server) createMerchantChannel(w http.ResponseWriter, r *http.Request) {
	_, session, _, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	repository, supported := s.services.Merchants.(SelfServiceChannelRepository)
	if !supported || s.services.AdminNewAPI == nil || s.services.Routing == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "channel_management_unavailable", "渠道创建服务尚未就绪")
		return
	}
	var request merchantChannelRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "渠道参数格式无效")
		return
	}
	name := strings.TrimSpace(request.ChannelName)
	if len([]rune(name)) < 2 || len([]rune(name)) > 100 {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "渠道名称需为 2 至 100 字")
		return
	}
	provider, err := normalizeChannelProvider(request.Provider)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	baseURL, err := normalizePublicHTTPSURL(request.BaseURL, false)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "基础地址必须是公开 HTTPS 地址")
		return
	}
	credential := []byte(strings.TrimSpace(request.APIKey))
	if len(credential) < 8 || len(credential) > 65536 {
		zeroBytes(credential)
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "API 密钥长度无效")
		return
	}
	matched, channelType, err := s.fetchAndMatchChannelModels(r, session, provider, baseURL, string(credential))
	if err != nil {
		zeroBytes(credential)
		writeAPIError(w, http.StatusBadGateway, "channel_models_sync_failed", err.Error())
		return
	}
	models := make([]string, 0, len(matched))
	storedModels := make([]StoreChannelModel, 0, len(matched))
	for _, model := range matched {
		if model.MatchStatus != "matched" || model.CanonicalModelID <= 0 {
			continue
		}
		models = append(models, model.UpstreamModelName)
		storedModels = append(storedModels, StoreChannelModel{UpstreamModelName: model.UpstreamModelName, CanonicalModelID: model.CanonicalModelID})
	}
	if len(models) == 0 {
		zeroBytes(credential)
		writeAPIError(w, http.StatusUnprocessableEntity, "no_matched_models", "上游没有平台已收录的同厂商模型")
		return
	}
	fingerprint := s.services.Credentials.Fingerprint(credential)
	newAPIChannelID, err := s.createSelfServiceCoreChannel(r.Context(), session.UserID, channelType, baseURL, string(credential), models)
	zeroBytes(credential)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "channel_import_failed", sanitizedChannelError(err, request.APIKey))
		return
	}
	channel, err := repository.CreateSelfServiceChannel(r.Context(), StoreMerchantChannel{
		UserID: session.UserID, NewAPIChannelID: newAPIChannelID, ChannelName: name,
		Provider: provider, ChannelType: channelType, AdapterType: adapterForChannelType(channelType),
		BaseURL: baseURL, CredentialFingerprint: fingerprint, Models: storedModels,
	})
	if err != nil {
		_, _ = s.services.AdminNewAPI.Do(r.Context(), http.MethodDelete, fmt.Sprintf("/api/channel/%d", newAPIChannelID), nil)
		s.writeMerchantStoreError(w, err)
		return
	}
	channel.ModelBindings = make([]ChannelModelBinding, 0, len(storedModels))
	for _, model := range storedModels {
		channel.ModelBindings = append(channel.ModelBindings, ChannelModelBinding{
			UpstreamModelName: model.UpstreamModelName, CanonicalModelID: model.CanonicalModelID,
			Model: model.UpstreamModelName, MatchStatus: "matched", HealthState: "pending",
		})
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": channel})
}

func (s *server) createSelfServiceCoreChannel(ctx context.Context, userID, channelType int, baseURL, credential string, models []string) (int, error) {
	randomName, err := temporaryChannelName()
	if err != nil {
		return 0, err
	}
	internalName := fmt.Sprintf("ModelRoute-U%d-%s", userID, strings.TrimPrefix(randomName, "ModelRoute-Test-"))
	if _, err := s.services.AdminNewAPI.Do(ctx, http.MethodPost, "/api/channel/", map[string]any{
		"mode": "single",
		"channel": map[string]any{
			"type": channelType, "key": credential, "name": internalName,
			"base_url": newAPIChannelBaseURL(channelType, baseURL),
			"models":   strings.Join(models, ","), "group": "default", "status": 1,
			"test_model": models[0], "tag": fmt.Sprintf("modelroute-u%d", userID),
		},
	}); err != nil {
		return 0, err
	}
	data, err := s.services.AdminNewAPI.Do(ctx, http.MethodGet, "/api/channel/search?"+url.Values{
		"keyword": {internalName}, "p": {"1"}, "page_size": {"20"}, "id_sort": {"true"},
	}.Encode(), nil)
	if err != nil {
		return 0, err
	}
	var result struct {
		Items []struct {
			ID int `json:"id"`
		} `json:"items"`
	}
	if json.Unmarshal(data, &result) != nil || len(result.Items) == 0 || result.Items[0].ID <= 0 {
		return 0, errors.New("核心服务未返回新渠道编号")
	}
	return result.Items[0].ID, nil
}

func (s *server) attachChannelModels(ctx context.Context, channels []MerchantChannel) {
	repository, ok := s.services.Merchants.(MultiModelMerchantRepository)
	if !ok {
		return
	}
	for index := range channels {
		models, err := repository.ChannelModels(ctx, channels[index].ID)
		if err == nil {
			channels[index].ModelBindings = models
		}
	}
}

func (s *server) adminMerchantApplications(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.merchantReady(w) {
		return
	}
	page, pageSize := pagination(r)
	items, total, err := s.services.Merchants.ListApplications(r.Context(), page, pageSize, r.URL.Query().Get("status"), r.URL.Query().Get("keyword"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取商户申请失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"page": page, "page_size": pageSize, "total": total, "items": items}})
}

func (s *server) adminMerchantApplication(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.merchantReady(w) {
		return
	}
	id, ok := pathInt64(w, r, "id", "申请编号无效")
	if !ok {
		return
	}
	application, err := s.services.Merchants.Application(r.Context(), id)
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": application})
}

func (s *server) adminMerchantChannels(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.merchantReady(w) {
		return
	}
	page, pageSize := pagination(r)
	var items []MerchantChannel
	var total int
	var err error
	if repository, filtered := s.services.Merchants.(FilteredMerchantChannelRepository); filtered {
		merchantUserID, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("merchant_user_id")))
		items, total, err = repository.ListChannelsFiltered(r.Context(), page, pageSize, r.URL.Query().Get("keyword"), merchantUserID)
	} else {
		items, total, err = s.services.Merchants.ListChannels(r.Context(), page, pageSize)
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取渠道列表失败")
		return
	}
	s.attachChannelModels(r.Context(), items)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"page": page, "page_size": pageSize, "total": total, "items": items}})
}

func (s *server) adminApproveApplication(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	id, ok := pathInt64(w, r, "id", "商户申请编号无效")
	if !ok {
		return
	}
	profile, err := s.services.Merchants.ApproveApplication(r.Context(), id, session.UserID)
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": profile})
}

func (s *server) adminRejectApplication(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	id, ok := pathInt64(w, r, "id", "商户申请编号无效")
	if !ok {
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	reason := ""
	if err := decodeJSON(r, &request); err == nil {
		reason = strings.TrimSpace(request.Reason)
	}
	if len([]rune(reason)) < 2 || len([]rune(reason)) > 500 {
		writeAPIError(w, http.StatusBadRequest, "invalid_reason", "请输入 2 至 500 字的驳回原因")
		return
	}
	if err := s.services.Merchants.RejectApplication(r.Context(), id, session.UserID, reason); err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"id": id, "status": "rejected"}})
}

func (s *server) adminUserMerchantStatus(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	userID, ok := pathID(w, r)
	if !ok {
		return
	}
	identity, err := s.services.Merchants.Identity(r.Context(), userID)
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	var request struct {
		Enabled *bool  `json:"enabled"`
		Reason  string `json:"reason"`
	}
	if err := decodeJSON(r, &request); err != nil || request.Enabled == nil || len(strings.TrimSpace(request.Reason)) < 2 {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "状态和原因不能为空")
		return
	}
	change, err := s.services.Merchants.SetMerchantStatus(r.Context(), identity.MerchantID, *request.Enabled, session.UserID, strings.TrimSpace(request.Reason))
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"user_id": change.UserID, "status": change.Status}})
}

type merchantChannelRequest struct {
	ChannelName string `json:"channel_name"`
	Provider    string `json:"provider"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"`
}

type merchantSubmissionRequest struct {
	DisplayName         string `json:"display_name"`
	WebsiteURL          string `json:"website_url"`
	BusinessDescription string `json:"business_description"`
}

type merchantDisplayName struct {
	Value string
	Key   string
	Units int
}

func validateMerchantDisplayName(value string) (merchantDisplayName, error) {
	value = strings.TrimSpace(value)
	result := merchantDisplayName{Value: value, Key: strings.ToLower(value)}
	for _, character := range value {
		switch {
		case unicode.Is(unicode.Han, character):
			result.Units += 2
		case character >= 'A' && character <= 'Z', character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			result.Units++
		default:
			return merchantDisplayName{}, errors.New("商户名称只能包含汉字、英文字母和数字")
		}
	}
	if result.Units < 2 || result.Units > 16 {
		return merchantDisplayName{}, errors.New("商户名称长度需为 2 至 16 单位，汉字计 2 单位，英文和数字计 1 单位")
	}
	return result, nil
}

func (s *server) buildApplicationCommand(userID int, request merchantSubmissionRequest) (CreateMerchantApplication, error) {
	displayName, err := validateMerchantDisplayName(request.DisplayName)
	if err != nil {
		return CreateMerchantApplication{}, err
	}
	description := strings.TrimSpace(request.BusinessDescription)
	if len([]rune(description)) < 10 || len([]rune(description)) > 2000 {
		return CreateMerchantApplication{}, errors.New("业务说明需为 10 至 2000 字")
	}
	website, err := normalizePublicHTTPSURL(request.WebsiteURL, false)
	if err != nil {
		return CreateMerchantApplication{}, errors.New("网站地址必须是公开 HTTPS 地址")
	}
	return CreateMerchantApplication{UserID: userID, DisplayName: displayName.Value, DisplayNameKey: displayName.Key, WebsiteURL: website, BusinessDescription: description}, nil
}

func (s *server) requireMerchant(w http.ResponseWriter, r *http.Request) (string, platformauth.Session, MerchantIdentity, bool) {
	token, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.merchantReady(w) {
		return "", platformauth.Session{}, MerchantIdentity{}, false
	}
	identity, err := s.services.Merchants.Identity(r.Context(), session.UserID)
	if err != nil || !identity.IsMerchant || identity.Status != "active" {
		writeAPIError(w, http.StatusForbidden, "merchant_required", "当前账号没有有效商户权限")
		return "", platformauth.Session{}, MerchantIdentity{}, false
	}
	return token, session, identity, true
}

func (s *server) merchantReady(w http.ResponseWriter) bool {
	if s.services.Merchants == nil || s.services.Credentials == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "service_not_ready", "商户服务正在初始化")
		return false
	}
	return true
}

func (s *server) writeMerchantStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrMerchantNotFound):
		writeAPIError(w, http.StatusNotFound, "merchant_not_found", "商户记录不存在")
	case errors.Is(err, ErrMerchantNameConflict):
		writeAPIError(w, http.StatusConflict, "merchant_name_conflict", "商户名称已被使用，请更换后重试")
	case errors.Is(err, ErrMerchantConflict), errors.Is(err, ErrMerchantInvalidState):
		writeAPIError(w, http.StatusConflict, "merchant_state_conflict", "记录状态已变化，请刷新后重试")
	default:
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "保存商户数据失败")
	}
}

func normalizePublicHTTPSURL(raw string, trimOpenAIV1 bool) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid URL")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", errors.New("private host")
	}
	if ip := net.ParseIP(host); ip != nil && (!ip.IsGlobalUnicast() || ip.IsPrivate()) {
		return "", errors.New("private address")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if trimOpenAIV1 && strings.HasSuffix(parsed.Path, "/v1") {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/v1")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func sanitizedChannelError(err error, credential string) string {
	message := strings.TrimSpace(err.Error())
	if credential != "" {
		message = strings.ReplaceAll(message, credential, "***")
	}
	if message == "" {
		return "渠道请求失败"
	}
	if len(message) > 500 {
		message = message[:500]
	}
	return message
}

func pathInt64(w http.ResponseWriter, r *http.Request, name, message string) (int64, bool) {
	value, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || value <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_id", message)
		return 0, false
	}
	return value, true
}

func pagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("p"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func zeroBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
