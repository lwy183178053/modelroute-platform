package httpapi

import (
	"context"
	"errors"
	"time"
)

var (
	ErrPaymentNotConfigured = errors.New("payment not configured")
	ErrPaymentNotFound      = errors.New("payment not found")
	ErrPaymentConflict      = errors.New("payment state conflict")
)

type PaymentSetting struct {
	ID                         int64     `json:"id"`
	Version                    int       `json:"version"`
	GatewayURL                 string    `json:"gateway_url"`
	MerchantPID                string    `json:"-"`
	EncryptedKey               []byte    `json:"-"`
	KeyFingerprint             string    `json:"-"`
	PaymentMethods             []string  `json:"payment_methods"`
	MinimumUSDCents            int64     `json:"minimum_usd_cents"`
	MaximumUSDCents            int64     `json:"maximum_usd_cents"`
	ExchangeRateCNYCentsPerUSD int64     `json:"exchange_rate_cny_cents_per_usd"`
	Enabled                    bool      `json:"enabled"`
	CreatedAt                  time.Time `json:"created_at"`
}

type CreatePaymentSettingCommand struct {
	AdminUserID                int
	GatewayURL                 string
	MerchantPID                string
	EncryptedKey               []byte
	KeyFingerprint             string
	PaymentMethods             []string
	MinimumUSDCents            int64
	MaximumUSDCents            int64
	ExchangeRateCNYCentsPerUSD int64
	Enabled                    bool
}

type PaymentOrder struct {
	ID                         int64      `json:"id"`
	OrderNo                    string     `json:"order_no"`
	UserID                     int        `json:"user_id"`
	SettingVersionID           int64      `json:"setting_version_id"`
	PaymentType                string     `json:"payment_type"`
	USDCents                   int64      `json:"usd_cents"`
	CNYCents                   int64      `json:"cny_cents"`
	ExchangeRateCNYCentsPerUSD int64      `json:"exchange_rate_cny_cents_per_usd"`
	NewAPIQuota                int64      `json:"newapi_quota"`
	PaymentStatus              string     `json:"payment_status"`
	CreditStatus               string     `json:"credit_status"`
	ThirdPartyTradeNo          string     `json:"third_party_trade_no,omitempty"`
	PaidAt                     *time.Time `json:"paid_at,omitempty"`
	CreditedAt                 *time.Time `json:"credited_at,omitempty"`
	ExpiresAt                  time.Time  `json:"expires_at"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}

type CreatePaymentOrderCommand struct {
	OrderNo                    string
	UserID                     int
	SettingVersionID           int64
	PaymentType                string
	USDCents                   int64
	CNYCents                   int64
	ExchangeRateCNYCentsPerUSD int64
	NewAPIQuota                int64
	ExpiresAt                  time.Time
}

type PaymentCallbackContext struct {
	Order   PaymentOrder
	Setting PaymentSetting
}

type AcceptPaymentCallbackCommand struct {
	OrderNo     string
	TradeNo     string
	CallbackKey string
	Payload     map[string]string
	PaidAt      time.Time
}

type PaymentRepository interface {
	CurrentPaymentSetting(context.Context) (PaymentSetting, error)
	CreatePaymentSetting(context.Context, CreatePaymentSettingCommand) (PaymentSetting, error)
	CreatePaymentOrder(context.Context, CreatePaymentOrderCommand) (PaymentOrder, error)
	PaymentCallbackContext(context.Context, string) (PaymentCallbackContext, error)
	AcceptPaymentCallback(context.Context, AcceptPaymentCallbackCommand) (PaymentOrder, bool, error)
	PaymentOrdersForUser(context.Context, int, int, int) ([]PaymentOrder, int, error)
	PaymentOrderForUser(context.Context, int, string) (PaymentOrder, error)
	ListPaymentOrders(context.Context, int, int, string) ([]PaymentOrder, int, error)
	RetryPaymentCredit(context.Context, int64, bool) error
	ConfirmPaymentCredit(context.Context, int64, time.Time) error
}
