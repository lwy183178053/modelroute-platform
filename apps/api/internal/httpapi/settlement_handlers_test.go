package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type settlementRepositoryStub struct {
	SettlementRepository
	balance     MerchantBalance
	created     CreateMerchantWithdrawalCommand
	withdrawals []MerchantWithdrawal
	action      string
}

func (s *settlementRepositoryStub) MerchantBalance(context.Context, int) (MerchantBalance, error) {
	return s.balance, nil
}

func (s *settlementRepositoryStub) MerchantWithdrawals(context.Context, int, int, int) ([]MerchantWithdrawal, int, error) {
	return s.withdrawals, len(s.withdrawals), nil
}

func (s *settlementRepositoryStub) CreateMerchantWithdrawal(_ context.Context, command CreateMerchantWithdrawalCommand) (MerchantWithdrawal, error) {
	s.created = command
	taxQuota, netPayoutQuota, _ := withdrawalBreakdown(command.AmountQuota, command.QuotaPerUnit, command.WithdrawalTaxBPS)
	return MerchantWithdrawal{
		ID: 8, WithdrawalNo: command.WithdrawalNo, MerchantID: 31, AmountQuota: command.AmountQuota,
		WithdrawalTaxBPS: command.WithdrawalTaxBPS, TaxQuota: taxQuota, NetPayoutQuota: netPayoutQuota,
		RecipientType: command.RecipientType, RecipientAccountMasked: command.RecipientAccountMasked,
		Status: "pending", CreatedAt: command.CreatedAt,
	}, nil
}

func (s *settlementRepositoryStub) ListMerchantWithdrawals(context.Context, int, int, string) ([]MerchantWithdrawal, int, error) {
	return s.withdrawals, len(s.withdrawals), nil
}

func (s *settlementRepositoryStub) ApproveMerchantWithdrawal(_ context.Context, id int64, adminID int, reason string, now time.Time) (MerchantWithdrawal, error) {
	s.action = "approve:" + reason
	return MerchantWithdrawal{ID: id, MerchantID: 31, Status: "approved", ReviewedByUserID: adminID, ReviewedAt: &now}, nil
}

func (s *settlementRepositoryStub) RejectMerchantWithdrawal(_ context.Context, id int64, adminID int, reason string, now time.Time) (MerchantWithdrawal, error) {
	s.action = "reject:" + reason
	return MerchantWithdrawal{ID: id, MerchantID: 31, Status: "rejected", ReviewedByUserID: adminID, ReviewedAt: &now}, nil
}

func (s *settlementRepositoryStub) PayMerchantWithdrawal(_ context.Context, id int64, adminID int, reference string, proof string, now time.Time) (MerchantWithdrawal, error) {
	s.action = "paid:" + reference + ":" + proof
	return MerchantWithdrawal{ID: id, MerchantID: 31, Status: "paid", PaidByUserID: adminID, PaidAt: &now, PayoutReference: reference}, nil
}

func TestMerchantWithdrawalEncryptsRecipientAndReturnsOnlyMask(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &settlementRepositoryStub{balance: MerchantBalance{MerchantID: 31, AvailableQuota: 100 * quotaPerCNY}}
	services.Merchants = &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
		return MerchantIdentity{MerchantID: 31, UserID: 7, Status: "active", IsMerchant: true}, nil
	}}
	services.Credentials = testCredentialBox(t)
	services.Settlement = repository
	services.Payments = &paymentRepositoryStub{currentSetting: func(context.Context) (PaymentSetting, error) {
		return PaymentSetting{WithdrawalTaxBPS: 750}, nil
	}}
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	account := "13800132621"
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/withdrawals", strings.NewReader(`{"amount_quota":25000000,"recipient_account":"`+account+`"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("状态码=%d 响应=%s", response.Code, response.Body.String())
	}
	if bytes.Contains(repository.created.EncryptedRecipientAccount, []byte(account)) || len(repository.created.EncryptedRecipientAccount) == 0 {
		t.Fatal("收款账户没有加密后写入仓储")
	}
	if repository.created.RecipientAccountMasked != "138****2621" || strings.Contains(response.Body.String(), account) {
		t.Fatalf("账户掩码=%q 响应=%s", repository.created.RecipientAccountMasked, response.Body.String())
	}
	if repository.created.RecipientType != "alipay" {
		t.Fatalf("提现收款方式=%q，期望固定为 alipay", repository.created.RecipientType)
	}
	if repository.created.WithdrawalTaxBPS != 750 || !strings.Contains(response.Body.String(), `"withdrawal_tax_bps":750`) || !strings.Contains(response.Body.String(), `"tax_quota":1875000`) || !strings.Contains(response.Body.String(), `"net_payout_quota":23125000`) {
		t.Fatalf("提现响应缺少税费或实际到账金额：%s", response.Body.String())
	}
}

