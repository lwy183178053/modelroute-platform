package httpapi

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

type paymentCreditRepositoryStub struct {
	staleBefore time.Time
	operations  []PaymentCreditOperation
	completed   []int64
	failed      map[int64]string
}

func (s *paymentCreditRepositoryStub) MarkStalePaymentCreditsUnknown(_ context.Context, before time.Time) (int, error) {
	s.staleBefore = before
	return 0, nil
}

func (s *paymentCreditRepositoryStub) ClaimPaymentCredits(context.Context, time.Time, int) ([]PaymentCreditOperation, error) {
	return s.operations, nil
}

func (s *paymentCreditRepositoryStub) CompletePaymentCredit(_ context.Context, operationID int64, _ time.Time) error {
	s.completed = append(s.completed, operationID)
	return nil
}

func (s *paymentCreditRepositoryStub) FailPaymentCredit(_ context.Context, operationID int64, status string, _ string, _ time.Time) error {
	if s.failed == nil {
		s.failed = make(map[int64]string)
	}
	s.failed[operationID] = status
	return nil
}

type paymentCreditAdminStub struct {
	err   error
	calls []map[string]any
}

func (s *paymentCreditAdminStub) Do(_ context.Context, method string, path string, body any) ([]byte, error) {
	if method != http.MethodPost || path != "/api/user/manage" {
		return nil, errors.New("unexpected request")
	}
	s.calls = append(s.calls, body.(map[string]any))
	return []byte(`null`), s.err
}

func TestPaymentCreditRunnerCreditsExactlyClaimedOperation(t *testing.T) {
	now := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	repository := &paymentCreditRepositoryStub{operations: []PaymentCreditOperation{{
		ID: 41, OrderID: 21, OrderNo: "MR202607220001", UserID: 7, NewAPIQuota: 50_000_000, Attempts: 1,
	}}}
	admin := &paymentCreditAdminStub{}
	runner := NewPaymentCreditRunner(repository, admin)
	if err := runner.RunCycle(context.Background(), now); err != nil {
		t.Fatalf("入账周期失败：%v", err)
	}
	if len(admin.calls) != 1 || admin.calls[0]["id"] != 7 || admin.calls[0]["action"] != "add_quota" || admin.calls[0]["mode"] != "add" || admin.calls[0]["value"] != int64(50_000_000) {
		t.Fatalf("New API 入账请求=%v", admin.calls)
	}
	if len(repository.completed) != 1 || repository.completed[0] != 41 || len(repository.failed) != 0 {
		t.Fatalf("入账落库 completed=%v failed=%v", repository.completed, repository.failed)
	}
	if repository.staleBefore != now.Add(-5*time.Minute) {
		t.Fatalf("处理中超时边界=%s", repository.staleBefore)
	}
}

func TestPaymentCreditRunnerRetriesOnlyExplicitNewAPIFailure(t *testing.T) {
	now := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		err        error
		wantStatus string
	}{
		{name: "明确响应失败", err: &newapi.APIError{StatusCode: http.StatusBadRequest, Message: "user not found"}, wantStatus: "failed"},
		{name: "网络结果未知", err: errors.New("请求 New API：context deadline exceeded"), wantStatus: "unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &paymentCreditRepositoryStub{operations: []PaymentCreditOperation{{
				ID: 41, OrderID: 21, OrderNo: "MR202607220001", UserID: 7, NewAPIQuota: 50_000_000, Attempts: 1,
			}}}
			admin := &paymentCreditAdminStub{err: test.err}
			runner := NewPaymentCreditRunner(repository, admin)
			if err := runner.RunCycle(context.Background(), now); err != nil {
				t.Fatalf("入账周期失败：%v", err)
			}
			if repository.failed[41] != test.wantStatus || len(repository.completed) != 0 {
				t.Fatalf("失败状态=%v，完成=%v", repository.failed, repository.completed)
			}
		})
	}
}
