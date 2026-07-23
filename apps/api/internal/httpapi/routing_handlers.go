package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

func (s *server) createCanonicalModel(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	var request struct {
		PublicName string        `json:"public_name"`
		Provider   string        `json:"provider"`
		Protocol   string        `json:"protocol"`
		Price      OfficialPrice `json:"price"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "标准模型格式无效")
		return
	}
	command := CreateCanonicalModelCommand{
		PublicName: strings.TrimSpace(request.PublicName), Provider: strings.TrimSpace(request.Provider),
		Protocol: strings.TrimSpace(request.Protocol), AdminID: session.UserID,
		Price: normalizeOfficialPrice(request.Price),
	}
	if err := validateCanonicalModel(command); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_model_price", err.Error())
		return
	}
	model, err := s.services.Routing.CreateCanonicalModel(r.Context(), command)
	if err != nil {
		writeAPIError(w, http.StatusConflict, "model_create_failed", "标准模型名称或价格版本冲突")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": model})
}

func (s *server) detectCanonicalModel(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	if s.services.ChannelTests == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "model_detect_unavailable", "模型检测服务尚未就绪")
		return
	}
	var request struct {
		PublicName string `json:"public_name"`
		Provider   string `json:"provider"`
		Protocol   string `json:"protocol"`
		BaseURL    string `json:"base_url"`
		APIKey     string `json:"api_key"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_model_detect", "模型检测参数无效")
		return
	}
	model := CanonicalModel{
		ID: 1, PublicName: strings.TrimSpace(request.PublicName), Provider: strings.TrimSpace(request.Provider),
		Protocol: strings.TrimSpace(request.Protocol), Status: "active",
	}
	baseURL, err := normalizePublicHTTPSURL(request.BaseURL, false)
	credential := strings.TrimSpace(request.APIKey)
	if err != nil || validateModelDescriptor(model.PublicName, model.Provider, model.Protocol) != nil || len(credential) < 8 || len(credential) > 65536 {
		writeAPIError(w, http.StatusBadRequest, "invalid_model_detect", "模型、基础地址或 API 密钥无效")
		return
	}
	outcome, err := s.services.ChannelTests.Test(r.Context(), ChannelTestRequest{Model: model, BaseURL: baseURL, APIKey: credential})
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "model_detect_failed", "模型 hi 检测失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": outcome})
}

func (s *server) deleteCanonicalModel(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	modelID, ok := pathInt64(w, r, "id", "标准模型编号无效")
	if !ok {
		return
	}
	result, err := s.services.Routing.DeleteCanonicalModel(r.Context(), modelID)
	if errors.Is(err, ErrRoutingNotFound) {
		writeAPIError(w, http.StatusNotFound, "model_not_found", "标准模型不存在")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusConflict, "model_delete_failed", "标准模型当前无法删除或归档")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]string{"result": result}})
}

func (s *server) models(w http.ResponseWriter, r *http.Request) {
	if !s.routingReady(w, false) {
		return
	}
	s.writeModels(w, r)
}

func (s *server) adminModels(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok || !s.routingReady(w, false) {
		return
	}
	s.writeModels(w, r)
}

func (s *server) writeModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.services.Routing.ListCanonicalModels(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "models_query_failed", "查询标准模型失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": models})
}

func (s *server) publishOfficialPrice(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	modelID, ok := pathInt64(w, r, "id", "标准模型编号无效")
	if !ok {
		return
	}
	var price OfficialPrice
	if err := decodeJSON(r, &price); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_model_price", "官方价格配置无效")
		return
	}
	published, err := s.services.Routing.PublishOfficialPrice(r.Context(), modelID, session.UserID, normalizeOfficialPrice(price))
	if err != nil {
		if errors.Is(err, ErrRoutingInvalidState) {
			writeAPIError(w, http.StatusBadRequest, "invalid_model_price", "官方价格配置无效")
			return
		}
		writeAPIError(w, http.StatusConflict, "price_publish_failed", "标准模型不存在或价格版本冲突")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": published})
}

