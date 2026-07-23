package httpapi

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSettlementNotFound          = errors.New("settlement object not found")
	ErrSettlementConflict          = errors.New("settlement state conflict")
	ErrInsufficientMerchantBalance = errors.New("insufficient merchant balance")
)

type MerchantBalance struct {
	MerchantID            int64     `json:"merchant_id"`
	AvailableQuota        int64     `json:"available_quota"`
	FrozenQuota           int64     `json:"frozen_quota"`
	LifetimeEarnedQuota   int64     `json:"lifetime_earned_quota"`
	LifetimeRefundedQuota int64     `json:"lifetime_refunded_quota"`
	LifetimePaidQuota     int64     `json:"lifetime_paid_quota"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type MerchantLedgerEntry struct {
	ID             int64     `json:"id"`
	MerchantID     int64     `json:"merchant_id"`
	EntryType      string    `json:"entry_type"`
	AvailableDelta int64     `json:"available_delta"`
	FrozenDelta    int64     `json:"frozen_delta"`
	AvailableAfter int64     `json:"available_after"`
	FrozenAfter    int64     `json:"frozen_after"`
	Description    string    `json:"description"`
	NewAPILogID    int64     `json:"newapi_log_id,omitempty"`
	SourceType     int       `json:"source_type,omitempty"`
	ModelName      string    `json:"model_name,omitempty"`
	GrossQuota     int64     `json:"gross_quota,omitempty"`
	WithdrawalNo   string    `json:"withdrawal_no,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type MerchantWithdrawal struct {
	ID                        int64      `json:"id"`
	WithdrawalNo              string     `json:"withdrawal_no"`
	MerchantID                int64      `json:"merchant_id"`
	MerchantName              string     `json:"merchant_name,omitempty"`
	AmountQuota               int64      `json:"amount_quota"`
	TaxQuota                  int64      `json:"tax_quota"`
	NetPayoutQuota            int64      `json:"net_payout_quota"`
	AvailableBeforeQuota      int64      `json:"available_before_quota"`
	RecipientType             string     `json:"recipient_type"`
	EncryptedRecipientAccount []byte     `json:"-"`
	RecipientAccountMasked    string     `json:"recipient_account_masked"`
	RecipientAccount          string     `json:"recipient_account,omitempty"`
	Status                    string     `json:"status"`
	ReviewReason              string     `json:"review_reason,omitempty"`
	ReviewedByUserID          int        `json:"reviewed_by_user_id,omitempty"`
	ReviewedAt                *time.Time `json:"reviewed_at,omitempty"`
	PayoutReference           string     `json:"payout_reference,omitempty"`
	PayoutProof               string     `json:"payout_proof,omitempty"`
	PaidByUserID              int        `json:"paid_by_user_id,omitempty"`
	PaidAt                    *time.Time `json:"paid_at,omitempty"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
}

type CreateMerchantWithdrawalCommand struct {
	WithdrawalNo              string
	UserID                    int
	AmountQuota               int64
	QuotaPerUnit              int64
	RecipientType             string
	EncryptedRecipientAccount []byte
	RecipientAccountMasked    string
	CreatedAt                 time.Time
}

type SettlementRepository interface {
	SettlementLogRepository
	MerchantBalance(context.Context, int) (MerchantBalance, error)
	MerchantWithdrawals(context.Context, int, int, int) ([]MerchantWithdrawal, int, error)
	CreateMerchantWithdrawal(context.Context, CreateMerchantWithdrawalCommand) (MerchantWithdrawal, error)
	ListMerchantWithdrawals(context.Context, int, int, string) ([]MerchantWithdrawal, int, error)
	ApproveMerchantWithdrawal(context.Context, int64, int, string, time.Time) (MerchantWithdrawal, error)
	RejectMerchantWithdrawal(context.Context, int64, int, string, time.Time) (MerchantWithdrawal, error)
	PayMerchantWithdrawal(context.Context, int64, int, string, string, time.Time) (MerchantWithdrawal, error)
}
