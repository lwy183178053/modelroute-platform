package httpapi

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type SQLMerchantStore struct {
	database *sql.DB
}

func NewSQLMerchantStore(database *sql.DB) *SQLMerchantStore {
	return &SQLMerchantStore{database: database}
}

func (s *SQLMerchantStore) SaveChannelTest(ctx context.Context, command CreateChannelTest) (ChannelTest, error) {
	capabilities, err := json.Marshal(command.Capabilities)
	if err != nil {
		return ChannelTest{}, fmt.Errorf("编码渠道检测能力：%w", err)
	}
	result := ChannelTest{
		CanonicalModelID: command.CanonicalModelID,
		AdapterType:      command.AdapterType,
		Capabilities:     append([]string(nil), command.Capabilities...),
		LatencyMS:        command.LatencyMS,
		ExpiresAt:        command.ExpiresAt,
	}
	err = s.database.QueryRowContext(ctx, `
INSERT INTO merchant_channel_tests
    (newapi_user_id, canonical_model_id, base_url, credential_fingerprint,
     adapter_type, capabilities, latency_ms, receipt_token_hash, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, expires_at
`, command.UserID, command.CanonicalModelID, command.BaseURL, command.CredentialFingerprint,
		command.AdapterType, capabilities, command.LatencyMS, command.ReceiptTokenHash, command.ExpiresAt,
	).Scan(&result.ID, &result.ExpiresAt)
	if err != nil {
		return ChannelTest{}, fmt.Errorf("保存渠道检测结果：%w", err)
	}
	return result, nil
}

func (s *SQLMerchantStore) Identity(ctx context.Context, userID int) (MerchantIdentity, error) {
	var identity MerchantIdentity
	err := s.database.QueryRowContext(ctx, `
SELECT id, newapi_user_id, display_name, status, is_platform_owned
FROM merchant_profiles
WHERE newapi_user_id = $1
`, userID).Scan(&identity.MerchantID, &identity.UserID, &identity.DisplayName, &identity.Status, &identity.IsPlatformOwned)
	if err != nil {
		return MerchantIdentity{}, merchantStoreError(err)
	}
	identity.IsMerchant = true
	return identity, nil
}

func (s *SQLMerchantStore) EnsurePlatformMerchant(ctx context.Context, userID int, publicOrigin string) (MerchantIdentity, error) {
	var identity MerchantIdentity
	var createdAt time.Time
	err := s.database.QueryRowContext(ctx, `
INSERT INTO merchant_profiles
    (newapi_user_id, approved_application_id, display_name, website_url, status,
     approved_by_newapi_user_id, is_platform_owned)
VALUES ($1, NULL, $2, $3, 'active', $1, true)
ON CONFLICT (newapi_user_id) DO UPDATE
SET approved_application_id = NULL,
    display_name = EXCLUDED.display_name,
    website_url = EXCLUDED.website_url,
    status = 'active',
    status_reason = NULL,
    approved_by_newapi_user_id = EXCLUDED.approved_by_newapi_user_id,
    approved_at = now(),
    is_platform_owned = true,
    updated_at = now()
RETURNING id, newapi_user_id, display_name, status, is_platform_owned, created_at
`, userID, "ModelRoute 官方", publicOrigin).Scan(
		&identity.MerchantID,
		&identity.UserID,
		&identity.DisplayName,
		&identity.Status,
		&identity.IsPlatformOwned,
		&createdAt,
	)
	if err != nil {
		return MerchantIdentity{}, merchantStoreError(err)
	}
	identity.IsMerchant = true
	return identity, nil
}

