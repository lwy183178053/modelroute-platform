package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLSettlementStore struct {
	database *sql.DB
}

func NewSQLSettlementStore(database *sql.DB) *SQLSettlementStore {
	return &SQLSettlementStore{database: database}
}

func (s *SQLSettlementStore) LastSettlementSync(ctx context.Context) (int64, error) {
	var through int64
	if err := s.database.QueryRowContext(ctx, `
SELECT last_synced_through FROM settlement_sync_state WHERE singleton
`).Scan(&through); err != nil {
		return 0, fmt.Errorf("读取结算水位：%w", err)
	}
	return through, nil
}

func (s *SQLSettlementStore) CompleteSettlementSync(ctx context.Context, through int64, completedAt time.Time) error {
	_, err := s.database.ExecContext(ctx, `
UPDATE settlement_sync_state
SET last_synced_through = GREATEST(last_synced_through, $1),
    last_success_at = $2, last_error = NULL, updated_at = now()
WHERE singleton
`, through, completedAt)
	if err != nil {
		return fmt.Errorf("保存结算水位：%w", err)
	}
	return nil
}

func (s *SQLSettlementStore) RecordSettlementSyncError(ctx context.Context, message string) error {
	_, err := s.database.ExecContext(ctx, `
UPDATE settlement_sync_state SET last_error = $1, updated_at = now() WHERE singleton
`, message)
	if err != nil {
		return fmt.Errorf("保存结算同步错误：%w", err)
	}
	return nil
}

func (s *SQLSettlementStore) ApplySettlementLog(ctx context.Context, item SettlementLog) (SettlementApplyResult, error) {
	if item.ID <= 0 || item.UserID <= 0 || (item.Type != 2 && item.Type != 6) || item.ModelName == "" || item.ChannelID <= 0 || item.Quota <= 0 {
		return SettlementApplyResult{}, errors.New("结算日志字段无效")
	}
	if len(item.Other) == 0 || !json.Valid(item.Other) {
		item.Other = json.RawMessage(`{}`)
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return SettlementApplyResult{}, fmt.Errorf("开始结算事务：%w", err)
	}
	defer tx.Rollback()

	source, inserted, err := insertOrLockSettlementSource(ctx, tx, item)
	if err != nil {
		return SettlementApplyResult{}, err
	}
	if !inserted && source.Status == "settled" {
		if err := tx.Commit(); err != nil {
			return SettlementApplyResult{}, fmt.Errorf("提交重复结算事务：%w", err)
		}
		return SettlementApplyResult{SourceID: source.ID, Status: source.Status}, nil
	}
	if !inserted {
		item = source.Log
	}
	status, err := settleSource(ctx, tx, source.ID, item)
	if err != nil {
		return SettlementApplyResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return SettlementApplyResult{}, fmt.Errorf("提交结算事务：%w", err)
	}
	return SettlementApplyResult{SourceID: source.ID, Status: status}, nil
}

