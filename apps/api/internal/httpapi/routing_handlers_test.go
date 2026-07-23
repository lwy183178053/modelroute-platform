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
	createModel          func(context.Context, CreateCanonicalModelCommand) (CanonicalModel, error)
	createOffer          func(context.Context, CreateOfferCommand) (OfferVersion, error)
	retireOffer          func(context.Context, int, int64) error
	offerState           func(context.Context, int, int64) (MerchantOfferState, error)
	deleteOffer          func(context.Context, int, int64) error
	createRoute          func(context.Context, int, RoutingPolicyInput) (RoutingPolicy, error)
	listRoutes           func(context.Context, int) ([]RoutingPolicy, error)
	getRoute             func(context.Context, int, int64) (RoutingPolicy, error)
	updateRoute          func(context.Context, int, int64, RoutingPolicyInput) (RoutingPolicy, error)
	deleteRoute          func(context.Context, int, int64) error
	bindToken            func(context.Context, int, int, string, int64) (TokenRoutingBinding, error)
	rebindToken          func(context.Context, int, int, int64) (TokenRoutingBinding, error)
	listBindings         func(context.Context, int) ([]TokenRoutingBinding, error)
	revokeToken          func(context.Context, int, int) error
	listRoutingMerchants func(context.Context, string, int, int) (RoutingMerchantPage, error)
	listOffers           func(context.Context, int, string) ([]MarketOffer, error)
	listMerchantOffers   func(context.Context, int) ([]MerchantOfferView, error)
	prepareHealth        func(context.Context, int64) (int, error)
	resetHealth          func(context.Context, int64) error
	listModels           func(context.Context) ([]CanonicalModel, error)
	deleteModel          func(context.Context, int64) (string, error)
}

func (s *routingRepositoryStub) CreateRoutingPolicy(ctx context.Context, userID int, input RoutingPolicyInput) (RoutingPolicy, error) {
	return s.createRoute(ctx, userID, input)
}

func (s *routingRepositoryStub) ListRoutingPolicies(ctx context.Context, userID int) ([]RoutingPolicy, error) {
	return s.listRoutes(ctx, userID)
}

func (s *routingRepositoryStub) RoutingPolicy(ctx context.Context, userID int, policyID int64) (RoutingPolicy, error) {
	return s.getRoute(ctx, userID, policyID)
}

func (s *routingRepositoryStub) UpdateRoutingPolicy(ctx context.Context, userID int, policyID int64, input RoutingPolicyInput) (RoutingPolicy, error) {
	return s.updateRoute(ctx, userID, policyID, input)
}

func (s *routingRepositoryStub) DeleteRoutingPolicy(ctx context.Context, userID int, policyID int64) error {
	return s.deleteRoute(ctx, userID, policyID)
}

func (s *routingRepositoryStub) BindTokenToPolicy(ctx context.Context, userID int, tokenID int, fingerprint string, policyID int64) (TokenRoutingBinding, error) {
	return s.bindToken(ctx, userID, tokenID, fingerprint, policyID)
}

func (s *routingRepositoryStub) RebindTokenPolicy(ctx context.Context, userID int, tokenID int, policyID int64) (TokenRoutingBinding, error) {
	return s.rebindToken(ctx, userID, tokenID, policyID)
}

func (s *routingRepositoryStub) ListTokenRoutingBindings(ctx context.Context, userID int) ([]TokenRoutingBinding, error) {
	return s.listBindings(ctx, userID)
}

