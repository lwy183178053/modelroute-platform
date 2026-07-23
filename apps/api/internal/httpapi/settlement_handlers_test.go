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
	return MerchantWithdrawal{
		ID: 8, WithdrawalNo: command.WithdrawalNo, MerchantID: 31, AmountQuota: command.AmountQuota,
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
	repository := &settlementRepositoryStub{balance: MerchantBalance{MerchantID: 31, AvailableQuota: 900}}
	services.Merchants = &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
		return MerchantIdentity{MerchantID: 31, UserID: 7, Status: "active", IsMerchant: true}, nil
	}}
	services.Credentials = testCredentialBox(t)
	services.Settlement = repository
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	account := "13800132621"
	request := httptest.NewRequest(http.MethodPost, "/api/merchant/withdrawals", strings.NewReader(`{"amount_quota":500,"recipient_type":"alipay","recipient_account":"`+account+`"}`))
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
}

func TestAdminWithdrawalListDecryptsRecipientAndSupportsActions(t *testing.T) {
	box := testCredentialBox(t)
	encrypted, err := box.Encrypt([]byte("6217000012341098"))
	if err != nil {
		t.Fatal(err)
	}
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &settlementRepositoryStub{withdrawals: []MerchantWithdrawal{{
		ID: 9, WithdrawalNo: "MW-9", MerchantID: 31, MerchantName: "北辰云智", AmountQuota: 500,
		RecipientType: "bank", EncryptedRecipientAccount: encrypted, RecipientAccountMasked: "6217 **** 1098", Status: "approved",
	}}}
	services.Credentials = box
	services.Settlement = repository
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)

	listRequest := httptest.NewRequest(http.MethodGet, "/api/admin/withdrawals?status=approved", nil)
	listRequest.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), `"recipient_account":"6217000012341098"`) {
		t.Fatalf("管理员提现列表=%d %s", listResponse.Code, listResponse.Body.String())
	}

	actions := []struct {
		path string
		body string
		want string
	}{
		{path: "/api/admin/withdrawals/9/approve", body: `{"reason":"账本与收款信息核对通过"}`, want: "approve:账本与收款信息核对通过"},
		{path: "/api/admin/withdrawals/9/reject", body: `{"reason":"收款信息需要重新提交"}`, want: "reject:收款信息需要重新提交"},
		{path: "/api/admin/withdrawals/9/paid", body: `{"payout_reference":"BANK-009","payout_proof":"银行回单 009"}`, want: "paid:BANK-009:银行回单 009"},
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
