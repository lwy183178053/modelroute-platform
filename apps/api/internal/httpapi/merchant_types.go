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
	ChannelType           int        `json:"channel_type"`
	BaseURL               string     `json:"base_url"`
	RequestedModels       []string   `json:"requested_models"`
	VerifiedModels        []string   `json:"verified_models,omitempty"`
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
	ID                    int64     `json:"id"`
	MerchantID            int64     `json:"merchant_id"`
	ReviewID              int64     `json:"review_id"`
	NewAPIChannelID       int       `json:"newapi_channel_id"`
	ChannelName           string    `json:"channel_name"`
	ChannelType           int       `json:"channel_type"`
	BaseURL               string    `json:"base_url"`
	Models                []string  `json:"models"`
	CredentialFingerprint string    `json:"credential_fingerprint"`
	Status                string    `json:"status"`
	CreatedAt             time.Time `json:"created_at"`
}

type ChannelSubmission struct {
	ChannelName           string
	ChannelType           int
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
	FinishVerification(context.Context, int64, int, []string, string) error
	FinishApproval(context.Context, int64, int, int) (MerchantApproval, error)
	CancelApproval(context.Context, int64, string) error
	RejectReview(context.Context, int64, int, string) error
	Merchant(context.Context, int64) (MerchantProfile, error)
	SetMerchantStatus(context.Context, int64, bool, int, string) (MerchantStatusChange, error)
}