func (s *SQLSettlementStore) RetrySettlementReconciliation(ctx context.Context, limit int) (int, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := s.database.QueryContext(ctx, `
SELECT s.newapi_log_id
FROM settlement_sources s
JOIN settlement_reconciliation_items r ON r.settlement_source_id = s.id
WHERE s.status = 'reconciliation' AND r.status = 'open'
ORDER BY r.id
LIMIT $1
`, limit)
	if err != nil {
		return 0, fmt.Errorf("查询待重试对账项：%w", err)
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("读取待重试对账项：%w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return 0, fmt.Errorf("关闭待重试对账结果：%w", err)
	}
	settled := 0
	for _, id := range ids {
		item, err := s.settlementLog(ctx, id)
		if err != nil {
			return settled, err
		}
		result, err := s.ApplySettlementLog(ctx, item)
		if err != nil {
			return settled, err
		}
		if result.Status == "settled" {
			settled++
		}
	}
	return settled, nil
}

type storedSettlementSource struct {
	ID     int64
	Status string
	Log    SettlementLog
}

func insertOrLockSettlementSource(ctx context.Context, tx *sql.Tx, item SettlementLog) (storedSettlementSource, bool, error) {
	var source storedSettlementSource
	err := tx.QueryRowContext(ctx, `
INSERT INTO settlement_sources
    (newapi_log_id, newapi_user_id, source_created_at, source_type, model_name,
     newapi_channel_id, newapi_token_id, request_id, task_id, gross_quota, other)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)
ON CONFLICT (newapi_log_id) DO NOTHING
RETURNING id, status
`, item.ID, item.UserID, time.Unix(item.CreatedAt, 0).UTC(), item.Type, item.ModelName,
		item.ChannelID, item.TokenID, item.RequestID, item.TaskID, item.Quota, string(item.Other)).Scan(&source.ID, &source.Status)
	if err == nil {
		source.Log = item
		return source, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return storedSettlementSource{}, false, fmt.Errorf("保存结算来源：%w", err)
	}
	err = tx.QueryRowContext(ctx, `
SELECT id, status, newapi_log_id, newapi_user_id, extract(epoch FROM source_created_at)::bigint,
       source_type, model_name, gross_quota, newapi_channel_id, newapi_token_id,
       COALESCE(request_id, ''), COALESCE(task_id, ''), other
FROM settlement_sources WHERE newapi_log_id = $1 FOR UPDATE
`, item.ID).Scan(
		&source.ID, &source.Status, &source.Log.ID, &source.Log.UserID, &source.Log.CreatedAt,
		&source.Log.Type, &source.Log.ModelName, &source.Log.Quota, &source.Log.ChannelID,
		&source.Log.TokenID, &source.Log.RequestID, &source.Log.TaskID, &source.Log.Other,
	)
	if err != nil {
		return storedSettlementSource{}, false, fmt.Errorf("锁定结算来源：%w", err)
	}
	return source, false, nil
}

func settleSource(ctx context.Context, tx *sql.Tx, sourceID int64, item SettlementLog) (string, error) {
	var merchantID, offerID int64
	err := tx.QueryRowContext(ctx, `
SELECT mc.merchant_profile_id, o.id
FROM offer_versions o
JOIN merchant_channels mc ON mc.id = o.merchant_channel_id
WHERE mc.newapi_channel_id = $1 AND o.internal_alias = $2
`, item.ChannelID, item.ModelName).Scan(&merchantID, &offerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "reconciliation", recordSettlementReconciliation(ctx, tx, sourceID, "offer_mapping_missing", map[string]any{
			"newapi_channel_id": item.ChannelID, "model_name": item.ModelName,
		})
	}
	if err != nil {
		return "", fmt.Errorf("查询结算报价归属：%w", err)
	}
	if item.Type == 2 {
		return settleConsumption(ctx, tx, sourceID, merchantID, offerID, item)
	}
	return settleRefund(ctx, tx, sourceID, merchantID, offerID, item)
}

func settleConsumption(ctx context.Context, tx *sql.Tx, sourceID, merchantID, offerID int64, item SettlementLog) (string, error) {
	merchantQuota, platformQuota := splitSettlementQuota(item.Quota)
	available, frozen, err := updateMerchantBalance(ctx, tx, merchantID, merchantQuota, 0, merchantQuota, 0, 0)
	if err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE settlement_sources
SET status = 'settled', merchant_profile_id = $2, offer_version_id = $3,
    merchant_quota = $4, platform_quota = $5, processed_at = now(), updated_at = now()
WHERE id = $1
`, sourceID, merchantID, offerID, merchantQuota, platformQuota); err != nil {
		return "", fmt.Errorf("完成消费结算来源：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_ledger_entries
    (merchant_profile_id, settlement_source_id, entry_type, available_delta, frozen_delta,
     available_after, frozen_after, description)
VALUES ($1, $2, 'earning', $3, 0, $4, $5, '模型调用结算')
`, merchantID, sourceID, merchantQuota, available, frozen); err != nil {
		return "", fmt.Errorf("写入消费结算账本：%w", err)
	}
	if err := resolveSettlementReconciliation(ctx, tx, sourceID); err != nil {
		return "", err
	}
	return "settled", nil
}

type refundCandidate struct {
	sourceID         int64
	gross            int64
	merchant         int64
	platform         int64
	exactMatch       bool
	previousRefunded int64
	refundGross      int64
	refundMerchant   int64
	refundPlatform   int64
}

