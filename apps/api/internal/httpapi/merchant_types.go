package httpapi

import (
	"context"
	"errors"
	"time"
)

var (
	ErrMerchantNotFound     = errors.New("merchant not found")
	ErrMerchantConflict     = errors.New("merchant state conflict")
	ErrMerchantNameConflict = errors.New("merchant display name conflict")
	ErrMerchantInvalidState = errors.New("merchant invalid state")
)

type CredentialProtector interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
	Fingerprint([]byte) string
}

type MerchantIdentity struct {
	MerchantID      int64  `json:"-"`
	UserID          int    `json:"user_id"`
	DisplayName     string `json:"merchant_name"`
	Status          string `json:"merchant_status"`
	IsMerchant      bool   `json:"is_merchant"`
	IsPlatformOwned bool   `json:"is_platform_merchant"`
}

type MerchantChannelReview struct {
	ID                    int64                `json:"id"`
	UserID                int                  `json:"user_id"`
	ApplicationID         int64                `json:"application_id,omitempty"`
	ReplacesChannelID     int64                `json:"replaces_channel_id,omitempty"`
	ChannelName           string               `json:"channel_name"`
	ChannelType           int                  `json:"-"`
	CanonicalModelID      int64                `json:"canonical_model_id"`
	Model                 string               `json:"model"`
	Provider              string               `json:"provider"`
	Protocol              string               `json:"protocol"`
	AdapterType           string               `json:"adapter_type"`
	ChannelTestID         int64                `json:"channel_test_id"`
	BaseURL               string               `json:"base_url"`
	RequestedModels       []string             `json:"-"`
	VerifiedModels        []string             `json:"-"`
	Status                string               `json:"status"`
	CredentialConfigured  bool                 `json:"credential_configured"`
	EncryptedCredential   []byte               `json:"-"`
	CredentialFingerprint string               `json:"-"`
	VerificationError     string               `json:"verification_error,omitempty"`
	ReviewReason          string               `json:"review_reason,omitempty"`
	VerifiedAt            *time.Time           `json:"verified_at,omitempty"`
	ReviewedAt            *time.Time           `json:"reviewed_at,omitempty"`
	CreatedAt             time.Time            `json:"created_at"`
	Models                []ReviewModelBinding `json:"models,omitempty"`
}

// ReviewModelBinding is the model-level state captured during channel review.
type ReviewModelBinding struct {
	ID                   int64      `json:"id"`
	ReviewID             int64      `json:"review_id,omitempty"`
	UpstreamModelName    string     `json:"upstream_model_name"`
	CanonicalModelID     int64      `json:"canonical_model_id,omitempty"`
	MatchStatus          string     `json:"match_status"`
	TestStatus           string     `json:"test_status"`
	TestedByNewAPIUserID int        `json:"tested_by_user_id,omitempty"`
	TestedAt             *time.Time `json:"tested_at,omitempty"`
	TestLatencyMS        int        `json:"test_latency_ms,omitempty"`
	TestError            string     `json:"test_error,omitempty"`
}

type ChannelModelBinding struct {
	ID                int64     `json:"id"`
	ChannelID         int64     `json:"channel_id,omitempty"`
	UpstreamModelName string    `json:"upstream_model_name"`
	CanonicalModelID  int64     `json:"canonical_model_id,omitempty"`
	MatchStatus       string    `json:"match_status"`
	LastSyncedAt      time.Time `json:"last_synced_at"`
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
	MerchantID            int64      `json:"-"`
	UserID                int        `json:"user_id"`
	MerchantName          string     `json:"merchant_name"`
	ReviewID              int64      `json:"review_id,omitempty"`
	NewAPIChannelID       int        `json:"-"`
	ChannelName           string     `json:"channel_name"`
	ChannelType           int        `json:"-"`
	CanonicalModelID      int64      `json:"canonical_model_id"`
	Model                 string     `json:"model"`
	Provider              string     `json:"provider"`
	Protocol              string     `json:"protocol"`
	AdapterType           string     `json:"adapter_type"`
	BaseURL               string     `json:"base_url"`
	Models                []string   `json:"matched_models,omitempty"`
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
	AdapterType           string
	CanonicalModelID      int64
	TestReceipt           string
	BaseURL               string
	Models                []string
	EncryptedCredential   []byte
	CredentialFingerprint string
	UpstreamModels        []SyncedChannelModel
}

type CreateMerchantApplication struct {
	UserID              int
	DisplayName         string
	DisplayNameKey      string
	WebsiteURL          string
	BusinessDescription string
	Channel             ChannelSubmission
}

type CreateMerchantChannel struct {
	UserID  int
	Channel ChannelSubmission
}

type MerchantApproval struct {
	MerchantID      int64 `json:"-"`
	UserID          int   `json:"user_id"`
	ChannelID       int64 `json:"channel_id"`
	NewAPIChannelID int   `json:"-"`
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
	ListChannels(context.Context, int, int, string) ([]MerchantChannel, int, error)
	ClaimReview(context.Context, int64, string) (MerchantChannelReview, error)
	FinishVerification(context.Context, int64, int, string, string) error
	FinishApproval(context.Context, int64, int, int) (MerchantApproval, error)
	CancelApproval(context.Context, int64, string) error
	RejectReview(context.Context, int64, int, string) error
	Merchant(context.Context, int64) (MerchantProfile, error)
	SetMerchantStatus(context.Context, int64, bool, int, string) (MerchantStatusChange, error)
}

// MultiModelMerchantRepository is optional so existing integrations and test
// doubles can keep the single-model repository contract while V2 flows opt in.
type MultiModelMerchantRepository interface {
	CreateMultiModelChannelReview(context.Context, CreateMerchantChannel) (MerchantChannelReview, error)
	SyncReviewModels(context.Context, int64, []SyncedChannelModel) error
	SyncChannelModels(context.Context, int64, []SyncedChannelModel) error
	ReviewModels(context.Context, int64) ([]ReviewModelBinding, error)
	ChannelModels(context.Context, int64) ([]ChannelModelBinding, error)
	SetReviewModelTest(context.Context, int64, int64, int, string, int, string) error
	PromoteReviewModels(context.Context, int64, int64) error
}

type MerchantChannelManagement interface {
	SetChannelStatus(context.Context, int64, int64, bool) error
	DeleteChannel(context.Context, int64, int64) error
}

type MerchantChannelReviewManagement interface {
	DeleteChannelReview(context.Context, int64, int64) (string, error)
}

type FilteredMerchantChannelRepository interface {
	ListChannelsFiltered(context.Context, int, int, string, string, int) ([]MerchantChannel, int, error)
}

type MerchantChannelLookup interface {
	Channel(context.Context, int64) (MerchantChannel, error)
}
