package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

type merchantChannelTestRequest struct {
	CanonicalModelID int64  `json:"canonical_model_id"`
	BaseURL          string `json:"base_url"`
	APIKey           string `json:"api_key"`
}

func (s *server) createMerchantChannelTest(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	if s.services.Routing == nil || s.services.ChannelTests == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "channel_test_unavailable", "渠道检测服务尚未就绪")
		return
	}
	var request merchantChannelTestRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "渠道检测格式无效")
		return
	}
	baseURL, err := normalizePublicHTTPSURL(request.BaseURL, false)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "基础地址必须是公开 HTTPS 地址")
		return
	}
	credential := []byte(strings.TrimSpace(request.APIKey))
	if request.CanonicalModelID <= 0 || len(credential) < 8 || len(credential) > 65536 {
		zeroBytes(credential)
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "模型或 API 密钥无效")
		return
	}
	models, err := s.services.Routing.ListCanonicalModels(r.Context())
	if err != nil {
		zeroBytes(credential)
		writeAPIError(w, http.StatusInternalServerError, "model_store_error", "读取标准模型失败")
		return
	}
	var model CanonicalModel
	for _, candidate := range models {
		if candidate.ID == request.CanonicalModelID && candidate.Status == "active" {
			model = candidate
			break
		}
	}
	if model.ID == 0 {
		zeroBytes(credential)
		writeAPIError(w, http.StatusBadRequest, "invalid_model", "标准模型不存在或未上架")
		return
	}
	fingerprint := s.services.Credentials.Fingerprint(credential)
	outcome, err := s.services.ChannelTests.Test(r.Context(), ChannelTestRequest{Model: model, BaseURL: baseURL, APIKey: string(credential)})
	zeroBytes(credential)
	if err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, "channel_test_failed", "渠道发送 hi 检测失败")
		return
	}
	rawReceipt := make([]byte, 32)
	if _, err := rand.Read(rawReceipt); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "receipt_error", "生成检测凭证失败")
		return
	}
	receipt := base64.RawURLEncoding.EncodeToString(rawReceipt)
	for index := range rawReceipt {
		rawReceipt[index] = 0
	}
	expiresAt := time.Now().UTC().Add(15 * time.Minute)
	record, err := s.services.Merchants.SaveChannelTest(r.Context(), CreateChannelTest{
		UserID: session.UserID, CanonicalModelID: model.ID, BaseURL: baseURL,
		CredentialFingerprint: fingerprint, AdapterType: outcome.AdapterType,
		Capabilities: outcome.Capabilities, LatencyMS: outcome.LatencyMS,
		ReceiptTokenHash: channelTestReceiptHash(receipt), ExpiresAt: expiresAt,
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "保存渠道检测结果失败")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": map[string]any{
		"id": record.ID, "canonical_model_id": record.CanonicalModelID,
		"adapter_type": record.AdapterType, "capabilities": record.Capabilities,
		"latency_ms": record.LatencyMS, "expires_at": record.ExpiresAt, "test_receipt": receipt,
	}})
}

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

func (s *server) merchantChannels(w http.ResponseWriter, r *http.Request) {
	_, session, identity, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	channels, err := s.services.Merchants.ChannelsForUser(r.Context(), session.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取商户渠道失败")
		return
	}
	reviews, err := s.services.Merchants.ChannelReviewsForUser(r.Context(), session.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取渠道审核记录失败")
		return
	}
	if repository, ok := s.services.Merchants.(MultiModelMerchantRepository); ok {
		for index := range channels {
			if models, modelErr := repository.ChannelModels(r.Context(), channels[index].ID); modelErr == nil {
				for _, model := range models {
					channels[index].Models = append(channels[index].Models, model.UpstreamModelName)
				}
			}
		}
		for index := range reviews {
			if models, modelErr := repository.ReviewModels(r.Context(), reviews[index].ID); modelErr == nil {
				reviews[index].Models = models
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"merchant": identity, "items": channels, "reviews": reviews}})
}

