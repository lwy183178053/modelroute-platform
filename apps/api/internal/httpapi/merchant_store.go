package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
	models, err := json.Marshal(command.Channel.Models)
	if err != nil {
		return MerchantApplication{}, fmt.Errorf("编码拟售模型：%w", err)
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantApplication{}, fmt.Errorf("开始商户申请事务：%w", err)
	}
	defer tx.Rollback()

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
		ChannelType: command.Channel.ChannelType, BaseURL: command.Channel.BaseURL, RequestedModels: command.Channel.Models,
		Status: "pending", CredentialConfigured: true,
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_channel_reviews
    (merchant_newapi_user_id, application_id, channel_name, channel_type, base_url,
     requested_models, encrypted_credential, credential_fingerprint)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
RETURNING id, created_at
`, command.UserID, application.ID, command.Channel.ChannelName, command.Channel.ChannelType,
		command.Channel.BaseURL, models, command.Channel.EncryptedCredential, command.Channel.CredentialFingerprint,
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
	models, err := json.Marshal(command.Channel.Models)
	if err != nil {
		return MerchantChannelReview{}, fmt.Errorf("编码拟售模型：%w", err)
	}
	review := MerchantChannelReview{
		UserID: command.UserID, ChannelName: command.Channel.ChannelName, ChannelType: command.Channel.ChannelType,
		BaseURL: command.Channel.BaseURL, RequestedModels: command.Channel.Models, Status: "pending", CredentialConfigured: true,
	}
	err = s.database.QueryRowContext(ctx, `
INSERT INTO merchant_channel_reviews
    (merchant_profile_id, merchant_newapi_user_id, channel_name, channel_type, base_url,
     requested_models, encrypted_credential, credential_fingerprint)
SELECT id, $1, $2, $3, $4, $5::jsonb, $6, $7
FROM merchant_profiles
WHERE newapi_user_id = $1 AND status = 'active'
RETURNING id, created_at
`, command.UserID, command.Channel.ChannelName, command.Channel.ChannelType, command.Channel.BaseURL,
		models, command.Channel.EncryptedCredential, command.Channel.CredentialFingerprint,
	).Scan(&review.ID, &review.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return MerchantChannelReview{}, ErrMerchantInvalidState
		}
		return MerchantChannelReview{}, merchantStoreError(err)
	}
	return review, nil
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
WHERE merchant_newapi_user_id = $1
ORDER BY id DESC
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
WHERE ($1 = '' OR status = $1)
ORDER BY id DESC
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

func (s *SQLMerchantStore) ListMerchants(ctx context.Context, page int, pageSize int, status string) ([]MerchantProfile, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	status = strings.TrimSpace(status)
	var total int
	if err := s.database.QueryRowContext(ctx, "SELECT count(*) FROM merchant_profiles WHERE ($1 = '' OR status = $1)", status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计商户：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, `
SELECT id, newapi_user_id, display_name, website_url, status,
       COALESCE(status_reason, ''), approved_at
FROM merchant_profiles
WHERE ($1 = '' OR status = $1)
ORDER BY id DESC
LIMIT $2 OFFSET $3
`, status, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询商户：%w", err)
	}
	defer rows.Close()
	items := make([]MerchantProfile, 0)
	for rows.Next() {
		var profile MerchantProfile
		if err := rows.Scan(&profile.ID, &profile.UserID, &profile.DisplayName, &profile.WebsiteURL, &profile.Status, &profile.StatusReason, &profile.ApprovedAt); err != nil {
			return nil, 0, fmt.Errorf("解析商户：%w", err)
		}
		items = append(items, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历商户：%w", err)
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
SELECT id, merchant_newapi_user_id, COALESCE(application_id, 0), COALESCE(replaces_channel_id, 0),
       channel_name, channel_type, base_url, requested_models, encrypted_credential,
       credential_fingerprint, status, COALESCE(verified_models, '[]'::jsonb),
       COALESCE(verification_error, ''), COALESCE(review_reason, ''), verified_at, reviewed_at, created_at
FROM claimed
`
	review, err := scanMerchantReview(s.database.QueryRowContext(ctx, query, reviewID))
	if errors.Is(err, ErrMerchantNotFound) {
		return MerchantChannelReview{}, ErrMerchantInvalidState
	}
	return review, err
}

func (s *SQLMerchantStore) FinishVerification(ctx context.Context, reviewID int64, adminID int, models []string, failure string) error {
	encodedModels, err := json.Marshal(models)
	if err != nil {
		return fmt.Errorf("编码验证模型：%w", err)
	}
	status := "verified"
	if failure != "" {
		status = "pending"
	}
	var returnedID int64
	err = s.database.QueryRowContext(ctx, `
WITH updated AS (
    UPDATE merchant_channel_reviews
    SET status = $1, verified_models = $2::jsonb, verification_error = NULLIF($3, ''),
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
`, status, encodedModels, failure, adminID, reviewID).Scan(&returnedID)
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
		ID, ApplicationID       int64
		UserID, ChannelType     int
		ChannelName, BaseURL    string
		Models                  []byte
		Fingerprint             string
		DisplayName, WebsiteURL string
	}
	err = tx.QueryRowContext(ctx, `
SELECT r.id, r.merchant_newapi_user_id, COALESCE(r.application_id, 0),
       r.channel_name, r.channel_type, r.base_url, r.requested_models,
       r.credential_fingerprint, COALESCE(a.display_name, ''), COALESCE(a.website_url, '')
FROM merchant_channel_reviews r
LEFT JOIN merchant_applications a ON a.id = r.application_id
WHERE r.id = $1 AND r.status = 'approving' AND r.encrypted_credential IS NOT NULL
FOR UPDATE OF r
`, reviewID).Scan(&review.ID, &review.UserID, &review.ApplicationID, &review.ChannelName, &review.ChannelType,
		&review.BaseURL, &review.Models, &review.Fingerprint, &review.DisplayName, &review.WebsiteURL)
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
    (merchant_profile_id, review_id, newapi_channel_id, channel_name, channel_type,
     base_url, models, credential_fingerprint)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
