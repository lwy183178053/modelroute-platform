package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/secretbox"
)

type merchantRepositoryStub struct {
	MerchantRepository
	identity          func(context.Context, int) (MerchantIdentity, error)
	createApplication func(context.Context, CreateMerchantApplication) (MerchantApplication, error)
	claimReview       func(context.Context, int64, string) (MerchantChannelReview, error)
	finishVerify      func(context.Context, int64, int, []string, string) error
	finishApproval    func(context.Context, int64, int, int) (MerchantApproval, error)
	cancelApproval    func(context.Context, int64, string) error
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

func (s *merchantRepositoryStub) ClaimReview(ctx context.Context, reviewID int64, targetStatus string) (MerchantChannelReview, error) {
	return s.claimReview(ctx, reviewID, targetStatus)
}

func (s *merchantRepositoryStub) FinishVerification(ctx context.Context, reviewID int64, adminID int, models []string, failure string) error {
	return s.finishVerify(ctx, reviewID, adminID, models, failure)
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

func TestMerchantApplicationEncryptsCredentialAndNeverReturnsIt(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("提交申请不应调用 New API")
	}))
	box := testCredentialBox(t)
	secret := "sk-upstream-secret"
	repository := &merchantRepositoryStub{}
	repository.createApplication = func(_ context.Context, command CreateMerchantApplication) (MerchantApplication, error) {
		if command.UserID != 7 || command.DisplayName != "北辰云智" || command.Channel.ChannelType != 1 {
			t.Fatalf("申请参数=%+v", command)
		}
		if bytes.Contains(command.Channel.EncryptedCredential, []byte(secret)) || len(command.Channel.EncryptedCredential) == 0 {
			t.Fatal("仓储收到未加密密钥或空密文")
		}
		if len(command.Channel.CredentialFingerprint) != 64 {
			t.Fatalf("密钥指纹=%q", command.Channel.CredentialFingerprint)
		}
		return MerchantApplication{ID: 11, UserID: 7, DisplayName: command.DisplayName, WebsiteURL: command.WebsiteURL, Status: "pending", InitialReview: MerchantChannelReview{ID: 21, ChannelName: command.Channel.ChannelName, Status: "pending", CredentialConfigured: true}}, nil
	}
	services.Merchants = repository
	services.Credentials = box
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	body := `{"display_name":"北辰云智","website_url":"https://merchant.example","business_description":"企业模型 API 服务","channel_name":"主线路","channel_type":1,"base_url":"https://api.merchant.example","api_key":"` + secret + `","models":["gpt-4o-mini"]}`
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/application", strings.NewReader(body))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("状态码=%d，响应=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), secret) || strings.Contains(response.Body.String(), "encrypted_credential") {
		t.Fatalf("申请响应泄露凭据：%s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"credential_configured":true`) {
		t.Fatalf("申请响应缺少凭据状态：%s", response.Body.String())
	}
}

func TestAdminVerifiesMerchantCredentialThroughNewAPI(t *testing.T) {
	box := testCredentialBox(t)
	encrypted, err := box.Encrypt([]byte("sk-upstream-secret"))
	if err != nil {
		t.Fatal(err)
	}
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/channel/fetch_models" {
			t.Fatalf("New API 请求=%s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Cookie") != "session=upstream" || r.Header.Get("New-Api-User") != "1" {
			t.Fatal("New API 管理鉴权缺失")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "sk-upstream-secret" || body["base_url"] != "https://api.merchant.example" {
			t.Fatalf("验证参数=%v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": []string{"gpt-4o-mini", "text-embedding-3-small"}})
	}))
	repository := &merchantRepositoryStub{}
	repository.claimReview = func(_ context.Context, reviewID int64, status string) (MerchantChannelReview, error) {
		if reviewID != 21 || status != "verifying" {
			t.Fatalf("认领审核参数=%d %s", reviewID, status)
		}
		return MerchantChannelReview{ID: 21, UserID: 7, ChannelType: 1, BaseURL: "https://api.merchant.example", RequestedModels: []string{"gpt-4o-mini"}, EncryptedCredential: encrypted}, nil
	}
	repository.finishVerify = func(_ context.Context, reviewID int64, adminID int, models []string, failure string) error {
		if reviewID != 21 || adminID != 1 || failure != "" || strings.Join(models, ",") != "gpt-4o-mini,text-embedding-3-small" {
			t.Fatalf("验证结果=%d %d %v %q", reviewID, adminID, models, failure)
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
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "gpt-4o-mini") {
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
			if body.Channel.Key != "sk-upstream-secret" || body.Channel.Models != "gpt-4o-mini" || body.Channel.Type != 1 {
				t.Fatalf("渠道创建参数=%+v", body.Channel)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case r.Method == http.MethodGet && r.URL.Path == "/api/channel/search":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"items": []map[string]any{{"id": 41, "name": "ModelRoute-R21-U7", "models": "gpt-4o-mini"}}, "total": 1}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/channel/test/41":
			if r.URL.Query().Get("model") != "gpt-4o-mini" {
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
		return MerchantChannelReview{ID: 21, UserID: 7, ApplicationID: 11, ChannelName: "主线路", ChannelType: 1, BaseURL: "https://api.merchant.example", RequestedModels: []string{"gpt-4o-mini"}, VerifiedModels: []string{"gpt-4o-mini"}, EncryptedCredential: encrypted, Status: "approving"}, nil
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
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"newapi_channel_id":41`) {
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

func TestMerchantConflictErrorIsStable(t *testing.T) {
	if !errors.Is(ErrMerchantConflict, ErrMerchantConflict) {
		t.Fatal("商户冲突错误必须可比较")
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
		return MerchantIdentity{MerchantID: 31, UserID: userID, Status: "active", IsMerchant: true}, nil
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
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"is_merchant":true`) || !strings.Contains(response.Body.String(), `"merchant_id":31`) {
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
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/application", strings.NewReader(`{"display_name":"北辰云智","website_url":"https://merchant.example","business_description":"企业模型 API 服务","channel_name":"主线路","channel_type":1,"base_url":"https://api.merchant.example","api_key":"sk-upstream-secret","models":["gpt-4o-mini"]}`))
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
		return MerchantChannelReview{ID: 21, UserID: 7, ChannelName: "主线路", ChannelType: 1, BaseURL: "https://api.merchant.example", RequestedModels: []string{"gpt-4o-mini"}, EncryptedCredential: encrypted}, nil
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
