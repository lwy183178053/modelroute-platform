package httpapi

import (
	"context"
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
	_, session, _, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) {
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
	channel, err := s.buildMultiModelSubmission(r, session, request.ChannelName, request.Provider, request.BaseURL, request.APIKey)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	command := CreateMerchantChannel{UserID: session.UserID, Channel: channel}
	repository, supported := s.services.Merchants.(MultiModelMerchantRepository)
	if !supported {
		writeAPIError(w, http.StatusServiceUnavailable, "multimodel_unavailable", "多模型渠道服务尚未就绪")
		return
	}
	review, err := repository.CreateMultiModelChannelReview(r.Context(), command)
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
	if repository, ok := s.services.Merchants.(MultiModelMerchantRepository); ok {
		for index := range items {
			if models, modelErr := repository.ChannelModels(r.Context(), items[index].ID); modelErr == nil {
				for _, model := range models {
					items[index].Models = append(items[index].Models, model.UpstreamModelName)
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"page": page, "page_size": pageSize, "total": total, "items": items}})
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
	repository, supported := s.services.Merchants.(MultiModelMerchantRepository)
	if !supported || s.services.Routing == nil || s.services.ChannelTests == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "multimodel_unavailable", "多模型审核服务尚未就绪")
		return
	}
	review, err := s.services.Merchants.ClaimReview(r.Context(), reviewID, "verifying")
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	verificationContext, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	go func() {
		defer cancel()
		s.runAdminReviewVerification(verificationContext, repository, review, session.UserID)
	}()
	writeJSON(w, http.StatusAccepted, map[string]any{"success": true, "data": map[string]any{
		"review_id": reviewID, "status": "verifying",
	}})
}

func (s *server) runAdminReviewVerification(ctx context.Context, repository MultiModelMerchantRepository, review MerchantChannelReview, adminID int) {
	bindings, err := repository.ReviewModels(ctx, review.ID)
	if err != nil {
		s.finishAdminReviewVerification(review.ID, adminID, "", "读取审核模型失败")
		return
	}
	credential, err := s.services.Credentials.Decrypt(review.EncryptedCredential)
	if err != nil {
		s.finishAdminReviewVerification(review.ID, adminID, "", "凭据解密失败")
		return
	}
	defer zeroBytes(credential)
	models, err := s.services.Routing.ListCanonicalModels(ctx)
	if err != nil {
		s.finishAdminReviewVerification(review.ID, adminID, "", "读取标准模型失败")
		return
	}
	modelsByID := make(map[int64]CanonicalModel, len(models))
	for _, model := range models {
		modelsByID[model.ID] = model
	}
	passed, failed := 0, 0
	adapterType := review.AdapterType
	firstFailure := ""
	for index, binding := range bindings {
		if binding.MatchStatus != "matched" || binding.CanonicalModelID <= 0 {
			continue
		}
		model, exists := modelsByID[binding.CanonicalModelID]
		status, testError, latency := "failed", "标准模型不存在", 0
		if exists {
			outcome, testErr := s.services.ChannelTests.Test(ctx, ChannelTestRequest{Model: model, BaseURL: review.BaseURL, APIKey: string(credential)})
			latency = outcome.LatencyMS
			if testErr == nil && (review.AdapterType == "" || outcome.AdapterType == review.AdapterType) {
				status, testError = "passed", ""
				adapterType = outcome.AdapterType
			} else if testErr != nil {
				testError = sanitizedChannelError(testErr, string(credential))
			} else {
				testError = "管理员复检得到的内部适配器与商户自测不一致"
			}
		}
		if err := repository.SetReviewModelTest(ctx, review.ID, binding.ID, adminID, status, latency, testError); err != nil {
			s.finishAdminReviewVerification(review.ID, adminID, "", "保存模型检测结果失败")
			return
		}
		bindings[index].TestStatus = status
		bindings[index].TestLatencyMS = latency
		bindings[index].TestError = testError
		if status == "passed" {
			passed++
		} else {
			failed++
			if firstFailure == "" {
				firstFailure = testError
			}
		}
	}
	if passed == 0 {
		failure := "所有已匹配模型验证失败"
		if firstFailure != "" {
			failure += "：" + firstFailure
		}
		s.finishAdminReviewVerification(review.ID, adminID, "", failure)
		return
	}
	s.finishAdminReviewVerification(review.ID, adminID, adapterType, "")
}

