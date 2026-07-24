package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/secretbox"
)

type merchantRepositoryStub struct {
	MerchantRepository
	identity           func(context.Context, int) (MerchantIdentity, error)
	channels           func(context.Context, int) ([]MerchantChannel, error)
	createApplication  func(context.Context, CreateMerchantApplication) (MerchantApplication, error)
	application        func(context.Context, int64) (MerchantApplication, error)
	approveApplication func(context.Context, int64, int) (MerchantProfile, error)
	rejectApplication  func(context.Context, int64, int, string) error
	merchant           func(context.Context, int64) (MerchantProfile, error)
	updateDescription  func(context.Context, int64, string) (MerchantProfile, error)
	setMerchantStatus  func(context.Context, int64, bool, int, string) (MerchantStatusChange, error)
}

func (s *merchantRepositoryStub) Identity(ctx context.Context, userID int) (MerchantIdentity, error) {
	if s.identity == nil {
		return MerchantIdentity{}, ErrMerchantNotFound
	}
	return s.identity(ctx, userID)
}

func (s *merchantRepositoryStub) ChannelsForUser(ctx context.Context, userID int) ([]MerchantChannel, error) {
	if s.channels == nil {
		return nil, nil
	}
	return s.channels(ctx, userID)
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

func (s *merchantRepositoryStub) Merchant(ctx context.Context, merchantID int64) (MerchantProfile, error) {
	return s.merchant(ctx, merchantID)
}

func (s *merchantRepositoryStub) UpdateMerchantDescription(ctx context.Context, merchantID int64, description string) (MerchantProfile, error) {
	return s.updateDescription(ctx, merchantID, description)
}

func (s *merchantRepositoryStub) SetMerchantStatus(ctx context.Context, merchantID int64, enabled bool, adminID int, reason string) (MerchantStatusChange, error) {
	return s.setMerchantStatus(ctx, merchantID, enabled, adminID, reason)
}

type filteredMerchantRepositoryStub struct {
	*merchantRepositoryStub
	listChannelsFiltered func(context.Context, int, int, string, int) ([]MerchantChannel, int, error)
}

func (s *filteredMerchantRepositoryStub) ListChannelsFiltered(ctx context.Context, page, pageSize int, keyword string, userID int) ([]MerchantChannel, int, error) {
	return s.listChannelsFiltered(ctx, page, pageSize, keyword, userID)
}

type filteredMultiModelMerchantRepositoryStub struct {
	*filteredMerchantRepositoryStub
	MultiModelMerchantRepository
	channelModels func(context.Context, int64) ([]ChannelModelBinding, error)
}

func (s *filteredMultiModelMerchantRepositoryStub) ChannelModels(ctx context.Context, channelID int64) ([]ChannelModelBinding, error) {
	return s.channelModels(ctx, channelID)
}

type channelTesterStub struct {
	test func(context.Context, ChannelTestRequest) (ChannelTestOutcome, error)
}

func (s channelTesterStub) Test(ctx context.Context, request ChannelTestRequest) (ChannelTestOutcome, error) {
	return s.test(ctx, request)
}

func testCredentialBox(t *testing.T) *secretbox.Box {
	t.Helper()
	box, err := secretbox.New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func createPlatformSession(t *testing.T, services Services, userID, role int) string {
	t.Helper()
	token, err := services.Sessions.Create(context.Background(), platformauth.Session{
		UserID: userID, Username: "user", Role: role, Status: 1, UpstreamCookie: "session=upstream",
	})
	if err != nil {
		t.Fatalf("创建平台会话失败：%v", err)
	}
	return token
}

func TestMerchantChannelResponseHidesInternalCoreFields(t *testing.T) {
	payload, err := json.Marshal(MerchantChannel{
		ID: 51, MerchantID: 31, UserID: 7, MerchantName: "北辰云智",
		NewAPIChannelID: 41, ChannelType: 14, AdapterType: "anthropic",
		Provider: "Anthropic", ChannelName: "主线路", ModelBindings: []ChannelModelBinding{{
			ID: 71, CanonicalModelID: 7, Model: "claude-sonnet-5", HealthState: "healthy",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, field := range []string{"channel_type", "adapter_type", "merchant_id", "newapi_channel_id"} {
		if strings.Contains(text, `"`+field+`"`) {
			t.Fatalf("渠道响应暴露内部字段 %s：%s", field, text)
		}
	}
	for _, field := range []string{"id", "provider", "user_id", "merchant_name", "models", "health_state"} {
		if !strings.Contains(text, `"`+field+`"`) {
			t.Fatalf("渠道响应缺少字段 %s：%s", field, text)
		}
	}
}

func TestMerchantApplicationOnlyStoresProfileFields(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("提交商户申请不应调用核心服务")
	}))
	repository := &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
		return MerchantIdentity{}, ErrMerchantNotFound
	}}
	repository.createApplication = func(_ context.Context, command CreateMerchantApplication) (MerchantApplication, error) {
		if command.UserID != 7 || command.DisplayName != "北辰云智" || command.WebsiteURL != "https://merchant.example" || command.BusinessDescription != "企业模型 API 服务" {
			t.Fatalf("商户申请=%+v", command)
		}
		return MerchantApplication{ID: 11, UserID: command.UserID, DisplayName: command.DisplayName, Status: "pending"}, nil
	}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, roleCommon)
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/application", strings.NewReader(`{"display_name":"北辰云智","website_url":"https://merchant.example","business_description":"企业模型 API 服务"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"id":11`) {
		t.Fatalf("商户申请响应=%d %s", response.Code, response.Body.String())
	}
}