func (s *server) createMerchantChannel(w http.ResponseWriter, r *http.Request) {
	_, session, _, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	var request merchantChannelRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "渠道申请格式无效")
		return
	}
	var channel ChannelSubmission
	var err error
	if strings.TrimSpace(request.ChannelType) != "" {
		channel, err = s.buildMultiModelSubmission(r, session, request.ChannelName, request.ChannelType, request.BaseURL, request.APIKey)
	} else {
		channel, err = s.buildChannelSubmission(request.ChannelName, request.CanonicalModelID, request.BaseURL, request.APIKey, request.TestReceipt)
	}
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	command := CreateMerchantChannel{UserID: session.UserID, Channel: channel}
	var review MerchantChannelReview
	if len(channel.UpstreamModels) > 0 {
		repository, supported := s.services.Merchants.(MultiModelMerchantRepository)
		if !supported {
			writeAPIError(w, http.StatusServiceUnavailable, "multimodel_unavailable", "多模型渠道服务尚未就绪")
			return
		}
		review, err = repository.CreateMultiModelChannelReview(r.Context(), command)
	} else {
		review, err = s.services.Merchants.CreateChannelReview(r.Context(), command)
	}
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": review})
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

func (s *server) adminChannelReviews(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.merchantReady(w) {
		return
	}
	page, pageSize := pagination(r)
	items, total, err := s.services.Merchants.ListChannelReviews(r.Context(), page, pageSize, r.URL.Query().Get("status"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取渠道审核失败")
		return
	}
	if repository, ok := s.services.Merchants.(MultiModelMerchantRepository); ok {
		for index := range items {
			if models, modelErr := repository.ReviewModels(r.Context(), items[index].ID); modelErr == nil {
				items[index].Models = models
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"page": page, "page_size": pageSize, "total": total, "items": items}})
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
		items, total, err = repository.ListChannelsFiltered(r.Context(), page, pageSize, r.URL.Query().Get("status"), r.URL.Query().Get("keyword"), merchantUserID)
	} else {
		items, total, err = s.services.Merchants.ListChannels(r.Context(), page, pageSize, r.URL.Query().Get("status"))
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取渠道列表失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"page": page, "page_size": pageSize, "total": total, "items": items}})
}

func (s *server) adminVerifyApplication(w http.ResponseWriter, r *http.Request) {
	application, ok := s.requireAdminApplication(w, r)
	if ok {
		s.adminVerifyReviewID(w, r, application.InitialReview.ID)
	}
}

func (s *server) adminVerifyReview(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt64(w, r, "id", "渠道审核编号无效")
	if ok {
		s.adminVerifyReviewID(w, r, id)
	}
}

func (s *server) adminVerifyReviewID(w http.ResponseWriter, r *http.Request, reviewID int64) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	review, err := s.services.Merchants.ClaimReview(r.Context(), reviewID, "verifying")
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	if repository, ok := s.services.Merchants.(MultiModelMerchantRepository); ok {
		if bindings, modelErr := repository.ReviewModels(r.Context(), reviewID); modelErr == nil && len(bindings) > 0 {
			if finishErr := s.services.Merchants.FinishVerification(r.Context(), reviewID, session.UserID, review.AdapterType, ""); finishErr != nil {
				writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "保存审核状态失败")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
				"review_id": reviewID, "adapter_type": review.AdapterType, "models": bindings,
			}})
			return
		}
	}
	credential, err := s.services.Credentials.Decrypt(review.EncryptedCredential)
	if err != nil {
		_ = s.services.Merchants.FinishVerification(r.Context(), reviewID, session.UserID, "", "凭据解密失败")
		writeAPIError(w, http.StatusInternalServerError, "credential_error", "读取待审核凭据失败")
		return
	}
	defer zeroBytes(credential)
	if s.services.ChannelTests == nil {
		_ = s.services.Merchants.FinishVerification(r.Context(), reviewID, session.UserID, "", "渠道检测服务尚未就绪")
		writeAPIError(w, http.StatusServiceUnavailable, "channel_test_unavailable", "渠道检测服务尚未就绪")
		return
	}
	outcome, verifyErr := s.services.ChannelTests.Test(r.Context(), ChannelTestRequest{
		Model: CanonicalModel{
			ID: review.CanonicalModelID, PublicName: review.Model, Provider: review.Provider,
			Status: "active", Price: OfficialPrice{PriceKind: review.PriceKind},
		},
		BaseURL: review.BaseURL, APIKey: string(credential),
	})
	if verifyErr != nil {
		failure := "管理员发送 hi 复检失败"
		_ = s.services.Merchants.FinishVerification(r.Context(), reviewID, session.UserID, "", failure)
		writeAPIError(w, http.StatusUnprocessableEntity, "channel_verification_failed", failure)
		return
	}
	if outcome.AdapterType != review.AdapterType {
		failure := "管理员复检得到的适配器与商户自测不一致"
		_ = s.services.Merchants.FinishVerification(r.Context(), reviewID, session.UserID, "", failure)
		writeAPIError(w, http.StatusUnprocessableEntity, "channel_adapter_mismatch", failure)
		return
	}
	if err := s.services.Merchants.FinishVerification(r.Context(), reviewID, session.UserID, outcome.AdapterType, ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "保存验证结果失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"review_id": reviewID, "adapter_type": outcome.AdapterType,
		"capabilities": outcome.Capabilities, "latency_ms": outcome.LatencyMS,
	}})
}

