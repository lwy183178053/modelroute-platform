package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type adminOperationCall struct {
	Method string
	Path   string
	Body   any
}

type adminOperationStub struct {
	calls   []adminOperationCall
	handler func(string, string, any) ([]byte, error)
}

type usageAliasResolverStub struct {
	aliases map[string]string
}

func (s usageAliasResolverStub) PublicModelNames(_ context.Context, aliases []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, alias := range aliases {
		if model := s.aliases[alias]; model != "" {
			result[alias] = model
		}
	}
	return result, nil
}

type adminOverviewStoreStub struct {
	snapshot AdminOverviewSnapshot
}

func (s adminOverviewStoreStub) Snapshot(context.Context, time.Time) (AdminOverviewSnapshot, error) {
	return s.snapshot, nil
}

func (s *adminOperationStub) Do(_ context.Context, method string, path string, body any) ([]byte, error) {
	s.calls = append(s.calls, adminOperationCall{Method: method, Path: path, Body: body})
	if s.handler == nil {
		return []byte("null"), nil
	}
	return s.handler(method, path, body)
}

func TestUsageReturnsRealNewAPIDataWithoutInternalOfferAlias(t *testing.T) {
	const alias = "mkt-ov-0123456789abcdef01234567"
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("New-Api-User") != "7" || r.Header.Get("Cookie") != "session=upstream" {
			t.Fatalf("用户用量上游鉴权头不符合预期")
		}
		var data any
		switch r.URL.Path {
		case "/api/log/self":
			data = map[string]any{
				"page": 1, "page_size": 20, "total": 1,
				"items": []any{map[string]any{
					"id": 1, "type": 2, "model_name": alias, "quota": 500000,
					"content": "模型 " + alias + " 消费", "other": `{"upstream_model_name":"` + alias + `"}`,
				}},
			}
		case "/api/log/self/stat":
			data = map[string]any{"quota": 500000, "rpm": 2, "tpm": 320}
		case "/api/data/self":
			data = []any{map[string]any{"created_at": 1784649600, "model_name": alias, "quota": 500000, "count": 1}}
		default:
			t.Fatalf("未知用户用量上游路径：%s", r.URL.RequestURI())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
	}))
	services.UsageRecords = &usageRecordReaderStub{page: UsageRecordPage{
		Page: 1, PageSize: 20, Total: 1,
		Items: []UsageRecord{{ID: 1, ModelName: "gpt-4o-mini", GrossQuota: 500000}},
	}}
	services.UsageAliases = usageAliasResolverStub{aliases: map[string]string{alias: "gpt-4o-mini"}}
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/usage?start_timestamp=1784649600&end_timestamp=1784736000&p=1&page_size=20&type=2", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("读取用户用量状态码=%d，响应=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), alias) || strings.Contains(response.Body.String(), "mkt-ov-") {
		t.Fatalf("用户用量响应泄露内部报价别名：%s", response.Body.String())
	}
	for _, expected := range []string{`"quota":500000`, `"rpm":2`, `"total":1`, `"trend"`, `gpt-4o-mini`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("用户用量响应缺少 %s：%s", expected, response.Body.String())
		}
	}
}