RETURNING id
`, merchantID, review.ID, newAPIChannelID, review.ChannelName, review.ChannelType,
		review.BaseURL, review.Models, review.Fingerprint).Scan(&channelID)
	if err != nil {
		return MerchantApproval{}, merchantStoreError(err)
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

const applicationSelect = `
SELECT a.id, a.newapi_user_id, a.display_name, a.website_url, a.business_description,
       a.status, COALESCE(a.review_reason, ''), a.reviewed_at, a.created_at,
       r.id, r.merchant_newapi_user_id, COALESCE(r.application_id, 0), COALESCE(r.replaces_channel_id, 0),
       r.channel_name, r.channel_type, r.base_url, r.requested_models, r.encrypted_credential,
       r.credential_fingerprint, r.status, COALESCE(r.verified_models, '[]'::jsonb),
       COALESCE(r.verification_error, ''), COALESCE(r.review_reason, ''), r.verified_at, r.reviewed_at, r.created_at
FROM merchant_applications a
JOIN merchant_channel_reviews r ON r.id = a.initial_channel_review_id
`

const reviewSelect = `
SELECT id, merchant_newapi_user_id, COALESCE(application_id, 0), COALESCE(replaces_channel_id, 0),
       channel_name, channel_type, base_url, requested_models, encrypted_credential,
       credential_fingerprint, status, COALESCE(verified_models, '[]'::jsonb),
       COALESCE(verification_error, ''), COALESCE(review_reason, ''), verified_at, reviewed_at, created_at
FROM merchant_channel_reviews
`

const channelSelect = `
SELECT c.id, c.merchant_profile_id, c.review_id, c.newapi_channel_id,
       c.channel_name, c.channel_type, c.base_url, c.models,
       c.credential_fingerprint, c.status, c.created_at
FROM merchant_channels c
JOIN merchant_profiles p ON p.id = c.merchant_profile_id
`

type rowScanner interface {
	Scan(...any) error
}

func scanMerchantApplication(scanner rowScanner) (MerchantApplication, error) {
	var application MerchantApplication
	var review MerchantChannelReview
	var applicationReviewedAt, reviewVerifiedAt, reviewReviewedAt sql.NullTime
	var requestedModels, verifiedModels []byte
	var applicationReason, verificationError, reviewReason string
	err := scanner.Scan(
		&application.ID, &application.UserID, &application.DisplayName, &application.WebsiteURL,
		&application.BusinessDescription, &application.Status, &applicationReason, &applicationReviewedAt, &application.CreatedAt,
		&review.ID, &review.UserID, &review.ApplicationID, &review.ReplacesChannelID,
		&review.ChannelName, &review.ChannelType, &review.BaseURL, &requestedModels, &review.EncryptedCredential,
		&review.CredentialFingerprint, &review.Status, &verifiedModels, &verificationError, &reviewReason,
		&reviewVerifiedAt, &reviewReviewedAt, &review.CreatedAt,
	)
	if err != nil {
		return MerchantApplication{}, merchantStoreError(err)
	}
	if err := json.Unmarshal(requestedModels, &review.RequestedModels); err != nil {
		return MerchantApplication{}, fmt.Errorf("解析拟售模型：%w", err)
	}
	if err := json.Unmarshal(verifiedModels, &review.VerifiedModels); err != nil {
		return MerchantApplication{}, fmt.Errorf("解析验证模型：%w", err)
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
	var requestedModels, verifiedModels []byte
	var verifiedAt, reviewedAt sql.NullTime
	err := scanner.Scan(
		&review.ID, &review.UserID, &review.ApplicationID, &review.ReplacesChannelID,
		&review.ChannelName, &review.ChannelType, &review.BaseURL, &requestedModels, &review.EncryptedCredential,
		&review.CredentialFingerprint, &review.Status, &verifiedModels, &review.VerificationError,
		&review.ReviewReason, &verifiedAt, &reviewedAt, &review.CreatedAt,
	)
	if err != nil {
		return MerchantChannelReview{}, merchantStoreError(err)
	}
	if err := json.Unmarshal(requestedModels, &review.RequestedModels); err != nil {
		return MerchantChannelReview{}, fmt.Errorf("解析拟售模型：%w", err)
	}
	if err := json.Unmarshal(verifiedModels, &review.VerifiedModels); err != nil {
		return MerchantChannelReview{}, fmt.Errorf("解析验证模型：%w", err)
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
		var channel MerchantChannel
		var models []byte
		var fingerprint string
		if err := rows.Scan(&channel.ID, &channel.MerchantID, &channel.ReviewID, &channel.NewAPIChannelID,
			&channel.ChannelName, &channel.ChannelType, &channel.BaseURL, &models, &fingerprint,
			&channel.Status, &channel.CreatedAt); err != nil {
			return nil, fmt.Errorf("解析商户渠道：%w", err)
		}
		if err := json.Unmarshal(models, &channel.Models); err != nil {
			return nil, fmt.Errorf("解析渠道模型：%w", err)
		}
		channel.CredentialFingerprint = shortFingerprint(fingerprint)
		items = append(items, channel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历商户渠道：%w", err)
	}
	return items, nil
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
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
