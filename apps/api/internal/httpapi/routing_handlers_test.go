package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type routingRepositoryStub struct {
	RoutingRepository
	createModel   func(context.Context, CreateCanonicalModelCommand) (CanonicalModel, error)
	createOffer   func(context.Context, CreateOfferCommand) (OfferVersion, error)
	bindPolicy    func(context.Context, int, int, string) (RoutingPolicy, error)
	updatePolicy  func(context.Context, int, int, RoutingPolicyInput) (RoutingPolicy, error)
	revokeToken   func(context.Context, int, int) error
	listOffers    func(context.Context, int, string) ([]MarketOffer, error)
	prepareHealth func(context.Context, int64) (int, error)
	resetHealth   func(context.Context, int64) error
	listModels    func(context.Context) ([]CanonicalModel, error)
	deleteModel   func(context.Context, int64) (string, error)
}

func (s *routingRepositoryStub) ListCanonicalModels(ctx context.Context) ([]CanonicalModel, error) {
	if s.listModels == nil {
		return nil, nil
	}
	return s.listModels(ctx)
}

func (s *routingRepositoryStub) ListMarketOffers(ctx context.Context, userID int, model string) ([]MarketOffer, error) {
	return s.listOffers(ctx, userID, model)
}

func (s *routingRepositoryStub) ChannelForHealthReset(ctx context.Context, channelID int64) (int, error) {
	return s.prepareHealth(ctx, channelID)
}

func (s *routingRepositoryStub) ResetChannelHealth(ctx context.Context, channelID int64) error {
	return s.resetHealth(ctx, channelID)
}

func (s *routingRepositoryStub) BindTokenAndDefaultPolicy(ctx context.Context, userID int, tokenID int, fingerprint string) (RoutingPolicy, error) {
	return s.bindPolicy(ctx, userID, tokenID, fingerprint)
}

func (s *routingRepositoryStub) UpdatePolicy(ctx context.Context, userID int, tokenID int, input RoutingPolicyInput) (RoutingPolicy, error) {
	return s.updatePolicy(ctx, userID, tokenID, input)
}

func (s *routingRepositoryStub) RevokeTokenBinding(ctx context.Context, userID int, tokenID int) error {
	if s.revokeToken == nil {
		return nil
	}
	return s.revokeToken(ctx, userID, tokenID)
}

func (s *routingRepositoryStub) CreateCanonicalModel(ctx context.Context, command CreateCanonicalModelCommand) (CanonicalModel, error) {
	return s.createModel(ctx, command)
}

func (s *routingRepositoryStub) CreateOffer(ctx context.Context, command CreateOfferCommand) (OfferVersion, error) {
	return s.createOffer(ctx, command)
}

func (s *routingRepositoryStub) DeleteCanonicalModel(ctx context.Context, modelID int64) (string, error) {
	if s.deleteModel == nil {
		return "", ErrRoutingNotFound
	}
	return s.deleteModel(ctx, modelID)
}

type offerSynchronizerStub struct {
	offerID int64
}

func (s *offerSynchronizerStub) SyncOffer(_ context.Context, offerID int64) error {
	s.offerID = offerID
	return nil
}

func TestAdminCreatesCanonicalModelWithIntegerOfficialPrice(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("创建标准模型不应直接调用 New API")
	}))
	repository := &routingRepositoryStub{}
	repository.createModel = func(_ context.Context, command CreateCanonicalModelCommand) (CanonicalModel, error) {
		if command.PublicName != "gpt-5.5" || command.Provider != "OpenAI" || command.Protocol != "responses" {
			t.Fatalf("标准模型参数=%+v", command)
		}
		if len(command.Price.Tiers) != 1 || command.Price.Tiers[0].InputPriceMicros != 1_250_000 || command.Price.Tiers[0].CacheReadPriceMicros == nil || *command.Price.Tiers[0].CacheReadPriceMicros != 100_000 {
			t.Fatalf("官方价格参数=%+v", command.Price)
		}
		return CanonicalModel{ID: 61, PublicName: command.PublicName, Provider: command.Provider, Protocol: command.Protocol, Status: "active", Price: command.Price}, nil
	}
	services.Routing = repository
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	body := `{"public_name":"gpt-5.5","provider":"OpenAI","protocol":"responses","price":{"tiers":[{"minimum_input_tokens":0,"maximum_input_tokens":null,"input_price_micros":1250000,"output_price_micros":3000000,"cache_read_price_micros":100000}]}}`
	request := httptest.NewRequest(http.MethodPost, "/api/admin/models", strings.NewReader(body))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"id":61`) {
		t.Fatalf("创建响应=%d %s", response.Code, response.Body.String())
	}
}

func TestAdminDetectsCustomModelWithoutPersistingCredentials(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	persisted := false
	repository.createModel = func(context.Context, CreateCanonicalModelCommand) (CanonicalModel, error) {
		persisted = true
		return CanonicalModel{}, nil
	}
	services.Routing = repository
	services.ChannelTests = channelTesterStub{test: func(_ context.Context, request ChannelTestRequest) (ChannelTestOutcome, error) {
		if request.Model.PublicName != "custom-chat-v1" || request.Model.Provider != "OpenAI" || request.Model.Protocol != "chat_completions" || request.BaseURL != "https://api.example.com" || request.APIKey != "sk-custom-secret" {
			t.Fatalf("模型检测参数=%+v", request)
		}
		return ChannelTestOutcome{AdapterType: "openai_compatible", Capabilities: []string{"chat_completions"}, LatencyMS: 123}, nil
	}}
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/models/detect", strings.NewReader(`{"public_name":"custom-chat-v1","provider":"OpenAI","protocol":"chat_completions","base_url":"https://api.example.com","api_key":"sk-custom-secret"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || persisted || strings.Contains(response.Body.String(), "sk-custom-secret") || !strings.Contains(response.Body.String(), `"latency_ms":123`) {
		t.Fatalf("检测响应=%d %s，持久化=%v", response.Code, response.Body.String(), persisted)
	}
}

