package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/secretbox"
)

func TestMerchantChannelResponsesHideInternalLegacyFields(t *testing.T) {
	review := MerchantChannelReview{
		ChannelType:      14,
		CanonicalModelID: 7,
		Model:            "claude-sonnet-4-6",
		AdapterType:      "anthropic",
		RequestedModels:  []string{"claude-sonnet-4-6"},
		VerifiedModels:   []string{"claude-sonnet-4-6"},
	}
	channel := MerchantChannel{
		MerchantID:       31,
		UserID:           7,
		MerchantName:     "北辰云智",
		NewAPIChannelID:  41,
		ChannelType:      14,
		CanonicalModelID: 7,
		Model:            "claude-sonnet-4-6",
		AdapterType:      "anthropic",
		Models:           []string{"claude-sonnet-4-6"},
	}

	payload, err := json.Marshal(struct {
		Review  MerchantChannelReview `json:"review"`
		Channel MerchantChannel       `json:"channel"`
	}{Review: review, Channel: channel})
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, field := range []string{"channel_type", "adapter_type", "requested_models", "verified_models", "models", "merchant_id", "newapi_channel_id"} {
		if strings.Contains(text, `"`+field+`"`) {
			t.Fatalf("V2 公共响应仍暴露内部字段 %s：%s", field, text)
		}
	}
	for _, field := range []string{"canonical_model_id", "model", "provider", "user_id", "merchant_name", "matched_models"} {
		if !strings.Contains(text, `"`+field+`"`) {
			t.Fatalf("V2 公共响应缺少单模型字段 %s：%s", field, text)
		}
	}
}

