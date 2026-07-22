package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrRoutingNotFound     = errors.New("路由对象不存在")
	ErrRoutingInvalidState = errors.New("路由对象状态无效")
)

type CreateOfferCommand struct {
	UserID            int
	MerchantChannelID int64
	CanonicalModelID  int64
	MultiplierBPS     int
	InternalAlias     string
}

type OfficialPrice struct {
	ID                         int64     `json:"id,omitempty"`
	Version                    int       `json:"version,omitempty"`
	BillingMode                string    `json:"billing_mode"`
	ModelRatioMicros           int64     `json:"model_ratio_micros,omitempty"`
	ModelPriceMicroUSD         int64     `json:"model_price_microusd,omitempty"`
	CompletionRatioMicros      int64     `json:"completion_ratio_micros"`
	CacheRatioMicros           int64     `json:"cache_ratio_micros"`
	CreateCacheRatioMicros     int64     `json:"create_cache_ratio_micros"`
	ImageRatioMicros           int64     `json:"image_ratio_micros"`
	AudioRatioMicros           int64     `json:"audio_ratio_micros"`
	AudioCompletionRatioMicros int64     `json:"audio_completion_ratio_micros"`
	PublishedAt                time.Time `json:"published_at,omitempty"`
}

type CreateCanonicalModelCommand struct {
	PublicName string
	Provider   string
	Protocol   string
	AdminID    int
	Price      OfficialPrice
}

type CanonicalModel struct {
	ID         int64         `json:"id"`
	PublicName string        `json:"public_name"`
	Provider   string        `json:"provider"`
	Protocol   string        `json:"protocol"`
	Status     string        `json:"status"`
	Price      OfficialPrice `json:"price"`
	CreatedAt  time.Time     `json:"created_at,omitempty"`
}

type RoutingRepository interface {
	CreateCanonicalModel(context.Context, CreateCanonicalModelCommand) (CanonicalModel, error)
	CreateOffer(context.Context, CreateOfferCommand) (OfferVersion, error)
	MarkOfferSyncFailed(context.Context, int64, string) error
	BindTokenAndDefaultPolicy(context.Context, int, int, string) (RoutingPolicy, error)
	UpdatePolicy(context.Context, int, int, RoutingPolicyInput) (RoutingPolicy, error)
	RevokeTokenBinding(context.Context, int, int) error
	ResolveRoute(context.Context, string, string) (RouteSelection, error)
	StartRouteDecision(context.Context, RouteDecision) error
	CompleteRouteDecision(context.Context, RouteDecisionResult) error
	RecordRouteObservation(context.Context, RouteObservation) error
	ListMarketOffers(context.Context, int, string) ([]MarketOffer, error)
	ChannelForHealthReset(context.Context, int64) (int, error)
	ResetChannelHealth(context.Context, int64) error
	PublishOfficialPrice(context.Context, int64, int, OfficialPrice) (OfficialPrice, error)
	ListCanonicalModels(context.Context) ([]CanonicalModel, error)
	ListMerchantOffers(context.Context, int) ([]MerchantOfferView, error)
}

type MarketOffer struct {
	OfferID           int64  `json:"offer_id"`
	MerchantID        int64  `json:"merchant_id"`
	MerchantName      string `json:"merchant_name"`
	MerchantChannelID int64  `json:"merchant_channel_id"`
	ChannelName       string `json:"channel_name"`
	Model             string `json:"model"`
	Protocol          string `json:"protocol"`
	MultiplierBPS     int    `json:"multiplier_bps"`
	HealthState       string `json:"health_state"`
	AvailabilityPPM   int    `json:"availability_ppm"`
	TTFTP95MS         int    `json:"ttft_p95_ms"`
	InternalAlias     string `json:"-"`
}

type MerchantOfferView struct {
	OfferID           int64  `json:"offer_id"`
	MerchantChannelID int64  `json:"merchant_channel_id"`
	ChannelName       string `json:"channel_name"`
	Model             string `json:"model"`
	Protocol          string `json:"protocol"`
	MultiplierBPS     int    `json:"multiplier_bps"`
	OfferStatus       string `json:"offer_status"`
	HealthState       string `json:"health_state"`
	AvailabilityPPM   int    `json:"availability_ppm"`
	TTFTP95MS         int    `json:"ttft_p95_ms"`
	SyncError         string `json:"sync_error,omitempty"`
	InternalAlias     string `json:"-"`
}

type RouteSelection struct {
	TokenBindingID   int64
	PolicyID         int64
	CanonicalModelID int64
	Mode             string
	Priority         string
	Protocol         string
	Candidates       []RouteCandidate
}

type RouteDecision struct {
	RequestID        string
	TokenBindingID   int64
	CanonicalModelID int64
	OfferID          int64
	Mode             string
	Priority         string
}

type RouteDecisionResult struct {
	RequestID   string
	OfferID     int64
	Attempts    int
	StatusCode  int
	FirstByteMS int
	Result      string
}

type RouteObservation struct {
	Candidate   RouteCandidate
	Outcome     string
	StatusCode  int
	FirstByteMS int
	RequestID   string
	Summary     string
}

type RoutingPolicyInput struct {
	Mode                 string  `json:"mode"`
	Priority             string  `json:"priority"`
	MinimumMultiplierBPS int     `json:"minimum_multiplier_bps"`
	MaximumMultiplierBPS int     `json:"maximum_multiplier_bps"`
	MerchantIDs          []int64 `json:"merchant_ids"`
}

