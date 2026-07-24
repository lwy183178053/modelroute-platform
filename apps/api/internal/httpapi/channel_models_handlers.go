package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

type channelModelsSyncRequest struct {
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
}

func (s *server) syncMerchantChannelModels(w http.ResponseWriter, r *http.Request) {
	_, session, _, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	var request channelModelsSyncRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "同步参数格式无效")
		return
	}
	models, _, err := s.fetchAndMatchChannelModels(r, session, request.Provider, request.BaseURL, request.APIKey)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "channel_models_sync_failed", err.Error())
		return
	}
	provider, _ := normalizeChannelProvider(request.Provider)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"provider": provider, "items": models, "synced_at": time.Now().UTC(),
	}})
}

func (s *server) fetchAndMatchChannelModels(r *http.Request, session platformauth.Session, provider, baseURL, apiKey string) ([]SyncedChannelModel, int, error) {
	if (s.services.NewAPI == nil && s.services.AdminNewAPI == nil) || s.services.Routing == nil {
		return nil, 0, errors.New("多模型同步服务尚未就绪")
	}
	provider, err := normalizeChannelProvider(provider)
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
	canonical, err := s.services.Routing.ListCanonicalModels(r.Context())
	if err != nil {
		return nil, 0, errors.New("读取平台模型目录失败")
	}
	for _, channelType := range channelTypesForProvider(provider) {
		requestBody := map[string]any{
			"type": channelType, "channel_type": channelType,
			"base_url": newAPIChannelBaseURL(channelType, normalizedURL),
			"key":      credential, "api_key": credential,
		}
		var data json.RawMessage
		if s.services.AdminNewAPI != nil {
			data, err = s.services.AdminNewAPI.Do(r.Context(), http.MethodPost, "/api/channel/fetch_models", requestBody)
		} else {
			data, err = s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/channel/fetch_models", requestBody, upstreamAuth(session))
		}
		if err != nil {
			continue
		}
		names := extractUpstreamModelNames(data)
		if len(names) > 0 {
			return matchUpstreamModelsForProvider(names, canonical, provider), channelType, nil
		}
	}
	return nil, 0, errors.New("上游模型同步失败")
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
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func (s *server) syncExistingMerchantChannelModels(w http.ResponseWriter, r *http.Request) {
	_, _, identity, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	repository, ok := s.services.Merchants.(MultiModelMerchantRepository)
	lookup, lookupReady := s.services.Merchants.(MerchantChannelLookup)
	if !ok || !lookupReady || s.services.AdminNewAPI == nil || s.services.Routing == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "multimodel_unavailable", "多模型渠道服务尚未就绪")
		return
	}
	channelID, ok := pathInt64(w, r, "id", "渠道编号无效")
	if !ok {
		return
	}
	channel, err := lookup.Channel(r.Context(), channelID)
	if err != nil || channel.MerchantID != identity.MerchantID {
		writeAPIError(w, http.StatusNotFound, "channel_not_found", "渠道不存在")
		return
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodGet, fmt.Sprintf("/api/channel/fetch_models/%d", channel.NewAPIChannelID), nil)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "channel_models_sync_failed", "同步渠道模型失败")
		return
	}
	canonical, err := s.services.Routing.ListCanonicalModels(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "model_store_error", "读取平台模型目录失败")
		return
	}
	matched := matchUpstreamModelsForProvider(extractUpstreamModelNames(data), canonical, channel.Provider)
	if err := s.updateCoreChannelModels(r.Context(), channel.NewAPIChannelID, matched); err != nil {
		writeAPIError(w, http.StatusBadGateway, "channel_models_sync_failed", "更新渠道模型失败")
		return
	}
	if err := repository.SyncChannelModels(r.Context(), channelID, matched); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "保存渠道模型失败")
		return
	}
	models, err := repository.ChannelModels(r.Context(), channelID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取渠道模型失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"items": models, "synced_at": time.Now().UTC()}})
}

func (s *server) updateCoreChannelModels(ctx context.Context, channelID int, matched []SyncedChannelModel) error {
	data, err := s.services.AdminNewAPI.Do(ctx, http.MethodGet, fmt.Sprintf("/api/channel/%d", channelID), nil)
	if err != nil {
		return err
	}
	currentModels, mapping, err := decodeChannelConfiguration(data)
	if err != nil {
		return err
	}
	models := make([]string, 0, len(matched))
	seen := map[string]struct{}{}
	for _, model := range matched {
		if model.MatchStatus != "matched" || model.CanonicalModelID <= 0 {
			continue
		}
		name := strings.TrimSpace(model.UpstreamModelName)
		if _, exists := seen[name]; !exists {
			seen[name] = struct{}{}
			models = append(models, name)
		}
	}
	for _, name := range strings.Split(currentModels, ",") {
		name = strings.TrimSpace(name)
		if !strings.HasPrefix(name, "mkt-o") {
			continue
		}
		if _, exists := seen[name]; !exists {
			seen[name] = struct{}{}
			models = append(models, name)
		}
	}
	sort.Strings(models)
	_, err = s.services.AdminNewAPI.Do(ctx, http.MethodPut, "/api/channel/", map[string]any{
		"id": channelID, "models": strings.Join(models, ","), "model_mapping": mapping,
	})
	return err
}

