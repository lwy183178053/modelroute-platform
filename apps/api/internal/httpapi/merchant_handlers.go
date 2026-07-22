package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	if err := s.recordAudit(r, session, "merchant.application.submit", int(application.ID), "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "申请已提交，但审计记录失败")
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
	channel, err := s.buildChannelSubmission(request.ChannelName, request.ChannelType, request.BaseURL, request.APIKey, request.Models)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	review, err := s.services.Merchants.CreateChannelReview(r.Context(), CreateMerchantChannel{UserID: session.UserID, Channel: channel})
	if err != nil {
		s.writeMerchantStoreError(w, err)
		return
	}
	if err := s.recordAudit(r, session, "merchant.channel.submit", int(review.ID), "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "渠道已提交，但审计记录失败")
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
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"page": page, "page_size": pageSize, "total": total, "items": items}})
}

func (s *server) adminMerchants(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.merchantReady(w) {
		return
	}
	page, pageSize := pagination(r)
	items, total, err := s.services.Merchants.ListMerchants(r.Context(), page, pageSize, r.URL.Query().Get("status"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取商户列表失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"page": page, "page_size": pageSize, "total": total, "items": items}})
}

func (s *server) adminMerchantChannels(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.merchantReady(w) {
		return
	}
	page, pageSize := pagination(r)
	items, total, err := s.services.Merchants.ListChannels(r.Context(), page, pageSize, r.URL.Query().Get("status"))
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
	credential, err := s.services.Credentials.Decrypt(review.EncryptedCredential)
	if err != nil {
		_ = s.services.Merchants.FinishVerification(r.Context(), reviewID, session.UserID, nil, "凭据解密失败")
		writeAPIError(w, http.StatusInternalServerError, "credential_error", "读取待审核凭据失败")
		return
	}
	defer zeroBytes(credential)
	data, verifyErr := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/channel/fetch_models", map[string]any{
		"base_url": review.BaseURL, "type": review.ChannelType, "key": string(credential),
	}, upstreamAuth(session))
	if verifyErr != nil {
		failure := "上游模型列表验证失败"
		_ = s.services.Merchants.FinishVerification(r.Context(), reviewID, session.UserID, nil, failure)
		_ = s.recordAudit(r, session, "merchant.channel.verify", int(reviewID), "failed", failure)
		writeAPIError(w, http.StatusUnprocessableEntity, "channel_verification_failed", failure)
		return
	}
	var models []string
	if err := json.Unmarshal(data, &models); err != nil || len(models) == 0 || !modelsContain(models, review.RequestedModels) {
		failure := "上游未返回全部拟售模型"
		_ = s.services.Merchants.FinishVerification(r.Context(), reviewID, session.UserID, models, failure)
		_ = s.recordAudit(r, session, "merchant.channel.verify", int(reviewID), "failed", failure)
		writeAPIError(w, http.StatusUnprocessableEntity, "channel_models_mismatch", failure)
		return
	}
	if err := s.services.Merchants.FinishVerification(r.Context(), reviewID, session.UserID, models, ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "保存验证结果失败")
		return
	}
	if err := s.recordAudit(r, session, "merchant.channel.verify", int(reviewID), "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "验证已完成，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"review_id": reviewID, "models": models}})
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
		_ = s.services.Merchants.CancelApproval(r.Context(), reviewID, "New API 渠道导入或测试失败")
		_ = s.recordAudit(r, session, "merchant.channel.approve", int(reviewID), "failed", "New API 渠道导入或测试失败")
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
	if err := s.recordAudit(r, session, "merchant.channel.approve", int(reviewID), "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "渠道已导入，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": approval})
}