type RoutingPolicy struct {
	ID                   int64     `json:"id"`
	NewAPITokenID        int       `json:"newapi_token_id"`
	Mode                 string    `json:"mode"`
	Priority             string    `json:"priority"`
	MinimumMultiplierBPS int       `json:"minimum_multiplier_bps"`
	MaximumMultiplierBPS int       `json:"maximum_multiplier_bps"`
	MerchantIDs          []int64   `json:"merchant_ids"`
	UpdatedAt            time.Time `json:"updated_at,omitempty"`
}

type OfferSynchronizer interface {
	SyncOffer(context.Context, int64, int64) error
}

type OfferVersion struct {
	ID                     int64     `json:"id"`
	MerchantChannelID      int64     `json:"merchant_channel_id"`
	CanonicalModelID       int64     `json:"canonical_model_id"`
	OfficialPriceVersionID int64     `json:"official_price_version_id"`
	PreviousOfferID        int64     `json:"previous_offer_id,omitempty"`
	MultiplierBPS          int       `json:"multiplier_bps"`
	InternalAlias          string    `json:"-"`
	Status                 string    `json:"status"`
	PublicModel            string    `json:"model"`
	Protocol               string    `json:"protocol"`
	NewAPIChannelID        int       `json:"newapi_channel_id"`
	CreatedAt              time.Time `json:"created_at,omitempty"`
}

type SQLRoutingStore struct {
	database *sql.DB
}

func NewSQLRoutingStore(database *sql.DB) *SQLRoutingStore {
	return &SQLRoutingStore{database: database}
}