func (s *routingRepositoryStub) ListRoutingMerchants(ctx context.Context, keyword string, page int, pageSize int) (RoutingMerchantPage, error) {
	return s.listRoutingMerchants(ctx, keyword, page, pageSize)
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

func (s *routingRepositoryStub) ListMerchantOffers(ctx context.Context, userID int) ([]MerchantOfferView, error) {
	return s.listMerchantOffers(ctx, userID)
}

func (s *routingRepositoryStub) ChannelForHealthReset(ctx context.Context, channelID int64) (int, error) {
	return s.prepareHealth(ctx, channelID)
}

func (s *routingRepositoryStub) ResetChannelHealth(ctx context.Context, channelID int64) error {
	return s.resetHealth(ctx, channelID)
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

func (s *routingRepositoryStub) RetireOffer(ctx context.Context, userID int, modelID int64) error {
	if s.retireOffer == nil {
		return ErrRoutingNotFound
	}
	return s.retireOffer(ctx, userID, modelID)
}

func (s *routingRepositoryStub) MerchantOfferState(ctx context.Context, userID int, modelID int64) (MerchantOfferState, error) {
	if s.offerState == nil {
		return MerchantOfferState{}, ErrRoutingNotFound
	}
	return s.offerState(ctx, userID, modelID)
}

func (s *routingRepositoryStub) DeleteMerchantOffer(ctx context.Context, userID int, modelID int64) error {
	if s.deleteOffer == nil {
		return ErrRoutingNotFound
	}
	return s.deleteOffer(ctx, userID, modelID)
}

func (s *routingRepositoryStub) DeleteCanonicalModel(ctx context.Context, modelID int64) (string, error) {
	if s.deleteModel == nil {
		return "", ErrRoutingNotFound
	}
	return s.deleteModel(ctx, modelID)
}

type offerSynchronizerStub struct {
	offerID  int64
	activate bool
}

func (s *offerSynchronizerStub) SyncOffer(_ context.Context, offerID int64, activate bool) error {
	s.offerID = offerID
	s.activate = activate
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
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
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
	if response.Code != http.StatusCreated || synchronizer.offerID != 42 || !synchronizer.activate {
		t.Fatalf("报价响应=%d %s，同步=%+v", response.Code, response.Body.String(), synchronizer)
	}
}

func TestMerchantCanChangeInactiveOfferPriceWithoutListingIt(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	repository.offerState = func(_ context.Context, userID int, modelID int64) (MerchantOfferState, error) {
		return MerchantOfferState{MultiplierBPS: 8500, Status: "inactive"}, nil
	}
	repository.createOffer = func(_ context.Context, command CreateOfferCommand) (OfferVersion, error) {
		return OfferVersion{ID: 43, CanonicalModelID: command.CanonicalModelID, MultiplierBPS: command.MultiplierBPS, Status: "syncing"}, nil
	}
	synchronizer := &offerSynchronizerStub{}
	services.Routing = repository
	services.OfferSync = synchronizer
	token := createPlatformSession(t, services, 7, 1)
	request := httptest.NewRequest(http.MethodPut, "/api/merchant/offers/61", strings.NewReader(`{"multiplier_bps":9000}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusCreated || synchronizer.offerID != 43 || synchronizer.activate {
		t.Fatalf("下架模型改价响应=%d %s，同步=%+v", response.Code, response.Body.String(), synchronizer)
	}
}

func TestMerchantCanChangeOfferListingStateAndDeleteIt(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	repository.offerState = func(_ context.Context, userID int, modelID int64) (MerchantOfferState, error) {
		return MerchantOfferState{MultiplierBPS: 8500, Status: "inactive"}, nil
	}
	repository.createOffer = func(_ context.Context, command CreateOfferCommand) (OfferVersion, error) {
		return OfferVersion{ID: 44, CanonicalModelID: command.CanonicalModelID, MultiplierBPS: command.MultiplierBPS, Status: "syncing"}, nil
	}
	repository.retireOffer = func(_ context.Context, userID int, modelID int64) error {
		if userID != 7 || modelID != 61 {
			t.Fatalf("取消报价参数 user=%d model=%d", userID, modelID)
		}
		return nil
	}
	repository.deleteOffer = func(_ context.Context, userID int, modelID int64) error {
		if userID != 7 || modelID != 61 {
			t.Fatalf("删除模型参数 user=%d model=%d", userID, modelID)
		}
		return nil
	}
	synchronizer := &offerSynchronizerStub{}
	services.Routing = repository
	services.OfferSync = synchronizer
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)

	for _, testCase := range []struct {
		method string
		path   string
		body   string
		status string
	}{
		{method: http.MethodPost, path: "/api/merchant/offers/61/status", body: `{"enabled":true}`, status: "active"},
		{method: http.MethodPost, path: "/api/merchant/offers/61/status", body: `{"enabled":false}`, status: "inactive"},
		{method: http.MethodDelete, path: "/api/merchant/offers/61", status: "deleted"},
	} {
		request := httptest.NewRequest(testCase.method, testCase.path, strings.NewReader(testCase.body))
		request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
		request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"`+testCase.status+`"`) {
			t.Fatalf("模型状态响应=%d %s", response.Code, response.Body.String())
		}
	}
	if synchronizer.offerID != 44 || !synchronizer.activate {
		t.Fatalf("重新上架没有同步最新报价：%+v", synchronizer)
	}
}

func TestCreateIndependentRoutingPolicyWithCacheFloor(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("创建路由不应访问核心服务")
	}))
	repository := &routingRepositoryStub{}
	repository.createRoute = func(_ context.Context, userID int, input RoutingPolicyInput) (RoutingPolicy, error) {
		if userID != 7 || input.Name != "缓存主路由" || input.MinimumCacheHitPPM == nil || *input.MinimumCacheHitPPM != 650_000 {
			t.Fatalf("创建路由参数 user=%d input=%+v", userID, input)
		}
		return RoutingPolicy{ID: 91, UserID: userID, Name: input.Name, Mode: input.Mode, Priority: input.Priority, MinimumCacheHitPPM: input.MinimumCacheHitPPM}, nil
	}
	services.Routing = repository
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/routing-policies", strings.NewReader(`{"name":"缓存主路由","mode":"smart","priority":"cache","minimum_multiplier_bps":0,"maximum_multiplier_bps":50000,"minimum_cache_hit_ppm":650000,"merchant_ids":[]}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"id":91`) || !strings.Contains(response.Body.String(), `"minimum_cache_hit_ppm":650000`) {
		t.Fatalf("创建路由响应=%d %s", response.Code, response.Body.String())
	}
}

