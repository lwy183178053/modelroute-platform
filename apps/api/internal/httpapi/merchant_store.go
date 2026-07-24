package httpapi

import (
	"context"
	"database/sql"
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
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantIdentity{}, fmt.Errorf("开始平台商户事务：%w", err)
	}
	defer tx.Rollback()
	var identity MerchantIdentity
	var createdAt time.Time
	err = tx.QueryRowContext(ctx, `
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
		&identity.MerchantID, &identity.UserID, &identity.DisplayName,
		&identity.Status, &identity.IsPlatformOwned, &createdAt,
	)
	if err != nil {
		return MerchantIdentity{}, merchantStoreError(err)
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO merchant_name_reservations
    (display_name_key, display_name, newapi_user_id, merchant_profile_id, status)
VALUES ($1, $2, $3, $4, 'active')
ON CONFLICT (display_name_key) DO UPDATE
SET display_name = EXCLUDED.display_name,
    merchant_profile_id = EXCLUDED.merchant_profile_id,
    status = 'active',
    updated_at = now()
WHERE merchant_name_reservations.newapi_user_id = EXCLUDED.newapi_user_id
`, strings.ToLower(identity.DisplayName), identity.DisplayName, identity.UserID, identity.MerchantID)
	if err != nil {
		return MerchantIdentity{}, merchantStoreError(err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return MerchantIdentity{}, ErrMerchantConflict
	}
	if err := tx.Commit(); err != nil {
		return MerchantIdentity{}, fmt.Errorf("提交平台商户事务：%w", err)
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
	application := MerchantApplication{
		UserID: command.UserID, DisplayName: command.DisplayName, WebsiteURL: command.WebsiteURL,
		BusinessDescription: command.BusinessDescription, Status: "pending",
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_applications
    (newapi_user_id, display_name, display_name_key, website_url, business_description)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at
`, command.UserID, command.DisplayName, command.DisplayNameKey, command.WebsiteURL, command.BusinessDescription).Scan(&application.ID, &application.CreatedAt)
	if err != nil {
		return MerchantApplication{}, merchantStoreError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_name_reservations
    (display_name_key, display_name, newapi_user_id, application_id, status)
VALUES ($1, $2, $3, $4, 'pending')
`, command.DisplayNameKey, command.DisplayName, command.UserID, application.ID); err != nil {
		return MerchantApplication{}, merchantStoreError(err)
	}
	if err := tx.Commit(); err != nil {
		return MerchantApplication{}, fmt.Errorf("提交商户申请事务：%w", err)
	}
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
	return scanMerchantApplication(s.database.QueryRowContext(ctx, applicationSelect+`WHERE a.id = $1`, id))
}

func (s *SQLMerchantStore) ApproveApplication(ctx context.Context, applicationID int64, adminID int) (MerchantProfile, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantProfile{}, fmt.Errorf("开始商户申请审批事务：%w", err)
	}
	defer tx.Rollback()
	var application MerchantApplication
	err = tx.QueryRowContext(ctx, `
SELECT id, newapi_user_id, display_name, website_url, business_description
FROM merchant_applications
WHERE id = $1 AND status = 'pending'
FOR UPDATE
`, applicationID).Scan(&application.ID, &application.UserID, &application.DisplayName, &application.WebsiteURL, &application.BusinessDescription)
	if errors.Is(err, sql.ErrNoRows) {
		return MerchantProfile{}, ErrMerchantInvalidState
	}
	if err != nil {
		return MerchantProfile{}, fmt.Errorf("锁定商户申请：%w", err)
	}
	var profile MerchantProfile
	err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_profiles
    (newapi_user_id, approved_application_id, display_name, website_url, business_description,
     status, approved_by_newapi_user_id, is_platform_owned)
VALUES ($1, $2, $3, $4, $5, 'active', $6, false)
ON CONFLICT (newapi_user_id) DO UPDATE
SET approved_application_id = EXCLUDED.approved_application_id,
    display_name = EXCLUDED.display_name,
    website_url = EXCLUDED.website_url,
    business_description = EXCLUDED.business_description,
    status = 'active', status_reason = NULL,
    approved_by_newapi_user_id = EXCLUDED.approved_by_newapi_user_id,
    approved_at = now(), is_platform_owned = false, updated_at = now()
RETURNING id, newapi_user_id, display_name, website_url, business_description, status,
          COALESCE(status_reason, ''), approved_at, is_platform_owned
`, application.UserID, application.ID, application.DisplayName, application.WebsiteURL,
		application.BusinessDescription, adminID).Scan(
		&profile.ID, &profile.UserID, &profile.DisplayName, &profile.WebsiteURL,
		&profile.BusinessDescription, &profile.Status, &profile.StatusReason,
		&profile.ApprovedAt, &profile.IsPlatformOwned,
	)
	if err != nil {
		return MerchantProfile{}, merchantStoreError(err)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE merchant_applications
SET status = 'approved', review_reason = NULL, reviewed_by_newapi_user_id = $1,
    reviewed_at = now(), updated_at = now()
WHERE id = $2 AND status = 'pending'
`, adminID, application.ID)
	if err != nil {
		return MerchantProfile{}, fmt.Errorf("完成商户申请审批：%w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return MerchantProfile{}, ErrMerchantInvalidState
	}
	result, err = tx.ExecContext(ctx, `
UPDATE merchant_name_reservations
SET merchant_profile_id = $1, status = 'active', updated_at = now()
WHERE application_id = $2 AND status = 'pending'
`, profile.ID, application.ID)
	if err != nil {
		return MerchantProfile{}, fmt.Errorf("激活商户名称：%w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return MerchantProfile{}, ErrMerchantInvalidState
	}
	if err := tx.Commit(); err != nil {
		return MerchantProfile{}, fmt.Errorf("提交商户申请审批事务：%w", err)
	}
	return profile, nil
}

func (s *SQLMerchantStore) RejectApplication(ctx context.Context, applicationID int64, adminID int, reason string) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始商户申请驳回事务：%w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
UPDATE merchant_applications
SET status = 'rejected', review_reason = $1, reviewed_by_newapi_user_id = $2,
    reviewed_at = now(), updated_at = now()
WHERE id = $3 AND status = 'pending'
`, reason, adminID, applicationID)
	if err != nil {
		return fmt.Errorf("驳回商户申请：%w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrMerchantInvalidState
	}
	result, err = tx.ExecContext(ctx, `DELETE FROM merchant_name_reservations WHERE application_id = $1 AND status = 'pending'`, applicationID)
	if err != nil {
		return fmt.Errorf("释放商户名称：%w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrMerchantInvalidState
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交商户申请驳回事务：%w", err)
	}
	return nil
}

func (s *SQLMerchantStore) ChannelsForUser(ctx context.Context, userID int) ([]MerchantChannel, error) {
	rows, err := s.database.QueryContext(ctx, channelSelect+`
WHERE c.deleted_at IS NULL AND p.newapi_user_id = $1
ORDER BY c.id DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询商户渠道：%w", err)
	}
	return scanMerchantChannels(rows)
}

