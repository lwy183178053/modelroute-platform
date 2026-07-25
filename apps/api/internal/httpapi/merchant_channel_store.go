package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func (s *SQLMerchantStore) CreateSelfServiceChannel(ctx context.Context, command StoreMerchantChannel) (MerchantChannel, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantChannel{}, fmt.Errorf("开始创建渠道事务：%w", err)
	}
	defer tx.Rollback()
	var result MerchantChannel
	err = tx.QueryRowContext(ctx, `
INSERT INTO merchant_channels
    (merchant_profile_id, newapi_channel_id, channel_name, base_url, provider,
     adapter_type, channel_type, credential_fingerprint, last_synced_at)
SELECT profile.id, $2, $3, $4, $5, $6, $7, $8, now()
FROM merchant_profiles profile
WHERE profile.newapi_user_id = $1 AND profile.status = 'active'
RETURNING id, merchant_profile_id, $1, $3, $5, $4, last_synced_at, created_at
`, command.UserID, command.NewAPIChannelID, command.ChannelName, command.BaseURL,
		command.Provider, command.AdapterType, command.ChannelType, command.CredentialFingerprint,
	).Scan(&result.ID, &result.MerchantID, &result.UserID, &result.ChannelName,
		&result.Provider, &result.BaseURL, &result.LastSyncedAt, &result.CreatedAt)
	if err != nil {
		return MerchantChannel{}, merchantStoreError(err)
	}
	for _, model := range command.Models {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_channel_model_bindings
    (merchant_channel_id, upstream_model_name, canonical_model_id, match_status, health_state, last_synced_at)
VALUES ($1, $2, $3, 'matched', 'pending', now())
`, result.ID, model.UpstreamModelName, model.CanonicalModelID); err != nil {
			return MerchantChannel{}, fmt.Errorf("保存渠道模型：%w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return MerchantChannel{}, fmt.Errorf("提交创建渠道事务：%w", err)
	}
	return result, nil
}

func (s *SQLMerchantStore) SyncChannelModels(ctx context.Context, channelID int64, models []SyncedChannelModel) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始同步渠道模型事务：%w", err)
	}
	defer tx.Rollback()
	var lockedID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM merchant_channels WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, channelID).Scan(&lockedID); err != nil {
		return merchantStoreError(err)
	}
	canonicalIDs := make([]int64, 0, len(models))
	for _, model := range models {
		if model.MatchStatus == "matched" && model.CanonicalModelID > 0 {
			canonicalIDs = append(canonicalIDs, model.CanonicalModelID)
		}
	}
	encodedIDs, _ := json.Marshal(canonicalIDs)
	if _, err := tx.ExecContext(ctx, `
DELETE FROM merchant_channel_model_bindings
WHERE merchant_channel_id = $1
  AND canonical_model_id NOT IN (
      SELECT value::bigint FROM jsonb_array_elements_text($2::jsonb) AS value
  )
`, channelID, encodedIDs); err != nil {
		return fmt.Errorf("移除已消失渠道模型：%w", err)
	}
	for _, model := range models {
		if model.MatchStatus != "matched" || model.CanonicalModelID <= 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_channel_model_bindings
    (merchant_channel_id, upstream_model_name, canonical_model_id, match_status, health_state, last_synced_at)
VALUES ($1, $2, $3, 'matched', 'pending', now())
ON CONFLICT (merchant_channel_id, upstream_model_name) DO UPDATE
SET canonical_model_id = EXCLUDED.canonical_model_id,
    match_status = 'matched',
    health_state = CASE
        WHEN merchant_channel_model_bindings.canonical_model_id = EXCLUDED.canonical_model_id
            THEN merchant_channel_model_bindings.health_state
        ELSE 'pending'
    END,
    consecutive_request_failures = CASE
        WHEN merchant_channel_model_bindings.canonical_model_id = EXCLUDED.canonical_model_id
            THEN merchant_channel_model_bindings.consecutive_request_failures
        ELSE 0
    END,
    automatic_check_failures = CASE
        WHEN merchant_channel_model_bindings.canonical_model_id = EXCLUDED.canonical_model_id
            THEN merchant_channel_model_bindings.automatic_check_failures
        ELSE 0
    END,
    next_check_at = CASE
        WHEN merchant_channel_model_bindings.canonical_model_id = EXCLUDED.canonical_model_id
            THEN merchant_channel_model_bindings.next_check_at
        ELSE NULL
    END,
    last_synced_at = now(), updated_at = now()
`, channelID, model.UpstreamModelName, model.CanonicalModelID); err != nil {
			return fmt.Errorf("保存同步渠道模型：%w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE merchant_channels SET last_synced_at = now(), updated_at = now() WHERE id = $1`, channelID); err != nil {
		return fmt.Errorf("更新渠道同步时间：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交同步渠道模型事务：%w", err)
	}
	return nil
}

