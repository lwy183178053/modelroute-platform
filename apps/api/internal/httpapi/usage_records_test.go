package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type usageRecordReaderStub struct {
	query UsageRecordQuery
	page  UsageRecordPage
}

func (s *usageRecordReaderStub) ListUsageRecords(_ context.Context, query UsageRecordQuery) (UsageRecordPage, error) {
	s.query = query
	return s.page, nil
}

func TestUsageRecordQueryDoesNotApplyDefaultRetentionWindow(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/usage?p=1&page_size=20", nil)
	response := httptest.NewRecorder()
	query, ok := usageRecordQueryFromRequest(response, request)
	if !ok || query.StartTimestamp != 0 || query.EndTimestamp != 0 {
		t.Fatalf("调用记录不应默认限制时间范围：ok=%v query=%+v body=%s", ok, query, response.Body.String())
	}
}

func TestUserUsageFiltersSettledConsumptionByModelAndMerchant(t *testing.T) {
	upstreamPaths := make([]string, 0, 2)
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPaths = append(upstreamPaths, r.URL.Path)
		var data any
		switch r.URL.Path {
		case "/api/log/self/stat":
			data = map[string]any{"quota": 900000, "rpm": 1, "tpm": 200}
		case "/api/data/self":
			data = []any{}
		default:
			t.Fatalf("用户调用记录不应读取上游日志明细：%s", r.URL.RequestURI())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
	}))
	reader := &usageRecordReaderStub{page: UsageRecordPage{
		Page: 2, PageSize: 200, Total: 1,
		Items: []UsageRecord{{
			ID: 901, CreatedAt: time.Unix(1784649600, 0).UTC(), FirstTokenMS: 286, ModelName: "gpt-5.6",
			TokenID: 12, PromptTokens: 120, CompletionTokens: 80, CacheTokens: 40,
			GrossQuota: 1000000, MerchantUserID: 17, MerchantName: "北辰云智", RequestID: "req-user-1",
		}},
		Filters: UsageRecordFilters{
			Models:    []string{"gpt-5.6"},
			Merchants: []UsageMerchantOption{{UserID: 17, Name: "北辰云智"}},
		},
	}}
	services.UsageRecords = reader
	token := createPlatformSession(t, services, 7, 1)
	request := httptest.NewRequest(http.MethodGet, "/api/usage?model_name=gpt-5.6&merchant_user_id=17&p=2&page_size=200", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("用户调用记录响应=%d %s", response.Code, response.Body.String())
	}
	if reader.query.ConsumerUserID != 7 || reader.query.MerchantUserID != 0 || reader.query.FilterMerchantUserID != 17 || reader.query.ModelName != "gpt-5.6" || reader.query.Page != 2 || reader.query.PageSize != 200 {
		t.Fatalf("用户调用记录查询=%+v", reader.query)
	}
	for _, expected := range []string{`"merchant_user_id":17`, `"merchant_name":"北辰云智"`, `"first_token_ms":286`, `"prompt_tokens":120`, `"completion_tokens":80`, `"filters"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("响应缺少 %s：%s", expected, response.Body.String())
		}
	}
	if strings.Contains(response.Body.String(), `"type":6`) || strings.Contains(response.Body.String(), "退款") {
		t.Fatalf("用户调用记录泄露退款：%s", response.Body.String())
	}
	if len(upstreamPaths) != 2 {
		t.Fatalf("上游统计调用=%v", upstreamPaths)
	}
}

func TestMerchantUsageUsesSessionMerchantAndReturnsNetIncome(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("商户调用记录不应读取上游用户日志")
	}))
	repository := &merchantRepositoryStub{identity: func(_ context.Context, userID int) (MerchantIdentity, error) {
		return MerchantIdentity{MerchantID: 31, UserID: userID, DisplayName: "北辰云智", Status: "active", IsMerchant: true}, nil
	}}
	services.Merchants = repository
	services.Credentials = testCredentialBox(t)
	reader := &usageRecordReaderStub{page: UsageRecordPage{
		Page: 1, PageSize: 20, Total: 1,
		Items: []UsageRecord{{
			ID: 902, CreatedAt: time.Unix(1784649700, 0).UTC(), FirstTokenMS: 1340, ModelName: "gpt-5.6",
			ChannelID: 51, ChannelName: "主线路", PromptTokens: 50, CompletionTokens: 25,
			MerchantIncomeQuota: 900000, RequestID: "req-merchant-1",
		}},
		Filters: UsageRecordFilters{Models: []string{"gpt-5.6"}},
	}}
	services.UsageRecords = reader
	token := createPlatformSession(t, services, 17, 1)
	request := httptest.NewRequest(http.MethodGet, "/api/merchant/usage?model_name=gpt-5.6&merchant_user_id=999&p=1&page_size=20", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("商户调用记录响应=%d %s", response.Code, response.Body.String())
	}
	if reader.query.MerchantUserID != 17 || reader.query.ConsumerUserID != 0 || reader.query.FilterMerchantUserID != 0 {
		t.Fatalf("商户调用记录查询=%+v", reader.query)
	}
	for _, expected := range []string{`"channel_id":51`, `"channel_name":"主线路"`, `"first_token_ms":1340`, `"merchant_income_quota":900000`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("商户响应缺少 %s：%s", expected, response.Body.String())
		}
	}
}

func TestUsageRejectsInvalidMerchantFilter(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.UsageRecords = &usageRecordReaderStub{}
	token := createPlatformSession(t, services, 7, 1)
	request := httptest.NewRequest(http.MethodGet, "/api/usage?merchant_user_id="+url.QueryEscape("not-an-id"), nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	NewServerWithServices("test", Dependencies{}, services).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("无效商户筛选响应=%d %s", response.Code, response.Body.String())
	}
}