func TestAdminApprovesMerchantApplicationWithoutChannel(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("通过商户申请不应创建渠道")
	}))
	repository := &merchantRepositoryStub{approveApplication: func(_ context.Context, applicationID int64, adminID int) (MerchantProfile, error) {
		if applicationID != 11 || adminID != 1 {
			t.Fatalf("审批参数=%d/%d", applicationID, adminID)
		}
		return MerchantProfile{ID: 31, UserID: 7, DisplayName: "北辰云智", Status: "active"}, nil
	}}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 1, roleAdmin)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/merchant-applications/11/approve", nil)
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":31`) {
		t.Fatalf("审批响应=%d %s", response.Code, response.Body.String())
	}
}

func TestAdminChannelsFiltersByMerchantUserID(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	filtered := &filteredMerchantRepositoryStub{merchantRepositoryStub: &merchantRepositoryStub{}}
	filtered.listChannelsFiltered = func(_ context.Context, page, pageSize int, keyword string, userID int) ([]MerchantChannel, int, error) {
		if page != 1 || pageSize != 50 || keyword != "主线路" || userID != 7 {
			t.Fatalf("渠道筛选=%d/%d/%q/%d", page, pageSize, keyword, userID)
		}
		return []MerchantChannel{{ID: 51, UserID: 7, MerchantName: "北辰云智", ChannelName: "主线路"}}, 1, nil
	}
	services.Merchants = filtered
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 1, roleAdmin)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/channels?p=1&page_size=50&keyword=%E4%B8%BB%E7%BA%BF%E8%B7%AF&merchant_user_id=7", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"total":1`) {
		t.Fatalf("管理员渠道响应=%d %s", response.Code, response.Body.String())
	}
}

func TestRemovedChannelReviewAndChannelStatusRoutesAreUnavailable(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.Merchants = &merchantRepositoryStub{}
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 1, roleAdmin)
	for _, target := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/admin/channel-reviews"},
		{http.MethodPost, "/api/admin/channels/51/status"},
		{http.MethodPost, "/api/admin/channels/51/health/reset"},
	} {
		request := httptest.NewRequest(target.method, target.path, nil)
		request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
		response := httptest.NewRecorder()
		NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
		if response.Code != http.StatusNotFound && response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("旧接口 %s %s 状态码=%d", target.method, target.path, response.Code)
		}
	}
}

func TestValidateMerchantDisplayNameUsesWeightedLengthAndAllowedCharacters(t *testing.T) {
	for _, valid := range []string{"八个汉字名称呀", "Abc123", "商户Ab12"} {
		if _, err := validateMerchantDisplayName(valid); err != nil {
			t.Fatalf("合法名称 %q 被拒绝：%v", valid, err)
		}
	}
	for _, invalid := range []string{"九个汉字名称超长呀", "abcdefghijklmnopq", "商户_1", "商户 1", "商户!"} {
		if _, err := validateMerchantDisplayName(invalid); err == nil {
			t.Fatalf("非法名称 %q 被接受", invalid)
		}
	}
}

func TestMerchantNameConflictReturnsClearMessage(t *testing.T) {
	response := httptest.NewRecorder()
	new(server).writeMerchantStoreError(response, ErrMerchantNameConflict)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"merchant_name_conflict"`) {
		t.Fatalf("重名响应=%d %s", response.Code, response.Body.String())
	}
}
