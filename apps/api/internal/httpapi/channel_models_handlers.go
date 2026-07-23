package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

type channelModelsSyncRequest struct {
	ChannelType string `json:"channel_type"`
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"`
}

func (s *server) syncMerchantChannelModels(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	var request channelModelsSyncRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "同步参数格式无效")
		return
	}
	models, channelType, err := s.fetchAndMatchChannelModels(r, session, request.ChannelType, request.BaseURL, request.APIKey)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "channel_models_sync_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"channel_type": channelType, "items": models, "synced_at": time.Now().UTC(),
	}})
}

func (s *server) fetchAndMatchChannelModels(r *http.Request, session platformauth.Session, channelTypeName, baseURL, apiKey string) ([]SyncedChannelModel, int, error) {
	if (s.services.NewAPI == nil && s.services.AdminNewAPI == nil) || s.services.Routing == nil {
		return nil, 0, errors.New("多模型同步服务尚未就绪")
	}
	channelType, err := parseChannelType(channelTypeName)
	if err != nil {
		return nil, 0, err
	}
	normalizedURL, err := normalizePublicHTTPSURL(baseURL, false)
	if err != nil {
		return nil, 0, errors.New("基础地址必须是公开 HTTPS 地址")
	}
	credential := strings.TrimSpace(apiKey)
	if len(credential) < 8 || len(credential) > 65536 {
		return nil, 0, errors.New("API 密钥长度无效")
	}
	requestBody := map[string]any{
		"type": channelType, "channel_type": channelType, "base_url": newAPIChannelBaseURL(channelType, normalizedURL), "key": credential, "api_key": credential,
	}
	var data json.RawMessage
	if s.services.AdminNewAPI != nil {
		data, err = s.services.AdminNewAPI.Do(r.Context(), http.MethodPost, "/api/channel/fetch_models", requestBody)
	} else {
		data, err = s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/channel/fetch_models", requestBody, upstreamAuth(session))
	}
	if err != nil {
		return nil, 0, fmt.Errorf("上游模型同步失败")
	}
	names := extractUpstreamModelNames(data)
	if len(names) == 0 {
		return nil, 0, errors.New("上游没有返回模型")
	}
	canonical, err := s.services.Routing.ListCanonicalModels(r.Context())
	if err != nil {
		return nil, 0, errors.New("读取平台模型目录失败")
	}
	return matchUpstreamModels(names, canonical), channelType, nil
}

func parseChannelType(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if value, ok := channelTypeForName(raw); ok {
		return value, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || (value != 1 && value != 14 && value != 24) {
		return 0, errors.New("渠道类型只支持 OpenAI 兼容、Anthropic 或 Gemini")
	}
	return value, nil
}

func extractUpstreamModelNames(data json.RawMessage) []string {
	var value any
	if json.Unmarshal(data, &value) != nil {
		return nil
	}
	var result []string
	var visit func(any)
	visit = func(node any) {
		switch item := node.(type) {
		case []any:
			for _, child := range item {
				visit(child)
			}
		case map[string]any:
			for _, key := range []string{"model", "model_name", "id", "name"} {
				if name, ok := item[key].(string); ok && strings.TrimSpace(name) != "" {
					result = append(result, strings.TrimSpace(name))
					return
				}
			}
			for _, key := range []string{"models", "data", "items", "list"} {
				if child, ok := item[key]; ok {
					visit(child)
				}
			}
		case string:
			if strings.TrimSpace(item) != "" {
				result = append(result, strings.TrimSpace(item))
			}
		}
	}
	visit(value)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(result))
	for _, name := range result {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func (s *server) syncExistingMerchantChannelModels(w http.ResponseWriter, r *http.Request) {
	_, session, identity, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	repository, ok := s.services.Merchants.(MultiModelMerchantRepository)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "multimodel_unavailable", "多模型渠道服务尚未就绪")
		return
	}
	if (s.services.NewAPI == nil && s.services.AdminNewAPI == nil) || s.services.Routing == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "multimodel_unavailable", "多模型同步服务尚未就绪")
		return
	}
	channelID, ok := pathInt64(w, r, "id", "渠道编号无效")
	if !ok {
		return
	}
	var channel MerchantChannel
	channels, err := s.services.Merchants.ChannelsForUser(r.Context(), session.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取商户渠道失败")
		return
	}
	for _, candidate := range channels {
		if candidate.ID == channelID {
			channel = candidate
			break
		}
	}
	if channel.ID == 0 || channel.MerchantID != identity.MerchantID {
		writeAPIError(w, http.StatusNotFound, "channel_not_found", "渠道不存在")
		return
	}
	var data json.RawMessage
	if s.services.AdminNewAPI != nil {
		data, err = s.services.AdminNewAPI.Do(r.Context(), http.MethodGet, fmt.Sprintf("/api/channel/fetch_models/%d", channel.NewAPIChannelID), nil)
	} else {
		data, err = s.services.NewAPI.Do(r.Context(), http.MethodGet, fmt.Sprintf("/api/channel/fetch_models/%d", channel.NewAPIChannelID), nil, upstreamAuth(session))
	}
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "channel_models_sync_failed", "同步已导入渠道模型失败")
		return
	}
	names := extractUpstreamModelNames(data)
	canonical, err := s.services.Routing.ListCanonicalModels(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "model_store_error", "读取平台模型目录失败")
		return
	}
	matched := matchUpstreamModels(names, canonical)
	if err := repository.SyncChannelModels(r.Context(), channelID, matched); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "保存渠道模型失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"items": matched, "synced_at": time.Now().UTC()}})
}