func settleRefund(ctx context.Context, tx *sql.Tx, sourceID, merchantID, offerID int64, item SettlementLog) (string, error) {
	if item.TaskID == "" && item.RequestID == "" {
		return "reconciliation", recordSettlementReconciliation(ctx, tx, sourceID, "refund_origin_missing", map[string]any{
			"reason": "退款日志缺少 task_id 和 request_id",
		})
	}
	rows, err := tx.QueryContext(ctx, `
SELECT id, gross_quota, merchant_quota, platform_quota,
       (($7 <> '' AND task_id = $7) OR ($7 = '' AND $8 <> '' AND request_id = $8)) AS exact_match
FROM settlement_sources
WHERE source_type = 2 AND status = 'settled' AND merchant_profile_id = $1 AND offer_version_id = $2
  AND newapi_user_id = $3 AND newapi_channel_id = $4 AND model_name = $5 AND source_created_at <= $6
  AND (
      (($7 <> '' AND task_id = $7) OR ($7 = '' AND $8 <> '' AND request_id = $8))
      OR (COALESCE(task_id, '') = '' AND COALESCE(request_id, '') = '')
  )
ORDER BY source_created_at, id
FOR UPDATE
`, merchantID, offerID, item.UserID, item.ChannelID, item.ModelName,
		time.Unix(item.CreatedAt, 0).UTC(), item.TaskID, item.RequestID)
	if err != nil {
		return "", fmt.Errorf("查询退款原消费：%w", err)
	}
	candidates := make([]refundCandidate, 0)
	for rows.Next() {
		var candidate refundCandidate
		if err := rows.Scan(&candidate.sourceID, &candidate.gross, &candidate.merchant, &candidate.platform, &candidate.exactMatch); err != nil {
			rows.Close()
			return "", fmt.Errorf("读取退款原消费：%w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Close(); err != nil {
		return "", fmt.Errorf("关闭退款原消费结果：%w", err)
	}
	exactFound := false
	exactCandidates := make([]refundCandidate, 0, len(candidates))
	fallbackCandidates := make([]refundCandidate, 0, len(candidates))
	for index := range candidates {
		if candidates[index].exactMatch {
			exactFound = true
		}
		if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(sum(gross_quota), 0) FROM settlement_refund_allocations WHERE consume_source_id = $1
`, candidates[index].sourceID).Scan(&candidates[index].previousRefunded); err != nil {
			return "", fmt.Errorf("读取原消费已退款额度：%w", err)
		}
		if candidates[index].previousRefunded >= candidates[index].gross {
			continue
		}
		if candidates[index].exactMatch {
			exactCandidates = append(exactCandidates, candidates[index])
		} else {
			fallbackCandidates = append(fallbackCandidates, candidates[index])
		}
	}
	if exactFound {
		candidates = exactCandidates
	} else {
		switch {
		case len(fallbackCandidates) == 0:
			return "reconciliation", recordSettlementReconciliation(ctx, tx, sourceID, "refund_origin_missing", map[string]any{
				"task_id": item.TaskID, "request_id": item.RequestID, "unmatched_quota": item.Quota,
			})
		case len(fallbackCandidates) > 1:
			return "reconciliation", recordSettlementReconciliation(ctx, tx, sourceID, "refund_origin_ambiguous", map[string]any{
				"task_id": item.TaskID, "request_id": item.RequestID, "candidate_count": len(fallbackCandidates),
			})
		default:
			available := fallbackCandidates[0].gross - fallbackCandidates[0].previousRefunded
			if available < item.Quota {
				return "reconciliation", recordSettlementReconciliation(ctx, tx, sourceID, "refund_origin_insufficient", map[string]any{
					"task_id": item.TaskID, "request_id": item.RequestID, "available_quota": available, "refund_quota": item.Quota,
				})
			}
			candidates = fallbackCandidates
		}
	}

	remaining := item.Quota
	var merchantRefund, platformRefund int64
	for index := range candidates {
		available := candidates[index].gross - candidates[index].previousRefunded
		if available > remaining {
			available = remaining
		}
		merchant, platform, err := refundSplitDelta(
			candidates[index].gross, candidates[index].merchant, candidates[index].platform,
			candidates[index].previousRefunded, available,
		)
		if err != nil {
			return "", fmt.Errorf("计算退款拆分：%w", err)
		}
		candidates[index].refundGross = available
		candidates[index].refundMerchant = merchant
		candidates[index].refundPlatform = platform
		merchantRefund += merchant
		platformRefund += platform
		remaining -= available
		if remaining == 0 {
			break
		}
	}
	if remaining != 0 {
		return "reconciliation", recordSettlementReconciliation(ctx, tx, sourceID, "refund_origin_missing", map[string]any{
			"task_id": item.TaskID, "request_id": item.RequestID, "unmatched_quota": remaining,
		})
	}

	available, frozen, err := updateMerchantBalance(ctx, tx, merchantID, -merchantRefund, 0, 0, merchantRefund, 0)
	if err != nil {
		return "", err
	}
	for _, candidate := range candidates {
		if candidate.refundGross == 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO settlement_refund_allocations
    (refund_source_id, consume_source_id, gross_quota, merchant_quota, platform_quota)
VALUES ($1, $2, $3, $4, $5)
`, sourceID, candidate.sourceID, candidate.refundGross, candidate.refundMerchant, candidate.refundPlatform); err != nil {
			return "", fmt.Errorf("记录退款原消费分配：%w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE settlement_sources
SET status = 'settled', merchant_profile_id = $2, offer_version_id = $3,
    merchant_quota = $4, platform_quota = $5, processed_at = now(), updated_at = now()
WHERE id = $1
`, sourceID, merchantID, offerID, -merchantRefund, -platformRefund); err != nil {
		return "", fmt.Errorf("完成退款结算来源：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_ledger_entries
    (merchant_profile_id, settlement_source_id, entry_type, available_delta, frozen_delta,
     available_after, frozen_after, description)
VALUES ($1, $2, 'refund', $3, 0, $4, $5, '模型调用退款冲正')
`, merchantID, sourceID, -merchantRefund, available, frozen); err != nil {
		return "", fmt.Errorf("写入退款冲正账本：%w", err)
	}
	if err := resolveSettlementReconciliation(ctx, tx, sourceID); err != nil {
		return "", err
	}
	return "settled", nil
}

func updateMerchantBalance(ctx context.Context, tx *sql.Tx, merchantID, availableDelta, frozenDelta, earnedDelta, refundedDelta, paidDelta int64) (int64, int64, error) {
	if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_balances (merchant_profile_id) VALUES ($1) ON CONFLICT DO NOTHING
`, merchantID); err != nil {
		return 0, 0, fmt.Errorf("初始化商户余额：%w", err)
	}
	var available, frozen int64
	err := tx.QueryRowContext(ctx, `
UPDATE merchant_balances
SET available_quota = available_quota + $2,
    frozen_quota = frozen_quota + $3,
    lifetime_earned_quota = lifetime_earned_quota + $4,
    lifetime_refunded_quota = lifetime_refunded_quota + $5,
    lifetime_paid_quota = lifetime_paid_quota + $6,
    updated_at = now()
WHERE merchant_profile_id = $1
RETURNING available_quota, frozen_quota
`, merchantID, availableDelta, frozenDelta, earnedDelta, refundedDelta, paidDelta).Scan(&available, &frozen)
	if err != nil {
		return 0, 0, fmt.Errorf("更新商户余额：%w", err)
	}
	return available, frozen, nil
}

func recordSettlementReconciliation(ctx context.Context, tx *sql.Tx, sourceID int64, reason string, details map[string]any) error {
	encoded, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("编码结算对账详情：%w", err)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO settlement_reconciliation_items (settlement_source_id, reason_code, details)
VALUES ($1, $2, $3::jsonb)
ON CONFLICT (settlement_source_id) DO UPDATE
SET reason_code = EXCLUDED.reason_code, details = EXCLUDED.details,
    status = CASE WHEN settlement_reconciliation_items.status = 'ignored' THEN 'ignored' ELSE 'open' END,
    updated_at = now()
`, sourceID, reason, string(encoded))
	if err != nil {
		return fmt.Errorf("记录结算对账项：%w", err)
	}
	return nil
}

func resolveSettlementReconciliation(ctx context.Context, tx *sql.Tx, sourceID int64) error {
	_, err := tx.ExecContext(ctx, `
UPDATE settlement_reconciliation_items
SET status = 'resolved', resolution_note = '映射恢复后自动结算', resolved_at = now(), updated_at = now()
WHERE settlement_source_id = $1 AND status = 'open'
`, sourceID)
	if err != nil {
		return fmt.Errorf("关闭结算对账项：%w", err)
	}
	return nil
}

func (s *SQLSettlementStore) settlementLog(ctx context.Context, newAPILogID int64) (SettlementLog, error) {
	var item SettlementLog
	err := s.database.QueryRowContext(ctx, `
SELECT newapi_log_id, newapi_user_id, extract(epoch FROM source_created_at)::bigint,
       source_type, model_name, gross_quota, newapi_channel_id, newapi_token_id,
       COALESCE(request_id, ''), COALESCE(task_id, ''), other
FROM settlement_sources WHERE newapi_log_id = $1
`, newAPILogID).Scan(
		&item.ID, &item.UserID, &item.CreatedAt, &item.Type, &item.ModelName, &item.Quota,
		&item.ChannelID, &item.TokenID, &item.RequestID, &item.TaskID, &item.Other,
	)
	if err != nil {
		return SettlementLog{}, fmt.Errorf("读取结算来源日志：%w", err)
	}
	return item, nil
}