func TestRoutingMerchantSearchReturnsPagedCacheMetrics(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	repository.listRoutingMerchants = func(_ context.Context, keyword string, page int, pageSize int) (RoutingMerchantPage, error) {
		if keyword != "官方" || page != 2 || pageSize != 50 {
			t.Fatalf("商户搜索参数 keyword=%q page=%d size=%d", keyword, page, pageSize)
		}
		return RoutingMerchantPage{
			Items: []RoutingMerchant{{ID: 31, UserID: 7, Name: "ModelRoute 官方", ModelCount: 8, MinimumMultiplierBPS: 6000, MaximumMultiplierBPS: 9000, AvailabilityPPM: 990000, TTFTP95MS: 420, CacheHitPPM: 720000, CacheSampleInputTokens: 2000000, CacheWeightReady: true}},
			Page:  2, PageSize: 50, Total: 51,
		}, nil
	}
	services.Routing = repository
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/routing-merchants?keyword=%E5%AE%98%E6%96%B9&p=2&page_size=50", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"cache_hit_ppm":720000`) || !strings.Contains(response.Body.String(), `"total":51`) {
		t.Fatalf("路由商户搜索响应=%d %s", response.Code, response.Body.String())
	}
}

func TestFixedRoutingPolicyRequiresAtLeastOneMerchant(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	repository.createRoute = func(context.Context, int, RoutingPolicyInput) (RoutingPolicy, error) {
		t.Fatal("固定商户为空时不应写仓储")
		return RoutingPolicy{}, nil
	}
	services.Routing = repository
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/routing-policies", strings.NewReader(`{"name":"固定路由","mode":"fixed","priority":"price","minimum_multiplier_bps":2000,"maximum_multiplier_bps":6000,"merchant_ids":[]}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("固定路由空名单响应=%d %s", response.Code, response.Body.String())
	}
}

func TestRoutingPolicyValidationSupportsWeightedPrioritiesAndSmartCacheFloor(t *testing.T) {
	cacheFloor := 750_000
	for _, priority := range []string{"balanced", "price", "success", "latency", "cache"} {
		input := RoutingPolicyInput{
			Name: "主路由", Mode: "smart", Priority: priority,
			MinimumMultiplierBPS: 0, MaximumMultiplierBPS: 50_000,
			MinimumCacheHitPPM: &cacheFloor,
		}
		if err := validateRoutingPolicy(input); err != nil {
			t.Fatalf("优先方式 %s 校验失败：%v", priority, err)
		}
	}
}