func TestMerchantNameConflictReturnsClearMessage(t *testing.T) {
	response := httptest.NewRecorder()
	new(server).writeMerchantStoreError(response, ErrMerchantNameConflict)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"merchant_name_conflict"`) ||
		!strings.Contains(response.Body.String(), "商户名称已被使用") {
		t.Fatalf("重名响应=%d %s", response.Code, response.Body.String())
	}
}

type filteredMerchantRepositoryStub struct {
	*merchantRepositoryStub
	listChannelsFiltered func(context.Context, int, int, string, string, int) ([]MerchantChannel, int, error)
}

func (s *filteredMerchantRepositoryStub) ListChannelsFiltered(ctx context.Context, page int, pageSize int, status, keyword string, userID int) ([]MerchantChannel, int, error) {
	return s.listChannelsFiltered(ctx, page, pageSize, status, keyword, userID)
}

type merchantRepositoryStub struct {
	MerchantRepository
	identity           func(context.Context, int) (MerchantIdentity, error)
	channels           func(context.Context, int) ([]MerchantChannel, error)
	createApplication  func(context.Context, CreateMerchantApplication) (MerchantApplication, error)
	application        func(context.Context, int64) (MerchantApplication, error)
	approveApplication func(context.Context, int64, int) (MerchantProfile, error)
	rejectApplication  func(context.Context, int64, int, string) error
	claimReview        func(context.Context, int64, string) (MerchantChannelReview, error)
	finishVerify       func(context.Context, int64, int, string, string) error
	finishApproval     func(context.Context, int64, int, int) (MerchantApproval, error)
	cancelApproval     func(context.Context, int64, string) error
	saveChannelTest    func(context.Context, CreateChannelTest) (ChannelTest, error)
	merchant           func(context.Context, int64) (MerchantProfile, error)
	updateDescription  func(context.Context, int64, string) (MerchantProfile, error)
	setMerchantStatus  func(context.Context, int64, bool, int, string) (MerchantStatusChange, error)
}

func (s *merchantRepositoryStub) SaveChannelTest(ctx context.Context, command CreateChannelTest) (ChannelTest, error) {
	return s.saveChannelTest(ctx, command)
}

type channelTesterStub struct {
	test func(context.Context, ChannelTestRequest) (ChannelTestOutcome, error)
}

func (s channelTesterStub) Test(ctx context.Context, request ChannelTestRequest) (ChannelTestOutcome, error) {
	return s.test(ctx, request)
}

func (s *merchantRepositoryStub) Identity(ctx context.Context, userID int) (MerchantIdentity, error) {
	if s.identity == nil {
		return MerchantIdentity{}, ErrMerchantNotFound
	}
	return s.identity(ctx, userID)
}

func (s *merchantRepositoryStub) CreateApplication(ctx context.Context, command CreateMerchantApplication) (MerchantApplication, error) {
	return s.createApplication(ctx, command)
}

func (s *merchantRepositoryStub) Application(ctx context.Context, applicationID int64) (MerchantApplication, error) {
	return s.application(ctx, applicationID)
}

func (s *merchantRepositoryStub) ApproveApplication(ctx context.Context, applicationID int64, adminID int) (MerchantProfile, error) {
	return s.approveApplication(ctx, applicationID, adminID)
}

func (s *merchantRepositoryStub) RejectApplication(ctx context.Context, applicationID int64, adminID int, reason string) error {
	return s.rejectApplication(ctx, applicationID, adminID, reason)
}

func (s *merchantRepositoryStub) ChannelsForUser(ctx context.Context, userID int) ([]MerchantChannel, error) {
	if s.channels == nil {
		return nil, nil
	}
	return s.channels(ctx, userID)
}

func (s *merchantRepositoryStub) ClaimReview(ctx context.Context, reviewID int64, targetStatus string) (MerchantChannelReview, error) {
	return s.claimReview(ctx, reviewID, targetStatus)
}

func (s *merchantRepositoryStub) FinishVerification(ctx context.Context, reviewID int64, adminID int, adapterType string, failure string) error {
	return s.finishVerify(ctx, reviewID, adminID, adapterType, failure)
}

func (s *merchantRepositoryStub) FinishApproval(ctx context.Context, reviewID int64, adminID int, channelID int) (MerchantApproval, error) {
	return s.finishApproval(ctx, reviewID, adminID, channelID)
}

func (s *merchantRepositoryStub) CancelApproval(ctx context.Context, reviewID int64, failure string) error {
	if s.cancelApproval == nil {
		return nil
	}
	return s.cancelApproval(ctx, reviewID, failure)
}

func (s *merchantRepositoryStub) Merchant(ctx context.Context, merchantID int64) (MerchantProfile, error) {
	return s.merchant(ctx, merchantID)
}

func (s *merchantRepositoryStub) UpdateMerchantDescription(ctx context.Context, merchantID int64, description string) (MerchantProfile, error) {
	return s.updateDescription(ctx, merchantID, description)
}

func (s *merchantRepositoryStub) SetMerchantStatus(ctx context.Context, merchantID int64, enabled bool, adminID int, reason string) (MerchantStatusChange, error) {
	return s.setMerchantStatus(ctx, merchantID, enabled, adminID, reason)
}

func testCredentialBox(t *testing.T) *secretbox.Box {
	t.Helper()
	box, err := secretbox.New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func createPlatformSession(t *testing.T, services Services, userID int, role int) string {
	t.Helper()
	token, err := services.Sessions.Create(context.Background(), platformauth.Session{
		UserID: userID, Username: "user", Role: role, Status: 1, UpstreamCookie: "session=upstream",
	})
	if err != nil {
		t.Fatalf("创建平台会话失败：%v", err)
	}
	return token
}

func TestMerchantApplicationOnlyStoresMerchantProfileFields(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("提交申请不应调用 New API")
	}))
	repository := &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
		return MerchantIdentity{}, ErrMerchantNotFound
	}}
	repository.createApplication = func(_ context.Context, command CreateMerchantApplication) (MerchantApplication, error) {
		if command.UserID != 7 || command.DisplayName != "北辰云智" || command.WebsiteURL != "https://merchant.example" || command.BusinessDescription != "企业模型 API 服务" {
			t.Fatalf("申请参数=%+v", command)
		}
		return MerchantApplication{ID: 11, UserID: 7, DisplayName: command.DisplayName, WebsiteURL: command.WebsiteURL, BusinessDescription: command.BusinessDescription, Status: "pending"}, nil
	}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	body := `{"display_name":"北辰云智","website_url":"https://merchant.example","business_description":"企业模型 API 服务"}`
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/application", strings.NewReader(body))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("状态码=%d，响应=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "initial_channel") || strings.Contains(response.Body.String(), "credential_configured") {
		t.Fatalf("商户申请仍包含渠道信息：%s", response.Body.String())
	}
}

func TestMerchantChannelTestRequiresActiveMerchant(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("普通用户不应触发渠道检测")
	}))
	services.Merchants = &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
		return MerchantIdentity{}, ErrMerchantNotFound
	}}
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/channel-tests", strings.NewReader(`{"canonical_model_id":61,"base_url":"https://api.merchant.example","api_key":"sk-upstream-secret"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"code":"merchant_required"`) {
		t.Fatalf("普通用户渠道检测响应=%d %s", response.Code, response.Body.String())
	}
}

func TestAdminApprovesMerchantApplicationWithoutCreatingChannel(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("商户身份审批不应调用渠道服务")
	}))
	repository := &merchantRepositoryStub{
		application: func(_ context.Context, applicationID int64) (MerchantApplication, error) {
			return MerchantApplication{ID: applicationID, UserID: 7, Status: "pending"}, nil
		},
		approveApplication: func(_ context.Context, applicationID int64, adminID int) (MerchantProfile, error) {
			if applicationID != 11 || adminID != 1 {
				t.Fatalf("审批参数=%d/%d", applicationID, adminID)
			}
			return MerchantProfile{ID: 31, UserID: 7, DisplayName: "北辰云智", BusinessDescription: "企业模型 API 服务", Status: "active"}, nil
		},
	}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 1, 10)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/merchant-applications/11/approve", nil)
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"display_name":"北辰云智"`) {
		t.Fatalf("商户审批响应=%d %s", response.Code, response.Body.String())
	}
}