func (s *SQLMerchantStore) ListApplications(ctx context.Context, page, pageSize int, status, keyword string) ([]MerchantApplication, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	status, keyword = strings.TrimSpace(status), strings.TrimSpace(keyword)
	var total int
	err := s.database.QueryRowContext(ctx, `
SELECT count(*) FROM merchant_applications
WHERE ($1 = '' OR status = $1)
  AND ($2 = '' OR display_name ILIKE '%' || $2 || '%' OR newapi_user_id::text = $2)
`, status, keyword).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("统计商户申请：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, applicationSelect+`
WHERE ($1 = '' OR a.status = $1)
  AND ($2 = '' OR a.display_name ILIKE '%' || $2 || '%' OR a.newapi_user_id::text = $2)
ORDER BY a.id DESC LIMIT $3 OFFSET $4
`, status, keyword, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询商户申请：%w", err)
	}
	defer rows.Close()
	items := make([]MerchantApplication, 0)
	for rows.Next() {
		item, err := scanMerchantApplication(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历商户申请：%w", err)
	}
	return items, total, nil
}

func (s *SQLMerchantStore) ListChannels(ctx context.Context, page, pageSize int) ([]MerchantChannel, int, error) {
	return s.ListChannelsFiltered(ctx, page, pageSize, "", 0)
}

func (s *SQLMerchantStore) Channel(ctx context.Context, channelID int64) (MerchantChannel, error) {
	return scanMerchantChannel(s.database.QueryRowContext(ctx, channelSelect+`
WHERE c.id = $1 AND c.deleted_at IS NULL
`, channelID))
}

func (s *SQLMerchantStore) ListChannelsFiltered(ctx context.Context, page, pageSize int, keyword string, userID int) ([]MerchantChannel, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	keyword = strings.TrimSpace(keyword)
	var total int
	if err := s.database.QueryRowContext(ctx, `
SELECT count(*)
FROM merchant_channels c
JOIN merchant_profiles p ON p.id = c.merchant_profile_id
WHERE c.deleted_at IS NULL
  AND ($1 = '' OR c.channel_name ILIKE '%' || $1 || '%' OR p.display_name ILIKE '%' || $1 || '%'
       OR c.base_url ILIKE '%' || $1 || '%' OR c.id::text = $1 OR p.newapi_user_id::text = $1)
  AND ($2 = 0 OR p.newapi_user_id = $2)
`, keyword, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计渠道：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, channelSelect+`
WHERE c.deleted_at IS NULL
  AND ($1 = '' OR c.channel_name ILIKE '%' || $1 || '%' OR p.display_name ILIKE '%' || $1 || '%'
       OR c.base_url ILIKE '%' || $1 || '%' OR c.id::text = $1 OR p.newapi_user_id::text = $1)
  AND ($2 = 0 OR p.newapi_user_id = $2)
ORDER BY c.id DESC LIMIT $3 OFFSET $4
`, keyword, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询渠道：%w", err)
	}
	items, err := scanMerchantChannels(rows)
	return items, total, err
}

func (s *SQLMerchantStore) Merchant(ctx context.Context, merchantID int64) (MerchantProfile, error) {
	var profile MerchantProfile
	err := s.database.QueryRowContext(ctx, `
SELECT id, newapi_user_id, display_name, website_url, business_description, status,
       COALESCE(status_reason, ''), approved_at, is_platform_owned
FROM merchant_profiles
WHERE id = $1
`, merchantID).Scan(&profile.ID, &profile.UserID, &profile.DisplayName, &profile.WebsiteURL,
		&profile.BusinessDescription, &profile.Status, &profile.StatusReason, &profile.ApprovedAt, &profile.IsPlatformOwned)
	if err != nil {
		return MerchantProfile{}, merchantStoreError(err)
	}
	rows, err := s.database.QueryContext(ctx, `
SELECT newapi_channel_id FROM merchant_channels
WHERE merchant_profile_id = $1 AND deleted_at IS NULL
ORDER BY id
`, merchantID)
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
	return profile, rows.Err()
}

func (s *SQLMerchantStore) UpdateMerchantDescription(ctx context.Context, merchantID int64, description string) (MerchantProfile, error) {
	var profile MerchantProfile
	err := s.database.QueryRowContext(ctx, `
UPDATE merchant_profiles
SET business_description = $2, updated_at = now()
WHERE id = $1 AND status = 'active'
RETURNING id, newapi_user_id, display_name, website_url, business_description, status,
          COALESCE(status_reason, ''), approved_at, is_platform_owned
`, merchantID, description).Scan(&profile.ID, &profile.UserID, &profile.DisplayName, &profile.WebsiteURL,
		&profile.BusinessDescription, &profile.Status, &profile.StatusReason, &profile.ApprovedAt, &profile.IsPlatformOwned)
	if err != nil {
		return MerchantProfile{}, merchantStoreError(err)
	}
	return profile, nil
}

func (s *SQLMerchantStore) SetMerchantStatus(ctx context.Context, merchantID int64, enabled bool, adminID int, reason string) (MerchantStatusChange, error) {
	status := "revoked"
	if enabled {
		status = "active"
	}
	var userID int
	err := s.database.QueryRowContext(ctx, `
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
	return MerchantStatusChange{MerchantID: merchantID, UserID: userID, Status: status}, nil
}

func (s *SQLMerchantStore) DeleteChannel(ctx context.Context, channelID, merchantID int64) error {
	query := `UPDATE merchant_channels SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`
	args := []any{channelID}
	if merchantID > 0 {
		query += " AND merchant_profile_id = $2"
		args = append(args, merchantID)
	}
	result, err := s.database.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("归档渠道：%w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrMerchantNotFound
	}
	return nil
}

const applicationSelect = `
SELECT a.id, a.newapi_user_id, a.display_name, a.website_url, a.business_description,
       a.status, COALESCE(a.review_reason, ''), a.reviewed_at, a.created_at
FROM merchant_applications a
`

const channelSelect = `
SELECT c.id, c.merchant_profile_id, p.newapi_user_id, p.display_name,
       c.newapi_channel_id, c.channel_name, c.channel_type, c.provider,
       c.adapter_type, c.base_url, c.credential_fingerprint, c.last_synced_at, c.created_at
FROM merchant_channels c
JOIN merchant_profiles p ON p.id = c.merchant_profile_id
`

type rowScanner interface {
	Scan(...any) error
}

func scanMerchantApplication(scanner rowScanner) (MerchantApplication, error) {
	var application MerchantApplication
	var reviewedAt sql.NullTime
	err := scanner.Scan(
		&application.ID, &application.UserID, &application.DisplayName, &application.WebsiteURL,
		&application.BusinessDescription, &application.Status, &application.ReviewReason, &reviewedAt, &application.CreatedAt,
	)
	if err != nil {
		return MerchantApplication{}, merchantStoreError(err)
	}
	application.ReviewedAt = nullableTime(reviewedAt)
	return application, nil
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
	var lastSyncedAt sql.NullTime
	err := scanner.Scan(
		&channel.ID, &channel.MerchantID, &channel.UserID, &channel.MerchantName,
		&channel.NewAPIChannelID, &channel.ChannelName, &channel.ChannelType,
		&channel.Provider, &channel.AdapterType, &channel.BaseURL, &fingerprint,
		&lastSyncedAt, &channel.CreatedAt,
	)
	if err != nil {
		return MerchantChannel{}, merchantStoreError(err)
	}
	channel.CredentialFingerprint = shortFingerprint(fingerprint)
	channel.LastSyncedAt = nullableTime(lastSyncedAt)
	return channel, nil
}

func merchantStoreError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMerchantNotFound
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		if postgresError.Code == "23505" && postgresError.ConstraintName == "merchant_name_reservations_pkey" {
			return ErrMerchantNameConflict
		}
		if postgresError.Code == "23505" || postgresError.Code == "23503" {
			return ErrMerchantConflict
		}
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

func boundedPagination(page, pageSize int) (int, int) {
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
