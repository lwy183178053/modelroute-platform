package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func (s *SQLSettlementStore) ListSettlementReconciliation(ctx context.Context, page int, pageSize int, status string) ([]SettlementReconciliationItem, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	var total int
	if err := s.database.QueryRowContext(ctx, `
SELECT count(*) FROM settlement_reconciliation_items WHERE ($1 = '' OR status = $1)
`, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计结算对账项：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, `
SELECT r.id, r.settlement_source_id, s.newapi_log_id, s.newapi_user_id, s.source_type,
       s.model_name, COALESCE(c.id, 0), s.newapi_channel_id, s.gross_quota, r.reason_code, r.details,
       r.status, COALESCE(r.resolution_note, ''), COALESCE(r.resolved_by_newapi_user_id, 0),
       r.resolved_at, r.created_at
FROM settlement_reconciliation_items r
JOIN settlement_sources s ON s.id = r.settlement_source_id
LEFT JOIN merchant_channels c ON c.newapi_channel_id = s.newapi_channel_id
WHERE ($1 = '' OR r.status = $1)
ORDER BY r.id DESC LIMIT $2 OFFSET $3
`, status, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询结算对账项：%w", err)
	}
	defer rows.Close()
	items := make([]SettlementReconciliationItem, 0)
	for rows.Next() {
		var item SettlementReconciliationItem
		if err := rows.Scan(
			&item.ID, &item.SourceID, &item.NewAPILogID, &item.NewAPIUserID, &item.SourceType,
			&item.ModelName, &item.ChannelID, &item.NewAPIChannelID, &item.GrossQuota, &item.ReasonCode,
			&item.Details, &item.Status, &item.ResolutionNote, &item.ResolvedByUserID,
			&item.ResolvedAt, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("读取结算对账项：%w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *SQLSettlementStore) IgnoreSettlementReconciliation(ctx context.Context, id int64, adminID int, note string, resolvedAt time.Time) (SettlementReconciliationItem, error) {
	var item SettlementReconciliationItem
	err := s.database.QueryRowContext(ctx, `
UPDATE settlement_reconciliation_items
SET status = 'ignored', resolution_note = $2, resolved_by_newapi_user_id = $3,
    resolved_at = $4, updated_at = now()
WHERE id = $1 AND status = 'open'
RETURNING id, settlement_source_id, status, resolution_note,
          resolved_by_newapi_user_id, resolved_at, created_at
`, id, note, adminID, resolvedAt).Scan(
		&item.ID, &item.SourceID, &item.Status, &item.ResolutionNote,
		&item.ResolvedByUserID, &item.ResolvedAt, &item.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return SettlementReconciliationItem{}, ErrSettlementConflict
	}
	if err != nil {
		return SettlementReconciliationItem{}, fmt.Errorf("忽略结算对账项：%w", err)
	}
	return item, nil
}