func TestMerchantProfileCanOnlyUpdateDescription(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &merchantRepositoryStub{
		identity: func(_ context.Context, userID int) (MerchantIdentity, error) {
			return MerchantIdentity{MerchantID: 31, UserID: userID, IsMerchant: true, Status: "active"}, nil
		},
		merchant: func(_ context.Context, merchantID int64) (MerchantProfile, error) {
			return MerchantProfile{ID: merchantID, UserID: 7, DisplayName: "北辰云智", WebsiteURL: "https://merchant.example", BusinessDescription: "原商户介绍", Status: "active"}, nil
		},
		updateDescription: func(_ context.Context, merchantID int64, description string) (MerchantProfile, error) {
			if merchantID != 31 || description != "更新后的商户介绍内容" {
				t.Fatalf("资料更新参数=%d/%q", merchantID, description)
			}
			return MerchantProfile{ID: merchantID, UserID: 7, DisplayName: "北辰云智", WebsiteURL: "https://merchant.example", BusinessDescription: description, Status: "active"}, nil
		},
	}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPut, "/api/merchant/profile", strings.NewReader(`{"business_description":"更新后的商户介绍内容"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"business_description":"更新后的商户介绍内容"`) {
		t.Fatalf("商户资料响应=%d %s", response.Code, response.Body.String())
	}
}

func TestMerchantApplicationVerifyRouteIsRemoved(t *testing.T) {
	handler := NewServerWithServices("test", Dependencies{}, Services{})
	request := httptest.NewRequest(http.MethodPost, "/api/admin/merchant-applications/11/verify", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("旧商户验证入口状态码=%d，响应=%s", response.Code, response.Body.String())
	}
}