func TestAdminDeletesOrArchivesCanonicalModel(t *testing.T) {
	services, _, audit := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	repository.deleteModel = func(_ context.Context, modelID int64) (string, error) {
		if modelID != 61 {
			t.Fatalf("删除模型编号=%d", modelID)
		}
		return "archived", nil
	}
	services.Routing = repository
	adminToken := createPlatformSession(t, services, 1, 100)
	userToken := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)

	for _, test := range []struct {
		token string
		want  int
	}{
		{token: userToken, want: http.StatusForbidden},
		{token: adminToken, want: http.StatusOK},
	} {
		request := httptest.NewRequest(http.MethodDelete, "/api/admin/models/61", nil)
		request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
		request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: test.token})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.want || test.want == http.StatusOK && !strings.Contains(response.Body.String(), `"result":"archived"`) {
			t.Fatalf("删除响应=%d %s", response.Code, response.Body.String())
		}
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != "model.delete" {
		t.Fatalf("删除审计=%+v", audit.entries)
	}
}

func TestMerchantOfferRejectsMultiplierAboveFiveTimesBeforeStore(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	called := false
	repository := &routingRepositoryStub{}
	repository.createOffer = func(context.Context, CreateOfferCommand) (OfferVersion, error) {
		called = true
		return OfferVersion{}, nil
	}
	services.Routing = repository
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPut, "/api/merchant/offers/61", strings.NewReader(`{"multiplier_bps":50001}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || called {
		t.Fatalf("越界报价响应=%d %s，仓储调用=%v", response.Code, response.Body.String(), called)
	}
}

func TestMerchantOfferActivatesOnlyAfterNewAPISync(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	repository.createOffer = func(_ context.Context, command CreateOfferCommand) (OfferVersion, error) {
		if command.UserID != 7 || command.CanonicalModelID != 61 || command.MultiplierBPS != 0 {
			t.Fatalf("报价参数=%+v", command)
		}
		return OfferVersion{ID: 42, CanonicalModelID: 61, MultiplierBPS: 0, Status: "syncing"}, nil
	}
	synchronizer := &offerSynchronizerStub{}
	services.Routing = repository
	services.OfferSync = synchronizer
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPut, "/api/merchant/offers/61", strings.NewReader(`{"multiplier_bps":0}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || synchronizer.offerID != 42 {
		t.Fatalf("报价响应=%d %s，同步=%+v", response.Code, response.Body.String(), synchronizer)
	}
}

func TestRoutingPolicyBindsRevealedKeyAsFingerprintOnly(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token/9/key" || r.Method != http.MethodPost {
			t.Fatalf("New API 请求=%s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"key":"sk-user-secret"}}`))
	}))
	box := testCredentialBox(t)
	repository := &routingRepositoryStub{}
	repository.bindPolicy = func(_ context.Context, userID int, tokenID int, fingerprint string) (RoutingPolicy, error) {
		if userID != 7 || tokenID != 9 || fingerprint == "" || strings.Contains(fingerprint, "sk-user-secret") {
			t.Fatalf("令牌绑定参数=%d %d %q", userID, tokenID, fingerprint)
		}
		if fingerprint != box.Fingerprint([]byte("sk-user-secret")) {
			t.Fatalf("令牌指纹不一致：%q", fingerprint)
		}
		return RoutingPolicy{NewAPITokenID: 9, Mode: "smart", Priority: "stability", MinimumMultiplierBPS: 1, MaximumMultiplierBPS: 10_000}, nil
	}
	services.Routing = repository
	services.Credentials = box
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/routing-policies/9", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "sk-user-secret") || !strings.Contains(response.Body.String(), `"mode":"smart"`) {
		t.Fatalf("路由策略响应=%d %s", response.Code, response.Body.String())
	}
}