func (s *server) createAndTestNewAPIChannel(r *http.Request, session platformauth.Session, review MerchantChannelReview, channelName string, credential string) (int, error) {
	models := strings.Join(review.RequestedModels, ",")
	testModel := review.RequestedModels[0]
	_, err := s.services.NewAPI.Do(r.Context(), http.MethodPost, "/api/channel/", map[string]any{
		"mode": "single",
		"channel": map[string]any{
			"type": review.ChannelType, "key": credential, "name": channelName,
			"base_url": review.BaseURL, "models": models, "group": "default", "status": 1,
			"test_model": testModel, "tag": fmt.Sprintf("modelroute-u%d", review.UserID),
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
		return 0, errors.New("New API 未返回新渠道编号")
	}
	_, err = s.services.NewAPI.Do(r.Context(), http.MethodGet, fmt.Sprintf("/api/channel/test/%d?%s", channelID, url.Values{"model": []string{testModel}}.Encode()), nil, upstreamAuth(session))
	if err != nil {
		_, _ = s.services.NewAPI.Do(r.Context(), http.MethodDelete, fmt.Sprintf("/api/channel/%d", channelID), nil, upstreamAuth(session))
		return 0, err
	}
	return channelID, nil
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
	if err := s.recordAudit(r, session, "merchant.channel.reject", int(reviewID), "success", strings.TrimSpace(request.Reason)); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "审核已驳回，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"review_id": reviewID, "status": "rejected"}})
}

func (s *server) adminMerchantStatus(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.merchantReady(w) {
		return
	}
	merchantID, ok := pathInt64(w, r, "id", "商户编号无效")
	if !ok {
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
	action := "revoke"
	if *request.Enabled {
		action = "enable"
	}
	if err := s.recordAudit(r, session, "merchant."+action, int(merchantID), "success", strings.TrimSpace(request.Reason)); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "商户状态已更新，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": change})
}

type merchantChannelRequest struct {
	ChannelName string   `json:"channel_name"`
	ChannelType int      `json:"channel_type"`
	BaseURL     string   `json:"base_url"`
	APIKey      string   `json:"api_key"`
	Models      []string `json:"models"`
}

type merchantSubmissionRequest struct {
	DisplayName         string `json:"display_name"`
	WebsiteURL          string `json:"website_url"`
	BusinessDescription string `json:"business_description"`
	merchantChannelRequest
}

func (s *server) buildApplicationCommand(userID int, request merchantSubmissionRequest) (CreateMerchantApplication, error) {
	displayName := strings.TrimSpace(request.DisplayName)
	description := strings.TrimSpace(request.BusinessDescription)
	if len(displayName) < 2 || len(displayName) > 80 {
		return CreateMerchantApplication{}, errors.New("商户名称需为 2 至 80 字")
	}
	if len(description) < 10 || len(description) > 2000 {
		return CreateMerchantApplication{}, errors.New("业务说明需为 10 至 2000 字")
	}
	website, err := normalizePublicHTTPSURL(request.WebsiteURL, false)
	if err != nil {
		return CreateMerchantApplication{}, errors.New("网站地址必须是公开 HTTPS 地址")
	}
	channel, err := s.buildChannelSubmission(request.ChannelName, request.ChannelType, request.BaseURL, request.APIKey, request.Models)
	if err != nil {
		return CreateMerchantApplication{}, err
	}
	return CreateMerchantApplication{UserID: userID, DisplayName: displayName, WebsiteURL: website, BusinessDescription: description, Channel: channel}, nil
}

func (s *server) buildChannelSubmission(name string, channelType int, baseURL string, apiKey string, models []string) (ChannelSubmission, error) {
	name = strings.TrimSpace(name)
	credential := []byte(strings.TrimSpace(apiKey))
	if len(name) < 2 || len(name) > 100 {
		return ChannelSubmission{}, errors.New("渠道名称需为 2 至 100 字")
	}
	if !supportedMerchantChannelType(channelType) {
		return ChannelSubmission{}, errors.New("暂不支持该渠道类型")
	}
	normalizedBase, err := normalizePublicHTTPSURL(baseURL, channelType == 1)
	if err != nil {
		return ChannelSubmission{}, errors.New("基础地址必须是公开 HTTPS 地址")
	}
	if len(credential) < 8 || len(credential) > 65536 {
		return ChannelSubmission{}, errors.New("API 密钥长度无效")
	}
	models = normalizeModels(models)
	if len(models) == 0 || len(models) > 100 {
		return ChannelSubmission{}, errors.New("拟售模型数量需为 1 至 100 个")
	}
	ciphertext, err := s.services.Credentials.Encrypt(credential)
	if err != nil {
		zeroBytes(credential)
		return ChannelSubmission{}, errors.New("加密渠道凭据失败")
	}
	fingerprint := s.services.Credentials.Fingerprint(credential)
	zeroBytes(credential)
	return ChannelSubmission{ChannelName: name, ChannelType: channelType, BaseURL: normalizedBase, Models: models, EncryptedCredential: ciphertext, CredentialFingerprint: fingerprint}, nil
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
	if session.Role >= roleAdmin && errors.Is(err, ErrMerchantNotFound) {
		return token, session, MerchantIdentity{UserID: session.UserID, Status: "active", IsMerchant: true}, true
	}
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

func normalizeModels(models []string) []string {
	seen := make(map[string]struct{}, len(models))
	result := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" || len(model) > 255 {
			continue
		}
		if _, exists := seen[model]; exists {
			continue
		}
		seen[model] = struct{}{}
		result = append(result, model)
	}
	return result
}

func modelsContain(have []string, requested []string) bool {
	set := make(map[string]struct{}, len(have))
	for _, model := range have {
		set[model] = struct{}{}
	}
	for _, model := range requested {
		if _, ok := set[model]; !ok {
			return false
		}
	}
	return true
}

func supportedMerchantChannelType(value int) bool {
	switch value {
	case 1, 14, 20, 24, 25, 27, 40, 42, 43, 48:
		return true
	default:
		return false
	}
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
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func zeroBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