func TestAdminOverviewUsesRepositorySnapshotAndRoleChecks(t *testing.T) {
	lastSuccess := time.Date(2026, 7, 22, 16, 0, 0, 0, time.UTC)
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.Overview = adminOverviewStoreStub{snapshot: AdminOverviewSnapshot{
		TodayGrossQuota:             4_500_000,
		TodayMerchantQuota:          4_050_000,
		TodayPlatformQuota:          450_000,
		TodayRefundQuota:            500_000,
		TodayRequestCount:           42,
		TodaySuccessfulRequests:     41,
		PendingMerchantApplications: 3,
		PendingChannelReviews:       2,
		PendingWithdrawals:          1,
		OpenReconciliation:          1,
		TotalChannels:               8,
		ActiveChannels:              6,
		CoolingChannels:             1,
		DisabledChannels:            1,
		PaymentEnabled:              true,
		LastSyncedThrough:           lastSuccess.Unix() - 5,
		LastSyncSuccessAt:           &lastSuccess,
	}}
	handler := NewServerWithServices("test", Dependencies{
		Database: stubDependency{}, Redis: stubDependency{}, NewAPI: stubDependency{},
	}, services)

	regularToken := createPlatformSession(t, services, 7, 1)
	regularRequest := httptest.NewRequest(http.MethodGet, "/api/admin/overview", nil)
	regularRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: regularToken})
	regularResponse := httptest.NewRecorder()
	handler.ServeHTTP(regularResponse, regularRequest)
	if regularResponse.Code != http.StatusForbidden {
		t.Fatalf("普通用户读取管理员总览状态码=%d", regularResponse.Code)
	}

	adminToken := createPlatformSession(t, services, 1, 100)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/overview", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: adminToken})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("管理员总览响应=%d %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{
		`"today_platform_quota":450000`, `"today_merchant_quota":4050000`, `"today_request_count":42`,
		`"pending_merchant_applications":3`, `"cooling_channels":1`, `"payment_enabled":true`,
		`"database":{"status":"healthy"`, `"new_api":{"status":"healthy"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("管理员总览缺少 %s：%s", expected, response.Body.String())
		}
	}
}

func TestAdminChannelLogsRequireAdminAndUseSharedUsageRecords(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	reader := &usageRecordReaderStub{page: UsageRecordPage{Page: 1, PageSize: 200, Total: 1, Items: []UsageRecord{{
		ID: 9, ConsumerUserID: 27, ModelName: "gpt-5.6", MerchantUserID: 7, MerchantName: "北辰云智",
		ChannelID: 51, ChannelName: "主线路", GrossQuota: 700000,
	}}}}
	services.UsageRecords = reader
	services.Merchants = &merchantChannelLookupStub{
		merchantRepositoryStub: &merchantRepositoryStub{},
		channel: func(context.Context, int64) (MerchantChannel, error) {
			return MerchantChannel{ID: 51, UserID: 7, NewAPIChannelID: 41}, nil
		},
	}
	handler := NewServerWithServices("test", Dependencies{}, services)

	regularToken := createPlatformSession(t, services, 7, 1)
	regularRequest := httptest.NewRequest(http.MethodGet, "/api/admin/channel-logs", nil)
	regularRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: regularToken})
	regularResponse := httptest.NewRecorder()
	handler.ServeHTTP(regularResponse, regularRequest)
	if regularResponse.Code != http.StatusForbidden {
		t.Fatalf("普通用户读取管理员用量状态码=%d", regularResponse.Code)
	}

	adminToken := createPlatformSession(t, services, 1, 100)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/channel-logs?merchant_user_id=7&channel_id=51&model_name=gpt-5.6&p=1&page_size=200", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: adminToken})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"total":1`) || !strings.Contains(response.Body.String(), `"user_id":27`) {
		t.Fatalf("管理员用量响应=%d %s", response.Code, response.Body.String())
	}
	if !reader.query.Admin || reader.query.FilterMerchantUserID != 7 || reader.query.ChannelID != 51 || reader.query.ModelName != "gpt-5.6" || reader.query.PageSize != 200 {
		t.Fatalf("管理员调用记录查询=%+v", reader.query)
	}
}

type merchantChannelLookupStub struct {
	*merchantRepositoryStub
	channel func(context.Context, int64) (MerchantChannel, error)
}

func (s *merchantChannelLookupStub) Channel(ctx context.Context, channelID int64) (MerchantChannel, error) {
	return s.channel(ctx, channelID)
}

func TestAdminChannelLogsRejectMismatchedMerchantAndChannel(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.UsageRecords = &usageRecordReaderStub{}
	services.Merchants = &merchantChannelLookupStub{
		merchantRepositoryStub: &merchantRepositoryStub{},
		channel: func(_ context.Context, channelID int64) (MerchantChannel, error) {
			if channelID != 51 {
				t.Fatalf("平台渠道 ID=%d", channelID)
			}
			return MerchantChannel{ID: 51, UserID: 7, NewAPIChannelID: 41}, nil
		},
	}
	token := createPlatformSession(t, services, 1, 100)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/channel-logs?channel_id=51&merchant_user_id=8", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("渠道与商户不匹配响应=%d %s", response.Code, response.Body.String())
	}
}