func TestRoutingPolicyValidationRejectsInvalidCacheFloor(t *testing.T) {
	tooHigh := 1_000_001
	if err := validateRoutingPolicy(RoutingPolicyInput{
		Name: "智能路由", Mode: "smart", Priority: "cache",
		MinimumMultiplierBPS: 0, MaximumMultiplierBPS: 50_000,
		MinimumCacheHitPPM: &tooHigh,
	}); err == nil {
		t.Fatal("超过 100% 的缓存命中率阈值应被拒绝")
	}
	zero := 0
	if err := validateRoutingPolicy(RoutingPolicyInput{
		Name: "固定路由", Mode: "fixed", Priority: "balanced",
		MinimumMultiplierBPS: 0, MaximumMultiplierBPS: 50_000,
		MinimumCacheHitPPM: &zero, MerchantIDs: []int64{31},
	}); err == nil {
		t.Fatal("固定路由不应接受缓存命中率阈值")
	}
}

func TestCreateKeyBindsOnlyNewTokenToSelectedPolicy(t *testing.T) {
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
	repository.bindToken = func(_ context.Context, userID int, tokenID int, fingerprint string, policyID int64) (TokenRoutingBinding, error) {
		if userID != 7 || policyID != 91 || fingerprint != box.Fingerprint([]byte("sk-new-key")) {
			t.Fatalf("绑定参数=%d %d %q %d", userID, tokenID, fingerprint, policyID)
		}
		boundToken = tokenID
		return TokenRoutingBinding{TokenID: tokenID, PolicyID: policyID, PolicyName: "主路由"}, nil
	}
	services.Routing = repository
	services.Credentials = box
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/keys", strings.NewReader(`{"name":"新密钥","expired_time":-1,"unlimited_quota":true,"routing_policy_id":91}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || boundToken != 9 || listCount != 2 {
		t.Fatalf("密钥创建响应=%d %s，绑定=%d，列表次数=%d", response.Code, response.Body.String(), boundToken, listCount)
	}
}

func TestListAndUpdateRoutingPoliciesByPolicyID(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	repository.listRoutes = func(_ context.Context, userID int) ([]RoutingPolicy, error) {
		return []RoutingPolicy{{ID: 91, UserID: userID, Name: "主路由", Mode: "smart", Priority: "balanced", BoundKeyCount: 2}}, nil
	}
	repository.updateRoute = func(_ context.Context, userID int, policyID int64, input RoutingPolicyInput) (RoutingPolicy, error) {
		if userID != 7 || policyID != 91 || input.Name != "低价路由" || input.Priority != "price" {
			t.Fatalf("修改路由参数 user=%d policy=%d input=%+v", userID, policyID, input)
		}
		return RoutingPolicy{ID: policyID, UserID: userID, Name: input.Name, Mode: input.Mode, Priority: input.Priority, BoundKeyCount: 2}, nil
	}
	services.Routing = repository
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)

	request := httptest.NewRequest(http.MethodGet, "/api/routing-policies", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"bound_key_count":2`) {
		t.Fatalf("路由列表响应=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPut, "/api/routing-policies/91", strings.NewReader(`{"name":"低价路由","mode":"smart","priority":"price","minimum_multiplier_bps":0,"maximum_multiplier_bps":50000,"merchant_ids":[]}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"name":"低价路由"`) {
		t.Fatalf("修改路由响应=%d %s", response.Code, response.Body.String())
	}
}

func TestCreateKeyRequiresRoutingPolicy(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("缺少路由时不应访问核心服务")
	}))
	services.Routing = &routingRepositoryStub{}
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/keys", strings.NewReader(`{"name":"新密钥","expired_time":-1,"unlimited_quota":true}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("缺少路由响应=%d %s", response.Code, response.Body.String())
	}
}

