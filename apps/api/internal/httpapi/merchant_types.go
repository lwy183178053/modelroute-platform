package httpapi

import (
	"context"
	"errors"
	"time"
)

var (
	ErrMerchantNotFound     = errors.New("merchant not found")
	ErrMerchantConflict     = errors.New("merchant state conflict")
	ErrMerchantInvalidState = errors.New("merchant invalid state")
)

type CredentialProtector interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
	Fingerprint([]byte) string
}

type MerchantIdentity struct {
	MerchantID      int64  `json:"merchant_id"`
	UserID          int    `json:"user_id"`
	DisplayName     string `json:"merchant_name"`
	Status          string `json:"merchant_status"`
	IsMerchant      bool   `json:"is_merchant"`
	IsPlatformOwned bool   `json:"is_platform_merchant"`
}

type MerchantChannelReview struct {
	ID                    int64      `json:"id"`
	UserID                int        `json:"user_id"`
	ApplicationID         int64      `json:"application_id,omitempty"`
	ReplacesChannelID     int64      `json:"replaces_channel_id,omitempty"`
	ChannelName           string     `json:"channel_name"`
	ChannelType           int        `json:"-"`
	CanonicalModelID      int64      `json:"canonical_model_id"`
	Model                 string     `json:"model"`
	Provider              string     `json:"provider"`
	Protocol              string     `json:"protocol"`
	AdapterType           string     `json:"adapter_type"`
	ChannelTestID         int64      `json:"channel_test_id"`
	BaseURL               string     `json:"base_url"`
	RequestedModels       []string   `json:"-"`
	VerifiedModels        []string   `json:"-"`
	Status                string     `json:"status"`
	CredentialConfigured  bool       `json:"credential_configured"`
	EncryptedCredential   []byte     `json:"-"`
	CredentialFingerprint string     `json:"-"`
	VerificationError     string     `json:"verification_error,omitempty"`
	ReviewReason          string     `json:"review_reason,omitempty"`
	VerifiedAt            *time.Time `json:"verified_at,omitempty"`
	ReviewedAt            *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}

type MerchantApplication struct {
	ID                  int64                 `json:"id"`
	UserID              int                   `json:"user_id"`
	DisplayName         string                `json:"display_name"`
	WebsiteURL          string                `json:"website_url"`
	BusinessDescription string                `json:"business_description"`
	Status              string                `json:"status"`
	ReviewReason        string                `json:"review_reason,omitempty"`
	InitialReview       MerchantChannelReview `json:"initial_channel"`
	ReviewedAt          *time.Time            `json:"reviewed_at,omitempty"`
	CreatedAt           time.Time             `json:"created_at"`
}

type MerchantProfile struct {
	ID               int64     `json:"id"`
	UserID           int       `json:"user_id"`
	DisplayName      string    `json:"display_name"`
	WebsiteURL       string    `json:"website_url"`
	Status           string    `json:"status"`
	StatusReason     string    `json:"status_reason,omitempty"`
	ApprovedAt       time.Time `json:"approved_at"`
	NewAPIChannelIDs []int     `json:"-"`
}

