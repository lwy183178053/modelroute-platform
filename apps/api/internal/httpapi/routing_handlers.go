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
		Price      OfficialPrice `json:"price"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "标准模型格式无效")
		return
	}
	command := CreateCanonicalModelCommand{
		PublicName: strings.TrimSpace(request.PublicName), Provider: strings.TrimSpace(request.Provider),
		AdminID: session.UserID, Price: normalizeOfficialPrice(request.Price),
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
		PriceKind  string `json:"price_kind"`
		BaseURL    string `json:"base_url"`
		APIKey     string `json:"api_key"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_model_detect", "模型检测参数无效")
		return
	}
	model := CanonicalModel{
		ID: 1, PublicName: strings.TrimSpace(request.PublicName), Provider: strings.TrimSpace(request.Provider),
		Status: "active", Price: OfficialPrice{PriceKind: strings.TrimSpace(request.PriceKind)},
	}
	baseURL, err := normalizePublicHTTPSURL(request.BaseURL, false)
	credential := strings.TrimSpace(request.APIKey)
	if err != nil || validateModelDescriptor(model.PublicName, model.Provider) != nil || (model.Price.PriceKind != "text" && model.Price.PriceKind != "image") || len(credential) < 8 || len(credential) > 65536 {
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

func (s *server) routingPolicies(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.routingReady(w, false) {
		return
	}
	policies, err := s.services.Routing.ListRoutingPolicies(r.Context(), session.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "routing_policy_query_failed", "读取路由列表失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": policies})
}

func (s *server) createRoutingPolicy(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	input, ok := decodeCreateRoutingPolicyInput(w, r)
	if !ok {
		return
	}
	policy, err := s.services.Routing.CreateRoutingPolicy(r.Context(), session.UserID, input)
	if err != nil {
		writeRoutingPolicyError(w, err, "创建路由失败")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": policy})
}

func (s *server) previewRoutingPolicy(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	repository, ok := s.services.Routing.(RoutingPreviewRepository)
	if !ok {
		writeAPIError(w, http.StatusServiceUnavailable, "routing_preview_unavailable", "路由范围预览服务尚未就绪")
		return
	}
	var input RoutingPreviewInput
	if err := decodeJSON(r, &input); err != nil || validateRoutingPreviewInput(input) != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_routing_preview", "路由预览参数无效")
		return
	}
	preview, err := repository.PreviewRoute(r.Context(), input)
	if errors.Is(err, ErrRoutingNotFound) {
		writeAPIError(w, http.StatusNotFound, "model_not_found", "预览模型不存在或未启用")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "routing_preview_failed", "统计路由范围失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": preview})
}

func validateRoutingPreviewInput(input RoutingPreviewInput) error {
	if input.CanonicalModelID <= 0 || input.MinimumMultiplierBPS < 0 || input.MaximumMultiplierBPS > 50_000 || input.MinimumMultiplierBPS > input.MaximumMultiplierBPS {
		return ErrRoutingInvalidState
	}
	if input.MinimumCacheHitPPM != nil && (*input.MinimumCacheHitPPM < 0 || *input.MinimumCacheHitPPM > 1_000_000) {
		return ErrRoutingInvalidState
	}
	return nil
}

func (s *server) routingPolicy(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.routingReady(w, false) {
		return
	}
	policyID, ok := pathInt64(w, r, "id", "路由编号无效")
	if !ok {
		return
	}
	policy, err := s.services.Routing.RoutingPolicy(r.Context(), session.UserID, policyID)
	if err != nil {
		writeRoutingPolicyError(w, err, "读取路由失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": policy})
}

func (s *server) updateRoutingPolicy(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	policyID, ok := pathInt64(w, r, "id", "路由编号无效")
	if !ok {
		return
	}
	input, ok := decodeRoutingPolicyInput(w, r)
	if !ok {
		return
	}
	policy, err := s.services.Routing.UpdateRoutingPolicy(r.Context(), session.UserID, policyID, input)
	if err != nil {
		writeRoutingPolicyError(w, err, "修改路由失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": policy})
}