func TestMerchantWithdrawalRejectsInvalidAmountsAndLegacyRecipientType(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.Merchants = &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
		return MerchantIdentity{MerchantID: 31, UserID: 7, Status: "active", IsMerchant: true}, nil
	}}
	services.Credentials = testCredentialBox(t)
	services.Settlement = &settlementRepositoryStub{}
	services.Payments = &paymentRepositoryStub{currentSetting: func(context.Context) (PaymentSetting, error) {
		return PaymentSetting{WithdrawalTaxBPS: 300}, nil
	}}
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)

	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "四十九元", body: `{"amount_quota":24500000,"recipient_account":"13800132621"}`, code: "invalid_withdrawal_amount"},
		{name: "五十点五元", body: `{"amount_quota":25250000,"recipient_account":"13800132621"}`, code: "invalid_withdrawal_amount"},
		{name: "旧银行卡类型", body: `{"amount_quota":25000000,"recipient_type":"bank","recipient_account":"13800132621"}`, code: "invalid_withdrawal"},
		{name: "支付宝账户过短", body: `{"amount_quota":25000000,"recipient_account":"1234"}`, code: "invalid_alipay_account"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/merchant/withdrawals", strings.NewReader(test.body))
			request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
			request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("响应=%d %s，期望错误=%s", response.Code, response.Body.String(), test.code)
			}
		})
	}
}

func TestAdminWithdrawalListDecryptsRecipientAndSupportsActions(t *testing.T) {
	box := testCredentialBox(t)
	encrypted, err := box.Encrypt([]byte("13800132621"))
	if err != nil {
		t.Fatal(err)
	}
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &settlementRepositoryStub{withdrawals: []MerchantWithdrawal{{
		ID: 9, WithdrawalNo: "MW-9", MerchantID: 31, MerchantName: "北辰云智", AmountQuota: 50 * quotaPerCNY,
		TaxQuota: 750_000, NetPayoutQuota: 24_250_000, RecipientType: "alipay",
		EncryptedRecipientAccount: encrypted, RecipientAccountMasked: "138****2621", Status: "approved",
	}}}
	services.Credentials = box
	services.Settlement = repository
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)

	listRequest := httptest.NewRequest(http.MethodGet, "/api/admin/withdrawals?status=approved", nil)
	listRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), `"recipient_account":"13800132621"`) || !strings.Contains(listResponse.Body.String(), `"net_payout_quota":24250000`) {
		t.Fatalf("管理员提现列表=%d %s", listResponse.Code, listResponse.Body.String())
	}

	actions := []struct {
		path string
		body string
		want string
	}{
		{path: "/api/admin/withdrawals/9/approve", body: `{"reason":"账本与收款信息核对通过"}`, want: "approve:账本与收款信息核对通过"},
		{path: "/api/admin/withdrawals/9/reject", body: `{"reason":"收款信息需要重新提交"}`, want: "reject:收款信息需要重新提交"},
		{path: "/api/admin/withdrawals/9/paid", body: `{"payout_reference":"ALIPAY-009","payout_proof":"支付宝转账凭证 009"}`, want: "paid:ALIPAY-009:支付宝转账凭证 009"},
	}
	for _, action := range actions {
		request := httptest.NewRequest(http.MethodPost, action.path, strings.NewReader(action.body))
		request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
		request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || repository.action != action.want {
			t.Fatalf("动作 %s 响应=%d %s，仓储=%q", action.path, response.Code, response.Body.String(), repository.action)
		}
	}
}

func TestRegularUserCannotReadAdminWithdrawals(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.Credentials = testCredentialBox(t)
	services.Settlement = &settlementRepositoryStub{}
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/withdrawals", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("普通用户读取管理员提现状态码=%d", response.Code)
	}
}

func TestProductRoutesDoNotExposeLedgerOrSettlementReconciliation(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	handler := NewServerWithServices("test", Dependencies{}, services)
	for _, path := range []string{
		"/api/merchant/ledger",
		"/api/admin/settlement-reconciliation",
		"/api/admin/settlement-reconciliation/12/ignore",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("旧产品接口 %s 仍然存在，状态码=%d", path, response.Code)
		}
	}
}