func (s *server) adminApproveApplication(w http.ResponseWriter, r *http.Request) {
	application, ok := s.requireAdminApplication(w, r)
	if ok {
		s.adminApproveReviewID(w, r, application.InitialReview.ID)
	}
}

func (s *server) adminApproveReview(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt64(w, r, "id", "渠道审核编号无效")
	if ok {
		s.adminApproveReviewID(w, r, id)
	}
}

func (s *server) adminApproveReviewID(w http.ResponseWriter, r *http.Request, reviewID int64) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	review, err := s.services.Merchants.ClaimReview(r.Context(), reviewID, "approving")
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	credential, err := s.services.Credentials.Decrypt(review.EncryptedCredential)
	if err != nil {
		_ = s.services.Merchants.CancelApproval(r.Context(), reviewID, "凭据解密失败")
		writeAPIError(w, http.StatusInternalServerError, "credential_error", "读取待审核凭据失败")
		return
	}
	defer zeroBytes(credential)
	channelName := fmt.Sprintf("ModelRoute-R%d-U%d", review.ID, review.UserID)
	channelID, err := s.createAndTestNewAPIChannel(r, session, review, channelName, string(credential))
	if err != nil {
		_ = s.services.Merchants.CancelApproval(r.Context(), reviewID, "核心服务渠道导入或测试失败")
		writeAPIError(w, http.StatusBadGateway, "channel_import_failed", "渠道导入或真实请求测试失败")
		return
	}
	approval, err := s.services.Merchants.FinishApproval(r.Context(), reviewID, session.UserID, channelID)
	if err != nil {
		_, _ = s.services.NewAPI.Do(r.Context(), http.MethodDelete, fmt.Sprintf("/api/channel/%d", channelID), nil, upstreamAuth(session))
		_ = s.services.Merchants.CancelApproval(r.Context(), reviewID, "平台审批事务失败")
		s.writeMerchantStoreError(w, err)
		return
	}
	if repository, ok := s.services.Merchants.(MultiModelMerchantRepository); ok {
		if err := repository.PromoteReviewModels(r.Context(), reviewID, approval.ChannelID); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "保存正式渠道模型失败")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": approval})
}

func (s *server) createAndTestNewAPIChannel(r *http.Request, session platformauth.Session, review MerchantChannelReview, channelName string, credential string) (int, error) {
	channelType, ok := channelTypeForAdapter(review.AdapterType)
	if !ok || review.Model == "" {
		return 0, errors.New("渠道内部适配器无效")
	}
	baseURL := newAPIChannelBaseURL(channelType, review.BaseURL)
	models := []string{review.Model}
	if repository, ok := s.services.Merchants.(MultiModelMerchantRepository); ok {
		if bindings, modelErr := repository.ReviewModels(r.Context(), review.ID); modelErr == nil {
			models = models[:0]
			for _, binding := range bindings {
				if binding.MatchStatus == "matched" && binding.CanonicalModelID > 0 {
					models = append(models, binding.UpstreamModelName)
				}
			}
			if len(models) == 0 {
				models = []string{review.Model}
			}
		}
	}
	modelList := strings.Join(models, ",")
	_, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/channel/", map[string]any{
		"mode": "single",
		"channel": map[string]any{
			"type": channelType, "key": credential, "name": channelName,
			"base_url": baseURL, "models": modelList, "group": "default", "status": 1,
			"test_model": models[0], "tag": fmt.Sprintf("modelroute-u%d", review.UserID),
		},
	}, upstreamAuth(session))
	if err != nil {
		return 0, err
	}
	data, err := s.services.NewAPI.Do(r.Context(), http.MethodGet, "/api/channel/search?"+url.Values{
		"keyword": []string{channelName}, "p": []string{"1"}, "page_size": []string{"20"}, "id_sort": []string{"true"},
	}.Encode(), nil, upstreamAuth(session))
	if err != nil {
		return 0, err
	}
	var result struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, err
	}
	channelID := 0
	for _, channel := range result.Items {
		if channel.Name == channelName {
			channelID = channel.ID
			break
		}
	}
	if channelID <= 0 {
		return 0, errors.New("核心服务未返回新渠道编号")
	}
	_, err = s.services.NewAPI.Do(r.Context(), http.MethodGet, fmt.Sprintf("/api/channel/test/%d?%s", channelID, url.Values{"model": []string{models[0]}}.Encode()), nil, upstreamAuth(session))
	if err != nil {
		_, _ = s.services.NewAPI.Do(r.Context(), http.MethodDelete, fmt.Sprintf("/api/channel/%d", channelID), nil, upstreamAuth(session))
		return 0, err
	}
	return channelID, nil
}