func (s *server) deleteRoutingPolicy(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.routingReady(w, false) {
		return
	}
	policyID, ok := pathInt64(w, r, "id", "路由编号无效")
	if !ok {
		return
	}
	affectedKeyCount, err := s.services.Routing.DeleteRoutingPolicy(r.Context(), session.UserID, policyID)
	if err != nil {
		writeRoutingPolicyError(w, err, "删除路由失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]int{"affected_key_count": affectedKeyCount}})
}

func decodeRoutingPolicyInput(w http.ResponseWriter, r *http.Request) (RoutingPolicyInput, bool) {
	var request struct {
		Name                 string `json:"name"`
		Priority             string `json:"priority"`
		MinimumMultiplierBPS int    `json:"minimum_multiplier_bps"`
		MaximumMultiplierBPS int    `json:"maximum_multiplier_bps"`
		MinimumCacheHitPPM   *int   `json:"minimum_cache_hit_ppm"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_routing_policy", "路由参数格式无效")
		return RoutingPolicyInput{}, false
	}
	input := RoutingPolicyInput{
		Name: request.Name, Priority: request.Priority,
		MinimumMultiplierBPS: request.MinimumMultiplierBPS,
		MaximumMultiplierBPS: request.MaximumMultiplierBPS,
		MinimumCacheHitPPM:   request.MinimumCacheHitPPM,
	}
	input.Name = strings.TrimSpace(input.Name)
	if err := validateRoutingPolicy(input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_routing_policy", err.Error())
		return RoutingPolicyInput{}, false
	}
	return input, true
}

func decodeCreateRoutingPolicyInput(w http.ResponseWriter, r *http.Request) (RoutingPolicyInput, bool) {
	var input RoutingPolicyInput
	if err := decodeJSON(r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_routing_policy", "路由参数格式无效")
		return RoutingPolicyInput{}, false
	}
	input.Name = strings.TrimSpace(input.Name)
	if err := validateCreateRoutingPolicy(input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_routing_policy", err.Error())
		return RoutingPolicyInput{}, false
	}
	return input, true
}

func writeRoutingPolicyError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrRoutingNotFound):
		writeAPIError(w, http.StatusNotFound, "routing_policy_not_found", "路由不存在")
	case errors.Is(err, ErrRoutingInvalidState):
		writeAPIError(w, http.StatusConflict, "routing_policy_conflict", "路由当前状态不允许操作")
	default:
		writeAPIError(w, http.StatusInternalServerError, "routing_policy_store_error", fallback)
	}
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

func (s *server) bindTokenRoutes(r *http.Request, userID int, cookie string, tokenID int, provider string, policyIDs []int64) (TokenRoutingBinding, error) {
	if s.services.Credentials == nil {
		return TokenRoutingBinding{}, errors.New("密钥指纹服务尚未就绪")
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/token/"+strconv.Itoa(tokenID)+"/key", nil, &newapi.Auth{Cookie: cookie, UserID: userID})
	if err != nil {
		return TokenRoutingBinding{}, err
	}
	var revealed struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(data, &revealed); err != nil || strings.TrimSpace(revealed.Key) == "" {
		return TokenRoutingBinding{}, errors.New("核心服务未返回有效密钥")
	}
	return s.services.Routing.CreateTokenBinding(
		r.Context(), userID, tokenID,
		s.services.Credentials.Fingerprint([]byte(publicAPIKey(revealed.Key))), provider, policyIDs,
	)
}

func validateRoutingPolicy(input RoutingPolicyInput) error {
	if name := strings.TrimSpace(input.Name); name == "" || len([]rune(name)) > 50 {
		return errors.New("路由名称需为 1 至 50 个字符")
	}
	if input.Priority != "balanced" && input.Priority != "price" && input.Priority != "success" && input.Priority != "latency" && input.Priority != "cache" {
		return errors.New("优先方式必须是均衡、价格、成功率、响应速度或缓存命中率优先")
	}
	if input.MinimumMultiplierBPS < 0 || input.MaximumMultiplierBPS > 50_000 || input.MinimumMultiplierBPS > input.MaximumMultiplierBPS {
		return errors.New("倍率区间必须在 0 到 50000 基点之间")
	}
	if input.MinimumCacheHitPPM != nil {
		if *input.MinimumCacheHitPPM < 0 || *input.MinimumCacheHitPPM > 1_000_000 {
			return errors.New("最低缓存命中率必须在 0% 到 100% 之间")
		}
	}
	return nil
}

func validateCreateRoutingPolicy(input RoutingPolicyInput) error {
	if input.CanonicalModelID <= 0 {
		return errors.New("请选择有效模型")
	}
	return validateRoutingPolicy(input)
}

func validateCanonicalModel(command CreateCanonicalModelCommand) error {
	if err := validateModelDescriptor(command.PublicName, command.Provider); err != nil {
		return err
	}
	return validateOfficialPrice(command.Price)
}

func validateModelDescriptor(publicName string, provider string) error {
	if publicName == "" || len(publicName) > 255 || strings.HasPrefix(publicName, "mkt-ov-") || provider == "" || len(provider) > 80 {
		return errors.New("模型名称或厂商无效")
	}
	if _, supported := supportedModelProviders[provider]; !supported {
		return errors.New("模型厂商不在平台支持范围")
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
