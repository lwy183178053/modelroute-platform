package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

type PaymentCreditOperation struct {
	ID          int64
	OrderID     int64
	OrderNo     string
	UserID      int
	NewAPIQuota int64
	Attempts    int
}

type PaymentCreditRepository interface {
	MarkStalePaymentCreditsUnknown(context.Context, time.Time) (int, error)
	ClaimPaymentCredits(context.Context, time.Time, int) ([]PaymentCreditOperation, error)
	CompletePaymentCredit(context.Context, int64, time.Time) error
	FailPaymentCredit(context.Context, int64, string, string, time.Time) error
}

type PaymentCreditRunner struct {
	repository PaymentCreditRepository
	admin      AdminNewAPI
}

func NewPaymentCreditRunner(repository PaymentCreditRepository, admin AdminNewAPI) *PaymentCreditRunner {
	return &PaymentCreditRunner{repository: repository, admin: admin}
}

func (r *PaymentCreditRunner) RunCycle(ctx context.Context, now time.Time) error {
	if _, err := r.repository.MarkStalePaymentCreditsUnknown(ctx, now.Add(-5*time.Minute)); err != nil {
		return fmt.Errorf("标记结果未知的充值入账：%w", err)
	}
	operations, err := r.repository.ClaimPaymentCredits(ctx, now, 20)
	if err != nil {
		return fmt.Errorf("领取充值入账任务：%w", err)
	}
	for _, operation := range operations {
		_, creditErr := r.admin.Do(ctx, http.MethodPost, "/api/user/manage", map[string]any{
			"id": operation.UserID, "action": "add_quota", "mode": "add", "value": operation.NewAPIQuota,
		})
		if creditErr == nil {
			if err := r.repository.CompletePaymentCredit(ctx, operation.ID, now); err != nil {
				return fmt.Errorf("确认充值入账完成：%w", err)
			}
			continue
		}
		status := "unknown"
		var upstreamError *newapi.APIError
		if errors.As(creditErr, &upstreamError) {
			status = "failed"
		}
		nextAttempt := now.Add(paymentCreditBackoff(operation.Attempts))
		if err := r.repository.FailPaymentCredit(ctx, operation.ID, status, creditErr.Error(), nextAttempt); err != nil {
			return fmt.Errorf("记录充值入账失败：%w", err)
		}
	}
	return nil
}

func paymentCreditBackoff(attempts int) time.Duration {
	switch attempts {
	case 0, 1:
		return time.Minute
	case 2:
		return 5 * time.Minute
	default:
		return 30 * time.Minute
	}
}