func (s *SQLRoutingStore) CreateCanonicalModel(ctx context.Context, command CreateCanonicalModelCommand) (CanonicalModel, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return CanonicalModel{}, fmt.Errorf("开始标准模型事务：%w", err)
	}
	defer tx.Rollback()
	model := CanonicalModel{PublicName: command.PublicName, Provider: command.Provider, Protocol: command.Protocol, Status: "active", Price: command.Price}
	err = tx.QueryRowContext(ctx, `
INSERT INTO canonical_models (public_name, provider, protocol)
VALUES ($1, $2, $3)
RETURNING id, status, created_at
`, command.PublicName, command.Provider, command.Protocol).Scan(&model.ID, &model.Status, &model.CreatedAt)
	if err != nil {
		return CanonicalModel{}, fmt.Errorf("创建标准模型：%w", err)
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO official_price_versions
    (canonical_model_id, version, billing_mode, model_ratio_micros, model_price_microusd,
     completion_ratio_micros, cache_ratio_micros, create_cache_ratio_micros,
     image_ratio_micros, audio_ratio_micros, audio_completion_ratio_micros,
     published_by_newapi_user_id)
VALUES ($1, 1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, version
`, model.ID, command.Price.BillingMode, nullablePositive(command.Price.ModelRatioMicros), nullablePositive(command.Price.ModelPriceMicroUSD),
		command.Price.CompletionRatioMicros, command.Price.CacheRatioMicros, command.Price.CreateCacheRatioMicros,
		command.Price.ImageRatioMicros, command.Price.AudioRatioMicros, command.Price.AudioCompletionRatioMicros,
		command.AdminID).Scan(&model.Price.ID, &model.Price.Version)
	if err != nil {
		return CanonicalModel{}, fmt.Errorf("创建官方价格版本：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE canonical_models SET current_official_price_version_id = $1, updated_at = now() WHERE id = $2`, model.Price.ID, model.ID); err != nil {
		return CanonicalModel{}, fmt.Errorf("发布官方价格版本：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return CanonicalModel{}, fmt.Errorf("提交标准模型事务：%w", err)
	}
	return model, nil
}

func nullablePositive(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func (s *SQLRoutingStore) CreateOffer(ctx context.Context, command CreateOfferCommand) (OfferVersion, error) {
	if command.UserID <= 0 || command.MerchantChannelID <= 0 || command.CanonicalModelID <= 0 || command.MultiplierBPS < 1 || command.MultiplierBPS > 10_000 || command.InternalAlias == "" {
		return OfferVersion{}, ErrRoutingInvalidState
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return OfferVersion{}, fmt.Errorf("开始报价事务：%w", err)
	}
	defer tx.Rollback()
	var offer OfferVersion
	err = tx.QueryRowContext(ctx, `
WITH eligible AS (
    SELECT mc.id AS merchant_channel_id, mc.newapi_channel_id, cm.id AS canonical_model_id,
           cm.public_name, cm.protocol, cm.current_official_price_version_id,
           (SELECT id FROM offer_versions old
            WHERE old.merchant_channel_id = mc.id AND old.canonical_model_id = cm.id AND old.status = 'active'
            ORDER BY old.id DESC LIMIT 1) AS previous_offer_version_id
    FROM merchant_channels mc
    JOIN merchant_profiles mp ON mp.id = mc.merchant_profile_id
    JOIN canonical_models cm ON cm.id = $3
    WHERE mp.newapi_user_id = $1 AND mp.status = 'active'
      AND mc.id = $2 AND mc.status = 'active'
      AND cm.status = 'active' AND cm.current_official_price_version_id IS NOT NULL
      AND mc.models ? cm.public_name
), inserted AS (
    INSERT INTO offer_versions
        (merchant_channel_id, canonical_model_id, official_price_version_id,
         previous_offer_version_id, multiplier_bps, internal_alias, created_by_newapi_user_id)
    SELECT merchant_channel_id, canonical_model_id, current_official_price_version_id,
           previous_offer_version_id, $4, $5, $1
    FROM eligible
    RETURNING id, merchant_channel_id, canonical_model_id, official_price_version_id,
              previous_offer_version_id, multiplier_bps, internal_alias, status
)
SELECT i.id, i.merchant_channel_id, i.canonical_model_id, i.official_price_version_id,
       COALESCE(i.previous_offer_version_id, 0), i.multiplier_bps, i.internal_alias, i.status,
       e.public_name, e.protocol, e.newapi_channel_id
FROM inserted i
JOIN eligible e ON e.merchant_channel_id = i.merchant_channel_id AND e.canonical_model_id = i.canonical_model_id
`, command.UserID, command.MerchantChannelID, command.CanonicalModelID, command.MultiplierBPS, command.InternalAlias).Scan(
		&offer.ID, &offer.MerchantChannelID, &offer.CanonicalModelID, &offer.OfficialPriceVersionID,
		&offer.PreviousOfferID, &offer.MultiplierBPS, &offer.InternalAlias, &offer.Status,
		&offer.PublicModel, &offer.Protocol, &offer.NewAPIChannelID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return OfferVersion{}, ErrRoutingNotFound
	}
	if err != nil {
		return OfferVersion{}, fmt.Errorf("创建报价版本：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return OfferVersion{}, fmt.Errorf("提交报价事务：%w", err)
	}
	return offer, nil
}

func (s *SQLRoutingStore) ActivateOffer(ctx context.Context, offerID int64) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始报价激活事务：%w", err)
	}
	defer tx.Rollback()
	var previous sql.NullInt64
	var merchantChannelID, canonicalModelID int64
	err = tx.QueryRowContext(ctx, `
SELECT previous_offer_version_id, merchant_channel_id, canonical_model_id
FROM offer_versions
WHERE id = $1 AND status = 'syncing'
FOR UPDATE
`, offerID).Scan(&previous, &merchantChannelID, &canonicalModelID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRoutingInvalidState
	}
	if err != nil {
		return fmt.Errorf("锁定报价版本：%w", err)
	}
	if previous.Valid {
		if _, err := tx.ExecContext(ctx, `UPDATE offer_versions SET status = 'inactive', retired_at = now() WHERE id = $1 AND status = 'active'`, previous.Int64); err != nil {
			return fmt.Errorf("退役旧报价：%w", err)
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE offer_versions SET status = 'active', activated_at = now(), sync_error = NULL WHERE id = $1 AND status = 'syncing'`, offerID)
	if err != nil {
		return fmt.Errorf("激活新报价：%w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrRoutingInvalidState
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO channel_model_health (merchant_channel_id, canonical_model_id, state, next_probe_at)
VALUES ($1, $2, 'pending', now())
ON CONFLICT (merchant_channel_id, canonical_model_id) DO UPDATE
SET state = CASE WHEN channel_model_health.state = 'disabled' THEN 'disabled' ELSE 'pending' END,
    next_probe_at = CASE WHEN channel_model_health.state = 'disabled' THEN channel_model_health.next_probe_at ELSE now() END,
    updated_at = now()
`, merchantChannelID, canonicalModelID); err != nil {
		return fmt.Errorf("初始化报价健康状态：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交报价激活事务：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) MarkOfferSyncFailed(ctx context.Context, offerID int64, summary string) error {
	_, err := s.database.ExecContext(ctx, `
UPDATE offer_versions
SET status = 'sync_failed', sync_error = $2
WHERE id = $1 AND status = 'syncing'
`, offerID, summary)
	if err != nil {
		return fmt.Errorf("记录报价同步失败：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) AliasPrices(ctx context.Context) ([]aliasPrice, error) {
	rows, err := s.database.QueryContext(ctx, `
SELECT o.internal_alias, p.billing_mode, o.multiplier_bps,
       p.model_ratio_micros, p.model_price_microusd,
       p.completion_ratio_micros, p.cache_ratio_micros, p.create_cache_ratio_micros,
       p.image_ratio_micros, p.audio_ratio_micros, p.audio_completion_ratio_micros
FROM offer_versions o
JOIN official_price_versions p ON p.id = o.official_price_version_id
WHERE o.status <> 'sync_failed'
ORDER BY o.id
`)
	if err != nil {
		return nil, fmt.Errorf("查询别名价格：%w", err)
	}
	defer rows.Close()
	prices := make([]aliasPrice, 0)
	for rows.Next() {
		var price aliasPrice
		var modelRatio, modelPrice sql.NullInt64
		if err := rows.Scan(
			&price.Alias, &price.BillingMode, &price.MultiplierBPS,
			&modelRatio, &modelPrice, &price.CompletionRatioMicros,
			&price.CacheRatioMicros, &price.CreateCacheRatioMicros, &price.ImageRatioMicros,
			&price.AudioRatioMicros, &price.AudioCompletionRatioMicros,
		); err != nil {
			return nil, fmt.Errorf("解析别名价格：%w", err)
		}
		if modelRatio.Valid {
			price.ModelRatioMicros = modelRatio.Int64
		}
		if modelPrice.Valid {
			price.ModelPriceMicroUSD = modelPrice.Int64
		}
		prices = append(prices, price)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历别名价格：%w", err)
	}
	return prices, nil
}

func (s *SQLRoutingStore) ChannelPriceRoute(ctx context.Context, merchantChannelID int64) (ChannelPriceRoute, error) {
	var route ChannelPriceRoute
	var modelsJSON []byte
	err := s.database.QueryRowContext(ctx, `
SELECT newapi_channel_id, models FROM merchant_channels WHERE id = $1
`, merchantChannelID).Scan(&route.NewAPIChannelID, &modelsJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return ChannelPriceRoute{}, ErrRoutingNotFound
	}
	if err != nil {
		return ChannelPriceRoute{}, fmt.Errorf("查询渠道模型：%w", err)
	}
	if err := json.Unmarshal(modelsJSON, &route.PublicModels); err != nil {
		return ChannelPriceRoute{}, fmt.Errorf("解析渠道原始模型：%w", err)
	}
	route.Aliases = make(map[string]string)
	rows, err := s.database.QueryContext(ctx, `
SELECT o.internal_alias, cm.public_name
FROM offer_versions o
JOIN canonical_models cm ON cm.id = o.canonical_model_id
WHERE o.merchant_channel_id = $1 AND o.status <> 'sync_failed'
ORDER BY o.id
`, merchantChannelID)
	if err != nil {
		return ChannelPriceRoute{}, fmt.Errorf("查询渠道报价别名：%w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var alias, publicModel string
		if err := rows.Scan(&alias, &publicModel); err != nil {
			return ChannelPriceRoute{}, fmt.Errorf("解析渠道报价别名：%w", err)
		}
		route.Aliases[alias] = publicModel
	}
	if err := rows.Err(); err != nil {
		return ChannelPriceRoute{}, fmt.Errorf("遍历渠道报价别名：%w", err)
	}
	return route, nil
}

func (s *SQLRoutingStore) BindTokenAndDefaultPolicy(ctx context.Context, userID int, tokenID int, fingerprint string) (RoutingPolicy, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return RoutingPolicy{}, fmt.Errorf("开始令牌绑定事务：%w", err)
	}
	defer tx.Rollback()
	var bindingID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO token_bindings (newapi_token_id, newapi_user_id, token_fingerprint)
VALUES ($1, $2, $3)
ON CONFLICT (newapi_token_id) DO UPDATE
SET token_fingerprint = EXCLUDED.token_fingerprint, status = 'active', synced_at = now()
WHERE token_bindings.newapi_user_id = EXCLUDED.newapi_user_id
RETURNING id
`, tokenID, userID, fingerprint).Scan(&bindingID)
	if errors.Is(err, sql.ErrNoRows) {
		return RoutingPolicy{}, ErrRoutingNotFound
	}
	if err != nil {
		return RoutingPolicy{}, fmt.Errorf("绑定 New API 令牌：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO routing_policies (token_binding_id)
VALUES ($1)
ON CONFLICT (token_binding_id) DO NOTHING
`, bindingID); err != nil {
		return RoutingPolicy{}, fmt.Errorf("创建默认路由策略：%w", err)
	}
	policy, err := policyByBinding(ctx, tx, bindingID, tokenID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if err := tx.Commit(); err != nil {
		return RoutingPolicy{}, fmt.Errorf("提交令牌绑定事务：%w", err)
	}
	return policy, nil
}

func (s *SQLRoutingStore) UpdatePolicy(ctx context.Context, userID int, tokenID int, input RoutingPolicyInput) (RoutingPolicy, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return RoutingPolicy{}, fmt.Errorf("开始路由策略事务：%w", err)
	}
	defer tx.Rollback()
	var policyID, bindingID int64
	err = tx.QueryRowContext(ctx, `
SELECT p.id, b.id
FROM routing_policies p
JOIN token_bindings b ON b.id = p.token_binding_id
WHERE b.newapi_user_id = $1 AND b.newapi_token_id = $2 AND b.status = 'active'
FOR UPDATE
`, userID, tokenID).Scan(&policyID, &bindingID)
	if errors.Is(err, sql.ErrNoRows) {
		return RoutingPolicy{}, ErrRoutingNotFound
	}
	if err != nil {
		return RoutingPolicy{}, fmt.Errorf("锁定路由策略：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE routing_policies
SET mode = $2, priority = $3, minimum_multiplier_bps = $4,
    maximum_multiplier_bps = $5, updated_at = now()
WHERE id = $1
`, policyID, input.Mode, input.Priority, input.MinimumMultiplierBPS, input.MaximumMultiplierBPS); err != nil {
		return RoutingPolicy{}, fmt.Errorf("更新路由策略：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM routing_policy_merchants WHERE routing_policy_id = $1`, policyID); err != nil {
		return RoutingPolicy{}, fmt.Errorf("清理固定商户：%w", err)
	}
	for _, merchantID := range uniqueInt64(input.MerchantIDs) {
		result, err := tx.ExecContext(ctx, `
INSERT INTO routing_policy_merchants (routing_policy_id, merchant_profile_id)
SELECT $1, id FROM merchant_profiles WHERE id = $2 AND status = 'active'
`, policyID, merchantID)
		if err != nil {
			return RoutingPolicy{}, fmt.Errorf("保存固定商户：%w", err)
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return RoutingPolicy{}, ErrRoutingNotFound
		}
	}
	policy, err := policyByBinding(ctx, tx, bindingID, tokenID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if err := tx.Commit(); err != nil {
		return RoutingPolicy{}, fmt.Errorf("提交路由策略事务：%w", err)
	}
	return policy, nil
}

func (s *SQLRoutingStore) RevokeTokenBinding(ctx context.Context, userID int, tokenID int) error {
	_, err := s.database.ExecContext(ctx, `
UPDATE token_bindings SET status = 'revoked', synced_at = now()
WHERE newapi_user_id = $1 AND newapi_token_id = $2
`, userID, tokenID)
	if err != nil {
		return fmt.Errorf("撤销令牌路由绑定：%w", err)
	}
	return nil
}

type policyQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func policyByBinding(ctx context.Context, queryer policyQueryer, bindingID int64, tokenID int) (RoutingPolicy, error) {
	var policy RoutingPolicy
	var merchantIDsJSON []byte
	err := queryer.QueryRowContext(ctx, `
SELECT p.id, p.mode, p.priority, p.minimum_multiplier_bps, p.maximum_multiplier_bps,
       COALESCE(json_agg(pm.merchant_profile_id ORDER BY pm.merchant_profile_id)
           FILTER (WHERE pm.merchant_profile_id IS NOT NULL), '[]'::json),
       p.updated_at
FROM routing_policies p
LEFT JOIN routing_policy_merchants pm ON pm.routing_policy_id = p.id
WHERE p.token_binding_id = $1
GROUP BY p.id
`, bindingID).Scan(&policy.ID, &policy.Mode, &policy.Priority, &policy.MinimumMultiplierBPS, &policy.MaximumMultiplierBPS, &merchantIDsJSON, &policy.UpdatedAt)
	if err != nil {
		return RoutingPolicy{}, fmt.Errorf("读取路由策略：%w", err)
	}
	policy.NewAPITokenID = tokenID
	if err := json.Unmarshal(merchantIDsJSON, &policy.MerchantIDs); err != nil {
		return RoutingPolicy{}, fmt.Errorf("解析固定商户：%w", err)
	}
	return policy, nil
}

func uniqueInt64(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (s *SQLRoutingStore) ResolveRoute(ctx context.Context, fingerprint string, publicModel string) (RouteSelection, error) {
	var selection RouteSelection
	err := s.database.QueryRowContext(ctx, `
SELECT b.id, p.id, cm.id, p.mode, p.priority, cm.protocol
FROM token_bindings b
JOIN routing_policies p ON p.token_binding_id = b.id
JOIN canonical_models cm ON cm.public_name = $2 AND cm.status = 'active'
WHERE b.token_fingerprint = $1 AND b.status = 'active'
`, fingerprint, publicModel).Scan(&selection.TokenBindingID, &selection.PolicyID, &selection.CanonicalModelID, &selection.Mode, &selection.Priority, &selection.Protocol)
	if errors.Is(err, sql.ErrNoRows) {
		return RouteSelection{}, ErrRoutingNotFound
	}
	if err != nil {
		return RouteSelection{}, fmt.Errorf("读取路由策略：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, `
SELECT o.id, mp.id, mc.id, mc.newapi_channel_id, cm.id, o.internal_alias,
       cm.public_name, cm.protocol, o.multiplier_bps, h.state,
       h.availability_ppm, h.ttft_p95_ms
FROM routing_policies p
JOIN canonical_models cm ON cm.id = $2
JOIN offer_versions o ON o.canonical_model_id = cm.id AND o.status = 'active'
JOIN merchant_channels mc ON mc.id = o.merchant_channel_id AND mc.status = 'active'
JOIN merchant_profiles mp ON mp.id = mc.merchant_profile_id AND mp.status = 'active'
JOIN channel_model_health h ON h.merchant_channel_id = mc.id AND h.canonical_model_id = cm.id
WHERE p.id = $1
  AND o.multiplier_bps BETWEEN p.minimum_multiplier_bps AND p.maximum_multiplier_bps
  AND h.state IN ('healthy', 'degraded')
  AND (p.mode = 'smart' OR EXISTS (
      SELECT 1 FROM routing_policy_merchants allowed
      WHERE allowed.routing_policy_id = p.id AND allowed.merchant_profile_id = mp.id
  ))
`, selection.PolicyID, selection.CanonicalModelID)
	if err != nil {
		return RouteSelection{}, fmt.Errorf("查询路由候选：%w", err)
	}
	defer rows.Close()
	selection.Candidates = make([]RouteCandidate, 0)
	for rows.Next() {
		var candidate RouteCandidate
		if err := rows.Scan(
			&candidate.OfferID, &candidate.MerchantID, &candidate.MerchantChannelID,
			&candidate.NewAPIChannelID, &candidate.CanonicalModelID, &candidate.InternalAlias,
			&candidate.PublicModel, &candidate.Protocol, &candidate.MultiplierBPS,
			&candidate.HealthState, &candidate.AvailabilityPPM, &candidate.TTFTP95MS,
		); err != nil {
			return RouteSelection{}, fmt.Errorf("解析路由候选：%w", err)
		}
		selection.Candidates = append(selection.Candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return RouteSelection{}, fmt.Errorf("遍历路由候选：%w", err)
	}
	sortCandidates(selection.Candidates, selection.Priority)
	return selection, nil
}

func (s *SQLRoutingStore) StartRouteDecision(ctx context.Context, decision RouteDecision) error {
	_, err := s.database.ExecContext(ctx, `
INSERT INTO routing_decisions
    (request_id, token_binding_id, canonical_model_id, offer_version_id, mode, priority)
VALUES ($1, $2, $3, NULLIF($4, 0), $5, $6)
`, decision.RequestID, decision.TokenBindingID, decision.CanonicalModelID, decision.OfferID, decision.Mode, decision.Priority)
	if err != nil {
		return fmt.Errorf("创建路由决策：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) CompleteRouteDecision(ctx context.Context, result RouteDecisionResult) error {
	_, err := s.database.ExecContext(ctx, `
UPDATE routing_decisions
SET offer_version_id = NULLIF($2, 0), attempt_count = $3,
    upstream_status_code = NULLIF($4, 0), first_byte_ms = $5,
    result = $6, completed_at = now()
WHERE request_id = $1
`, result.RequestID, result.OfferID, result.Attempts, result.StatusCode, result.FirstByteMS, result.Result)
	if err != nil {
		return fmt.Errorf("完成路由决策：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) RecordRouteObservation(ctx context.Context, observation RouteObservation) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始健康观察事务：%w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO channel_health_events
    (merchant_channel_id, canonical_model_id, offer_version_id, source,
     outcome, status_code, first_byte_ms, request_id, failure_summary)
VALUES ($1, $2, $3, 'request', $4, NULLIF($5, 0), $6, $7, NULLIF($8, ''))
`, observation.Candidate.MerchantChannelID, observation.Candidate.CanonicalModelID,
		observation.Candidate.OfferID, observation.Outcome, observation.StatusCode,
		observation.FirstByteMS, observation.RequestID, observation.Summary); err != nil {
		return fmt.Errorf("记录健康事件：%w", err)
	}
	success := observation.Outcome == "success"
	temporary := observation.Outcome == "temporary_failure"
	if _, err := tx.ExecContext(ctx, `
UPDATE channel_model_health
SET observations = observations + CASE WHEN $3 OR $4 THEN 1 ELSE 0 END,
    successes = successes + CASE WHEN $3 THEN 1 ELSE 0 END,
    availability_ppm = CASE
        WHEN observations + CASE WHEN $3 OR $4 THEN 1 ELSE 0 END = 0 THEN availability_ppm
        ELSE ((successes + CASE WHEN $3 THEN 1 ELSE 0 END) * 1000000 /
             (observations + CASE WHEN $3 OR $4 THEN 1 ELSE 0 END))::integer END,
    consecutive_failures = CASE WHEN $3 THEN 0 WHEN $4 THEN consecutive_failures + 1 ELSE consecutive_failures END,
    state = CASE
        WHEN state = 'disabled' THEN state
        WHEN $3 THEN 'healthy'
        WHEN $4 AND consecutive_failures + 1 >= 3 THEN 'cooling'
        WHEN $4 THEN 'degraded'
        ELSE state END,
    cooldown_until = CASE WHEN $4 AND consecutive_failures + 1 >= 3 THEN now() + interval '1 minute' ELSE cooldown_until END,
    next_probe_at = CASE WHEN $4 AND consecutive_failures + 1 >= 3 THEN now() + interval '1 minute' ELSE next_probe_at END,
    last_success_at = CASE WHEN $3 THEN now() ELSE last_success_at END,
    last_failure_at = CASE WHEN $4 THEN now() ELSE last_failure_at END,
    last_failure_class = CASE WHEN $4 THEN 'temporary' ELSE last_failure_class END,
    last_status_code = NULLIF($5, 0), updated_at = now()
WHERE merchant_channel_id = $1 AND canonical_model_id = $2
`, observation.Candidate.MerchantChannelID, observation.Candidate.CanonicalModelID,
		success, temporary, observation.StatusCode); err != nil {
		return fmt.Errorf("更新渠道健康状态：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交健康观察事务：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) ClaimHealthProbes(ctx context.Context, now time.Time, limit int) ([]HealthProbe, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.database.QueryContext(ctx, `
WITH due AS (
    SELECT h.merchant_channel_id, h.canonical_model_id, o.id AS offer_id,
           mc.newapi_channel_id, o.internal_alias, cm.protocol, h.state,
           h.consecutive_failures, h.cooldown_level, h.recovery_successes,
           h.healthy_since
    FROM channel_model_health h
    JOIN merchant_channels mc ON mc.id = h.merchant_channel_id AND mc.status = 'active'
    JOIN canonical_models cm ON cm.id = h.canonical_model_id AND cm.status = 'active'
    JOIN LATERAL (
        SELECT id, internal_alias FROM offer_versions
        WHERE merchant_channel_id = h.merchant_channel_id
          AND canonical_model_id = h.canonical_model_id AND status = 'active'
        ORDER BY id DESC LIMIT 1
    ) o ON true
    WHERE h.state <> 'disabled' AND h.next_probe_at <= $1
    ORDER BY h.next_probe_at, h.merchant_channel_id, h.canonical_model_id
    LIMIT $2
    FOR UPDATE OF h SKIP LOCKED
), claimed AS (
    UPDATE channel_model_health h
    SET next_probe_at = $1 + interval '30 seconds', updated_at = now()
    FROM due
    WHERE h.merchant_channel_id = due.merchant_channel_id
      AND h.canonical_model_id = due.canonical_model_id
    RETURNING h.merchant_channel_id, h.canonical_model_id
)
SELECT d.offer_id, d.merchant_channel_id, d.canonical_model_id, d.newapi_channel_id,
       d.internal_alias, d.protocol, d.state, d.consecutive_failures,
       d.cooldown_level, d.recovery_successes, d.healthy_since
FROM due d
JOIN claimed c USING (merchant_channel_id, canonical_model_id)
`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("领取探活任务：%w", err)
	}
	defer rows.Close()
	probes := make([]HealthProbe, 0)
	for rows.Next() {
		var probe HealthProbe
		var healthySince sql.NullTime
		if err := rows.Scan(
			&probe.OfferID, &probe.MerchantChannelID, &probe.CanonicalModelID,
			&probe.NewAPIChannelID, &probe.InternalAlias, &probe.Protocol, &probe.State,
			&probe.ConsecutiveFailures, &probe.CooldownLevel, &probe.RecoverySuccesses,
			&healthySince,
		); err != nil {
			return nil, fmt.Errorf("解析探活任务：%w", err)
		}
		if healthySince.Valid {
			probe.HealthySince = healthySince.Time
		}
		probes = append(probes, probe)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历探活任务：%w", err)
	}
	return probes, nil
}

func (s *SQLRoutingStore) RegisterProbeToken(ctx context.Context, tokenID int) error {
	_, err := s.database.ExecContext(ctx, `
INSERT INTO platform_runtime_metadata (key, value, updated_at)
VALUES ('newapi_probe_token_id', $1, now())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
`, fmt.Sprintf("%d", tokenID))
	if err != nil {
		return fmt.Errorf("登记 New API 探活令牌：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) SaveProbeResult(ctx context.Context, result HealthProbeResult) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始探活结果事务：%w", err)
	}
	defer tx.Rollback()
	outcome := "ignored"
	switch result.Class {
	case healthFailureNone:
		outcome = "success"
	case healthFailureTemporary:
		outcome = "temporary_failure"
	case healthFailureHard:
		outcome = "hard_failure"
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO channel_health_events
    (merchant_channel_id, canonical_model_id, offer_version_id, source,
     outcome, status_code, first_byte_ms, failure_summary)
VALUES ($1, $2, $3, 'probe', $4, NULLIF($5, 0), $6, NULLIF($7, ''))
`, result.Probe.MerchantChannelID, result.Probe.CanonicalModelID, result.Probe.OfferID,
		outcome, result.StatusCode, result.FirstByteMS, result.Summary); err != nil {
		return fmt.Errorf("记录探活事件：%w", err)
	}
	validObservation := result.Class != healthFailureIgnored
	success := result.Class == healthFailureNone
	if _, err := tx.ExecContext(ctx, `
UPDATE channel_model_health
SET state = $3, consecutive_failures = $4, cooldown_level = $5,
    recovery_successes = $6, next_probe_at = COALESCE($7, next_probe_at),
    cooldown_until = CASE WHEN $3 = 'cooling' THEN $7 ELSE NULL END,
    observations = observations + CASE WHEN $8 THEN 1 ELSE 0 END,
    successes = successes + CASE WHEN $9 THEN 1 ELSE 0 END,
    availability_ppm = CASE
        WHEN observations + CASE WHEN $8 THEN 1 ELSE 0 END = 0 THEN availability_ppm
        ELSE ((successes + CASE WHEN $9 THEN 1 ELSE 0 END) * 1000000 /
             (observations + CASE WHEN $8 THEN 1 ELSE 0 END))::integer END,
    last_success_at = CASE WHEN $9 THEN now() ELSE last_success_at END,
    last_failure_at = CASE WHEN $10 IN ('temporary', 'hard') THEN now() ELSE last_failure_at END,
    last_failure_class = CASE WHEN $10 IN ('temporary', 'hard') THEN $10 ELSE last_failure_class END,
    last_status_code = NULLIF($11, 0), healthy_since = NULLIF($12, '0001-01-01 00:00:00+00'::timestamptz),
    updated_at = now()
WHERE merchant_channel_id = $1 AND canonical_model_id = $2
`, result.Probe.MerchantChannelID, result.Probe.CanonicalModelID, result.State.State,
		result.State.ConsecutiveFailures, result.State.CooldownLevel, result.State.RecoverySuccesses,
		timeOrNil(result.State.NextProbeAt), validObservation, success, string(result.Class),
		result.StatusCode, result.State.HealthySince); err != nil {
		return fmt.Errorf("更新探活健康状态：%w", err)
	}
	if result.Class == healthFailureHard {
		if _, err := tx.ExecContext(ctx, `UPDATE merchant_channels SET status = 'disabled', updated_at = now() WHERE id = $1`, result.Probe.MerchantChannelID); err != nil {
			return fmt.Errorf("停用硬故障渠道：%w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交探活结果事务：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) ListMarketOffers(ctx context.Context, _ int, publicModel string) ([]MarketOffer, error) {
	rows, err := s.database.QueryContext(ctx, `
SELECT o.id, mp.id, mp.display_name, mc.id, mc.channel_name,
       cm.public_name, cm.protocol, o.multiplier_bps, h.state,
       h.availability_ppm, h.ttft_p95_ms, o.internal_alias
FROM offer_versions o
JOIN canonical_models cm ON cm.id = o.canonical_model_id
JOIN merchant_channels mc ON mc.id = o.merchant_channel_id
JOIN merchant_profiles mp ON mp.id = mc.merchant_profile_id
JOIN channel_model_health h ON h.merchant_channel_id = mc.id AND h.canonical_model_id = cm.id
WHERE o.status = 'active' AND mc.status = 'active' AND mp.status = 'active'
  AND ($1 = '' OR cm.public_name = $1)
ORDER BY cm.public_name, o.multiplier_bps, o.id
`, strings.TrimSpace(publicModel))
	if err != nil {
		return nil, fmt.Errorf("查询市场报价：%w", err)
	}
	defer rows.Close()
	offers := make([]MarketOffer, 0)
	for rows.Next() {
		var offer MarketOffer
		if err := rows.Scan(
			&offer.OfferID, &offer.MerchantID, &offer.MerchantName,
			&offer.MerchantChannelID, &offer.ChannelName, &offer.Model, &offer.Protocol,
			&offer.MultiplierBPS, &offer.HealthState, &offer.AvailabilityPPM,
			&offer.TTFTP95MS, &offer.InternalAlias,
		); err != nil {
			return nil, fmt.Errorf("解析市场报价：%w", err)
		}
		offers = append(offers, offer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历市场报价：%w", err)
	}
	return offers, nil
}

func (s *SQLRoutingStore) ChannelForHealthReset(ctx context.Context, channelID int64) (int, error) {
	var newAPIChannelID int
	err := s.database.QueryRowContext(ctx, `SELECT newapi_channel_id FROM merchant_channels WHERE id = $1`, channelID).Scan(&newAPIChannelID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrRoutingNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("读取待恢复渠道：%w", err)
	}
	return newAPIChannelID, nil
}

func (s *SQLRoutingStore) ResetChannelHealth(ctx context.Context, channelID int64) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始渠道恢复事务：%w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE merchant_channels SET status = 'active', updated_at = now() WHERE id = $1`, channelID); err != nil {
		return fmt.Errorf("恢复平台渠道：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE channel_model_health
SET state = 'pending', consecutive_failures = 0, recovery_successes = 0,
    next_probe_at = now(), cooldown_until = NULL, updated_at = now()
WHERE merchant_channel_id = $1
`, channelID); err != nil {
		return fmt.Errorf("重置渠道健康状态：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交渠道恢复事务：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) PublishOfficialPrice(ctx context.Context, modelID int64, adminID int, price OfficialPrice) (OfficialPrice, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return OfficialPrice{}, fmt.Errorf("开始官方价格事务：%w", err)
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(max(version), 0) + 1 FROM official_price_versions WHERE canonical_model_id = $1
`, modelID).Scan(&price.Version); err != nil {
		return OfficialPrice{}, fmt.Errorf("生成官方价格版本：%w", err)
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO official_price_versions
    (canonical_model_id, version, billing_mode, model_ratio_micros, model_price_microusd,
     completion_ratio_micros, cache_ratio_micros, create_cache_ratio_micros,
     image_ratio_micros, audio_ratio_micros, audio_completion_ratio_micros,
     published_by_newapi_user_id)
SELECT id, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
FROM canonical_models WHERE id = $1
RETURNING id, published_at
`, modelID, price.Version, price.BillingMode, nullablePositive(price.ModelRatioMicros), nullablePositive(price.ModelPriceMicroUSD),
		price.CompletionRatioMicros, price.CacheRatioMicros, price.CreateCacheRatioMicros,
		price.ImageRatioMicros, price.AudioRatioMicros, price.AudioCompletionRatioMicros,
		adminID).Scan(&price.ID, &price.PublishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return OfficialPrice{}, ErrRoutingNotFound
	}
	if err != nil {
		return OfficialPrice{}, fmt.Errorf("创建官方价格版本：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE canonical_models SET current_official_price_version_id = $1, updated_at = now() WHERE id = $2`, price.ID, modelID); err != nil {
		return OfficialPrice{}, fmt.Errorf("发布官方价格版本：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE offer_versions SET status = 'inactive', retired_at = now() WHERE canonical_model_id = $1 AND status = 'active'`, modelID); err != nil {
		return OfficialPrice{}, fmt.Errorf("停止旧官方价格报价：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return OfficialPrice{}, fmt.Errorf("提交官方价格事务：%w", err)
	}
	return price, nil
}

func (s *SQLRoutingStore) ListCanonicalModels(ctx context.Context) ([]CanonicalModel, error) {
	rows, err := s.database.QueryContext(ctx, `
SELECT cm.id, cm.public_name, cm.provider, cm.protocol, cm.status, cm.created_at,
       p.id, p.version, p.billing_mode, p.model_ratio_micros, p.model_price_microusd,
       p.completion_ratio_micros, p.cache_ratio_micros, p.create_cache_ratio_micros,
       p.image_ratio_micros, p.audio_ratio_micros, p.audio_completion_ratio_micros,
       p.published_at
FROM canonical_models cm
JOIN official_price_versions p ON p.id = cm.current_official_price_version_id
ORDER BY cm.public_name
`)
	if err != nil {
		return nil, fmt.Errorf("查询标准模型：%w", err)
	}
	defer rows.Close()
	models := make([]CanonicalModel, 0)
	for rows.Next() {
		var model CanonicalModel
		var ratio, fixed sql.NullInt64
		if err := rows.Scan(
			&model.ID, &model.PublicName, &model.Provider, &model.Protocol, &model.Status, &model.CreatedAt,
			&model.Price.ID, &model.Price.Version, &model.Price.BillingMode, &ratio, &fixed,
			&model.Price.CompletionRatioMicros, &model.Price.CacheRatioMicros, &model.Price.CreateCacheRatioMicros,
			&model.Price.ImageRatioMicros, &model.Price.AudioRatioMicros, &model.Price.AudioCompletionRatioMicros,
			&model.Price.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("解析标准模型：%w", err)
		}
		if ratio.Valid {
			model.Price.ModelRatioMicros = ratio.Int64
		}
		if fixed.Valid {
			model.Price.ModelPriceMicroUSD = fixed.Int64
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历标准模型：%w", err)
	}
	return models, nil
}

func (s *SQLRoutingStore) ListMerchantOffers(ctx context.Context, userID int) ([]MerchantOfferView, error) {
	rows, err := s.database.QueryContext(ctx, `
SELECT o.id, mc.id, mc.channel_name, cm.public_name, cm.protocol,
       o.multiplier_bps, o.status, COALESCE(h.state, 'pending'),
       COALESCE(h.availability_ppm, 0), COALESCE(h.ttft_p95_ms, 0),
       COALESCE(o.sync_error, ''), o.internal_alias
FROM offer_versions o
JOIN merchant_channels mc ON mc.id = o.merchant_channel_id
JOIN merchant_profiles mp ON mp.id = mc.merchant_profile_id
JOIN canonical_models cm ON cm.id = o.canonical_model_id
LEFT JOIN channel_model_health h ON h.merchant_channel_id = mc.id AND h.canonical_model_id = cm.id
WHERE mp.newapi_user_id = $1
ORDER BY o.id DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询商户报价：%w", err)
	}
	defer rows.Close()
	offers := make([]MerchantOfferView, 0)
	for rows.Next() {
		var offer MerchantOfferView
		if err := rows.Scan(
			&offer.OfferID, &offer.MerchantChannelID, &offer.ChannelName, &offer.Model,
			&offer.Protocol, &offer.MultiplierBPS, &offer.OfferStatus, &offer.HealthState,
			&offer.AvailabilityPPM, &offer.TTFTP95MS, &offer.SyncError, &offer.InternalAlias,
		); err != nil {
			return nil, fmt.Errorf("解析商户报价：%w", err)
		}
		offers = append(offers, offer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历商户报价：%w", err)
	}
	return offers, nil
}

func timeOrNil(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
