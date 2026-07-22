package httpapi

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type AdminOverviewSnapshot struct {
	TodayGrossQuota             int64      `json:"today_gross_quota"`
	TodayMerchantQuota          int64      `json:"today_merchant_quota"`
	TodayPlatformQuota          int64      `json:"today_platform_quota"`
	TodayRefundQuota            int64      `json:"today_refund_quota"`
	TodayRequestCount           int64      `json:"today_request_count"`
	TodaySuccessfulRequests     int64      `json:"today_successful_requests"`
	PendingMerchantApplications int64      `json:"pending_merchant_applications"`
	PendingChannelReviews       int64      `json:"pending_channel_reviews"`
	PendingWithdrawals          int64      `json:"pending_withdrawals"`
	OpenReconciliation          int64      `json:"open_reconciliation"`
	TotalChannels               int64      `json:"total_channels"`
	ActiveChannels              int64      `json:"active_channels"`
	CoolingChannels             int64      `json:"cooling_channels"`
	DisabledChannels            int64      `json:"disabled_channels"`
	PaymentEnabled              bool       `json:"payment_enabled"`
	LastSyncedThrough           int64      `json:"last_synced_through"`
	LastSyncSuccessAt           *time.Time `json:"last_sync_success_at,omitempty"`
	LastSyncError               string     `json:"last_sync_error,omitempty"`
}

type SQLAdminOverviewStore struct {
	database *sql.DB
}

func NewSQLAdminOverviewStore(database *sql.DB) *SQLAdminOverviewStore {
	return &SQLAdminOverviewStore{database: database}
}

func (s *SQLAdminOverviewStore) Snapshot(ctx context.Context, now time.Time) (AdminOverviewSnapshot, error) {
	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)
	local := now.In(shanghai)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, shanghai).UTC()
	var snapshot AdminOverviewSnapshot
	var lastSuccess sql.NullTime
	var lastError sql.NullString
	err := s.database.QueryRowContext(ctx, `
SELECT
    (SELECT count(*) FROM merchant_applications WHERE status IN ('pending', 'verifying', 'verified', 'approving')),
    (SELECT count(*) FROM merchant_channel_reviews WHERE status IN ('pending', 'verifying', 'verified', 'approving')),
    (SELECT count(*) FROM merchant_withdrawals WHERE status = 'pending'),
    (SELECT count(*) FROM settlement_reconciliation_items WHERE status = 'open'),
    (SELECT count(*) FROM merchant_channels),
    (SELECT count(*) FROM merchant_channels WHERE status = 'active'),
    (SELECT count(DISTINCT merchant_channel_id) FROM channel_model_health WHERE state = 'cooling'),
    (SELECT count(*) FROM merchant_channels mc
       WHERE mc.status = 'disabled'
          OR EXISTS (SELECT 1 FROM channel_model_health h WHERE h.merchant_channel_id = mc.id AND h.state = 'disabled')),
    COALESCE((SELECT enabled FROM payment_setting_versions WHERE is_current LIMIT 1), false),
    COALESCE((SELECT sum(CASE WHEN source_type = 2 THEN gross_quota ELSE -gross_quota END)
                FROM settlement_sources WHERE status = 'settled' AND source_created_at >= $1), 0),
    COALESCE((SELECT sum(CASE WHEN source_type = 2 THEN merchant_quota ELSE -merchant_quota END)
                FROM settlement_sources WHERE status = 'settled' AND source_created_at >= $1), 0),
    COALESCE((SELECT sum(CASE WHEN source_type = 2 THEN platform_quota ELSE -platform_quota END)
                FROM settlement_sources WHERE status = 'settled' AND source_created_at >= $1), 0),
    COALESCE((SELECT sum(gross_quota) FROM settlement_sources
                WHERE status = 'settled' AND source_type = 6 AND source_created_at >= $1), 0),
    (SELECT count(*) FROM routing_decisions WHERE created_at >= $1),
    (SELECT count(*) FROM routing_decisions
       WHERE created_at >= $1 AND result = 'completed' AND upstream_status_code BETWEEN 200 AND 399),
    last_synced_through, last_success_at, last_error
FROM settlement_sync_state
WHERE singleton
`, dayStart).Scan(
		&snapshot.PendingMerchantApplications, &snapshot.PendingChannelReviews,
		&snapshot.PendingWithdrawals, &snapshot.OpenReconciliation,
		&snapshot.TotalChannels, &snapshot.ActiveChannels, &snapshot.CoolingChannels, &snapshot.DisabledChannels,
		&snapshot.PaymentEnabled, &snapshot.TodayGrossQuota, &snapshot.TodayMerchantQuota, &snapshot.TodayPlatformQuota,
		&snapshot.TodayRefundQuota, &snapshot.TodayRequestCount, &snapshot.TodaySuccessfulRequests,
		&snapshot.LastSyncedThrough, &lastSuccess, &lastError,
	)
	if err != nil {
		return AdminOverviewSnapshot{}, fmt.Errorf("读取管理员总览：%w", err)
	}
	if lastSuccess.Valid {
		value := lastSuccess.Time.UTC()
		snapshot.LastSyncSuccessAt = &value
	}
	if lastError.Valid {
		snapshot.LastSyncError = lastError.String
	}
	return snapshot, nil
}