func TestMerchantChannelTestReturnsBoundReceiptWithoutPersistingCredential(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("渠道检测器已隔离，不应走用户 New API 会话")
	}))
	services.Routing = &routingRepositoryStub{listModels: func(context.Context) ([]CanonicalModel, error) {
		return []CanonicalModel{{ID: 61, PublicName: "claude-fable-5", Provider: "Anthropic", Status: "active", Price: OfficialPrice{PriceKind: "text"}}}, nil
	}}
	services.ChannelTests = channelTesterStub{test: func(_ context.Context, request ChannelTestRequest) (ChannelTestOutcome, error) {
		if request.Model.ID != 61 || request.BaseURL != "https://api.merchant.example" || request.APIKey != "sk-upstream-secret" {
			t.Fatalf("检测请求=%+v", request)
		}
		return ChannelTestOutcome{AdapterType: "anthropic", Capabilities: []string{"responses"}, LatencyMS: 125}, nil
	}}
	repository := &merchantRepositoryStub{identity: func(_ context.Context, userID int) (MerchantIdentity, error) {
		return MerchantIdentity{MerchantID: 31, UserID: userID, DisplayName: "北辰云智", Status: "active", IsMerchant: true}, nil
	}}
	var savedReceiptHash string
	repository.saveChannelTest = func(_ context.Context, command CreateChannelTest) (ChannelTest, error) {
		if command.UserID != 7 || command.CanonicalModelID != 61 || command.AdapterType != "anthropic" || command.LatencyMS != 125 {
			t.Fatalf("检测记录=%+v", command)
		}
		if len(command.CredentialFingerprint) != 64 || len(command.ReceiptTokenHash) != 64 || strings.Contains(command.ReceiptTokenHash, "sk-upstream") {
			t.Fatalf("检测摘要无效：%+v", command)
		}
		savedReceiptHash = command.ReceiptTokenHash
		return ChannelTest{ID: 81, CanonicalModelID: 61, AdapterType: "anthropic", Capabilities: []string{"responses"}, LatencyMS: 125, ExpiresAt: time.Now().Add(15 * time.Minute)}, nil
	}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/channel-tests", strings.NewReader(`{"canonical_model_id":61,"base_url":"https://api.merchant.example","api_key":"sk-upstream-secret"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"test_receipt":"`) {
		t.Fatalf("检测响应=%d %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "sk-upstream-secret") || strings.Contains(response.Body.String(), "credential_fingerprint") {
		t.Fatalf("检测响应泄露凭据：%s", response.Body.String())
	}
	var payload struct {
		Data struct {
			TestReceipt string `json:"test_receipt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	receiptDigest := sha256.Sum256([]byte(payload.Data.TestReceipt))
	if savedReceiptHash != hex.EncodeToString(receiptDigest[:]) {
		t.Fatalf("检测凭证保存摘要=%q，提交事务期望=%q", savedReceiptHash, hex.EncodeToString(receiptDigest[:]))
	}
}

func TestAdminVerifiesMerchantCredentialThroughNewAPI(t *testing.T) {
	box := testCredentialBox(t)
	encrypted, err := box.Encrypt([]byte("sk-upstream-secret"))
	if err != nil {
		t.Fatal(err)
	}
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("管理员复检应使用隔离检测器")
	}))
	services.ChannelTests = channelTesterStub{test: func(_ context.Context, request ChannelTestRequest) (ChannelTestOutcome, error) {
		if request.Model.PublicName != "gpt-5.6" || request.APIKey != "sk-upstream-secret" {
			t.Fatalf("管理员复检参数=%+v", request)
		}
		return ChannelTestOutcome{AdapterType: "openai_compatible", Capabilities: []string{"responses"}, LatencyMS: 88}, nil
	}}
	repository := &merchantRepositoryStub{}
	repository.claimReview = func(_ context.Context, reviewID int64, status string) (MerchantChannelReview, error) {
		if reviewID != 21 || status != "verifying" {
			t.Fatalf("认领审核参数=%d %s", reviewID, status)
		}
		return MerchantChannelReview{ID: 21, UserID: 7, CanonicalModelID: 61, Model: "gpt-5.6", Provider: "OpenAI", PriceKind: "text", AdapterType: "openai_compatible", BaseURL: "https://api.merchant.example", EncryptedCredential: encrypted}, nil
	}
	repository.finishVerify = func(_ context.Context, reviewID int64, adminID int, adapterType string, failure string) error {
		if reviewID != 21 || adminID != 1 || failure != "" || adapterType != "openai_compatible" {
			t.Fatalf("验证结果=%d %d %s %q", reviewID, adminID, adapterType, failure)
		}
		return nil
	}
	services.Merchants = repository
	services.Credentials = box
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/channel-reviews/21/verify", nil)
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"adapter_type":"openai_compatible"`) {
		t.Fatalf("验证响应=%d %s", response.Code, response.Body.String())
	}
}