func TestFixedRoutingPolicyRequiresAtLeastOneMerchant(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"key":"sk-user-secret"}}`))
	}))
	repository := &routingRepositoryStub{}
	repository.bindPolicy = func(context.Context, int, int, string) (RoutingPolicy, error) {
		return RoutingPolicy{NewAPITokenID: 9}, nil
	}
	repository.updatePolicy = func(context.Context, int, int, RoutingPolicyInput) (RoutingPolicy, error) {
		t.Fatal("固定商户为空时不应写仓储")
		return RoutingPolicy{}, nil
	}
	services.Routing = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPut, "/api/routing-policies/9", strings.NewReader(`{"mode":"fixed","priority":"price","minimum_multiplier_bps":2000,"maximum_multiplier_bps":6000,"merchant_ids":[]}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("固定路由空名单响应=%d %s", response.Code, response.Body.String())
	}
}

func TestCreateKeyBindsOnlyNewTokenToDefaultPolicy(t *testing.T) {
	listCount := 0
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/token/":
			listCount++
			items := `[{"id":8,"name":"旧密钥"}]`
			if listCount == 2 {
				items = `[{"id":9,"name":"新密钥"},{"id":8,"name":"旧密钥"}]`
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"items":` + items + `,"total":` + fmt.Sprint(len(items)) + `}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/token/":
			_, _ = w.Write([]byte(`{"success":true,"data":null}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/token/9/key":
			_, _ = w.Write([]byte(`{"success":true,"data":{"key":"sk-new-key"}}`))
		default:
			t.Fatalf("未知 New API 调用=%s %s", r.Method, r.URL.RequestURI())
		}
	}))
	box := testCredentialBox(t)
	repository := &routingRepositoryStub{}
	boundToken := 0
	repository.bindPolicy = func(_ context.Context, userID int, tokenID int, fingerprint string) (RoutingPolicy, error) {
		if userID != 7 || fingerprint != box.Fingerprint([]byte("sk-new-key")) {
			t.Fatalf("绑定参数=%d %d %q", userID, tokenID, fingerprint)
		}
		boundToken = tokenID
		return RoutingPolicy{}, nil
	}
	services.Routing = repository
	services.Credentials = box
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/keys", strings.NewReader(`{"name":"新密钥","expired_time":-1,"unlimited_quota":true}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || boundToken != 9 || listCount != 2 {
		t.Fatalf("密钥创建响应=%d %s，绑定=%d，列表次数=%d", response.Code, response.Body.String(), boundToken, listCount)
	}
}

func TestDeleteKeyRevokesRoutingBindingAfterNewAPIDelete(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/token/9" {
			t.Fatalf("New API 删除调用=%s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":null}`))
	}))
	repository := &routingRepositoryStub{}
	revoked := false
	repository.revokeToken = func(_ context.Context, userID int, tokenID int) error {
		revoked = userID == 7 && tokenID == 9
		return nil
	}
	services.Routing = repository
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodDelete, "/api/keys/9", nil)
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !revoked {
		t.Fatalf("删除响应=%d %s，撤销=%v", response.Code, response.Body.String(), revoked)
	}
}

func TestMarketOffersNeverExposeInternalAlias(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	repository.listOffers = func(_ context.Context, userID int, model string) ([]MarketOffer, error) {
		if userID != 0 || model != "gpt-5.5" {
			t.Fatalf("市场查询参数=%d %q", userID, model)
		}
		return []MarketOffer{{OfferID: 42, Model: "gpt-5.5", MerchantName: "北辰云智", MultiplierBPS: 4000, HealthState: "healthy", InternalAlias: "mkt-ov-secret"}}, nil
	}
	services.Routing = repository
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/offers?model=gpt-5.5", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "mkt-ov-secret") || !strings.Contains(response.Body.String(), "北辰云智") {
		t.Fatalf("市场响应=%d %s", response.Code, response.Body.String())
	}
}

type adminStatusStub struct {
	channelID int
}

func (s *adminStatusStub) Do(_ context.Context, method string, path string, body any) ([]byte, error) {
	if method != http.MethodPost || path != "/api/channel/41/status" || body.(map[string]int)["status"] != 1 {
		return nil, fmt.Errorf("恢复 New API 参数=%s %s %v", method, path, body)
	}
	s.channelID = 41
	return []byte(`true`), nil
}

func TestAdminHealthResetEnablesNewAPIBeforePlatformChannel(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	admin := &adminStatusStub{}
	repository.prepareHealth = func(_ context.Context, channelID int64) (int, error) {
		if channelID != 51 {
			t.Fatalf("准备恢复渠道=%d", channelID)
		}
		return 41, nil
	}
	repository.resetHealth = func(_ context.Context, channelID int64) error {
		if channelID != 51 || admin.channelID != 41 {
			t.Fatalf("平台恢复早于 New API：channel=%d upstream=%d", channelID, admin.channelID)
		}
		return nil
	}
	services.Routing = repository
	services.AdminNewAPI = admin
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/channels/51/health/reset", nil)
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("恢复响应=%d %s", response.Code, response.Body.String())
	}
}