func (s *server) checkMerchantChannelModel(w http.ResponseWriter, r *http.Request) {
	_, _, identity, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	repository, ok := s.services.Merchants.(MultiModelMerchantRepository)
	lookup, lookupReady := s.services.Merchants.(MerchantChannelLookup)
	tester, testerReady := s.services.ChannelTests.(ExistingChannelTester)
	if !ok || !lookupReady || !testerReady || s.services.Routing == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "channel_test_unavailable", "模型检查服务尚未就绪")
		return
	}
	channelID, valid := pathInt64(w, r, "id", "渠道编号无效")
	if !valid {
		return
	}
	bindingID, valid := pathInt64(w, r, "binding_id", "模型编号无效")
	if !valid {
		return
	}
	channel, err := lookup.Channel(r.Context(), channelID)
	if err != nil || channel.MerchantID != identity.MerchantID {
		writeAPIError(w, http.StatusNotFound, "channel_not_found", "渠道不存在")
		return
	}
	bindings, err := repository.ChannelModels(r.Context(), channelID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取渠道模型失败")
		return
	}
	var binding ChannelModelBinding
	for _, item := range bindings {
		if item.ID == bindingID {
			binding = item
			break
		}
	}
	if binding.ID == 0 || binding.CanonicalModelID <= 0 {
		writeAPIError(w, http.StatusNotFound, "channel_model_not_found", "渠道模型不存在")
		return
	}
	models, err := s.services.Routing.ListCanonicalModels(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "model_store_error", "读取平台模型目录失败")
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
		writeAPIError(w, http.StatusNotFound, "model_not_found", "平台模型不存在")
		return
	}
	outcome, testErr := tester.TestExisting(r.Context(), channel.NewAPIChannelID, model)
	class, summary := healthFailureNone, ""
	if testErr != nil {
		class = healthFailureTemporary
		summary = sanitizedChannelError(testErr, "")
	}
	current := ModelHealthState{
		State: binding.HealthState, ConsecutiveRequestFailures: binding.ConsecutiveRequestFailures,
		AutomaticCheckFailures: binding.AutomaticCheckFailures,
	}
	if binding.NextCheckAt != nil {
		current.NextCheckAt = *binding.NextCheckAt
	}
	updated, err := repository.SaveManualModelCheck(
		r.Context(), channelID, bindingID, identity.MerchantID,
		nextModelHealthAfterManualCheck(current, class, time.Now().UTC()), class, outcome.LatencyMS, summary,
	)
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	if testErr != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, "channel_model_check_failed", summary)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": updated})
}

func (s *server) deleteMerchantChannel(w http.ResponseWriter, r *http.Request) {
	_, _, identity, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	s.deleteChannel(w, r, identity.MerchantID)
}

func (s *server) adminDeleteChannel(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	s.deleteChannel(w, r, 0)
}

func (s *server) deleteChannel(w http.ResponseWriter, r *http.Request, merchantID int64) {
	manager, ok := s.services.Merchants.(MerchantChannelManagement)
	lookup, lookupReady := s.services.Merchants.(MerchantChannelLookup)
	if !ok || !lookupReady || s.services.AdminNewAPI == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "channel_management_unavailable", "渠道管理服务尚未就绪")
		return
	}
	id, ok := pathInt64(w, r, "id", "渠道编号无效")
	if !ok {
		return
	}
	channel, err := lookup.Channel(r.Context(), id)
	if err != nil || (merchantID > 0 && channel.MerchantID != merchantID) {
		writeAPIError(w, http.StatusNotFound, "channel_not_found", "渠道不存在")
		return
	}
	if _, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodDelete, fmt.Sprintf("/api/channel/%d", channel.NewAPIChannelID), nil); err != nil {
		writeAPIError(w, http.StatusBadGateway, "channel_delete_failed", "删除核心渠道失败")
		return
	}
	if err := manager.DeleteChannel(r.Context(), id, merchantID); err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"id": id, "deleted": true}})
}