func TestAdminApprovalCreatesAndTestsNewAPIChannelBeforeFinalizing(t *testing.T) {
	box := testCredentialBox(t)
	encrypted, _ := box.Encrypt([]byte("sk-upstream-secret"))
	var calls []string
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/channel/":
			var body struct {
				Channel struct {
					Key, Name, Models string
					Status            int
					Type              int `json:"type"`
				} `json:"channel"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Channel.Key != "sk-upstream-secret" || body.Channel.Models != "gpt-5.6" || body.Channel.Type != 1 {
				t.Fatalf("渠道创建参数=%+v", body.Channel)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/channel/search":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"items": []map[string]any{{"id": 41, "name": "ModelRoute-R21-U7", "models": "gpt-5.6"}}, "total": 1}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/channel/test/41":
			if r.URL.Query().Get("model") != "gpt-5.6" {
				t.Fatalf("测试模型=%q", r.URL.Query().Get("model"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": nil})
		default:
			t.Fatalf("未知 New API 请求=%s %s", r.Method, r.URL.RequestURI())
		}
	}))
	repository := &merchantRepositoryStub{}
	repository.claimReview = func(_ context.Context, _ int64, status string) (MerchantChannelReview, error) {
		if status != "approving" {
			t.Fatalf("审批状态=%s", status)
		}
		return MerchantChannelReview{ID: 21, UserID: 7, ApplicationID: 11, CanonicalModelID: 61, Model: "gpt-5.6", Provider: "OpenAI", PriceKind: "text", ChannelName: "主线路", AdapterType: "openai_compatible", BaseURL: "https://api.merchant.example", EncryptedCredential: encrypted, Status: "approving"}, nil
	}
	repository.finishApproval = func(_ context.Context, reviewID int64, adminID int, channelID int) (MerchantApproval, error) {
		if reviewID != 21 || adminID != 1 || channelID != 41 {
			t.Fatalf("完成审批参数=%d %d %d", reviewID, adminID, channelID)
		}
		return MerchantApproval{MerchantID: 31, UserID: 7, ChannelID: 51, NewAPIChannelID: 41}, nil
	}
	services.Merchants = repository
	services.Credentials = box
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/channel-reviews/21/approve", nil)
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"channel_id":51`) || strings.Contains(response.Body.String(), `"newapi_channel_id"`) {
		t.Fatalf("审批响应=%d %s", response.Code, response.Body.String())
	}
	if strings.Join(calls, "|") != "POST /api/channel/|GET /api/channel/search|GET /api/channel/test/41" {
		t.Fatalf("New API 调用顺序=%v", calls)
	}
}

func TestMerchantAndAdminEndpointsEnforceServerSideIdentity(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("越权请求不应调用 New API")
	}))
	services.Merchants = &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
		return MerchantIdentity{}, ErrMerchantNotFound
	}}
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	for _, path := range []string{"/api/admin/merchant-applications", "/api/merchant/channels"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s 状态码=%d，响应=%s", path, response.Code, response.Body.String())
		}
	}
}

