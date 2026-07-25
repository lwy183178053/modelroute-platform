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

type ChannelModelBinding struct {
	ID                         int64      `json:"id"`
	ChannelID                  int64      `json:"channel_id,omitempty"`
	UpstreamModelName          string     `json:"upstream_model_name"`
	CanonicalModelID           int64      `json:"canonical_model_id,omitempty"`
	Model                      string     `json:"model"`
	MatchStatus                string     `json:"match_status"`
	HealthState                string     `json:"health_state"`
	ConsecutiveRequestFailures int        `json:"consecutive_request_failures"`
	AutomaticCheckFailures     int        `json:"automatic_check_failures"`
	NextCheckAt                *time.Time `json:"next_check_at,omitempty"`
	LastCheckedAt              *time.Time `json:"last_checked_at,omitempty"`
	LastCheckLatencyMS         int        `json:"last_check_latency_ms,omitempty"`
	LastStatusCode             int        `json:"last_status_code,omitempty"`
	LastCheckError             string     `json:"last_check_error,omitempty"`
	LastSyncedAt               time.Time  `json:"last_synced_at"`
}

type MerchantApplication struct {
	ID                  int64      `json:"id"`
	UserID              int        `json:"user_id"`
	DisplayName         string     `json:"display_name"`
	WebsiteURL          string     `json:"website_url"`
	BusinessDescription string     `json:"business_description"`
	Status              string     `json:"status"`
	ReviewReason        string     `json:"review_reason,omitempty"`
	ReviewedAt          *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type MerchantProfile struct {
	ID                  int64     `json:"id"`
	UserID              int       `json:"user_id"`
	DisplayName         string    `json:"display_name"`
	WebsiteURL          string    `json:"website_url"`
	BusinessDescription string    `json:"business_description"`
	Status              string    `json:"status"`
	StatusReason        string    `json:"status_reason,omitempty"`
	ApprovedAt          time.Time `json:"approved_at"`
	IsPlatformOwned     bool      `json:"is_platform_merchant"`
	NewAPIChannelIDs    []int     `json:"-"`
}

type MerchantChannel struct {
	ID                    int64                 `json:"id"`
	MerchantID            int64                 `json:"-"`
	UserID                int                   `json:"user_id"`
	MerchantName          string                `json:"merchant_name"`
	NewAPIChannelID       int                   `json:"-"`
	ChannelName           string                `json:"channel_name"`
	ChannelType           int                   `json:"-"`
	Provider              string                `json:"provider"`
	AdapterType           string                `json:"-"`
	BaseURL               string                `json:"base_url"`
	ModelBindings         []ChannelModelBinding `json:"models,omitempty"`
	CredentialFingerprint string                `json:"credential_fingerprint"`
	LastSyncedAt          *time.Time            `json:"last_synced_at,omitempty"`
	CreatedAt             time.Time             `json:"created_at"`
}

type StoreChannelModel struct {
	UpstreamModelName string
	CanonicalModelID  int64
}

type StoreMerchantChannel struct {
	UserID                int
	NewAPIChannelID       int
	ChannelName           string
	Provider              string
	ChannelType           int
	AdapterType           string
	BaseURL               string
	CredentialFingerprint string
	Models                []StoreChannelModel
}

type SelfServiceChannelRepository interface {
	CreateSelfServiceChannel(context.Context, StoreMerchantChannel) (MerchantChannel, error)
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

type ExistingChannelTester interface {
	TestExisting(context.Context, int, CanonicalModel) (ChannelTestOutcome, error)
}

type ChannelTester interface {
	Test(context.Context, ChannelTestRequest) (ChannelTestOutcome, error)
}

type CreateMerchantApplication struct {
	UserID              int
	DisplayName         string
	DisplayNameKey      string
	WebsiteURL          string
	BusinessDescription string
}

type MerchantStatusChange struct {
	MerchantID int64  `json:"merchant_id"`
	UserID     int    `json:"user_id"`
	Status     string `json:"status"`
}

type MerchantRepository interface {
	Identity(context.Context, int) (MerchantIdentity, error)
	EnsurePlatformMerchant(context.Context, int, string) (MerchantIdentity, error)
	CreateApplication(context.Context, CreateMerchantApplication) (MerchantApplication, error)
	ApplicationForUser(context.Context, int) (MerchantApplication, error)
	ApproveApplication(context.Context, int64, int) (MerchantProfile, error)
	RejectApplication(context.Context, int64, int, string) error
	ChannelsForUser(context.Context, int) ([]MerchantChannel, error)
	ListApplications(context.Context, int, int, string, string) ([]MerchantApplication, int, error)
	Application(context.Context, int64) (MerchantApplication, error)
	ListChannels(context.Context, int, int) ([]MerchantChannel, int, error)
	Merchant(context.Context, int64) (MerchantProfile, error)
	UpdateMerchantDescription(context.Context, int64, string) (MerchantProfile, error)
	SetMerchantStatus(context.Context, int64, bool, int, string) (MerchantStatusChange, error)
}

type MultiModelMerchantRepository interface {
	SyncChannelModels(context.Context, int64, []SyncedChannelModel) error
	ChannelModels(context.Context, int64) ([]ChannelModelBinding, error)
	SaveManualModelCheck(context.Context, int64, int64, int64, ModelHealthState, healthFailureClass, int, int, string) (ChannelModelBinding, error)
}

type MerchantChannelManagement interface {
	DeleteChannel(context.Context, int64, int64) error
}

type FilteredMerchantChannelRepository interface {
	ListChannelsFiltered(context.Context, int, int, string, int) ([]MerchantChannel, int, error)
}

type MerchantChannelLookup interface {
	Channel(context.Context, int64) (MerchantChannel, error)
}