func (s *server) adminTestReviewModel(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	repository, ok := s.services.Merchants.(MultiModelMerchantRepository)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "multimodel_unavailable", "多模型审核服务尚未就绪")
		return
	}
	if s.services.ChannelTests == nil || s.services.Routing == nil || s.services.Credentials == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "channel_test_unavailable", "模型检查服务尚未就绪")
		return
	}
	reviewID, ok := pathInt64(w, r, "id", "审核编号无效")
	if !ok {
		return
	}
	var request struct {
		BindingID int64 `json:"binding_id"`
	}
	if err := decodeJSON(r, &request); err != nil || request.BindingID <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "模型审核编号无效")
		return
	}
	reviews, _, err := s.services.Merchants.ListChannelReviews(r.Context(), 1, 100, "")
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取渠道审核失败")
		return
	}
	var review MerchantChannelReview
	for _, item := range reviews {
		if item.ID == reviewID {
			review = item
			break
		}
	}
	if review.ID == 0 || (review.Status != "pending" && review.Status != "verified" && review.Status != "verifying") {
		writeAPIError(w, http.StatusConflict, "review_state_conflict", "渠道审核状态不允许检测")
		return
	}
	bindings, err := repository.ReviewModels(r.Context(), reviewID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取审核模型失败")
		return
	}
	var binding ReviewModelBinding
	for _, item := range bindings {
		if item.ID == request.BindingID {
			binding = item
			break
		}
	}
	if binding.ID == 0 || binding.CanonicalModelID <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_model", "该模型未被平台收录")
		return
	}
	credential, err := s.services.Credentials.Decrypt(review.EncryptedCredential)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "credential_error", "读取待审核凭据失败")
		return
	}
	defer zeroBytes(credential)
	models, err := s.services.Routing.ListCanonicalModels(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "model_store_error", "读取标准模型失败")
		return
	}
	var model CanonicalModel
	for _, item := range models {
		if item.ID == binding.CanonicalModelID {
			model = item
			break
		}
	}
	if model.ID == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_model", "标准模型不存在")
		return
	}
	outcome, testErr := s.services.ChannelTests.Test(r.Context(), ChannelTestRequest{Model: model, BaseURL: review.BaseURL, APIKey: string(credential)})
	status, message := "passed", ""
	if testErr != nil {
		status, message = "failed", testErr.Error()
	}
	if err := repository.SetReviewModelTest(r.Context(), reviewID, binding.ID, session.UserID, status, outcome.LatencyMS, message); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "保存模型检测结果失败")
		return
	}
	if testErr != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, "channel_model_test_failed", "模型 hi 检查失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"binding_id": binding.ID, "status": status, "latency_ms": outcome.LatencyMS}})
}

func (s *server) updateMerchantChannelStatus(w http.ResponseWriter, r *http.Request) {
	_, _, identity, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	manager, ok := s.services.Merchants.(MerchantChannelManagement)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "channel_management_unavailable", "渠道管理服务尚未就绪")
		return
	}
	id, ok := pathInt64(w, r, "id", "渠道编号无效")
	if !ok {
		return
	}
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeJSON(r, &request); err != nil || request.Enabled == nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "渠道状态无效")
		return
	}
	if err := manager.SetChannelStatus(r.Context(), id, identity.MerchantID, *request.Enabled); err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"id": id, "status": map[bool]string{true: "active", false: "disabled"}[*request.Enabled]}})
}

func (s *server) deleteMerchantChannel(w http.ResponseWriter, r *http.Request) {
	_, _, identity, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	manager, ok := s.services.Merchants.(MerchantChannelManagement)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "channel_management_unavailable", "渠道管理服务尚未就绪")
		return
	}
	id, ok := pathInt64(w, r, "id", "渠道编号无效")
	if !ok {
		return
	}
	if err := manager.DeleteChannel(r.Context(), id, identity.MerchantID); err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"id": id, "status": "archived"}})
}

func (s *server) deleteMerchantChannelReview(w http.ResponseWriter, r *http.Request) {
	_, _, identity, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	manager, ok := s.services.Merchants.(MerchantChannelReviewManagement)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "channel_review_management_unavailable", "渠道审核管理服务尚未就绪")
		return
	}
	id, ok := pathInt64(w, r, "id", "渠道审核编号无效")
	if !ok {
		return
	}
	action, err := manager.DeleteChannelReview(r.Context(), id, identity.MerchantID)
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"id": id, "action": action}})
}

func (s *server) adminUpdateChannelStatus(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	manager, ok := s.services.Merchants.(MerchantChannelManagement)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "channel_management_unavailable", "渠道管理服务尚未就绪")
		return
	}
	id, ok := pathInt64(w, r, "id", "渠道编号无效")
	if !ok {
		return
	}
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeJSON(r, &request); err != nil || request.Enabled == nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "渠道状态无效")
		return
	}
	if err := manager.SetChannelStatus(r.Context(), id, 0, *request.Enabled); err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"id": id}})
}

func (s *server) adminDeleteChannel(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	manager, ok := s.services.Merchants.(MerchantChannelManagement)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "channel_management_unavailable", "渠道管理服务尚未就绪")
		return
	}
	id, ok := pathInt64(w, r, "id", "渠道编号无效")
	if !ok {
		return
	}
	if err := manager.DeleteChannel(r.Context(), id, 0); err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"id": id, "status": "archived"}})
}