func channelTypeForAdapter(adapter string) (int, bool) {
	for _, candidate := range []channelAdapter{
		{name: "openai_compatible", channelType: 1},
		{name: "anthropic", channelType: 14},
		{name: "gemini", channelType: 24},
	} {
		if candidate.name == adapter {
			return candidate.channelType, true
		}
	}
	return 0, false
}

func (s *server) adminRejectApplication(w http.ResponseWriter, r *http.Request) {
	application, ok := s.requireAdminApplication(w, r)
	if ok {
		s.adminRejectReviewID(w, r, application.InitialReview.ID)
	}
}

func (s *server) adminRejectReview(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt64(w, r, "id", "渠道审核编号无效")
	if ok {
		s.adminRejectReviewID(w, r, id)
	}
}

func (s *server) adminRejectReviewID(w http.ResponseWriter, r *http.Request, reviewID int64) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &request); err != nil || len(strings.TrimSpace(request.Reason)) < 2 || len(request.Reason) > 500 {
		writeAPIError(w, http.StatusBadRequest, "invalid_reason", "请输入 2 至 500 字的驳回原因")
		return
	}
	if err := s.services.Merchants.RejectReview(r.Context(), reviewID, session.UserID, strings.TrimSpace(request.Reason)); err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"review_id": reviewID, "status": "rejected"}})
}

func (s *server) adminUserMerchantStatus(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	if s.services.Merchants == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "service_not_ready", "商户服务正在初始化")
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
	merchantID := identity.MerchantID
	var request struct {
		Enabled *bool  `json:"enabled"`
		Reason  string `json:"reason"`
	}
	if err := decodeJSON(r, &request); err != nil || request.Enabled == nil || len(strings.TrimSpace(request.Reason)) < 2 {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "状态和原因不能为空")
		return
	}
	merchant, err := s.services.Merchants.Merchant(r.Context(), merchantID)
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	changedUpstream := make([]int, 0)
	if !*request.Enabled {
		for _, channelID := range merchant.NewAPIChannelIDs {
			if _, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, fmt.Sprintf("/api/channel/%d/status", channelID), map[string]int{"status": 2}, upstreamAuth(session)); err != nil {
				for _, rollbackID := range changedUpstream {
					_, _ = s.services.NewAPI.Do(r.Context(), http.MethodPost, fmt.Sprintf("/api/channel/%d/status", rollbackID), map[string]int{"status": 1}, upstreamAuth(session))
				}
				writeAPIError(w, http.StatusBadGateway, "channel_disable_failed", "停用商户渠道失败")
				return
			}
			changedUpstream = append(changedUpstream, channelID)
		}
	}
	change, err := s.services.Merchants.SetMerchantStatus(r.Context(), merchantID, *request.Enabled, session.UserID, strings.TrimSpace(request.Reason))
	if err != nil {
		for _, channelID := range changedUpstream {
			_, _ = s.services.NewAPI.Do(r.Context(), http.MethodPost, fmt.Sprintf("/api/channel/%d/status", channelID), map[string]int{"status": 1}, upstreamAuth(session))
		}
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"user_id": change.UserID, "status": change.Status}})
}