func TestAdminChangesMerchantStatusByUserID(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &merchantRepositoryStub{
		identity: func(_ context.Context, userID int) (MerchantIdentity, error) {
			if userID != 7 {
				t.Fatalf("商户用户 ID=%d", userID)
			}
			return MerchantIdentity{MerchantID: 31, UserID: 7, Status: "active", IsMerchant: true}, nil
		},
		merchant: func(_ context.Context, merchantID int64) (MerchantProfile, error) {
			return MerchantProfile{ID: merchantID, UserID: 7, Status: "active"}, nil
		},
		setMerchantStatus: func(_ context.Context, merchantID int64, enabled bool, adminID int, reason string) (MerchantStatusChange, error) {
			if merchantID != 31 || enabled || adminID != 1 || reason != "资料异常" {
				t.Fatalf("商户状态参数=%d %v %d %q", merchantID, enabled, adminID, reason)
			}
			return MerchantStatusChange{MerchantID: 31, UserID: 7, Status: "revoked"}, nil
		},
	}
	services.Merchants = repository
	token := createPlatformSession(t, services, 1, 100)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/users/7/merchant-status", strings.NewReader(`{"enabled":false,"reason":"资料异常"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"user_id":7`) || strings.Contains(response.Body.String(), `"merchant_id"`) {
		t.Fatalf("商户状态响应=%d %s", response.Code, response.Body.String())
	}
}

func TestAdminChannelsFiltersByMerchantUserID(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &filteredMerchantRepositoryStub{merchantRepositoryStub: &merchantRepositoryStub{}}
	repository.listChannelsFiltered = func(_ context.Context, page, pageSize int, status, keyword string, userID int) ([]MerchantChannel, int, error) {
		if page != 1 || pageSize != 50 || status != "active" || keyword != "主线路" || userID != 7 {
			t.Fatalf("渠道筛选参数=%d %d %q %q %d", page, pageSize, status, keyword, userID)
		}
		return []MerchantChannel{{ID: 51, UserID: 7, MerchantName: "北辰云智", ChannelName: "主线路", NewAPIChannelID: 41}}, 1, nil
	}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 1, 100)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/channels?p=1&page_size=50&status=active&keyword=%E4%B8%BB%E7%BA%BF%E8%B7%AF&merchant_user_id=7", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"user_id":7`) || !strings.Contains(response.Body.String(), `"merchant_name":"北辰云智"`) || strings.Contains(response.Body.String(), `"newapi_channel_id"`) {
		t.Fatalf("渠道筛选响应=%d %s", response.Code, response.Body.String())
	}
}

func TestAdministratorMerchantEndpointRequiresPersistedMerchantIdentity(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("缺少商户身份时不应调用 New API")
	}))
	services.Merchants = &merchantRepositoryStub{
		identity: func(context.Context, int) (MerchantIdentity, error) {
			return MerchantIdentity{}, ErrMerchantNotFound
		},
		channels: func(context.Context, int) ([]MerchantChannel, error) {
			t.Fatal("缺少商户身份时不应读取商户渠道")
			return nil, nil
		},
	}
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/merchant/channels", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("状态码=%d，响应=%s", response.Code, response.Body.String())
	}
}

func TestMerchantConflictErrorIsStable(t *testing.T) {
	if !errors.Is(ErrMerchantConflict, ErrMerchantConflict) {
		t.Fatal("商户冲突错误必须可比较")
	}
}

func TestValidateMerchantDisplayNameUsesWeightedLengthAndAllowedCharacters(t *testing.T) {
	valid := []string{"云", "AB", "北辰云智科技平台", "ModelRoute2026", "云API2026"}
	for _, name := range valid {
		if _, err := validateMerchantDisplayName(name); err != nil {
			t.Fatalf("名称 %q 应通过校验：%v", name, err)
		}
	}
	invalid := []string{"A", "北辰云智科技平台站", "ABCDEFGHIJKLMNOPQ", "北辰 云智", "北辰_云智", "北辰-云智", "北辰🙂"}
	for _, name := range invalid {
		if _, err := validateMerchantDisplayName(name); err == nil {
			t.Fatalf("名称 %q 应被拒绝", name)
		}
	}
}