func (s *server) createMerchantOffer(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	var request struct {
		MultiplierBPS int `json:"multiplier_bps"`
	}
	modelID, ok := pathInt64(w, r, "model_id", "标准模型编号无效")
	if !ok {
		return
	}
	if err := decodeJSON(r, &request); err != nil || request.MultiplierBPS < 0 || request.MultiplierBPS > 50_000 {
		writeAPIError(w, http.StatusBadRequest, "invalid_offer", "报价倍率必须在 0 到 50000 基点之间")
		return
	}
	if s.services.OfferSync == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "routing_unavailable", "报价同步服务尚未就绪")
		return
	}
	activate := true
	if lifecycle, ready := s.services.Routing.(OfferLifecycleRepository); ready {
		state, stateErr := lifecycle.MerchantOfferState(r.Context(), session.UserID, modelID)
		switch {
		case stateErr == nil && state.Status == "inactive":
			activate = false
		case stateErr == nil, errors.Is(stateErr, ErrRoutingNotFound):
		case stateErr != nil:
			writeAPIError(w, http.StatusInternalServerError, "offer_state_failed", "读取上架模型状态失败")
			return
		}
	}
	offer, err := s.services.Routing.CreateOffer(r.Context(), CreateOfferCommand{
		UserID: session.UserID, CanonicalModelID: modelID, MultiplierBPS: request.MultiplierBPS,
	})
	if err != nil {
		status := http.StatusConflict
		message := "渠道、模型或当前报价状态不允许上架"
		if errors.Is(err, ErrRoutingNotFound) {
			status = http.StatusNotFound
			message = "未找到可上架的已审核渠道或标准模型"
		}
		writeAPIError(w, status, "offer_create_failed", message)
		return
	}
	if err := s.services.OfferSync.SyncOffer(r.Context(), offer.ID, activate); err != nil {
		_ = s.services.Routing.MarkOfferSyncFailed(r.Context(), offer.ID, "核心服务价格或渠道映射同步失败")
		writeAPIError(w, http.StatusBadGateway, "offer_sync_failed", "模型价格和渠道映射同步失败")
		return
	}
	if activate {
		offer.Status = "active"
	} else {
		offer.Status = "inactive"
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": offer})
}

func (s *server) updateMerchantOfferStatus(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	modelID, ok := pathInt64(w, r, "model_id", "标准模型编号无效")
	if !ok {
		return
	}
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decodeJSON(r, &request); err != nil || request.Enabled == nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_offer_status", "上架状态参数无效")
		return
	}
	if !*request.Enabled {
		if err := s.services.Routing.RetireOffer(r.Context(), session.UserID, modelID); err != nil {
			if errors.Is(err, ErrRoutingNotFound) {
				writeAPIError(w, http.StatusNotFound, "offer_not_found", "当前模型没有上架中的价格")
				return
			}
			writeAPIError(w, http.StatusConflict, "offer_retire_failed", "当前模型状态不允许下架")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"model_id": modelID, "status": "inactive"}})
		return
	}
	lifecycle, ready := s.services.Routing.(OfferLifecycleRepository)
	if !ready || s.services.OfferSync == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "offer_lifecycle_unavailable", "上架模型服务尚未就绪")
		return
	}
	state, err := lifecycle.MerchantOfferState(r.Context(), session.UserID, modelID)
	if err != nil || state.Status != "inactive" {
		if errors.Is(err, ErrRoutingNotFound) {
			writeAPIError(w, http.StatusNotFound, "offer_not_found", "待上架模型不存在")
			return
		}
		writeAPIError(w, http.StatusConflict, "offer_activate_failed", "当前模型状态不允许上架")
		return
	}
	offer, err := s.services.Routing.CreateOffer(r.Context(), CreateOfferCommand{UserID: session.UserID, CanonicalModelID: modelID, MultiplierBPS: state.MultiplierBPS})
	if err != nil {
		writeAPIError(w, http.StatusConflict, "offer_activate_failed", "当前渠道状态不允许上架")
		return
	}
	if err := s.services.OfferSync.SyncOffer(r.Context(), offer.ID, true); err != nil {
		_ = s.services.Routing.MarkOfferSyncFailed(r.Context(), offer.ID, "核心服务价格或渠道映射同步失败")
		writeAPIError(w, http.StatusBadGateway, "offer_sync_failed", "模型价格和渠道映射同步失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"model_id": modelID, "status": "active"}})
}

func (s *server) deleteMerchantOffer(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	modelID, ok := pathInt64(w, r, "model_id", "标准模型编号无效")
	if !ok {
		return
	}
	lifecycle, ready := s.services.Routing.(OfferLifecycleRepository)
	if !ready {
		writeAPIError(w, http.StatusServiceUnavailable, "offer_lifecycle_unavailable", "上架模型服务尚未就绪")
		return
	}
	if err := lifecycle.DeleteMerchantOffer(r.Context(), session.UserID, modelID); err != nil {
		if errors.Is(err, ErrRoutingNotFound) {
			writeAPIError(w, http.StatusNotFound, "offer_not_found", "待删除模型不存在")
			return
		}
		writeAPIError(w, http.StatusConflict, "offer_delete_failed", "当前模型状态不允许删除")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"model_id": modelID, "status": "deleted"}})
}

func (s *server) routingPolicy(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.routingReady(w, false) {
		return
	}
	tokenID, ok := pathID(w, r)
	if !ok {
		return
	}
	policy, ok := s.bindTokenPolicy(w, r, session.UserID, session.UpstreamCookie, tokenID)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": policy})
}