func (s *SQLMerchantStore) CreateApplication(ctx context.Context, command CreateMerchantApplication) (MerchantApplication, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantApplication{}, fmt.Errorf("开始商户申请事务：%w", err)
	}
	defer tx.Rollback()
	test, err := claimChannelTest(ctx, tx, command.UserID, command.Channel)
	if err != nil {
		return MerchantApplication{}, err
	}

	application := MerchantApplication{
		UserID: command.UserID, DisplayName: command.DisplayName, WebsiteURL: command.WebsiteURL,
		BusinessDescription: command.BusinessDescription, Status: "pending",
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_applications
    (newapi_user_id, display_name, website_url, business_description)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at
`, command.UserID, command.DisplayName, command.WebsiteURL, command.BusinessDescription).Scan(&application.ID, &application.CreatedAt)
	if err != nil {
		return MerchantApplication{}, merchantStoreError(err)
	}

	review := MerchantChannelReview{
		UserID: command.UserID, ApplicationID: application.ID, ChannelName: command.Channel.ChannelName,
		CanonicalModelID: command.Channel.CanonicalModelID, Model: test.Model, Provider: test.Provider,
		Protocol: test.Protocol, AdapterType: test.AdapterType, ChannelTestID: test.ID,
		BaseURL: command.Channel.BaseURL, Status: "pending", CredentialConfigured: true,
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_channel_reviews
    (merchant_newapi_user_id, application_id, canonical_model_id, channel_test_id,
     channel_name, base_url, adapter_type, encrypted_credential, credential_fingerprint)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, created_at
`, command.UserID, application.ID, command.Channel.CanonicalModelID, test.ID,
		command.Channel.ChannelName, command.Channel.BaseURL, test.AdapterType,
		command.Channel.EncryptedCredential, command.Channel.CredentialFingerprint,
	).Scan(&review.ID, &review.CreatedAt)
	if err != nil {
		return MerchantApplication{}, merchantStoreError(err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE merchant_applications SET initial_channel_review_id = $1 WHERE id = $2", review.ID, application.ID); err != nil {
		return MerchantApplication{}, fmt.Errorf("关联初始渠道审核：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return MerchantApplication{}, fmt.Errorf("提交商户申请事务：%w", err)
	}
	application.InitialReview = review
	return application, nil
}

func (s *SQLMerchantStore) ApplicationForUser(ctx context.Context, userID int) (MerchantApplication, error) {
	return scanMerchantApplication(s.database.QueryRowContext(ctx, applicationSelect+`
WHERE a.newapi_user_id = $1
ORDER BY a.id DESC
LIMIT 1
`, userID))
}

func (s *SQLMerchantStore) Application(ctx context.Context, id int64) (MerchantApplication, error) {
	return scanMerchantApplication(s.database.QueryRowContext(ctx, applicationSelect+`
WHERE a.id = $1
`, id))
}

func (s *SQLMerchantStore) CreateChannelReview(ctx context.Context, command CreateMerchantChannel) (MerchantChannelReview, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantChannelReview{}, fmt.Errorf("开始渠道审核事务：%w", err)
	}
	defer tx.Rollback()
	test, err := claimChannelTest(ctx, tx, command.UserID, command.Channel)
	if err != nil {
		return MerchantChannelReview{}, err
	}
	review := MerchantChannelReview{
		UserID: command.UserID, ChannelName: command.Channel.ChannelName,
		CanonicalModelID: command.Channel.CanonicalModelID, Model: test.Model, Provider: test.Provider,
		Protocol: test.Protocol, AdapterType: test.AdapterType, ChannelTestID: test.ID,
		BaseURL: command.Channel.BaseURL, Status: "pending", CredentialConfigured: true,
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_channel_reviews
    (merchant_profile_id, merchant_newapi_user_id, canonical_model_id, channel_test_id,
     channel_name, base_url, adapter_type, encrypted_credential, credential_fingerprint)
SELECT id, $1, $2, $3, $4, $5, $6, $7, $8
FROM merchant_profiles
WHERE newapi_user_id = $1 AND status = 'active'
RETURNING id, created_at
`, command.UserID, command.Channel.CanonicalModelID, test.ID, command.Channel.ChannelName,
		command.Channel.BaseURL, test.AdapterType, command.Channel.EncryptedCredential,
		command.Channel.CredentialFingerprint,
	).Scan(&review.ID, &review.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MerchantChannelReview{}, ErrMerchantInvalidState
		}
		return MerchantChannelReview{}, merchantStoreError(err)
	}
	if err := tx.Commit(); err != nil {
		return MerchantChannelReview{}, fmt.Errorf("提交渠道审核事务：%w", err)
	}
	return review, nil
}

// CreateMultiModelChannelReview stores one encrypted credential and one review
// binding per upstream model. The legacy canonical_model_id keeps the first
// matched model so older list and routing code remains compatible.
func (s *SQLMerchantStore) CreateMultiModelChannelReview(ctx context.Context, command CreateMerchantChannel) (MerchantChannelReview, error) {
	if len(command.Channel.UpstreamModels) == 0 {
		return MerchantChannelReview{}, ErrMerchantInvalidState
	}
	var primary SyncedChannelModel
	for _, model := range command.Channel.UpstreamModels {
		if model.MatchStatus == "matched" && model.CanonicalModelID > 0 {
			primary = model
			break
		}
	}
	if primary.CanonicalModelID <= 0 {
		return MerchantChannelReview{}, ErrMerchantInvalidState
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantChannelReview{}, fmt.Errorf("开始多模型渠道审核事务：%w", err)
	}
	defer tx.Rollback()
	adapter := command.Channel.AdapterType
	if adapter == "" {
		adapter = adapterForChannelType(command.Channel.ChannelType)
	}
	review := MerchantChannelReview{
		UserID: command.UserID, ChannelName: command.Channel.ChannelName,
		CanonicalModelID: primary.CanonicalModelID, Model: primary.Model,
		Provider: primary.Provider, Protocol: primary.Protocol, AdapterType: adapter,
		BaseURL: command.Channel.BaseURL, Status: "pending", CredentialConfigured: true,
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_channel_reviews
    (merchant_profile_id, merchant_newapi_user_id, canonical_model_id, channel_test_id,
     channel_type, channel_name, base_url, adapter_type, encrypted_credential, credential_fingerprint,
     last_synced_at)
SELECT id, $1, $2, NULL, $3, $4, $5, $6, $7, $8, now()
FROM merchant_profiles
WHERE newapi_user_id = $1 AND status = 'active'
RETURNING id, created_at
`, command.UserID, primary.CanonicalModelID, command.Channel.ChannelType, command.Channel.ChannelName,
		command.Channel.BaseURL, adapter, command.Channel.EncryptedCredential, command.Channel.CredentialFingerprint,
	).Scan(&review.ID, &review.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return MerchantChannelReview{}, ErrMerchantInvalidState
	}
	if err != nil {
		return MerchantChannelReview{}, merchantStoreError(err)
	}
	for _, model := range command.Channel.UpstreamModels {
		status := model.MatchStatus
		if status != "matched" {
			status = "unmatched"
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_review_model_bindings
    (review_id, upstream_model_name, canonical_model_id, match_status)
VALUES ($1, $2, NULLIF($3, 0), $4)
ON CONFLICT (review_id, upstream_model_name) DO UPDATE
SET canonical_model_id = EXCLUDED.canonical_model_id, match_status = EXCLUDED.match_status,
    updated_at = now()
`, review.ID, strings.TrimSpace(model.UpstreamModelName), model.CanonicalModelID, status); err != nil {
			return MerchantChannelReview{}, fmt.Errorf("保存审核模型绑定：%w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return MerchantChannelReview{}, fmt.Errorf("提交多模型渠道审核事务：%w", err)
	}
	return review, nil
}

func adapterForChannelType(channelType int) string {
	switch channelType {
	case 14:
		return "anthropic"
	case 24:
		return "gemini"
	default:
		return "openai_compatible"
	}
}

func (s *SQLMerchantStore) SyncReviewModels(ctx context.Context, reviewID int64, models []SyncedChannelModel) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, model := range models {
		status := "unmatched"
		if model.MatchStatus == "matched" {
			status = "matched"
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_review_model_bindings (review_id, upstream_model_name, canonical_model_id, match_status)
VALUES ($1, $2, NULLIF($3, 0), $4)
ON CONFLICT (review_id, upstream_model_name) DO UPDATE
SET canonical_model_id = EXCLUDED.canonical_model_id, match_status = EXCLUDED.match_status, updated_at = now()
`, reviewID, model.UpstreamModelName, model.CanonicalModelID, status); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE merchant_channel_reviews SET last_synced_at = now(), updated_at = now() WHERE id = $1`, reviewID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLMerchantStore) SyncChannelModels(ctx context.Context, channelID int64, models []SyncedChannelModel) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, model := range models {
		status := "unmatched"
		if model.MatchStatus == "matched" {
			status = "matched"
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_channel_model_bindings (merchant_channel_id, upstream_model_name, canonical_model_id, match_status)
VALUES ($1, $2, NULLIF($3, 0), $4)
ON CONFLICT (merchant_channel_id, upstream_model_name) DO UPDATE
SET canonical_model_id = EXCLUDED.canonical_model_id, match_status = EXCLUDED.match_status,
    last_synced_at = now(), updated_at = now()
`, channelID, model.UpstreamModelName, model.CanonicalModelID, status); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE merchant_channels SET last_synced_at = now(), updated_at = now() WHERE id = $1`, channelID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLMerchantStore) PromoteReviewModels(ctx context.Context, reviewID int64, channelID int64) error {
	_, err := s.database.ExecContext(ctx, `
INSERT INTO merchant_channel_model_bindings
    (merchant_channel_id, upstream_model_name, canonical_model_id, match_status)
SELECT $1, upstream_model_name, canonical_model_id, match_status
FROM merchant_review_model_bindings
WHERE review_id = $2
ON CONFLICT (merchant_channel_id, upstream_model_name) DO UPDATE
SET canonical_model_id = EXCLUDED.canonical_model_id, match_status = EXCLUDED.match_status,
    last_synced_at = now(), updated_at = now()
`, channelID, reviewID)
	return err
}

func (s *SQLMerchantStore) ReviewModels(ctx context.Context, reviewID int64) ([]ReviewModelBinding, error) {
	rows, err := s.database.QueryContext(ctx, `SELECT id, review_id, upstream_model_name, COALESCE(canonical_model_id, 0), match_status, test_status, COALESCE(tested_by_newapi_user_id, 0), tested_at, COALESCE(test_latency_ms, 0), COALESCE(test_error, '') FROM merchant_review_model_bindings WHERE review_id = $1 ORDER BY id`, reviewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ReviewModelBinding, 0)
	for rows.Next() {
		var item ReviewModelBinding
		var testedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.ReviewID, &item.UpstreamModelName, &item.CanonicalModelID, &item.MatchStatus, &item.TestStatus, &item.TestedByNewAPIUserID, &testedAt, &item.TestLatencyMS, &item.TestError); err != nil {
			return nil, err
		}
		item.TestedAt = nullableTime(testedAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLMerchantStore) ChannelModels(ctx context.Context, channelID int64) ([]ChannelModelBinding, error) {
	rows, err := s.database.QueryContext(ctx, `SELECT id, merchant_channel_id, upstream_model_name, COALESCE(canonical_model_id, 0), match_status, last_synced_at FROM merchant_channel_model_bindings WHERE merchant_channel_id = $1 ORDER BY id`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ChannelModelBinding, 0)
	for rows.Next() {
		var item ChannelModelBinding
		if err := rows.Scan(&item.ID, &item.ChannelID, &item.UpstreamModelName, &item.CanonicalModelID, &item.MatchStatus, &item.LastSyncedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLMerchantStore) SetReviewModelTest(ctx context.Context, reviewID int64, bindingID int64, adminID int, status string, latency int, testError string) error {
	if status != "passed" && status != "failed" {
		return ErrMerchantInvalidState
	}
	result, err := s.database.ExecContext(ctx, `UPDATE merchant_review_model_bindings SET test_status = $1, tested_by_newapi_user_id = $2, tested_at = now(), test_latency_ms = $3, test_error = NULLIF($4, ''), updated_at = now() WHERE id = $5 AND review_id = $6`, status, adminID, latency, testError, bindingID, reviewID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return ErrMerchantInvalidState
	}
	return nil
}

type claimedChannelTest struct {
	ID          int64
	AdapterType string
	Model       string
	Provider    string
	Protocol    string
}

func claimChannelTest(ctx context.Context, tx *sql.Tx, userID int, channel ChannelSubmission) (claimedChannelTest, error) {
	receiptHash := channelTestReceiptHash(channel.TestReceipt)
	var result claimedChannelTest
	err := tx.QueryRowContext(ctx, `
SELECT t.id, t.adapter_type, cm.public_name, cm.provider, cm.protocol
FROM merchant_channel_tests t
JOIN canonical_models cm ON cm.id = t.canonical_model_id
WHERE t.receipt_token_hash = $1 AND t.newapi_user_id = $2
  AND t.canonical_model_id = $3 AND t.base_url = $4
  AND t.credential_fingerprint = $5 AND t.expires_at > now()
  AND t.consumed_at IS NULL AND cm.status = 'active'
FOR UPDATE OF t
`, receiptHash, userID, channel.CanonicalModelID, channel.BaseURL, channel.CredentialFingerprint).Scan(
		&result.ID, &result.AdapterType, &result.Model, &result.Provider, &result.Protocol,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return claimedChannelTest{}, ErrMerchantInvalidState
	}
	if err != nil {
		return claimedChannelTest{}, fmt.Errorf("校验渠道检测凭证：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE merchant_channel_tests SET consumed_at = now() WHERE id = $1`, result.ID); err != nil {
		return claimedChannelTest{}, fmt.Errorf("使用渠道检测凭证：%w", err)
	}
	return result, nil
}