type MerchantChannel struct {
	ID                    int64      `json:"id"`
	MerchantID            int64      `json:"merchant_id"`
	ReviewID              int64      `json:"review_id"`
	NewAPIChannelID       int        `json:"newapi_channel_id"`
	ChannelName           string     `json:"channel_name"`
	ChannelType           int        `json:"-"`
	CanonicalModelID      int64      `json:"canonical_model_id"`
	Model                 string     `json:"model"`
	Provider              string     `json:"provider"`
	Protocol              string     `json:"protocol"`
	AdapterType           string     `json:"adapter_type"`
	BaseURL               string     `json:"base_url"`
	Models                []string   `json:"-"`
	CredentialFingerprint string     `json:"credential_fingerprint"`
	Status                string     `json:"status"`
	HealthState           string     `json:"health_state"`
	AvailabilityPPM       int        `json:"availability_ppm"`
	TTFTP95MS             int        `json:"ttft_p95_ms"`
	ConsecutiveFailures   int        `json:"consecutive_failures"`
	CooldownUntil         *time.Time `json:"cooldown_until,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}

type ChannelTest struct {
	ID               int64     `json:"id"`
	CanonicalModelID int64     `json:"canonical_model_id"`
	AdapterType      string    `json:"adapter_type"`
	Capabilities     []string  `json:"capabilities"`
	LatencyMS        int       `json:"latency_ms"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type CreateChannelTest struct {
	UserID                int
	CanonicalModelID      int64
	BaseURL               string
	CredentialFingerprint string
	AdapterType           string
	Capabilities          []string
	LatencyMS             int
	ReceiptTokenHash      string
	ExpiresAt             time.Time
}

type ChannelTestRequest struct {
	Model   CanonicalModel
	BaseURL string
	APIKey  string
}

type ChannelTestOutcome struct {
	AdapterType  string   `json:"adapter_type"`
	Capabilities []string `json:"capabilities"`
	LatencyMS    int      `json:"latency_ms"`
}

type ChannelTester interface {
	Test(context.Context, ChannelTestRequest) (ChannelTestOutcome, error)
}

type ChannelSubmission struct {
	ChannelName           string
	ChannelType           int
	CanonicalModelID      int64
	TestReceipt           string
	BaseURL               string
	Models                []string
	EncryptedCredential   []byte
	CredentialFingerprint string
}

type CreateMerchantApplication struct {
	UserID              int
	DisplayName         string
	WebsiteURL          string
	BusinessDescription string
	Channel             ChannelSubmission
}

type CreateMerchantChannel struct {
	UserID  int
	Channel ChannelSubmission
}

type MerchantApproval struct {
	MerchantID      int64 `json:"merchant_id"`
	UserID          int   `json:"user_id"`
	ChannelID       int64 `json:"channel_id"`
	NewAPIChannelID int   `json:"newapi_channel_id"`
}

type MerchantStatusChange struct {
	MerchantID int64  `json:"merchant_id"`
	UserID     int    `json:"user_id"`
	Status     string `json:"status"`
}

type MerchantRepository interface {
	Identity(context.Context, int) (MerchantIdentity, error)
	EnsurePlatformMerchant(context.Context, int, string) (MerchantIdentity, error)
	SaveChannelTest(context.Context, CreateChannelTest) (ChannelTest, error)
	CreateApplication(context.Context, CreateMerchantApplication) (MerchantApplication, error)
	ApplicationForUser(context.Context, int) (MerchantApplication, error)
	CreateChannelReview(context.Context, CreateMerchantChannel) (MerchantChannelReview, error)
	ChannelsForUser(context.Context, int) ([]MerchantChannel, error)
	ChannelReviewsForUser(context.Context, int) ([]MerchantChannelReview, error)
	ListApplications(context.Context, int, int, string, string) ([]MerchantApplication, int, error)
	Application(context.Context, int64) (MerchantApplication, error)
	ListChannelReviews(context.Context, int, int, string) ([]MerchantChannelReview, int, error)
	ListMerchants(context.Context, int, int, string) ([]MerchantProfile, int, error)
	ListChannels(context.Context, int, int, string) ([]MerchantChannel, int, error)
	ClaimReview(context.Context, int64, string) (MerchantChannelReview, error)
	FinishVerification(context.Context, int64, int, string, string) error
	FinishApproval(context.Context, int64, int, int) (MerchantApproval, error)
	CancelApproval(context.Context, int64, string) error
	RejectReview(context.Context, int64, int, string) error
	Merchant(context.Context, int64) (MerchantProfile, error)
	SetMerchantStatus(context.Context, int64, bool, int, string) (MerchantStatusChange, error)
}
