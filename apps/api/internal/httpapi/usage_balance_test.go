package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type usageTokenResolverStub struct {
	RoutingRepository
	userID int
	err    error
}

func (s usageTokenResolverStub) TokenAccountID(context.Context, string) (int, error) {
	return s.userID, s.err
}

type usageAdminNewAPIStub struct {
	data []byte
	err  error
	path string
}

func (s *usageAdminNewAPIStub) Do(_ context.Context, method string, path string, _ any) ([]byte, error) {
	if method != http.MethodGet {
		return nil, fmt.Errorf("unexpected method %s", method)
	}
	s.path = path
	return s.data, s.err
}

func TestUsageBalanceReturnsCurrentAccountQuotaForActiveKey(t *testing.T) {
	credentials := testCredentialBox(t)
	admin := &usageAdminNewAPIStub{data: []byte(`{"id":7,"status":1,"quota":257}`)}
	handler := NewServerWithServices("test", Dependencies{}, Services{
		Routing:      usageTokenResolverStub{userID: 7},
		Credentials:  credentials,
		AdminNewAPI:  admin,
		QuotaPerUnit: 500_000,
	})
	request := httptest.NewRequest(http.MethodGet, "/v1/usage", nil)
	request.Header.Set("Authorization", "Bearer sk-active-key")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("额度查询响应=%d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("缓存策略=%q", response.Header().Get("Cache-Control"))
	}
	if admin.path != "/api/user/7" {
		t.Fatalf("账户查询路径=%q", admin.path)
	}
	var body struct {
		IsValid        bool    `json:"isValid"`
		IsActive       bool    `json:"is_active"`
		Remaining      float64 `json:"remaining"`
		Balance        float64 `json:"balance"`
		Unit           string  `json:"unit"`
		RemainingQuota int64   `json:"remaining_quota"`
		QuotaPerUnit   int64   `json:"quota_per_unit"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.IsValid || !body.IsActive || body.Remaining != 0.000514 || body.Balance != 0.000514 || body.Unit != "CNY" || body.RemainingQuota != 257 || body.QuotaPerUnit != 500_000 {
		t.Fatalf("额度响应=%+v", body)
	}
}

func TestUsageBalanceRejectsInvalidKeysWithoutReadingAccount(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		header string
		err    error
	}{
		{name: "缺少 Bearer", header: "", err: nil},
		{name: "停用或删除密钥", header: "Bearer sk-revoked", err: ErrTokenNotFound},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			admin := &usageAdminNewAPIStub{data: []byte(`{"id":7,"status":1,"quota":257}`)}
			handler := NewServerWithServices("test", Dependencies{}, Services{
				Routing:      usageTokenResolverStub{err: testCase.err},
				Credentials:  testCredentialBox(t),
				AdminNewAPI:  admin,
				QuotaPerUnit: 500_000,
			})
			request := httptest.NewRequest(http.MethodGet, "/v1/usage", nil)
			if testCase.header != "" {
				request.Header.Set("Authorization", testCase.header)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"code":"invalid_api_key"`) {
				t.Fatalf("无效密钥响应=%d %s", response.Code, response.Body.String())
			}
			if admin.path != "" {
				t.Fatalf("无效密钥不应查询账户，实际=%q", admin.path)
			}
		})
	}
}

func TestUsageBalanceDoesNotExposeDisabledOrUnavailableAccountBalance(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		data       []byte
		adminErr   error
		statusCode int
		code       string
	}{
		{name: "账号停用", data: []byte(`{"id":7,"status":2,"quota":257}`), statusCode: http.StatusForbidden, code: "account_inactive"},
		{name: "账户编号不匹配", data: []byte(`{"id":8,"status":1,"quota":257}`), statusCode: http.StatusServiceUnavailable, code: "balance_unavailable"},
		{name: "内部查询失败", adminErr: errors.New("core unavailable"), statusCode: http.StatusServiceUnavailable, code: "balance_unavailable"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			handler := NewServerWithServices("test", Dependencies{}, Services{
				Routing:      usageTokenResolverStub{userID: 7},
				Credentials:  testCredentialBox(t),
				AdminNewAPI:  &usageAdminNewAPIStub{data: testCase.data, err: testCase.adminErr},
				QuotaPerUnit: 500_000,
			})
			request := httptest.NewRequest(http.MethodGet, "/v1/usage", nil)
			request.Header.Set("Authorization", "Bearer sk-active-key")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.statusCode || !strings.Contains(response.Body.String(), `"code":"`+testCase.code+`"`) || strings.Contains(response.Body.String(), `"remaining"`) {
				t.Fatalf("账户异常响应=%d %s", response.Code, response.Body.String())
			}
		})
	}
}