func channelTestReceiptHash(receipt string) string {
	digest := sha256.Sum256([]byte(receipt))
	return hex.EncodeToString(digest[:])
}

func (s *SQLMerchantStore) ChannelsForUser(ctx context.Context, userID int) ([]MerchantChannel, error) {
	rows, err := s.database.QueryContext(ctx, channelSelect+`
WHERE p.newapi_user_id = $1
ORDER BY c.id DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询商户渠道：%w", err)
	}
	return scanMerchantChannels(rows)
}

func (s *SQLMerchantStore) ChannelReviewsForUser(ctx context.Context, userID int) ([]MerchantChannelReview, error) {
	rows, err := s.database.QueryContext(ctx, reviewSelect+`
WHERE r.merchant_newapi_user_id = $1
ORDER BY r.id DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询商户渠道审核：%w", err)
	}
	defer rows.Close()
	items := make([]MerchantChannelReview, 0)
	for rows.Next() {
		review, err := scanMerchantReview(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历商户渠道审核：%w", err)
	}
	return items, nil
}

func (s *SQLMerchantStore) ListApplications(ctx context.Context, page int, pageSize int, status string, keyword string) ([]MerchantApplication, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	status = strings.TrimSpace(status)
	keyword = strings.TrimSpace(keyword)
	var total int
	err := s.database.QueryRowContext(ctx, `
SELECT count(*)
FROM merchant_applications a
WHERE ($1 = '' OR a.status = $1)
  AND ($2 = '' OR a.display_name ILIKE '%' || $2 || '%' OR a.website_url ILIKE '%' || $2 || '%' OR a.newapi_user_id::text = $2)
`, status, keyword).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("统计商户申请：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, applicationSelect+`
WHERE ($1 = '' OR a.status = $1)
  AND ($2 = '' OR a.display_name ILIKE '%' || $2 || '%' OR a.website_url ILIKE '%' || $2 || '%' OR a.newapi_user_id::text = $2)
ORDER BY a.id DESC
LIMIT $3 OFFSET $4
`, status, keyword, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询商户申请：%w", err)
	}
	defer rows.Close()
	items := make([]MerchantApplication, 0)
	for rows.Next() {
		application, err := scanMerchantApplication(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, application)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历商户申请：%w", err)
	}
	return items, total, nil
}

func (s *SQLMerchantStore) ListChannelReviews(ctx context.Context, page int, pageSize int, status string) ([]MerchantChannelReview, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	status = strings.TrimSpace(status)
	var total int
	if err := s.database.QueryRowContext(ctx, "SELECT count(*) FROM merchant_channel_reviews WHERE ($1 = '' OR status = $1)", status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计渠道审核：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, reviewSelect+`
WHERE ($1 = '' OR r.status = $1)
ORDER BY r.id DESC
LIMIT $2 OFFSET $3
`, status, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询渠道审核：%w", err)
	}
	defer rows.Close()
	items := make([]MerchantChannelReview, 0)
	for rows.Next() {
		review, err := scanMerchantReview(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, review)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历渠道审核：%w", err)
	}
	return items, total, nil
}

func (s *SQLMerchantStore) ListChannels(ctx context.Context, page int, pageSize int, status string) ([]MerchantChannel, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	status = strings.TrimSpace(status)
	var total int
	if err := s.database.QueryRowContext(ctx, "SELECT count(*) FROM merchant_channels WHERE ($1 = '' OR status = $1)", status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计商户渠道：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, channelSelect+`
WHERE ($1 = '' OR c.status = $1)
ORDER BY c.id DESC
LIMIT $2 OFFSET $3
`, status, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询商户渠道：%w", err)
	}
	returnChannels, err := scanMerchantChannels(rows)
	if err != nil {
		return nil, 0, err
	}
	return returnChannels, total, nil
}

func (s *SQLMerchantStore) Channel(ctx context.Context, channelID int64) (MerchantChannel, error) {
	channel, err := scanMerchantChannel(s.database.QueryRowContext(ctx, channelSelect+"WHERE c.id = $1", channelID))
	if err != nil {
		return MerchantChannel{}, merchantStoreError(err)
	}
	return channel, nil
}

func (s *SQLMerchantStore) ListChannelsFiltered(ctx context.Context, page int, pageSize int, status, keyword string, userID int) ([]MerchantChannel, int, error) {
	items, _, err := s.ListChannels(ctx, 1, 100, status)
	if err != nil {
		return nil, 0, err
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	filtered := make([]MerchantChannel, 0, len(items))
	for _, item := range items {
		if userID > 0 && item.UserID != userID {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.ChannelName), keyword) && !strings.Contains(strings.ToLower(item.Model), keyword) && !strings.Contains(strings.ToLower(item.MerchantName), keyword) && !strings.Contains(strconv.Itoa(item.UserID), keyword) {
			continue
		}
		filtered = append(filtered, item)
	}
	page, pageSize = boundedPagination(page, pageSize)
	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return []MerchantChannel{}, len(filtered), nil
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], len(filtered), nil
}

func (s *SQLMerchantStore) ClaimReview(ctx context.Context, reviewID int64, targetStatus string) (MerchantChannelReview, error) {
	var allowed string
	switch targetStatus {
	case "verifying":
		allowed = "status IN ('pending', 'verified')"
	case "approving":
		allowed = "status = 'verified'"
	default:
		return MerchantChannelReview{}, ErrMerchantInvalidState
	}
	query := `
WITH claimed AS (
    UPDATE merchant_channel_reviews
    SET status = '` + targetStatus + `', updated_at = now()
    WHERE id = $1 AND ` + allowed + ` AND encrypted_credential IS NOT NULL
    RETURNING *
), updated_application AS (
    UPDATE merchant_applications a
    SET status = '` + targetStatus + `', updated_at = now()
    FROM claimed
    WHERE a.id = claimed.application_id
)
SELECT c.id, c.merchant_newapi_user_id, COALESCE(c.application_id, 0), COALESCE(c.replaces_channel_id, 0),
	   c.channel_name, c.canonical_model_id, cm.public_name, cm.provider, cm.protocol,
	   COALESCE(c.channel_test_id, 0), c.adapter_type, c.base_url, c.encrypted_credential,
	   c.credential_fingerprint, c.status, COALESCE(c.verification_error, ''),
	   COALESCE(c.review_reason, ''), c.verified_at, c.reviewed_at, c.created_at
FROM claimed c
JOIN canonical_models cm ON cm.id = c.canonical_model_id
`
	review, err := scanMerchantReview(s.database.QueryRowContext(ctx, query, reviewID))
	if errors.Is(err, ErrMerchantNotFound) {
		return MerchantChannelReview{}, ErrMerchantInvalidState
	}
	return review, err
}

func (s *SQLMerchantStore) FinishVerification(ctx context.Context, reviewID int64, adminID int, adapterType string, failure string) error {
	status := "verified"
	if failure != "" {
		status = "pending"
	}
	var returnedID int64
	err := s.database.QueryRowContext(ctx, `
WITH updated AS (
    UPDATE merchant_channel_reviews
	SET status = $1, adapter_type = CASE WHEN $2 = '' THEN adapter_type ELSE $2 END, verification_error = NULLIF($3, ''),
        verified_by_newapi_user_id = $4, verified_at = now(), updated_at = now()
    WHERE id = $5 AND status = 'verifying'
    RETURNING id, application_id
), updated_application AS (
    UPDATE merchant_applications a
    SET status = $1, updated_at = now()
    FROM updated
    WHERE a.id = updated.application_id
)
SELECT id FROM updated
`, status, adapterType, failure, adminID, reviewID).Scan(&returnedID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMerchantInvalidState
	}
	if err != nil {
		return fmt.Errorf("保存渠道验证结果：%w", err)
	}
	return nil
}

func (s *SQLMerchantStore) FinishApproval(ctx context.Context, reviewID int64, adminID int, newAPIChannelID int) (MerchantApproval, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantApproval{}, fmt.Errorf("开始商户审批事务：%w", err)
	}
	defer tx.Rollback()
	var review struct {
		ID, ApplicationID, CanonicalModelID int64
		UserID                              int
		ChannelName, BaseURL, AdapterType   string
		Fingerprint                         string
		DisplayName, WebsiteURL             string
	}
	err = tx.QueryRowContext(ctx, `
SELECT r.id, r.merchant_newapi_user_id, COALESCE(r.application_id, 0),
	   r.canonical_model_id, r.channel_name, r.base_url, r.adapter_type,
       r.credential_fingerprint, COALESCE(a.display_name, ''), COALESCE(a.website_url, '')
FROM merchant_channel_reviews r
LEFT JOIN merchant_applications a ON a.id = r.application_id
WHERE r.id = $1 AND r.status = 'approving' AND r.encrypted_credential IS NOT NULL
FOR UPDATE OF r
`, reviewID).Scan(&review.ID, &review.UserID, &review.ApplicationID, &review.CanonicalModelID,
		&review.ChannelName, &review.BaseURL, &review.AdapterType, &review.Fingerprint,
		&review.DisplayName, &review.WebsiteURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MerchantApproval{}, ErrMerchantInvalidState
		}
		return MerchantApproval{}, fmt.Errorf("锁定渠道审核：%w", err)
	}
	var merchantID int64
	if review.ApplicationID > 0 {
		err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_profiles
    (newapi_user_id, approved_application_id, display_name, website_url, status, approved_by_newapi_user_id)
VALUES ($1, $2, $3, $4, 'active', $5)
ON CONFLICT (newapi_user_id) DO UPDATE
SET approved_application_id = EXCLUDED.approved_application_id,
    display_name = EXCLUDED.display_name,
    website_url = EXCLUDED.website_url,
    status = 'active', status_reason = NULL,
    approved_by_newapi_user_id = EXCLUDED.approved_by_newapi_user_id,
    approved_at = now(), updated_at = now()
RETURNING id
`, review.UserID, review.ApplicationID, review.DisplayName, review.WebsiteURL, adminID).Scan(&merchantID)
	} else {
		err = tx.QueryRowContext(ctx, "SELECT id FROM merchant_profiles WHERE newapi_user_id = $1 AND status = 'active' FOR UPDATE", review.UserID).Scan(&merchantID)
	}
	if err != nil {
		return MerchantApproval{}, merchantStoreError(err)
	}
	var channelID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_channels
	(merchant_profile_id, review_id, canonical_model_id, newapi_channel_id,
	 channel_name, base_url, adapter_type, credential_fingerprint)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id
`, merchantID, review.ID, review.CanonicalModelID, newAPIChannelID, review.ChannelName,
		review.BaseURL, review.AdapterType, review.Fingerprint).Scan(&channelID)
	if err != nil {
		return MerchantApproval{}, merchantStoreError(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO channel_health (merchant_channel_id) VALUES ($1)`, channelID); err != nil {
		return MerchantApproval{}, fmt.Errorf("初始化渠道健康状态：%w", err)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE merchant_channel_reviews
SET status = 'approved', encrypted_credential = NULL, verification_error = NULL,
    reviewed_by_newapi_user_id = $1, reviewed_at = now(), merchant_profile_id = $3, updated_at = now()
WHERE id = $2 AND status = 'approving'
`, adminID, review.ID, merchantID)
	if err != nil {
		return MerchantApproval{}, fmt.Errorf("完成渠道审批：%w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return MerchantApproval{}, ErrMerchantInvalidState
	}
	if review.ApplicationID > 0 {
		result, err = tx.ExecContext(ctx, `
UPDATE merchant_applications
SET status = 'approved', review_reason = NULL, reviewed_by_newapi_user_id = $1,
    reviewed_at = now(), updated_at = now()
WHERE id = $2 AND status = 'approving'
`, adminID, review.ApplicationID)
		if err != nil {
			return MerchantApproval{}, fmt.Errorf("完成商户申请审批：%w", err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return MerchantApproval{}, ErrMerchantInvalidState
		}
	}
	if err := tx.Commit(); err != nil {
		return MerchantApproval{}, fmt.Errorf("提交商户审批事务：%w", err)
	}
	return MerchantApproval{MerchantID: merchantID, UserID: review.UserID, ChannelID: channelID, NewAPIChannelID: newAPIChannelID}, nil
}

func (s *SQLMerchantStore) CancelApproval(ctx context.Context, reviewID int64, failure string) error {
	var returnedID int64
	err := s.database.QueryRowContext(ctx, `
WITH updated AS (
    UPDATE merchant_channel_reviews
    SET status = 'verified', verification_error = NULLIF($1, ''), updated_at = now()
    WHERE id = $2 AND status = 'approving'
    RETURNING id, application_id
), updated_application AS (
    UPDATE merchant_applications a
    SET status = 'verified', updated_at = now()
    FROM updated
    WHERE a.id = updated.application_id
)
SELECT id FROM updated
`, failure, reviewID).Scan(&returnedID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMerchantInvalidState
	}
	if err != nil {
		return fmt.Errorf("恢复渠道审批状态：%w", err)
	}
	return nil
}

func (s *SQLMerchantStore) RejectReview(ctx context.Context, reviewID int64, adminID int, reason string) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始驳回事务：%w", err)
	}
	defer tx.Rollback()
	var applicationID sql.NullInt64
	err = tx.QueryRowContext(ctx, `
UPDATE merchant_channel_reviews
SET status = 'rejected', encrypted_credential = NULL, review_reason = $1,
    reviewed_by_newapi_user_id = $2, reviewed_at = now(), updated_at = now()
WHERE id = $3 AND status IN ('pending', 'verified')
RETURNING application_id
`, reason, adminID, reviewID).Scan(&applicationID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMerchantInvalidState
	}
	if err != nil {
		return fmt.Errorf("驳回渠道审核：%w", err)
	}
	if applicationID.Valid {
		result, err := tx.ExecContext(ctx, `
UPDATE merchant_applications
SET status = 'rejected', review_reason = $1, reviewed_by_newapi_user_id = $2,
    reviewed_at = now(), updated_at = now()
WHERE id = $3 AND status IN ('pending', 'verified')
`, reason, adminID, applicationID.Int64)
		if err != nil {
			return fmt.Errorf("驳回商户申请：%w", err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return ErrMerchantInvalidState
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交驳回事务：%w", err)
	}
	return nil
}

func (s *SQLMerchantStore) Merchant(ctx context.Context, merchantID int64) (MerchantProfile, error) {
	var profile MerchantProfile
	err := s.database.QueryRowContext(ctx, `
SELECT id, newapi_user_id, display_name, website_url, status,
       COALESCE(status_reason, ''), approved_at
FROM merchant_profiles
WHERE id = $1
`, merchantID).Scan(&profile.ID, &profile.UserID, &profile.DisplayName, &profile.WebsiteURL, &profile.Status, &profile.StatusReason, &profile.ApprovedAt)
	if err != nil {
		return MerchantProfile{}, merchantStoreError(err)
	}
	rows, err := s.database.QueryContext(ctx, "SELECT newapi_channel_id FROM merchant_channels WHERE merchant_profile_id = $1 AND status = 'active' ORDER BY id", merchantID)
	if err != nil {
		return MerchantProfile{}, fmt.Errorf("查询商户渠道编号：%w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return MerchantProfile{}, fmt.Errorf("解析商户渠道编号：%w", err)
		}
		profile.NewAPIChannelIDs = append(profile.NewAPIChannelIDs, id)
	}
	if err := rows.Err(); err != nil {
		return MerchantProfile{}, fmt.Errorf("遍历商户渠道编号：%w", err)
	}
	return profile, nil
}

func (s *SQLMerchantStore) SetMerchantStatus(ctx context.Context, merchantID int64, enabled bool, adminID int, reason string) (MerchantStatusChange, error) {
	status := "revoked"
	if enabled {
		status = "active"
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantStatusChange{}, fmt.Errorf("开始商户状态事务：%w", err)
	}
	defer tx.Rollback()
	var userID int
	err = tx.QueryRowContext(ctx, `
UPDATE merchant_profiles
SET status = $1, status_reason = $2, status_changed_by_newapi_user_id = $3,
    status_changed_at = now(), updated_at = now()
WHERE id = $4 AND status <> $1
RETURNING newapi_user_id
`, status, reason, adminID, merchantID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return MerchantStatusChange{}, ErrMerchantInvalidState
	}
	if err != nil {
		return MerchantStatusChange{}, fmt.Errorf("更新商户状态：%w", err)
	}
	if !enabled {
		if _, err := tx.ExecContext(ctx, "UPDATE merchant_channels SET status = 'disabled', updated_at = now() WHERE merchant_profile_id = $1 AND status = 'active'", merchantID); err != nil {
			return MerchantStatusChange{}, fmt.Errorf("停用商户渠道：%w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return MerchantStatusChange{}, fmt.Errorf("提交商户状态事务：%w", err)
	}
	return MerchantStatusChange{MerchantID: merchantID, UserID: userID, Status: status}, nil
}

func (s *SQLMerchantStore) SetChannelStatus(ctx context.Context, channelID int64, merchantID int64, enabled bool) error {
	status := "disabled"
	if enabled {
		status = "active"
	}
	query := `UPDATE merchant_channels SET status = $1, updated_at = now() WHERE id = $2 AND status <> 'archived'`
	args := []any{status, channelID}
	if merchantID > 0 {
		query += " AND merchant_profile_id = $3"
		args = append(args, merchantID)
	}
	result, err := s.database.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return ErrMerchantInvalidState
	}
	return nil
}

func (s *SQLMerchantStore) DeleteChannel(ctx context.Context, channelID int64, merchantID int64) error {
	query := `UPDATE merchant_channels SET status = 'archived', updated_at = now() WHERE id = $1`
	args := []any{channelID}
	if merchantID > 0 {
		query += " AND merchant_profile_id = $2"
		args = append(args, merchantID)
	}
	result, err := s.database.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return ErrMerchantInvalidState
	}
	return nil
}

func (s *SQLMerchantStore) DeleteChannelReview(ctx context.Context, reviewID int64, merchantID int64) (string, error) {
	if reviewID <= 0 || merchantID <= 0 {
		return "", ErrMerchantInvalidState
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("开始删除渠道审核事务：%w", err)
	}
	defer tx.Rollback()
	var status string
	var applicationID sql.NullInt64
	var channelTestID sql.NullInt64
	err = tx.QueryRowContext(ctx, `
SELECT r.status, r.application_id, r.channel_test_id
FROM merchant_channel_reviews r
WHERE r.id = $1 AND r.merchant_profile_id = $2
FOR UPDATE
`, reviewID, merchantID).Scan(&status, &applicationID, &channelTestID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrMerchantNotFound
	}
	if err != nil {
		return "", fmt.Errorf("锁定渠道审核记录：%w", err)
	}
	if applicationID.Valid || (status != "pending" && status != "verified" && status != "rejected") {
		return "", ErrMerchantInvalidState
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM merchant_channel_reviews WHERE id = $1", reviewID); err != nil {
		return "", fmt.Errorf("删除渠道审核记录：%w", err)
	}
	if channelTestID.Valid {
		if _, err := tx.ExecContext(ctx, `
DELETE FROM merchant_channel_tests test
WHERE test.id = $1
  AND NOT EXISTS (SELECT 1 FROM merchant_channel_reviews review WHERE review.channel_test_id = test.id)
`, channelTestID.Int64); err != nil {
			return "", fmt.Errorf("清理渠道检测凭证：%w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("提交删除渠道审核事务：%w", err)
	}
	if status == "rejected" {
		return "deleted", nil
	}
	return "withdrawn", nil
}

const applicationSelect = `
SELECT a.id, a.newapi_user_id, a.display_name, a.website_url, a.business_description,
       a.status, COALESCE(a.review_reason, ''), a.reviewed_at, a.created_at,
       r.id, r.merchant_newapi_user_id, COALESCE(r.application_id, 0), COALESCE(r.replaces_channel_id, 0),
	   r.channel_name, r.canonical_model_id, cm.public_name, cm.provider, cm.protocol,
	   COALESCE(r.channel_test_id, 0), r.adapter_type, r.base_url, r.encrypted_credential,
	   r.credential_fingerprint, r.status, COALESCE(r.verification_error, ''),
	   COALESCE(r.review_reason, ''), r.verified_at, r.reviewed_at, r.created_at
FROM merchant_applications a
JOIN merchant_channel_reviews r ON r.id = a.initial_channel_review_id
JOIN canonical_models cm ON cm.id = r.canonical_model_id
`

const reviewSelect = `
SELECT r.id, r.merchant_newapi_user_id, COALESCE(r.application_id, 0), COALESCE(r.replaces_channel_id, 0),
	   r.channel_name, r.canonical_model_id, cm.public_name, cm.provider, cm.protocol,
	   COALESCE(r.channel_test_id, 0), r.adapter_type, r.base_url, r.encrypted_credential,
	   r.credential_fingerprint, r.status, COALESCE(r.verification_error, ''),
	   COALESCE(r.review_reason, ''), r.verified_at, r.reviewed_at, r.created_at
FROM merchant_channel_reviews r
JOIN canonical_models cm ON cm.id = r.canonical_model_id
`

const channelSelect = `
SELECT c.id, c.merchant_profile_id, p.newapi_user_id, p.display_name, c.review_id, c.newapi_channel_id,
	   c.channel_name, c.canonical_model_id, cm.public_name, cm.provider, cm.protocol,
	   c.adapter_type, c.base_url,
	   c.credential_fingerprint, c.status, c.created_at,
	   COALESCE(h.state, 'healthy'), COALESCE(h.availability_ppm, 0),
	   COALESCE(h.ttft_p95_ms, 0), COALESCE(h.consecutive_failures, 0), h.cooldown_until
FROM merchant_channels c
JOIN merchant_profiles p ON p.id = c.merchant_profile_id
JOIN canonical_models cm ON cm.id = c.canonical_model_id
LEFT JOIN channel_health h ON h.merchant_channel_id = c.id
`

type rowScanner interface {
	Scan(...any) error
}

func scanMerchantApplication(scanner rowScanner) (MerchantApplication, error) {
	var application MerchantApplication
	var review MerchantChannelReview
	var applicationReviewedAt, reviewVerifiedAt, reviewReviewedAt sql.NullTime
	var applicationReason, verificationError, reviewReason string
	err := scanner.Scan(
		&application.ID, &application.UserID, &application.DisplayName, &application.WebsiteURL,
		&application.BusinessDescription, &application.Status, &applicationReason, &applicationReviewedAt, &application.CreatedAt,
		&review.ID, &review.UserID, &review.ApplicationID, &review.ReplacesChannelID,
		&review.ChannelName, &review.CanonicalModelID, &review.Model, &review.Provider, &review.Protocol,
		&review.ChannelTestID, &review.AdapterType, &review.BaseURL, &review.EncryptedCredential,
		&review.CredentialFingerprint, &review.Status, &verificationError, &reviewReason,
		&reviewVerifiedAt, &reviewReviewedAt, &review.CreatedAt,
	)
	if err != nil {
		return MerchantApplication{}, merchantStoreError(err)
	}
	application.ReviewReason = applicationReason
	application.ReviewedAt = nullableTime(applicationReviewedAt)
	review.VerificationError = verificationError
	review.ReviewReason = reviewReason
	review.VerifiedAt = nullableTime(reviewVerifiedAt)
	review.ReviewedAt = nullableTime(reviewReviewedAt)
	review.CredentialConfigured = len(review.EncryptedCredential) > 0
	application.InitialReview = review
	return application, nil
}

func scanMerchantReview(scanner rowScanner) (MerchantChannelReview, error) {
	var review MerchantChannelReview
	var verifiedAt, reviewedAt sql.NullTime
	err := scanner.Scan(
		&review.ID, &review.UserID, &review.ApplicationID, &review.ReplacesChannelID,
		&review.ChannelName, &review.CanonicalModelID, &review.Model, &review.Provider, &review.Protocol,
		&review.ChannelTestID, &review.AdapterType, &review.BaseURL, &review.EncryptedCredential,
		&review.CredentialFingerprint, &review.Status, &review.VerificationError,
		&review.ReviewReason, &verifiedAt, &reviewedAt, &review.CreatedAt,
	)
	if err != nil {
		return MerchantChannelReview{}, merchantStoreError(err)
	}
	review.VerifiedAt = nullableTime(verifiedAt)
	review.ReviewedAt = nullableTime(reviewedAt)
	review.CredentialConfigured = len(review.EncryptedCredential) > 0
	return review, nil
}

func scanMerchantChannels(rows *sql.Rows) ([]MerchantChannel, error) {
	defer rows.Close()
	items := make([]MerchantChannel, 0)
	for rows.Next() {
		channel, err := scanMerchantChannel(rows)
		if err != nil {
			return nil, fmt.Errorf("解析商户渠道：%w", err)
		}
		items = append(items, channel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历商户渠道：%w", err)
	}
	return items, nil
}

func scanMerchantChannel(scanner rowScanner) (MerchantChannel, error) {
	var channel MerchantChannel
	var fingerprint string
	var cooldownUntil sql.NullTime
	err := scanner.Scan(&channel.ID, &channel.MerchantID, &channel.UserID, &channel.MerchantName, &channel.ReviewID, &channel.NewAPIChannelID,
		&channel.ChannelName, &channel.CanonicalModelID, &channel.Model, &channel.Provider,
		&channel.Protocol, &channel.AdapterType, &channel.BaseURL, &fingerprint,
		&channel.Status, &channel.CreatedAt, &channel.HealthState, &channel.AvailabilityPPM,
		&channel.TTFTP95MS, &channel.ConsecutiveFailures, &cooldownUntil)
	if err != nil {
		return MerchantChannel{}, err
	}
	channel.CredentialFingerprint = shortFingerprint(fingerprint)
	channel.CooldownUntil = nullableTime(cooldownUntil)
	return channel, nil
}

func merchantStoreError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMerchantNotFound
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && (postgresError.Code == "23505" || postgresError.Code == "23503") {
		return ErrMerchantConflict
	}
	return err
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func shortFingerprint(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 12 {
		return value
	}
	return value[:6] + "..." + value[len(value)-6:]
}

func boundedPagination(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