type merchantChannelRequest struct {
	ChannelName      string `json:"channel_name"`
	CanonicalModelID int64  `json:"canonical_model_id"`
	TestReceipt      string `json:"test_receipt"`
	ChannelType      string `json:"channel_type"`
	BaseURL          string `json:"base_url"`
	APIKey           string `json:"api_key"`
}

type merchantSubmissionRequest struct {
	DisplayName         string `json:"display_name"`
	WebsiteURL          string `json:"website_url"`
	BusinessDescription string `json:"business_description"`
	merchantChannelRequest
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
	if len(description) < 10 || len(description) > 2000 {
		return CreateMerchantApplication{}, errors.New("业务说明需为 10 至 2000 字")
	}
	website, err := normalizePublicHTTPSURL(request.WebsiteURL, false)
	if err != nil {
		return CreateMerchantApplication{}, errors.New("网站地址必须是公开 HTTPS 地址")
	}
	channel, err := s.buildChannelSubmission(request.ChannelName, request.CanonicalModelID, request.BaseURL, request.APIKey, request.TestReceipt)
	if err != nil {
		return CreateMerchantApplication{}, err
	}
	return CreateMerchantApplication{UserID: userID, DisplayName: displayName.Value, DisplayNameKey: displayName.Key, WebsiteURL: website, BusinessDescription: description, Channel: channel}, nil
}

func (s *server) buildChannelSubmission(name string, canonicalModelID int64, baseURL string, apiKey string, testReceipt string) (ChannelSubmission, error) {
	name = strings.TrimSpace(name)
	testReceipt = strings.TrimSpace(testReceipt)
	credential := []byte(strings.TrimSpace(apiKey))
	if len(name) < 2 || len(name) > 100 {
		return ChannelSubmission{}, errors.New("渠道名称需为 2 至 100 字")
	}
	if canonicalModelID <= 0 || len(testReceipt) < 16 || len(testReceipt) > 256 {
		return ChannelSubmission{}, errors.New("必须选择模型并先完成渠道检测")
	}
	normalizedBase, err := normalizePublicHTTPSURL(baseURL, false)
	if err != nil {
		return ChannelSubmission{}, errors.New("基础地址必须是公开 HTTPS 地址")
	}
	if len(credential) < 8 || len(credential) > 65536 {
		return ChannelSubmission{}, errors.New("API 密钥长度无效")
	}
	ciphertext, err := s.services.Credentials.Encrypt(credential)
	if err != nil {
		zeroBytes(credential)
		return ChannelSubmission{}, errors.New("加密渠道凭据失败")
	}
	fingerprint := s.services.Credentials.Fingerprint(credential)
	zeroBytes(credential)
	return ChannelSubmission{
		ChannelName: name, CanonicalModelID: canonicalModelID, TestReceipt: testReceipt,
		BaseURL: normalizedBase, EncryptedCredential: ciphertext, CredentialFingerprint: fingerprint,
	}, nil
}

func (s *server) buildMultiModelSubmission(r *http.Request, session platformauth.Session, name, channelType, baseURL, apiKey string) (ChannelSubmission, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 100 {
		return ChannelSubmission{}, errors.New("渠道名称需为 2 至 100 字")
	}
	models, numericType, err := s.fetchAndMatchChannelModels(r, session, channelType, baseURL, apiKey)
	if err != nil {
		return ChannelSubmission{}, err
	}
	credential := []byte(strings.TrimSpace(apiKey))
	ciphertext, err := s.services.Credentials.Encrypt(credential)
	if err != nil {
		zeroBytes(credential)
		return ChannelSubmission{}, errors.New("加密渠道凭据失败")
	}
	fingerprint := s.services.Credentials.Fingerprint(credential)
	zeroBytes(credential)
	return ChannelSubmission{ChannelName: name, ChannelType: numericType, AdapterType: adapterForChannelType(numericType), BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), UpstreamModels: models, EncryptedCredential: ciphertext, CredentialFingerprint: fingerprint}, nil
}

func (s *server) requireAdminApplication(w http.ResponseWriter, r *http.Request) (MerchantApplication, bool) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return MerchantApplication{}, false
	}
	id, ok := pathInt64(w, r, "id", "申请编号无效")
	if !ok {
		return MerchantApplication{}, false
	}
	application, err := s.services.Merchants.Application(r.Context(), id)
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return MerchantApplication{}, false
	}
	return application, true
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

func pathInt64(w http.ResponseWriter, r *http.Request, name string, message string) (int64, bool) {
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