func (s *server) finishAdminReviewVerification(reviewID int64, adminID int, adapterType string, failure string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.services.Merchants.FinishVerification(ctx, reviewID, adminID, adapterType, failure)
}

func (s *server) adminApproveApplication(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	id, ok := pathInt64(w, r, "id", "申请编号无效")
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
	repository, supported := s.services.Merchants.(MultiModelMerchantRepository)
	if !supported {
		_ = s.services.Merchants.CancelApproval(r.Context(), reviewID, "多模型审核服务尚未就绪")
		writeAPIError(w, http.StatusServiceUnavailable, "multimodel_unavailable", "多模型审核服务尚未就绪")
		return
	}
	bindings, err := repository.ReviewModels(r.Context(), reviewID)
	if err != nil {
		_ = s.services.Merchants.CancelApproval(r.Context(), reviewID, "读取审核模型失败")
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取审核模型失败")
		return
	}
	models := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		if binding.MatchStatus == "matched" && binding.TestStatus == "passed" && binding.CanonicalModelID > 0 {
			models = append(models, binding.UpstreamModelName)
		}
	}
	if len(models) == 0 {
		failure := "没有验证成功的模型，渠道不能通过"
		_ = s.services.Merchants.CancelApproval(r.Context(), reviewID, failure)
		writeAPIError(w, http.StatusConflict, "review_models_unverified", failure)
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
	channelID, err := s.createNewAPIChannel(r, session, review, channelName, string(credential), models)
	if err != nil {
		failure := sanitizedChannelError(err, string(credential))
		_ = s.services.Merchants.CancelApproval(r.Context(), reviewID, failure)
		writeAPIError(w, http.StatusBadGateway, "channel_import_failed", failure)
		return
	}
	approval, err := s.services.Merchants.FinishApproval(r.Context(), reviewID, session.UserID, channelID)
	if err != nil {
		_, _ = s.services.NewAPI.Do(r.Context(), http.MethodDelete, fmt.Sprintf("/api/channel/%d", channelID), nil, upstreamAuth(session))
		_ = s.services.Merchants.CancelApproval(r.Context(), reviewID, "平台审批事务失败")
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": approval})
}

func (s *server) createNewAPIChannel(r *http.Request, session platformauth.Session, review MerchantChannelReview, channelName string, credential string, models []string) (int, error) {
	channelType, ok := channelTypeForAdapter(review.AdapterType)
	if !ok || len(models) == 0 {
		return 0, errors.New("渠道内部适配器无效")
	}
	baseURL := newAPIChannelBaseURL(channelType, review.BaseURL)
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
	return channelID, nil
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
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	id, ok := pathInt64(w, r, "id", "申请编号无效")
	if !ok {
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &request); err != nil || len(strings.TrimSpace(request.Reason)) < 2 || len(request.Reason) > 500 {
		writeAPIError(w, http.StatusBadRequest, "invalid_reason", "请输入 2 至 500 字的驳回原因")
		return
	}
	if err := s.services.Merchants.RejectApplication(r.Context(), id, session.UserID, strings.TrimSpace(request.Reason)); err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"application_id": id, "status": "rejected"}})
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

func (s *server) buildMultiModelSubmission(r *http.Request, session platformauth.Session, name, provider, baseURL, apiKey string) (ChannelSubmission, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 100 {
		return ChannelSubmission{}, errors.New("渠道名称需为 2 至 100 字")
	}
	provider, err := normalizeChannelProvider(provider)
	if err != nil {
		return ChannelSubmission{}, err
	}
	models, numericType, err := s.fetchAndMatchChannelModels(r, session, provider, baseURL, apiKey)
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
	return ChannelSubmission{ChannelName: name, Provider: provider, ChannelType: numericType, AdapterType: adapterForChannelType(numericType), BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), UpstreamModels: models, EncryptedCredential: ciphertext, CredentialFingerprint: fingerprint}, nil
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