func TestPlatformSettingsAreWhitelistedAndEmptySMTPPasswordIsPreserved(t *testing.T) {
	admin := &adminOperationStub{handler: func(method string, path string, body any) ([]byte, error) {
		if method == http.MethodGet && path == "/api/option/" {
			return []byte(`[
				{"key":"SMTPServer","value":"smtp.example.com"},
				{"key":"SMTPPort","value":"465"},
				{"key":"SMTPAccount","value":"notice@example.com"},
				{"key":"SMTPFrom","value":"ModelRoute <notice@example.com>"},
				{"key":"SMTPSSLEnabled","value":"true"},
				{"key":"RegisterEnabled","value":"true"},
				{"key":"PasswordLoginEnabled","value":"true"},
				{"key":"PasswordRegisterEnabled","value":"true"},
				{"key":"EmailVerificationEnabled","value":"true"},
				{"key":"EpayKey","value":"should-never-be-returned"}
			]`), nil
		}
		if method == http.MethodPut && path == "/api/option/" {
			option, ok := body.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("设置请求类型=%T", body)
			}
			if option["key"] == "SMTPToken" {
				return nil, fmt.Errorf("空 SMTP 密码覆盖了旧密码")
			}
			return []byte("null"), nil
		}
		return nil, fmt.Errorf("未知设置调用 %s %s", method, path)
	}}
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.AdminNewAPI = admin
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/admin/platform-settings", nil)
	getRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK || strings.Contains(getResponse.Body.String(), "EpayKey") || strings.Contains(getResponse.Body.String(), "should-never") {
		t.Fatalf("平台设置读取响应=%d %s", getResponse.Code, getResponse.Body.String())
	}

	body := `{
		"smtp":{"server":"smtp.example.com","port":465,"account":"notice@example.com","from":"ModelRoute <notice@example.com>","password":"","ssl_enabled":true,"starttls_enabled":false,"insecure_skip_verify":false,"force_auth_login":false},
		"registration":{"email_verification_enabled":true,"register_enabled":true,"password_login_enabled":true,"password_register_enabled":true}
	}`
	putRequest := httptest.NewRequest(http.MethodPut, "/api/admin/platform-settings", strings.NewReader(body))
	putRequest.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	putRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	putResponse := httptest.NewRecorder()
	handler.ServeHTTP(putResponse, putRequest)
	if putResponse.Code != http.StatusOK {
		t.Fatalf("平台设置保存响应=%d %s", putResponse.Code, putResponse.Body.String())
	}
	for _, call := range admin.calls {
		if call.Method == http.MethodPut {
			option := call.Body.(map[string]any)
			if option["key"] == "SMTPToken" {
				t.Fatalf("空 SMTP 密码被写入上游")
			}
		}
	}
}

func TestAdminGroupsEndpointsAreNotExposed(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/admin/groups", nil)
	getRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusNotFound {
		t.Fatalf("分组读取端点仍暴露：%d %s", getResponse.Code, getResponse.Body.String())
	}

	putRequest := httptest.NewRequest(http.MethodPut, "/api/admin/groups", strings.NewReader(`{"groups":[{"name":"default","ratio_bps":10000},{"name":"vip","ratio_bps":7500}]}`))
	putRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	putResponse := httptest.NewRecorder()
	handler.ServeHTTP(putResponse, putRequest)
	if putResponse.Code != http.StatusNotFound {
		t.Fatalf("分组保存端点仍暴露：%d %s", putResponse.Code, putResponse.Body.String())
	}
}