func (s *server) updateRoutingPolicy(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	tokenID, ok := pathID(w, r)
	if !ok {
		return
	}
	var input RoutingPolicyInput
	if err := decodeJSON(r, &input); err != nil || validateRoutingPolicy(input) != nil {
		message := "路由策略参数无效"
		if err := validateRoutingPolicy(input); err != nil {
			message = err.Error()
		}
		writeAPIError(w, http.StatusBadRequest, "invalid_routing_policy", message)
		return
	}
	if _, ok := s.bindTokenPolicy(w, r, session.UserID, session.UpstreamCookie, tokenID); !ok {
		return
	}
	policy, err := s.services.Routing.UpdatePolicy(r.Context(), session.UserID, tokenID, input)
	if err != nil {
		writeAPIError(w, http.StatusConflict, "routing_policy_update_failed", "路由策略或固定商户状态已变化")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": policy})
}

func (s *server) marketOffers(w http.ResponseWriter, r *http.Request) {
	if !s.routingReady(w, false) {
		return
	}
	offers, err := s.services.Routing.ListMarketOffers(r.Context(), 0, strings.TrimSpace(r.URL.Query().Get("model")))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "offers_query_failed", "查询市场报价失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": offers})
}

func (s *server) merchantOffers(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.routingReady(w, false) {
		return
	}
	offers, err := s.services.Routing.ListMerchantOffers(r.Context(), session.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_offers_query_failed", "查询商户报价失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": offers})
}

func (s *server) adminResetChannelHealth(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) || s.services.AdminNewAPI == nil {
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
	if _, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodPost, "/api/channel/"+strconv.Itoa(newAPIChannelID)+"/status", map[string]int{"status": 1}); err != nil {
		writeAPIError(w, http.StatusBadGateway, "channel_enable_failed", "恢复 New API 渠道失败")
		return
	}
	if err := s.services.Routing.ResetChannelHealth(r.Context(), channelID); err != nil {
		_, _ = s.services.AdminNewAPI.Do(r.Context(), http.MethodPost, "/api/channel/"+strconv.Itoa(newAPIChannelID)+"/status", map[string]int{"status": 2})
		writeAPIError(w, http.StatusInternalServerError, "channel_reset_failed", "恢复平台渠道失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"channel_id": channelID, "health_state": "pending"}})
}

func (s *server) bindTokenPolicy(w http.ResponseWriter, r *http.Request, userID int, cookie string, tokenID int) (RoutingPolicy, bool) {
	if s.services.Credentials == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "routing_unavailable", "令牌指纹服务尚未就绪")
		return RoutingPolicy{}, false
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/token/"+strconv.Itoa(tokenID)+"/key", nil, &newapi.Auth{Cookie: cookie, UserID: userID})
	if err != nil {
		writeUpstreamError(w, err, http.StatusNotFound)
		return RoutingPolicy{}, false
	}
	var revealed struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(data, &revealed); err != nil || strings.TrimSpace(revealed.Key) == "" {
		writeAPIError(w, http.StatusBadGateway, "token_reveal_failed", "核心服务未返回有效密钥")
		return RoutingPolicy{}, false
	}
	policy, err := s.services.Routing.BindTokenAndDefaultPolicy(r.Context(), userID, tokenID, s.services.Credentials.Fingerprint([]byte(strings.TrimSpace(revealed.Key))))
	if err != nil {
		writeAPIError(w, http.StatusConflict, "token_bind_failed", "密钥路由绑定冲突")
		return RoutingPolicy{}, false
	}
	return policy, true
}

func validateRoutingPolicy(input RoutingPolicyInput) error {
	if input.Mode != "smart" && input.Mode != "fixed" {
		return errors.New("路由模式必须是智能路由或固定路由")
	}
	if input.Priority != "price" && input.Priority != "stability" {
		return errors.New("优先方式必须是价格优先或稳定性优先")
	}
	if input.MinimumMultiplierBPS < 0 || input.MaximumMultiplierBPS > 50_000 || input.MinimumMultiplierBPS > input.MaximumMultiplierBPS {
		return errors.New("倍率区间必须在 0 到 50000 基点之间")
	}
	if input.Mode == "fixed" && len(uniqueInt64(input.MerchantIDs)) == 0 {
		return errors.New("固定路由至少选择一个商户")
	}
	return nil
}

func validateCanonicalModel(command CreateCanonicalModelCommand) error {
	if err := validateModelDescriptor(command.PublicName, command.Provider, command.Protocol); err != nil {
		return err
	}
	return validateOfficialPrice(command.Price, command.Protocol)
}

func validateModelDescriptor(publicName string, provider string, protocol string) error {
	if publicName == "" || len(publicName) > 255 || strings.HasPrefix(publicName, "mkt-ov-") || provider == "" || len(provider) > 80 {
		return errors.New("模型名称或厂商无效")
	}
	if _, supported := supportedModelProviders[provider]; !supported {
		return errors.New("模型厂商不在平台支持范围")
	}
	switch protocol {
	case "chat_completions", "responses", "image_generation":
	default:
		return errors.New("模型协议不在平台开放范围")
	}
	return nil
}

func (s *server) routingReady(w http.ResponseWriter, requireSync bool) bool {
	if s.services.Routing == nil || (requireSync && s.services.OfferSync == nil) {
		writeAPIError(w, http.StatusServiceUnavailable, "routing_unavailable", "模型路由服务尚未就绪")
		return false
	}
	return true
}