func TestNormalizeMerchantDisplayNameIgnoresASCIICase(t *testing.T) {
	left, err := validateMerchantDisplayName("Model2026")
	if err != nil {
		t.Fatal(err)
	}
	right, err := validateMerchantDisplayName("model2026")
	if err != nil {
		t.Fatal(err)
	}
	if left.Key != right.Key || left.Units != 9 || right.Units != 9 {
		t.Fatalf("标准化结果 left=%+v right=%+v", left, right)
	}
}

func TestMeIncludesPlatformMerchantIdentity(t *testing.T) {
	services, store, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/self" {
			t.Fatalf("上游路径=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{
			"id": 7, "username": "alice", "display_name": "Alice", "role": 1, "status": 1, "group": "default",
		}})
	}))
	services.Merchants = &merchantRepositoryStub{identity: func(_ context.Context, userID int) (MerchantIdentity, error) {
		return MerchantIdentity{MerchantID: 31, UserID: userID, DisplayName: "ModelRoute 官方", Status: "active", IsMerchant: true, IsPlatformOwned: true}, nil
	}}
	token, err := store.Create(context.Background(), platformauth.Session{UserID: 7, Username: "alice", Role: 1, Status: 1, UpstreamCookie: "session=user"})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"is_merchant":true`) ||
		!strings.Contains(response.Body.String(), `"merchant_name":"ModelRoute 官方"`) ||
		!strings.Contains(response.Body.String(), `"is_platform_merchant":true`) ||
		strings.Contains(response.Body.String(), `"merchant_id"`) {
		t.Fatalf("me 响应=%d %s", response.Code, response.Body.String())
	}
}

func TestActiveMerchantCannotSubmitAnotherApplication(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("重复申请不应调用 New API")
	}))
	repository := &merchantRepositoryStub{identity: func(_ context.Context, userID int) (MerchantIdentity, error) {
		return MerchantIdentity{MerchantID: 31, UserID: userID, Status: "active", IsMerchant: true}, nil
	}}
	repository.createApplication = func(context.Context, CreateMerchantApplication) (MerchantApplication, error) {
		t.Fatal("有效商户不应创建新申请")
		return MerchantApplication{}, nil
	}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/application", strings.NewReader(`{"display_name":"北辰云智","website_url":"https://merchant.example","business_description":"企业模型 API 服务","channel_name":"主线路","canonical_model_id":61,"base_url":"https://api.merchant.example","api_key":"sk-upstream-secret","test_receipt":"receipt-token"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("状态码=%d，响应=%s", response.Code, response.Body.String())
	}
}

func TestApprovalFailureDeletesNewAPIChannelAndRestoresReview(t *testing.T) {
	box := testCredentialBox(t)
	encrypted, _ := box.Encrypt([]byte("sk-upstream-secret"))
	var calls []string
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/channel/":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/channel/search":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"items": []map[string]any{{"id": 41, "name": "ModelRoute-R21-U7"}}}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/channel/test/41":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "message": "upstream failed"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/channel/41":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			t.Fatalf("未知 New API 请求=%s %s", r.Method, r.URL.RequestURI())
		}
	}))
	restored := false
	repository := &merchantRepositoryStub{}
	repository.claimReview = func(context.Context, int64, string) (MerchantChannelReview, error) {
		return MerchantChannelReview{ID: 21, UserID: 7, CanonicalModelID: 61, Model: "gpt-5.6", Provider: "OpenAI", PriceKind: "text", ChannelName: "主线路", AdapterType: "openai_compatible", BaseURL: "https://api.merchant.example", EncryptedCredential: encrypted}, nil
	}
	repository.cancelApproval = func(_ context.Context, reviewID int64, _ string) error {
		restored = reviewID == 21
		return nil
	}
	services.Merchants = repository
	services.Credentials = box
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/channel-reviews/21/approve", nil)
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadGateway || !restored {
		t.Fatalf("审批失败响应=%d %s，恢复=%v", response.Code, response.Body.String(), restored)
	}
	if strings.Join(calls, "|") != "POST /api/channel/|GET /api/channel/search|GET /api/channel/test/41|DELETE /api/channel/41" {
		t.Fatalf("补偿调用=%v", calls)
	}
}