func (s *SQLMerchantStore) ChannelModels(ctx context.Context, channelID int64) ([]ChannelModelBinding, error) {
	rows, err := s.database.QueryContext(ctx, `
SELECT binding.id, binding.merchant_channel_id, binding.upstream_model_name,
       COALESCE(binding.canonical_model_id, 0), COALESCE(model.public_name, ''), binding.match_status,
       binding.health_state, binding.consecutive_request_failures, binding.automatic_check_failures,
       binding.next_check_at, binding.last_checked_at, COALESCE(binding.last_check_latency_ms, 0),
       COALESCE(binding.last_status_code, 0), COALESCE(binding.last_check_error, ''), binding.last_synced_at
FROM merchant_channel_model_bindings binding
LEFT JOIN canonical_models model ON model.id = binding.canonical_model_id
WHERE binding.merchant_channel_id = $1
ORDER BY binding.id
`, channelID)
	if err != nil {
		return nil, fmt.Errorf("读取渠道模型：%w", err)
	}
	defer rows.Close()
	items := make([]ChannelModelBinding, 0)
	for rows.Next() {
		var item ChannelModelBinding
		var nextCheckAt, lastCheckedAt sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.ChannelID, &item.UpstreamModelName, &item.CanonicalModelID,
			&item.Model, &item.MatchStatus, &item.HealthState,
			&item.ConsecutiveRequestFailures, &item.AutomaticCheckFailures,
			&nextCheckAt, &lastCheckedAt, &item.LastCheckLatencyMS,
			&item.LastStatusCode, &item.LastCheckError, &item.LastSyncedAt,
		); err != nil {
			return nil, fmt.Errorf("解析渠道模型：%w", err)
		}
		item.NextCheckAt = nullableTime(nextCheckAt)
		item.LastCheckedAt = nullableTime(lastCheckedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历渠道模型：%w", err)
	}
	return items, nil
}

func (s *SQLMerchantStore) SaveManualModelCheck(
	ctx context.Context,
	channelID, bindingID, merchantID int64,
	state ModelHealthState,
	class healthFailureClass,
	latencyMS int,
	statusCode int,
	summary string,
) (ChannelModelBinding, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return ChannelModelBinding{}, fmt.Errorf("开始保存模型检查事务：%w", err)
	}
	defer tx.Rollback()
	var ownerID int64
	if err := tx.QueryRowContext(ctx, `
SELECT channel.merchant_profile_id
FROM merchant_channel_model_bindings binding
JOIN merchant_channels channel ON channel.id = binding.merchant_channel_id
WHERE binding.id = $1 AND binding.merchant_channel_id = $2 AND channel.deleted_at IS NULL
FOR UPDATE OF binding
`, bindingID, channelID).Scan(&ownerID); err != nil {
		return ChannelModelBinding{}, merchantStoreError(err)
	}
	if merchantID > 0 && ownerID != merchantID {
		return ChannelModelBinding{}, ErrMerchantNotFound
	}
	var item ChannelModelBinding
	var nextCheckAt, lastCheckedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
UPDATE merchant_channel_model_bindings binding
SET health_state = $3, consecutive_request_failures = $4, automatic_check_failures = $5,
    next_check_at = $6, check_lease_until = NULL, last_checked_at = now(),
    last_check_latency_ms = NULLIF($7, 0), last_status_code = NULLIF($8, 0),
    last_check_error = NULLIF($9, ''), updated_at = now()
FROM canonical_models model
WHERE binding.id = $1 AND binding.merchant_channel_id = $2
  AND model.id = binding.canonical_model_id
RETURNING binding.id, binding.merchant_channel_id, binding.upstream_model_name,
          binding.canonical_model_id, model.public_name, binding.match_status,
          binding.health_state, binding.consecutive_request_failures, binding.automatic_check_failures,
          binding.next_check_at, binding.last_checked_at,
          COALESCE(binding.last_check_latency_ms, 0), COALESCE(binding.last_status_code, 0),
          COALESCE(binding.last_check_error, ''),
          binding.last_synced_at
`, bindingID, channelID, state.State, state.ConsecutiveRequestFailures,
		state.AutomaticCheckFailures, timeOrNil(state.NextCheckAt), latencyMS, statusCode, summary).Scan(
		&item.ID, &item.ChannelID, &item.UpstreamModelName, &item.CanonicalModelID,
		&item.Model, &item.MatchStatus, &item.HealthState,
		&item.ConsecutiveRequestFailures, &item.AutomaticCheckFailures,
		&nextCheckAt, &lastCheckedAt, &item.LastCheckLatencyMS,
		&item.LastStatusCode, &item.LastCheckError, &item.LastSyncedAt,
	)
	if err != nil {
		return ChannelModelBinding{}, merchantStoreError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO channel_model_health_events
    (channel_model_binding_id, source, outcome, status_code, first_byte_ms, failure_summary)
VALUES ($1, 'manual_check', $2, NULLIF($3, 0), NULLIF($4, 0), NULLIF($5, ''))
`, bindingID, healthEventOutcome(class), statusCode, latencyMS, summary); err != nil {
		return ChannelModelBinding{}, fmt.Errorf("记录模型检查事件：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return ChannelModelBinding{}, fmt.Errorf("提交模型检查事务：%w", err)
	}
	item.NextCheckAt = nullableTime(nextCheckAt)
	item.LastCheckedAt = nullableTime(lastCheckedAt)
	return item, nil
}

func healthEventOutcome(class healthFailureClass) string {
	switch class {
	case healthFailureNone:
		return "success"
	case healthFailureHard:
		return "hard_failure"
	default:
		return "temporary_failure"
	}
}

var _ SelfServiceChannelRepository = (*SQLMerchantStore)(nil)
var _ MultiModelMerchantRepository = (*SQLMerchantStore)(nil)