func TestCreateKeyDeletesNewTokenWhenRouteBindingFails(t *testing.T) {
	listCount := 0
	deleted := false
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/token/":
			listCount++
			items := `[{"id":8,"name":"旧密钥"}]`
			if listCount == 2 {
				items = `[{"id":9,"name":"新密钥"},{"id":8,"name":"旧密钥"}]`
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"items":` + items + `,"total":1}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/token/":
			_, _ = w.Write([]byte(`{"success":true,"data":null}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/token/9/key":
			_, _ = w.Write([]byte(`{"success":true,"data":{"key":"sk-new-key"}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/token/9":
			deleted = true
			_, _ = w.Write([]byte(`{"success":true,"data":null}`))
		default:
			t.Fatalf("未知核心调用=%s %s", r.Method, r.URL.RequestURI())
		}
	}))
	repository := &routingRepositoryStub{}
	repository.bindToken = func(context.Context, int, int, string, int64) (TokenRoutingBinding, error) {
		return TokenRoutingBinding{}, ErrRoutingNotFound
	}
	services.Routing = repository
	services.Credentials = testCredentialBox(t)
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/keys", strings.NewReader(`{"name":"新密钥","expired_time":-1,"unlimited_quota":true,"routing_policy_id":91}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !deleted {
		t.Fatalf("绑定失败补偿响应=%d %s deleted=%v", response.Code, response.Body.String(), deleted)
	}
}

func TestKeyListIncludesRoutingPolicyAndSupportsRebind(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/token/" {
			t.Fatalf("密钥列表核心请求=%s %s", r.Method, r.URL.RequestURI())
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"page":1,"page_size":20,"total":1,"items":[{"id":9,"name":"主密钥"}]}}`))
	}))
	repository := &routingRepositoryStub{}
	repository.listBindings = func(_ context.Context, userID int) ([]TokenRoutingBinding, error) {
		return []TokenRoutingBinding{{TokenID: 9, PolicyID: 91, PolicyName: "主路由"}}, nil
	}
	repository.rebindToken = func(_ context.Context, userID int, tokenID int, policyID int64) (TokenRoutingBinding, error) {
		if userID != 7 || tokenID != 9 || policyID != 92 {
			t.Fatalf("重新绑定参数=%d %d %d", userID, tokenID, policyID)
		}
		return TokenRoutingBinding{TokenID: tokenID, PolicyID: policyID, PolicyName: "备用路由"}, nil
	}
	services.Routing = repository
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)

	request := httptest.NewRequest(http.MethodGet, "/api/keys?p=1&page_size=20", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"routing_policy_id":91`) || !strings.Contains(response.Body.String(), `"routing_policy_name":"主路由"`) {
		t.Fatalf("密钥路由列表=%d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPut, "/api/keys/9/routing-policy", strings.NewReader(`{"routing_policy_id":92}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"routing_policy_name":"备用路由"`) {
		t.Fatalf("重新绑定响应=%d %s", response.Code, response.Body.String())
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
		return []MarketOffer{{OfferID: 42, Model: "gpt-5.5", MerchantName: "北辰云智", MultiplierBPS: 4000, HealthState: "healthy", CacheHitPPM: 720000, CacheSampleInputTokens: 2000000, CacheWeightReady: true, InternalAlias: "mkt-ov-secret"}}, nil
	}
	services.Routing = repository
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/offers?model=gpt-5.5", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "mkt-ov-secret") || !strings.Contains(response.Body.String(), "北辰云智") || !strings.Contains(response.Body.String(), `"cache_hit_ppm":720000`) {
		t.Fatalf("市场响应=%d %s", response.Code, response.Body.String())
	}
}

func TestMerchantOffersReturnAggregateAndChannelCacheMetrics(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &routingRepositoryStub{}
	repository.listMerchantOffers = func(_ context.Context, userID int) ([]MerchantOfferView, error) {
		return []MerchantOfferView{{
			OfferID: 42, Model: "gpt-5.5", CacheHitPPM: 720000, CacheSampleInputTokens: 2000000, CacheWeightReady: true,
			Channels: []OfferChannelView{{MerchantChannelID: 51, ChannelName: "主渠道", CacheHitPPM: 680000, CacheSampleInputTokens: 1200000, CacheWeightReady: true}},
		}}, nil
	}
	services.Routing = repository
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/merchant/offers", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"cache_hit_ppm":720000`) || !strings.Contains(response.Body.String(), `"cache_hit_ppm":680000`) {
		t.Fatalf("商户模型缓存响应=%d %s", response.Code, response.Body.String())
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