func TestRedemptionOperationsKeepQuotaAsInteger(t *testing.T) {
	admin := &adminOperationStub{handler: func(method string, path string, body any) ([]byte, error) {
		switch {
		case method == http.MethodGet && strings.HasPrefix(path, "/api/redemption/?"):
			return []byte(`{"page":1,"page_size":20,"total":1,"items":[{"id":6,"name":"活动码","key":"FULL_CODE","status":1,"quota":500000,"count":1,"expired_time":0}]}`), nil
		case method == http.MethodPost && path == "/api/redemption/":
			request := body.(map[string]any)
			if request["quota"] != 500000 || request["count"] != 2 {
				return nil, fmt.Errorf("兑换码整数参数=%v", request)
			}
			return []byte(`["CODE_A","CODE_B"]`), nil
		default:
			return nil, fmt.Errorf("未知兑换码调用 %s %s", method, path)
		}
	}}
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.AdminNewAPI = admin
	handler := NewServerWithServices("test", Dependencies{}, services)

	regularToken := createPlatformSession(t, services, 7, 1)
	regularRequest := httptest.NewRequest(http.MethodGet, "/api/admin/redemptions", nil)
	regularRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: regularToken})
	regularResponse := httptest.NewRecorder()
	handler.ServeHTTP(regularResponse, regularRequest)
	if regularResponse.Code != http.StatusForbidden {
		t.Fatalf("普通用户读取兑换码状态码=%d", regularResponse.Code)
	}

	adminToken := createPlatformSession(t, services, 1, 100)
	listRequest := httptest.NewRequest(http.MethodGet, "/api/admin/redemptions?p=1&page_size=20", nil)
	listRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: adminToken})
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), "FULL_CODE") {
		t.Fatalf("兑换码列表响应=%d %s", listResponse.Code, listResponse.Body.String())
	}

	createRequest := httptest.NewRequest(http.MethodPost, "/api/admin/redemptions", strings.NewReader(`{"name":"活动码","quota":500000,"count":2,"expired_time":0}`))
	createRequest.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	createRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: adminToken})
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated || !strings.Contains(createResponse.Body.String(), "CODE_A") {
		t.Fatalf("创建兑换码响应=%d %s", createResponse.Code, createResponse.Body.String())
	}
}

func TestAdminRedemptionUpdateStatusAndDeleteLifecycle(t *testing.T) {
	admin := &adminOperationStub{handler: func(method string, path string, body any) ([]byte, error) {
		switch {
		case method == http.MethodPut && path == "/api/redemption/":
			request := body.(map[string]any)
			if request["id"] != 6 || request["name"] != "更新活动码" || request["quota"] != 700000 {
				return nil, fmt.Errorf("编辑兑换码参数=%v", request)
			}
			if _, exists := request["count"]; exists {
				return nil, fmt.Errorf("编辑兑换码不应携带 count")
			}
			return []byte(`{"id":6,"name":"更新活动码","quota":700000,"status":1}`), nil
		case method == http.MethodPut && path == "/api/redemption/?status_only=true":
			request := body.(map[string]any)
			if request["id"] != 6 || request["status"] != 2 {
				return nil, fmt.Errorf("停用兑换码参数=%v", request)
			}
			return []byte(`{"id":6,"status":2}`), nil
		case method == http.MethodDelete && path == "/api/redemption/6":
			return []byte("null"), nil
		case method == http.MethodDelete && path == "/api/redemption/invalid":
			return []byte("3"), nil
		default:
			return nil, fmt.Errorf("未知兑换码生命周期调用 %s %s", method, path)
		}
	}}
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.AdminNewAPI = admin
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPut, path: "/api/admin/redemptions/6", body: `{"name":"更新活动码","quota":700000,"expired_time":0}`},
		{method: http.MethodPost, path: "/api/admin/redemptions/6/status", body: `{"enabled":false}`},
		{method: http.MethodDelete, path: "/api/admin/redemptions/6"},
		{method: http.MethodDelete, path: "/api/admin/redemptions/invalid"},
	}
	for _, test := range tests {
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
		request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("兑换码生命周期 %s %s 响应=%d %s", test.method, test.path, response.Code, response.Body.String())
		}
	}
}

func TestUserRedeemsCodeThroughOwnNewAPISession(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/user/topup" || r.Header.Get("New-Api-User") != "7" || r.Header.Get("Cookie") != "session=upstream" {
			t.Fatalf("兑换码上游请求=%s %s user=%s cookie=%s", r.Method, r.URL.Path, r.Header.Get("New-Api-User"), r.Header.Get("Cookie"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "CODE_A" {
			t.Fatalf("兑换码参数=%v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": 500000})
	}))
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/redemptions/redeem", strings.NewReader(`{"key":"CODE_A"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"data":500000`) {
		t.Fatalf("兑换响应=%d %s", response.Code, response.Body.String())
	}
}
