package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrRoutingNotFound     = errors.New("路由对象不存在")
	ErrRoutingInvalidState = errors.New("路由对象状态无效")
)

type CreateOfferCommand struct {
	UserID           int
	CanonicalModelID int64
	MultiplierBPS    int
}

type OfficialPrice struct {
	ID          int64            `json:"id,omitempty"`
	Version     int              `json:"version,omitempty"`
	PriceKind   string           `json:"price_kind"`
	Tiers       []PriceTier      `json:"tiers,omitempty"`
	ImagePrices []ImagePriceItem `json:"image_prices,omitempty"`
	PublishedAt time.Time        `json:"published_at,omitempty"`
}

type CreateCanonicalModelCommand struct {
	PublicName string
	Provider   string
	AdminID    int
	Price      OfficialPrice
}

type CanonicalModel struct {
	ID          int64         `json:"id"`
	PublicName  string        `json:"public_name"`
	Provider    string        `json:"provider"`
	Status      string        `json:"status"`
	Price       OfficialPrice `json:"price"`
	CreatedAt   time.Time     `json:"created_at,omitempty"`
}

type RoutingRepository interface {
	CreateCanonicalModel(context.Context, CreateCanonicalModelCommand) (CanonicalModel, error)
	CreateOffer(context.Context, CreateOfferCommand) (OfferVersion, error)
	RetireOffer(context.Context, int, int64) error
	MarkOfferSyncFailed(context.Context, int64, string) error
	CreateRoutingPolicy(context.Context, int, RoutingPolicyInput) (RoutingPolicy, error)
	ListRoutingPolicies(context.Context, int) ([]RoutingPolicy, error)
	RoutingPolicy(context.Context, int, int64) (RoutingPolicy, error)
	UpdateRoutingPolicy(context.Context, int, int64, RoutingPolicyInput) (RoutingPolicy, error)
	DeleteRoutingPolicy(context.Context, int, int64) error
	BindTokenToPolicy(context.Context, int, int, string, int64) (TokenRoutingBinding, error)
	RebindTokenPolicy(context.Context, int, int, int64) (TokenRoutingBinding, error)
	ListTokenRoutingBindings(context.Context, int) ([]TokenRoutingBinding, error)
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
	DeleteCanonicalModel(context.Context, int64) (string, error)
	ImagePriceItems(context.Context, int64) ([]ImagePriceItem, error)
}

type MerchantOfferState struct {
	MultiplierBPS int    `json:"multiplier_bps"`
	Status        string `json:"status"`
}

type OfferLifecycleRepository interface {
	MerchantOfferState(context.Context, int, int64) (MerchantOfferState, error)
	DeleteMerchantOffer(context.Context, int, int64) error
}

type MarketOffer struct {
	OfferID                int64            `json:"offer_id"`
	MerchantID             int64            `json:"merchant_id"`
	MerchantName           string           `json:"merchant_name"`
	Model                  string           `json:"model"`
	MultiplierBPS          int              `json:"multiplier_bps"`
	PriceTiers             []OfferPriceTier `json:"price_tiers"`
	ImagePrices            []ImagePriceItem `json:"image_prices,omitempty"`
	ChannelCount           int              `json:"channel_count"`
	HealthState            string           `json:"health_state"`
	AvailabilityPPM        int              `json:"availability_ppm"`
	TTFTP95MS              int              `json:"ttft_p95_ms"`
	CallCount              int64            `json:"call_count"`
	TokenUsage             int64            `json:"token_usage"`
	LastCalledAt           *time.Time       `json:"last_called_at,omitempty"`
	CacheHitPPM            int              `json:"cache_hit_ppm"`
	CacheSampleInputTokens int64            `json:"cache_sample_input_tokens"`
	CacheWeightReady       bool             `json:"cache_weight_ready"`
	MetricsUpdatedAt       *time.Time       `json:"metrics_updated_at,omitempty"`
	InternalAlias          string           `json:"-"`
	OfficialPrice          OfficialPrice    `json:"-"`
}

type MerchantOfferView struct {
	OfferID                int64              `json:"offer_id"`
	Model                  string             `json:"model"`
	MultiplierBPS          int                `json:"multiplier_bps"`
	ChannelCount           int                `json:"channel_count"`
	CallCount              int64              `json:"call_count"`
	TokenUsage             int64              `json:"token_usage"`
	LastCalledAt           *time.Time         `json:"last_called_at,omitempty"`
	PriceTiers             []OfferPriceTier   `json:"price_tiers"`
	ImagePrices            []ImagePriceItem   `json:"image_prices,omitempty"`
	Channels               []OfferChannelView `json:"channels"`
	OfferStatus            string             `json:"offer_status"`
	HealthState            string             `json:"health_state"`
	AvailabilityPPM        int                `json:"availability_ppm"`
	TTFTP95MS              int                `json:"ttft_p95_ms"`
	CacheHitPPM            int                `json:"cache_hit_ppm"`
	CacheSampleInputTokens int64              `json:"cache_sample_input_tokens"`
	CacheWeightReady       bool               `json:"cache_weight_ready"`
	MetricsUpdatedAt       *time.Time         `json:"metrics_updated_at,omitempty"`
	SyncError              string             `json:"sync_error,omitempty"`
	InternalAlias          string             `json:"-"`
	OfficialPrice          OfficialPrice      `json:"-"`
}

type OfferChannelView struct {
	MerchantChannelID      int64      `json:"merchant_channel_id"`
	ChannelName            string     `json:"channel_name"`
	HealthState            string     `json:"health_state"`
	AvailabilityPPM        int        `json:"availability_ppm"`
	TTFTP95MS              int        `json:"ttft_p95_ms"`
	CacheHitPPM            int        `json:"cache_hit_ppm"`
	CacheSampleInputTokens int64      `json:"cache_sample_input_tokens"`
	CacheWeightReady       bool       `json:"cache_weight_ready"`
	MetricsUpdatedAt       *time.Time `json:"metrics_updated_at,omitempty"`
}

type RouteSelection struct {
	TokenBindingID     int64
	PolicyID           int64
	CanonicalModelID   int64
	Priority           string
	MinimumCacheHitPPM *int
	PriceKind          string
	Candidates         []RouteCandidate
}

type RouteDecision struct {
	RequestID           string
	TokenBindingID      int64
	PolicyID            int64
	CanonicalModelID    int64
	OfferChannelAliasID int64
	Priority            string
	AlgorithmVersion    string
	CandidateCount      int
	SelectedMetrics     map[string]any
}

type RouteDecisionResult struct {
	RequestID           string
	OfferChannelAliasID int64
	Attempts            int
	StatusCode          int
	FirstByteMS         int
	Result              string
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
	Name                 string  `json:"name"`
	Priority             string  `json:"priority"`
	MinimumMultiplierBPS int     `json:"minimum_multiplier_bps"`
	MaximumMultiplierBPS int     `json:"maximum_multiplier_bps"`
	MinimumCacheHitPPM   *int    `json:"minimum_cache_hit_ppm"`
}

type RoutingPolicy struct {
	ID                   int64     `json:"id"`
	Name                 string    `json:"name"`
	UserID               int       `json:"user_id,omitempty"`
	Priority             string    `json:"priority"`
	MinimumMultiplierBPS int       `json:"minimum_multiplier_bps"`
	MaximumMultiplierBPS int       `json:"maximum_multiplier_bps"`
	MinimumCacheHitPPM   *int      `json:"minimum_cache_hit_ppm"`
	BoundKeyCount        int       `json:"bound_key_count"`
	UpdatedAt            time.Time `json:"updated_at,omitempty"`
}

type TokenRoutingBinding struct {
	TokenID    int    `json:"token_id"`
	PolicyID   int64  `json:"routing_policy_id"`
	PolicyName string `json:"routing_policy_name"`
}

type OfferSynchronizer interface {
	SyncOffer(context.Context, int64, bool) error
}

type OfferVersion struct {
	ID                     int64               `json:"id"`
	CanonicalModelID       int64               `json:"canonical_model_id"`
	OfficialPriceVersionID int64               `json:"official_price_version_id"`
	PreviousOfferID        int64               `json:"previous_offer_id,omitempty"`
	MultiplierBPS          int                 `json:"multiplier_bps"`
	Status                 string              `json:"status"`
	PublicModel            string              `json:"model"`
	PriceKind              string              `json:"-"`
	CreatedAt              time.Time           `json:"created_at,omitempty"`
	MerchantID             int64               `json:"merchant_id"`
	Version                int                 `json:"version"`
	Aliases                []OfferChannelAlias `json:"channels"`
}

type OfferChannelAlias struct {
	ID                int64  `json:"id"`
	MerchantChannelID int64  `json:"merchant_channel_id"`
	NewAPIChannelID   int    `json:"newapi_channel_id"`
	ChannelName       string `json:"channel_name"`
	InternalAlias     string `json:"-"`
}

type SQLRoutingStore struct {
	database *sql.DB
}

func NewSQLRoutingStore(database *sql.DB) *SQLRoutingStore {
	return &SQLRoutingStore{database: database}
}

func (s *SQLRoutingStore) RunCycle(ctx context.Context, now time.Time) error {
	return s.RefreshRouteMetricSnapshots(ctx, now)
}

func (s *SQLRoutingStore) CreateRoutingPolicy(ctx context.Context, userID int, input RoutingPolicyInput) (RoutingPolicy, error) {
	input.Name = strings.TrimSpace(input.Name)
	if userID <= 0 || validateRoutingPolicy(input) != nil {
		return RoutingPolicy{}, ErrRoutingInvalidState
	}
	policy := RoutingPolicy{
		Name: input.Name, UserID: userID, Priority: input.Priority,
		MinimumMultiplierBPS: input.MinimumMultiplierBPS, MaximumMultiplierBPS: input.MaximumMultiplierBPS,
		MinimumCacheHitPPM: input.MinimumCacheHitPPM,
	}
	err := s.database.QueryRowContext(ctx, `
INSERT INTO routing_policies
    (newapi_user_id, name, priority, minimum_multiplier_bps,
     maximum_multiplier_bps, minimum_cache_hit_ppm)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, updated_at
`, userID, input.Name, input.Priority, input.MinimumMultiplierBPS,
		input.MaximumMultiplierBPS, input.MinimumCacheHitPPM).Scan(&policy.ID, &policy.UpdatedAt)
	if err != nil {
		return RoutingPolicy{}, routingStoreError(err)
	}
	return policy, nil
}

func (s *SQLRoutingStore) RoutingPolicy(ctx context.Context, userID int, policyID int64) (RoutingPolicy, error) {
	if userID <= 0 || policyID <= 0 {
		return RoutingPolicy{}, ErrRoutingInvalidState
	}
	return scanRoutingPolicy(s.database.QueryRowContext(ctx, routingPolicySelect+`
WHERE policy.newapi_user_id = $1 AND policy.id = $2
`, userID, policyID))
}

func (s *SQLRoutingStore) ListRoutingPolicies(ctx context.Context, userID int) ([]RoutingPolicy, error) {
	if userID <= 0 {
		return nil, ErrRoutingInvalidState
	}
	rows, err := s.database.QueryContext(ctx, routingPolicySelect+`
WHERE policy.newapi_user_id = $1
ORDER BY policy.updated_at DESC, policy.id DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询路由列表：%w", err)
	}
	defer rows.Close()
	items := make([]RoutingPolicy, 0)
	for rows.Next() {
		policy, err := scanRoutingPolicy(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历路由列表：%w", err)
	}
	return items, nil
}

func (s *SQLRoutingStore) UpdateRoutingPolicy(ctx context.Context, userID int, policyID int64, input RoutingPolicyInput) (RoutingPolicy, error) {
	input.Name = strings.TrimSpace(input.Name)
	if userID <= 0 || policyID <= 0 || validateRoutingPolicy(input) != nil {
		return RoutingPolicy{}, ErrRoutingInvalidState
	}
	policy := RoutingPolicy{
		ID: policyID, UserID: userID, Name: input.Name, Priority: input.Priority,
		MinimumMultiplierBPS: input.MinimumMultiplierBPS, MaximumMultiplierBPS: input.MaximumMultiplierBPS,
		MinimumCacheHitPPM: input.MinimumCacheHitPPM,
	}
	err := s.database.QueryRowContext(ctx, `
UPDATE routing_policies
SET name = $1, priority = $2,
	minimum_multiplier_bps = $3, maximum_multiplier_bps = $4,
	minimum_cache_hit_ppm = $5, updated_at = now()
WHERE newapi_user_id = $6 AND id = $7
RETURNING id,
    (SELECT count(*) FROM token_bindings binding
     WHERE binding.routing_policy_id = routing_policies.id AND binding.status = 'active'),
    updated_at
`, input.Name, input.Priority, input.MinimumMultiplierBPS,
		input.MaximumMultiplierBPS, input.MinimumCacheHitPPM, userID, policyID).
		Scan(&policy.ID, &policy.BoundKeyCount, &policy.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return RoutingPolicy{}, ErrRoutingNotFound
	}
	if err != nil {
		return RoutingPolicy{}, routingStoreError(err)
	}
	return policy, nil
}

const routingPolicySelect = `
SELECT policy.id, policy.newapi_user_id, policy.name, policy.priority,
       policy.minimum_multiplier_bps, policy.maximum_multiplier_bps,
       policy.minimum_cache_hit_ppm,
       (SELECT count(*) FROM token_bindings binding
        WHERE binding.routing_policy_id = policy.id AND binding.status = 'active'),
       policy.updated_at
FROM routing_policies policy
`

func scanRoutingPolicy(scanner rowScanner) (RoutingPolicy, error) {
	var policy RoutingPolicy
	var cacheFloor sql.NullInt64
	err := scanner.Scan(
		&policy.ID, &policy.UserID, &policy.Name, &policy.Priority,
		&policy.MinimumMultiplierBPS, &policy.MaximumMultiplierBPS, &cacheFloor,
		&policy.BoundKeyCount, &policy.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return RoutingPolicy{}, ErrRoutingNotFound
	}
	if err != nil {
		return RoutingPolicy{}, fmt.Errorf("读取路由：%w", err)
	}
	if cacheFloor.Valid {
		value := int(cacheFloor.Int64)
		policy.MinimumCacheHitPPM = &value
	}
	return policy, nil
}

func (s *SQLRoutingStore) BindTokenToPolicy(ctx context.Context, userID int, tokenID int, fingerprint string, policyID int64) (TokenRoutingBinding, error) {
	var binding TokenRoutingBinding
	err := s.database.QueryRowContext(ctx, `
INSERT INTO token_bindings
    (newapi_token_id, newapi_user_id, token_fingerprint, routing_policy_id)
SELECT $1, $2, $3, policy.id
FROM routing_policies policy
WHERE policy.id = $4 AND policy.newapi_user_id = $2
ON CONFLICT (newapi_token_id) DO UPDATE
SET token_fingerprint = EXCLUDED.token_fingerprint,
    routing_policy_id = EXCLUDED.routing_policy_id,
    status = 'active', synced_at = now()
WHERE token_bindings.newapi_user_id = EXCLUDED.newapi_user_id
RETURNING newapi_token_id, routing_policy_id,
    (SELECT name FROM routing_policies WHERE id = routing_policy_id)
`, tokenID, userID, fingerprint, policyID).Scan(&binding.TokenID, &binding.PolicyID, &binding.PolicyName)
	if errors.Is(err, sql.ErrNoRows) {
		return TokenRoutingBinding{}, ErrRoutingNotFound
	}
	if err != nil {
		return TokenRoutingBinding{}, routingStoreError(err)
	}
	return binding, nil
}

func (s *SQLRoutingStore) RebindTokenPolicy(ctx context.Context, userID int, tokenID int, policyID int64) (TokenRoutingBinding, error) {
	var binding TokenRoutingBinding
	err := s.database.QueryRowContext(ctx, `
UPDATE token_bindings binding
SET routing_policy_id = policy.id, synced_at = now()
FROM routing_policies policy
WHERE binding.newapi_user_id = $1 AND binding.newapi_token_id = $2
  AND binding.status = 'active'
  AND policy.id = $3 AND policy.newapi_user_id = $1
RETURNING binding.newapi_token_id, binding.routing_policy_id, policy.name
`, userID, tokenID, policyID).Scan(&binding.TokenID, &binding.PolicyID, &binding.PolicyName)
	if errors.Is(err, sql.ErrNoRows) {
		return TokenRoutingBinding{}, ErrRoutingNotFound
	}
	if err != nil {
		return TokenRoutingBinding{}, routingStoreError(err)
	}
	return binding, nil
}

func (s *SQLRoutingStore) ListTokenRoutingBindings(ctx context.Context, userID int) ([]TokenRoutingBinding, error) {
	rows, err := s.database.QueryContext(ctx, `
SELECT binding.newapi_token_id, policy.id, policy.name
FROM token_bindings binding
JOIN routing_policies policy ON policy.id = binding.routing_policy_id
WHERE binding.newapi_user_id = $1 AND binding.status = 'active'
ORDER BY binding.newapi_token_id DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询密钥路由绑定：%w", err)
	}
	defer rows.Close()
	items := make([]TokenRoutingBinding, 0)
	for rows.Next() {
		var item TokenRoutingBinding
		if err := rows.Scan(&item.TokenID, &item.PolicyID, &item.PolicyName); err != nil {
			return nil, fmt.Errorf("解析密钥路由绑定：%w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历密钥路由绑定：%w", err)
	}
	return items, nil
}

func (s *SQLRoutingStore) DeleteRoutingPolicy(ctx context.Context, userID int, policyID int64) error {
	var boundKeyCount int
	var deletedID sql.NullInt64
	err := s.database.QueryRowContext(ctx, `
WITH target AS (
    SELECT policy.id,
           count(binding.id) FILTER (WHERE binding.status = 'active') AS bound_key_count
    FROM routing_policies policy
    LEFT JOIN token_bindings binding ON binding.routing_policy_id = policy.id
    WHERE policy.newapi_user_id = $1 AND policy.id = $2
    GROUP BY policy.id
), deleted AS (
    DELETE FROM routing_policies policy
    USING target
    WHERE policy.id = target.id AND target.bound_key_count = 0
    RETURNING policy.id
)
SELECT target.bound_key_count, deleted.id
FROM target LEFT JOIN deleted ON true
`, userID, policyID).Scan(&boundKeyCount, &deletedID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRoutingNotFound
	}
	if err != nil {
		return routingStoreError(err)
	}
	if boundKeyCount > 0 || !deletedID.Valid {
		return ErrRoutingInvalidState
	}
	return nil
}

func routingStoreError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRoutingNotFound
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return ErrRoutingInvalidState
	}
	return err
}

func (s *SQLRoutingStore) CreateCanonicalModel(ctx context.Context, command CreateCanonicalModelCommand) (CanonicalModel, error) {
	command.Price = normalizeOfficialPrice(command.Price)
	if err := validateCanonicalModel(command); err != nil {
		return CanonicalModel{}, ErrRoutingInvalidState
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return CanonicalModel{}, fmt.Errorf("开始标准模型事务：%w", err)
	}
	defer tx.Rollback()
	model := CanonicalModel{PublicName: command.PublicName, Provider: command.Provider, Status: "active", Price: command.Price}
	err = tx.QueryRowContext(ctx, `
INSERT INTO canonical_models (public_name, provider, status)
VALUES ($1, $2, $3)
RETURNING id, status, created_at
`, command.PublicName, command.Provider, model.Status).Scan(&model.ID, &model.Status, &model.CreatedAt)
	if err != nil {
		return CanonicalModel{}, fmt.Errorf("创建标准模型：%w", err)
	}
	model.Price, err = insertOfficialPriceVersion(ctx, tx, model.ID, 1, command.AdminID, command.Price)
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

func insertOfficialPriceVersion(ctx context.Context, tx *sql.Tx, modelID int64, version int, adminID int, price OfficialPrice) (OfficialPrice, error) {
	price = normalizeOfficialPrice(price)
	if err := validateOfficialPrice(price); err != nil {
		return OfficialPrice{}, err
	}
	err := tx.QueryRowContext(ctx, `
INSERT INTO official_price_versions
	(canonical_model_id, version, price_kind, published_by_newapi_user_id)
VALUES ($1, $2, $3, $4)
RETURNING id, version, published_at
`, modelID, version, price.PriceKind, adminID).Scan(
		&price.ID, &price.Version, &price.PublishedAt,
	)
	if err != nil {
		return OfficialPrice{}, err
	}
	for index, tier := range price.Tiers {
		result, err := tx.ExecContext(ctx, `
INSERT INTO text_price_tiers
    (official_price_version_id, tier_order, minimum_input_tokens, maximum_input_tokens,
     input_price_micros, output_price_micros, cache_read_price_micros, cache_write_price_micros)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, price.ID, index+1, tier.MinimumInputTokens, tier.MaximumInputTokens,
			tier.InputPriceMicros, tier.OutputPriceMicros, tier.CacheReadPriceMicros,
			tier.CacheWritePriceMicros)
		if err != nil {
			return OfficialPrice{}, err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return OfficialPrice{}, ErrRoutingInvalidState
		}
		price.Tiers[index].TierOrder = index + 1
	}
	for _, item := range price.ImagePrices {
		result, err := tx.ExecContext(ctx, `
INSERT INTO image_price_items (official_price_version_id, quality, size, price_micros)
VALUES ($1, $2, $3, $4)
`, price.ID, item.Quality, item.Size, item.PriceMicros)
		if err != nil {
			return OfficialPrice{}, err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return OfficialPrice{}, ErrRoutingInvalidState
		}
	}
	return price, nil
}

func (s *SQLRoutingStore) CreateOffer(ctx context.Context, command CreateOfferCommand) (OfferVersion, error) {
	if command.UserID <= 0 || command.CanonicalModelID <= 0 || command.MultiplierBPS < 0 || command.MultiplierBPS > 50_000 {
		return OfferVersion{}, ErrRoutingInvalidState
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return OfferVersion{}, fmt.Errorf("开始报价事务：%w", err)
	}
	defer tx.Rollback()
	var offer OfferVersion
	offer.CanonicalModelID = command.CanonicalModelID
	offer.MultiplierBPS = command.MultiplierBPS
	err = tx.QueryRowContext(ctx, `
SELECT mp.id, cm.current_official_price_version_id, cm.public_name, price.price_kind,
	   COALESCE((SELECT id FROM offer_versions old
	             WHERE old.merchant_profile_id = mp.id AND old.canonical_model_id = cm.id AND old.status = 'active'
	             ORDER BY old.id DESC LIMIT 1), 0),
	   COALESCE((SELECT max(version) + 1 FROM offer_versions versions
	             WHERE versions.merchant_profile_id = mp.id AND versions.canonical_model_id = cm.id), 1)
FROM merchant_profiles mp
JOIN canonical_models cm ON cm.id = $2
JOIN official_price_versions price ON price.id = cm.current_official_price_version_id
WHERE mp.newapi_user_id = $1 AND mp.status = 'active'
  AND cm.status = 'active' AND cm.current_official_price_version_id IS NOT NULL
FOR UPDATE OF mp, cm
	`, command.UserID, command.CanonicalModelID).Scan(
		&offer.MerchantID, &offer.OfficialPriceVersionID, &offer.PublicModel,
		&offer.PriceKind, &offer.PreviousOfferID, &offer.Version,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return OfferVersion{}, ErrRoutingNotFound
	}
	if err != nil {
		return OfferVersion{}, fmt.Errorf("锁定报价商户与模型：%w", err)
	}
	rows, err := tx.QueryContext(ctx, `
SELECT id, newapi_channel_id, channel_name FROM merchant_channels
WHERE merchant_profile_id = $1 AND status = 'active'
  AND (canonical_model_id = $2 OR EXISTS (
      SELECT 1 FROM merchant_channel_model_bindings binding
      WHERE binding.merchant_channel_id = merchant_channels.id
        AND binding.canonical_model_id = $2 AND binding.match_status = 'matched'
  ))
ORDER BY id
`, offer.MerchantID, command.CanonicalModelID)
	if err != nil {
		return OfferVersion{}, fmt.Errorf("查询报价渠道：%w", err)
	}
	for rows.Next() {
		var alias OfferChannelAlias
		if err := rows.Scan(&alias.MerchantChannelID, &alias.NewAPIChannelID, &alias.ChannelName); err != nil {
			rows.Close()
			return OfferVersion{}, fmt.Errorf("解析报价渠道：%w", err)
		}
		offer.Aliases = append(offer.Aliases, alias)
	}
	if err := rows.Close(); err != nil {
		return OfferVersion{}, fmt.Errorf("关闭报价渠道结果：%w", err)
	}
	if len(offer.Aliases) == 0 {
		return OfferVersion{}, ErrRoutingNotFound
	}
	var previous any
	if offer.PreviousOfferID > 0 {
		previous = offer.PreviousOfferID
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO offer_versions
	(merchant_profile_id, canonical_model_id, official_price_version_id,
	 previous_offer_version_id, multiplier_bps, version, created_by_newapi_user_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, status, created_at
`, offer.MerchantID, offer.CanonicalModelID, offer.OfficialPriceVersionID, previous,
		offer.MultiplierBPS, offer.Version, command.UserID).Scan(&offer.ID, &offer.Status, &offer.CreatedAt)
	if err != nil {
		return OfferVersion{}, fmt.Errorf("创建报价版本：%w", err)
	}
	for index := range offer.Aliases {
		alias := &offer.Aliases[index]
		alias.InternalAlias = fmt.Sprintf("mkt-o%d-c%d", offer.ID, alias.MerchantChannelID)
		if err := tx.QueryRowContext(ctx, `
INSERT INTO offer_channel_aliases (offer_version_id, merchant_channel_id, internal_alias)
VALUES ($1, $2, $3)
RETURNING id
`, offer.ID, alias.MerchantChannelID, alias.InternalAlias).Scan(&alias.ID); err != nil {
			return OfferVersion{}, fmt.Errorf("创建渠道报价别名：%w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return OfferVersion{}, fmt.Errorf("提交报价事务：%w", err)
	}
	return offer, nil
}

func (s *SQLRoutingStore) RetireOffer(ctx context.Context, userID int, modelID int64) error {
	if userID <= 0 || modelID <= 0 {
		return ErrRoutingInvalidState
	}
	result, err := s.database.ExecContext(ctx, `
UPDATE offer_versions offer
SET status = 'inactive', retired_at = now()
FROM merchant_profiles merchant
WHERE offer.merchant_profile_id = merchant.id
  AND merchant.newapi_user_id = $1
  AND offer.canonical_model_id = $2
  AND offer.status = 'active'
`, userID, modelID)
	if err != nil {
		return fmt.Errorf("取消商户报价：%w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrRoutingNotFound
	}
	return nil
}

func (s *SQLRoutingStore) MerchantOfferState(ctx context.Context, userID int, modelID int64) (MerchantOfferState, error) {
	if userID <= 0 || modelID <= 0 {
		return MerchantOfferState{}, ErrRoutingInvalidState
	}
	var state MerchantOfferState
	err := s.database.QueryRowContext(ctx, `
SELECT offer.multiplier_bps, offer.status
FROM offer_versions offer
JOIN merchant_profiles merchant ON merchant.id = offer.merchant_profile_id
WHERE merchant.newapi_user_id = $1 AND offer.canonical_model_id = $2
ORDER BY offer.version DESC
LIMIT 1
`, userID, modelID).Scan(&state.MultiplierBPS, &state.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return MerchantOfferState{}, ErrRoutingNotFound
	}
	if err != nil {
		return MerchantOfferState{}, fmt.Errorf("读取商户模型状态：%w", err)
	}
	return state, nil
}

func (s *SQLRoutingStore) DeleteMerchantOffer(ctx context.Context, userID int, modelID int64) error {
	if userID <= 0 || modelID <= 0 {
		return ErrRoutingInvalidState
	}
	result, err := s.database.ExecContext(ctx, `
WITH latest AS (
    SELECT offer.id
    FROM offer_versions offer
    JOIN merchant_profiles merchant ON merchant.id = offer.merchant_profile_id
    WHERE merchant.newapi_user_id = $1 AND offer.canonical_model_id = $2
      AND offer.status <> 'deleted'
    ORDER BY offer.version DESC
    LIMIT 1
)
UPDATE offer_versions
SET status = 'deleted', retired_at = COALESCE(retired_at, now()), sync_error = NULL
WHERE id = (SELECT id FROM latest)
  AND status IN ('active', 'inactive', 'sync_failed')
`, userID, modelID)
	if err != nil {
		return fmt.Errorf("删除商户上架模型：%w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrRoutingNotFound
	}
	return nil
}

func (s *SQLRoutingStore) ActivateOffer(ctx context.Context, offerID int64) error {
	return s.FinalizeOffer(ctx, offerID, true)
}

func (s *SQLRoutingStore) FinalizeOffer(ctx context.Context, offerID int64, activate bool) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始报价激活事务：%w", err)
	}
	defer tx.Rollback()
	var previous sql.NullInt64
	var merchantID, canonicalModelID int64
	err = tx.QueryRowContext(ctx, `
SELECT previous_offer_version_id, merchant_profile_id, canonical_model_id
FROM offer_versions
WHERE id = $1 AND status = 'syncing'
FOR UPDATE
`, offerID).Scan(&previous, &merchantID, &canonicalModelID)
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
	targetStatus := "inactive"
	if activate {
		targetStatus = "active"
	}
	result, err := tx.ExecContext(ctx, `
UPDATE offer_versions
SET status = $2,
    activated_at = CASE WHEN $2 = 'active' THEN now() ELSE NULL END,
    retired_at = CASE WHEN $2 = 'inactive' THEN now() ELSE NULL END,
    sync_error = NULL
WHERE id = $1 AND status = 'syncing'
`, offerID, targetStatus)
	if err != nil {
		return fmt.Errorf("激活新报价：%w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrRoutingInvalidState
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
SELECT aliases.internal_alias, price.price_kind, o.multiplier_bps, o.official_price_version_id
FROM offer_channel_aliases aliases
JOIN offer_versions o ON o.id = aliases.offer_version_id
JOIN official_price_versions price ON price.id = o.official_price_version_id
WHERE o.status IN ('syncing', 'active', 'inactive')
ORDER BY aliases.id
`)
	if err != nil {
		return nil, fmt.Errorf("查询别名价格：%w", err)
	}
	defer rows.Close()
	prices := make([]aliasPrice, 0)
	priceCache := make(map[int64]OfficialPrice)
	for rows.Next() {
		var price aliasPrice
		var priceID int64
		if err := rows.Scan(&price.Alias, &price.PriceKind, &price.MultiplierBPS, &priceID); err != nil {
			return nil, fmt.Errorf("解析别名价格：%w", err)
		}
		loaded, exists := priceCache[priceID]
		if !exists {
			loaded, err = loadOfficialPrice(ctx, s.database, priceID)
			if err != nil {
				return nil, fmt.Errorf("读取别名官方价格：%w", err)
			}
			priceCache[priceID] = loaded
		}
		price.Price = loaded
		prices = append(prices, price)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历别名价格：%w", err)
	}
	return prices, nil
}

type officialPriceQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func loadOfficialPrice(ctx context.Context, queryer officialPriceQueryer, priceID int64) (OfficialPrice, error) {
	var price OfficialPrice
	if err := queryer.QueryRowContext(ctx, `
SELECT id, version, price_kind, published_at
FROM official_price_versions
WHERE id = $1
`, priceID).Scan(&price.ID, &price.Version, &price.PriceKind, &price.PublishedAt); err != nil {
		return OfficialPrice{}, err
	}
	if price.PriceKind == "image" {
		rows, err := queryer.QueryContext(ctx, `
SELECT quality, size, price_micros
FROM image_price_items
WHERE official_price_version_id = $1
ORDER BY quality, size
`, priceID)
		if err != nil {
			return OfficialPrice{}, err
		}
		defer rows.Close()
		for rows.Next() {
			var item ImagePriceItem
			if err := rows.Scan(&item.Quality, &item.Size, &item.PriceMicros); err != nil {
				return OfficialPrice{}, err
			}
			price.ImagePrices = append(price.ImagePrices, item)
		}
		return price, rows.Err()
	}
	rows, err := queryer.QueryContext(ctx, `
SELECT id, tier_order, minimum_input_tokens, maximum_input_tokens,
       input_price_micros, output_price_micros, cache_read_price_micros,
       cache_write_price_micros
FROM text_price_tiers
WHERE official_price_version_id = $1
ORDER BY tier_order
`, priceID)
	if err != nil {
		return OfficialPrice{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var tier PriceTier
		var maximum, output, cacheRead, cacheWrite sql.NullInt64
		if err := rows.Scan(
			&tier.ID, &tier.TierOrder, &tier.MinimumInputTokens, &maximum,
			&tier.InputPriceMicros, &output, &cacheRead, &cacheWrite,
		); err != nil {
			return OfficialPrice{}, err
		}
		assignNullableTierPrices(&tier, maximum, output, cacheRead, cacheWrite)
		price.Tiers = append(price.Tiers, tier)
	}
	return price, rows.Err()
}

func nullInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func (s *SQLRoutingStore) OfferChannelRoutes(ctx context.Context, offerID int64) ([]ChannelPriceRoute, error) {
	rows, err := s.database.QueryContext(ctx, `
SELECT mc.id, mc.newapi_channel_id, cm.public_name, aliases.internal_alias
FROM offer_channel_aliases target
JOIN merchant_channels mc ON mc.id = target.merchant_channel_id
JOIN offer_channel_aliases aliases ON aliases.merchant_channel_id = mc.id
JOIN offer_versions o ON o.id = aliases.offer_version_id
JOIN canonical_models cm ON cm.id = o.canonical_model_id
WHERE target.offer_version_id = $1 AND (o.status = 'active' OR o.id = $1)
ORDER BY mc.id, aliases.id
`, offerID)
	if err != nil {
		return nil, fmt.Errorf("查询报价渠道映射：%w", err)
	}
	defer rows.Close()
	routes := make([]ChannelPriceRoute, 0)
	indexes := make(map[int64]int)
	modelSets := make(map[int64]map[string]struct{})
	for rows.Next() {
		var channelID int64
		var newAPIChannelID int
		var publicModel, alias string
		if err := rows.Scan(&channelID, &newAPIChannelID, &publicModel, &alias); err != nil {
			return nil, fmt.Errorf("解析报价渠道映射：%w", err)
		}
		index, exists := indexes[channelID]
		if !exists {
			index = len(routes)
			indexes[channelID] = index
			modelSets[channelID] = make(map[string]struct{})
			routes = append(routes, ChannelPriceRoute{MerchantChannelID: channelID, NewAPIChannelID: newAPIChannelID, Aliases: make(map[string]string)})
		}
		routes[index].Aliases[alias] = publicModel
		modelSets[channelID][publicModel] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历报价渠道映射：%w", err)
	}
	for index := range routes {
		for model := range modelSets[routes[index].MerchantChannelID] {
			routes[index].PublicModels = append(routes[index].PublicModels, model)
		}
		sort.Strings(routes[index].PublicModels)
	}
	return routes, nil
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

func (s *SQLRoutingStore) ResolveRoute(ctx context.Context, fingerprint string, publicModel string) (RouteSelection, error) {
	var selection RouteSelection
	var cacheFloor sql.NullInt64
	err := s.database.QueryRowContext(ctx, `
SELECT b.id, p.id, cm.id, p.priority, price.price_kind, p.minimum_cache_hit_ppm
FROM token_bindings b
JOIN routing_policies p ON p.id = b.routing_policy_id
JOIN canonical_models cm ON cm.public_name = $2 AND cm.status = 'active'
JOIN official_price_versions price ON price.id = cm.current_official_price_version_id
WHERE b.token_fingerprint = $1 AND b.status = 'active'
`, fingerprint, publicModel).Scan(&selection.TokenBindingID, &selection.PolicyID, &selection.CanonicalModelID, &selection.Priority, &selection.PriceKind, &cacheFloor)
	if errors.Is(err, sql.ErrNoRows) {
		return RouteSelection{}, ErrRoutingNotFound
	}
	if err != nil {
		return RouteSelection{}, fmt.Errorf("读取路由策略：%w", err)
	}
	if cacheFloor.Valid {
		value := int(cacheFloor.Int64)
		selection.MinimumCacheHitPPM = &value
	}
	rows, err := s.database.QueryContext(ctx, `
WITH merchant_metrics AS (
    SELECT channel.merchant_profile_id, snapshot.canonical_model_id,
           sum(snapshot.prompt_tokens) AS prompt_tokens,
           CASE WHEN sum(snapshot.prompt_tokens) = 0 THEN 0
                ELSE LEAST(1000000, (sum(snapshot.cache_tokens) * 1000000 / sum(snapshot.prompt_tokens))::integer) END AS cache_hit_ppm,
           sum(snapshot.prompt_tokens) >= 1000000 AS cache_weight_ready
    FROM route_metric_snapshots snapshot
    JOIN merchant_channels channel ON channel.id = snapshot.merchant_channel_id AND channel.status = 'active'
    GROUP BY channel.merchant_profile_id, snapshot.canonical_model_id
)
SELECT o.id, aliases.id, mp.id, mc.id, mc.newapi_channel_id, cm.id, aliases.internal_alias,
       cm.public_name, o.multiplier_bps, h.state,
       COALESCE(metric.availability_ppm, 0), COALESCE(metric.average_ttft_ms, 0),
       COALESCE(metric.request_count, 0), COALESCE(metric.cache_hit_ppm, 0),
       COALESCE(metric.prompt_tokens, 0), COALESCE(metric.cache_weight_ready, false),
       COALESCE(merchant_metric.cache_hit_ppm, 0), COALESCE(merchant_metric.cache_weight_ready, false)
FROM routing_policies p
JOIN canonical_models cm ON cm.id = $2
JOIN offer_versions o ON o.canonical_model_id = cm.id AND o.status = 'active'
JOIN offer_channel_aliases aliases ON aliases.offer_version_id = o.id
JOIN merchant_channels mc ON mc.id = aliases.merchant_channel_id AND mc.status = 'active'
JOIN merchant_profiles mp ON mp.id = mc.merchant_profile_id AND mp.status = 'active'
JOIN channel_health h ON h.merchant_channel_id = mc.id
LEFT JOIN route_metric_snapshots metric
    ON metric.merchant_channel_id = mc.id AND metric.canonical_model_id = cm.id
LEFT JOIN merchant_metrics merchant_metric
    ON merchant_metric.merchant_profile_id = mp.id AND merchant_metric.canonical_model_id = cm.id
WHERE p.id = $1
  AND (mc.canonical_model_id = $2 OR EXISTS (
      SELECT 1 FROM merchant_channel_model_bindings binding
      WHERE binding.merchant_channel_id = mc.id
        AND binding.canonical_model_id = $2 AND binding.match_status = 'matched'
  ))
  AND o.multiplier_bps BETWEEN p.minimum_multiplier_bps AND p.maximum_multiplier_bps
  AND h.state IN ('healthy', 'degraded')
`, selection.PolicyID, selection.CanonicalModelID)
	if err != nil {
		return RouteSelection{}, fmt.Errorf("查询路由候选：%w", err)
	}
	defer rows.Close()
	selection.Candidates = make([]RouteCandidate, 0)
	for rows.Next() {
		var candidate RouteCandidate
		if err := rows.Scan(
			&candidate.OfferID, &candidate.OfferChannelAliasID, &candidate.MerchantID, &candidate.MerchantChannelID,
			&candidate.NewAPIChannelID, &candidate.CanonicalModelID, &candidate.InternalAlias,
			&candidate.PublicModel, &candidate.MultiplierBPS,
			&candidate.HealthState, &candidate.AvailabilityPPM, &candidate.TTFTP95MS,
			&candidate.MetricRequestCount, &candidate.CacheHitPPM, &candidate.CacheSampleInputTokens,
			&candidate.CacheWeightReady, &candidate.MerchantCacheHitPPM, &candidate.MerchantCacheWeightReady,
		); err != nil {
			return RouteSelection{}, fmt.Errorf("解析路由候选：%w", err)
		}
		if routeCandidateMeetsCacheFloor(selection, candidate) {
			selection.Candidates = append(selection.Candidates, candidate)
		}
	}
	if err := rows.Err(); err != nil {
		return RouteSelection{}, fmt.Errorf("遍历路由候选：%w", err)
	}
	selection.Candidates = weightedCandidateOrder(selection.Candidates, selection.Priority, systemRouteRandom{})
	return selection, nil
}

func (s *SQLRoutingStore) StartRouteDecision(ctx context.Context, decision RouteDecision) error {
	selectedMetrics, err := json.Marshal(decision.SelectedMetrics)
	if err != nil {
		return fmt.Errorf("编码路由决策指标：%w", err)
	}
	_, err = s.database.ExecContext(ctx, `
INSERT INTO routing_decisions
    (request_id, token_binding_id, routing_policy_id, canonical_model_id,
     offer_channel_alias_id, priority, algorithm_version, candidate_count, selected_metrics)
VALUES ($1, $2, $3, $4, NULLIF($5, 0), $6, $7, $8, $9)
`, decision.RequestID, decision.TokenBindingID, decision.PolicyID, decision.CanonicalModelID,
		decision.OfferChannelAliasID, decision.Priority,
		decision.AlgorithmVersion, decision.CandidateCount, selectedMetrics)
	if err != nil {
		return fmt.Errorf("创建路由决策：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) CompleteRouteDecision(ctx context.Context, result RouteDecisionResult) error {
	_, err := s.database.ExecContext(ctx, `
UPDATE routing_decisions
SET offer_channel_alias_id = NULLIF($2, 0), attempt_count = $3,
    upstream_status_code = NULLIF($4, 0), first_byte_ms = $5,
    result = $6, completed_at = now()
WHERE request_id = $1
`, result.RequestID, result.OfferChannelAliasID, result.Attempts, result.StatusCode, result.FirstByteMS, result.Result)
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
	class := healthFailureIgnored
	switch observation.Outcome {
	case "success":
		class = healthFailureNone
	case "temporary_failure":
		class = healthFailureTemporary
	case "hard_failure":
		class = healthFailureHard
	}
	var probe HealthProbe
	probe.MerchantChannelID = observation.Candidate.MerchantChannelID
	if err := tx.QueryRowContext(ctx, `
SELECT state, consecutive_failures, cooldown_level
FROM channel_health
WHERE merchant_channel_id = $1
FOR UPDATE
`, probe.MerchantChannelID).Scan(&probe.State, &probe.ConsecutiveFailures, &probe.CooldownLevel); err != nil {
		return fmt.Errorf("锁定渠道健康状态：%w", err)
	}
	next := nextProbeState(probe, class, time.Now().UTC(), 0)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO channel_health_events
    (merchant_channel_id, offer_channel_alias_id, source, outcome,
     status_code, first_byte_ms, request_id, failure_summary)
VALUES ($1, $2, 'request', $3, NULLIF($4, 0), $5, $6, NULLIF($7, ''))
`, observation.Candidate.MerchantChannelID, observation.Candidate.OfferChannelAliasID,
		observation.Outcome, observation.StatusCode, observation.FirstByteMS,
		observation.RequestID, observation.Summary); err != nil {
		return fmt.Errorf("记录健康事件：%w", err)
	}
	validObservation := class != healthFailureIgnored
	success := class == healthFailureNone
	if _, err := tx.ExecContext(ctx, `
UPDATE channel_health
SET state = $2, consecutive_failures = $3, cooldown_level = $4,
	    next_probe_at = $5::timestamptz,
	    cooldown_until = CASE WHEN $2 = 'cooling' THEN $5::timestamptz ELSE NULL END,
	    observations = observations + CASE WHEN $6 THEN 1 ELSE 0 END,
	    successes = successes + CASE WHEN $7 THEN 1 ELSE 0 END,
	    availability_ppm = CASE
	        WHEN observations + CASE WHEN $6 THEN 1 ELSE 0 END = 0 THEN availability_ppm
	        ELSE ((successes + CASE WHEN $7 THEN 1 ELSE 0 END) * 1000000 /
	             (observations + CASE WHEN $6 THEN 1 ELSE 0 END))::integer END,
	    ttft_p95_ms = CASE WHEN $6 AND $8 > 0 THEN $8 ELSE ttft_p95_ms END,
	    last_success_at = CASE WHEN $7 THEN now() ELSE last_success_at END,
	    last_failure_at = CASE WHEN $9 IN ('temporary', 'hard') THEN now() ELSE last_failure_at END,
	    last_failure_class = CASE WHEN $9 IN ('temporary', 'hard') THEN $9 ELSE last_failure_class END,
	    last_status_code = NULLIF($10, 0), updated_at = now()
WHERE merchant_channel_id = $1
`, observation.Candidate.MerchantChannelID, next.State, next.ConsecutiveFailures,
		next.CooldownLevel, timeOrNil(next.NextProbeAt), validObservation, success,
		observation.FirstByteMS, string(class), observation.StatusCode); err != nil {
		return fmt.Errorf("更新渠道健康状态：%w", err)
	}
	if validObservation {
		requestCount, successCount, ttftTotal, ttftCount := int64(1), int64(0), int64(0), int64(0)
		if success {
			successCount = 1
			if observation.FirstByteMS > 0 {
				ttftTotal = int64(observation.FirstByteMS)
				ttftCount = 1
			}
		}
		bucketStart := time.Now().UTC().Truncate(time.Hour)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO route_metric_buckets
    (merchant_channel_id, canonical_model_id, bucket_start,
     request_count, success_count, ttft_total_ms, ttft_count)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (merchant_channel_id, canonical_model_id, bucket_start) DO UPDATE
SET request_count = route_metric_buckets.request_count + EXCLUDED.request_count,
    success_count = route_metric_buckets.success_count + EXCLUDED.success_count,
    ttft_total_ms = route_metric_buckets.ttft_total_ms + EXCLUDED.ttft_total_ms,
    ttft_count = route_metric_buckets.ttft_count + EXCLUDED.ttft_count,
    updated_at = now()
`, observation.Candidate.MerchantChannelID, observation.Candidate.CanonicalModelID,
			bucketStart, requestCount, successCount, ttftTotal, ttftCount); err != nil {
			return fmt.Errorf("记录路由小时指标：%w", err)
		}
	}
	if class == healthFailureHard {
		if _, err := tx.ExecContext(ctx, `UPDATE merchant_channels SET status = 'disabled', updated_at = now() WHERE id = $1`, observation.Candidate.MerchantChannelID); err != nil {
			return fmt.Errorf("停用硬故障渠道：%w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交健康观察事务：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) RefreshRouteMetricSnapshots(ctx context.Context, now time.Time) error {
	now = now.UTC()
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始刷新路由指标事务：%w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
DELETE FROM route_metric_snapshots snapshot
WHERE NOT EXISTS (
    SELECT 1 FROM route_metric_buckets bucket
    WHERE bucket.merchant_channel_id = snapshot.merchant_channel_id
      AND bucket.canonical_model_id = snapshot.canonical_model_id
      AND bucket.bucket_start >= date_trunc('hour', $1::timestamptz - interval '7 days')
)
`, now); err != nil {
		return fmt.Errorf("清理过期路由快照：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO route_metric_snapshots
    (merchant_channel_id, canonical_model_id, request_count, success_count,
     availability_ppm, average_ttft_ms, prompt_tokens, cache_tokens,
     cache_hit_ppm, cache_weight_ready, metrics_updated_at)
SELECT bucket.merchant_channel_id, bucket.canonical_model_id,
       sum(bucket.request_count), sum(bucket.success_count),
       CASE WHEN sum(bucket.request_count) = 0 THEN 0
            ELSE (sum(bucket.success_count) * 1000000 / sum(bucket.request_count))::integer END,
       CASE WHEN sum(bucket.ttft_count) = 0 THEN 0
            ELSE (sum(bucket.ttft_total_ms) / sum(bucket.ttft_count))::integer END,
       sum(bucket.prompt_tokens), sum(bucket.cache_tokens),
       CASE WHEN sum(bucket.prompt_tokens) = 0 THEN 0
            ELSE LEAST(1000000, (sum(bucket.cache_tokens) * 1000000 / sum(bucket.prompt_tokens))::integer) END,
       sum(bucket.prompt_tokens) >= 1000000,
       $1
FROM route_metric_buckets bucket
WHERE bucket.bucket_start >= date_trunc('hour', $1::timestamptz - interval '7 days')
GROUP BY bucket.merchant_channel_id, bucket.canonical_model_id
ON CONFLICT (merchant_channel_id, canonical_model_id) DO UPDATE
SET request_count = EXCLUDED.request_count,
    success_count = EXCLUDED.success_count,
    availability_ppm = EXCLUDED.availability_ppm,
    average_ttft_ms = EXCLUDED.average_ttft_ms,
    prompt_tokens = EXCLUDED.prompt_tokens,
    cache_tokens = EXCLUDED.cache_tokens,
    cache_hit_ppm = EXCLUDED.cache_hit_ppm,
    cache_weight_ready = EXCLUDED.cache_weight_ready,
    metrics_updated_at = EXCLUDED.metrics_updated_at
`, now); err != nil {
		return fmt.Errorf("刷新七天路由快照：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
DELETE FROM route_metric_buckets
WHERE bucket_start < date_trunc('hour', $1::timestamptz - interval '8 days')
`, now); err != nil {
		return fmt.Errorf("清理路由小时指标：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交路由指标事务：%w", err)
	}
	return nil
}

func (s *SQLRoutingStore) ClaimHealthProbes(ctx context.Context, now time.Time, limit int) ([]HealthProbe, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.database.QueryContext(ctx, `
WITH due AS (
    SELECT h.merchant_channel_id, mc.canonical_model_id, route.offer_id,
           route.offer_channel_alias_id, mc.newapi_channel_id, route.internal_alias,
           route.price_kind, h.state, h.consecutive_failures, h.cooldown_level
    FROM channel_health h
    JOIN merchant_channels mc ON mc.id = h.merchant_channel_id AND mc.status = 'active'
    JOIN canonical_models cm ON cm.id = mc.canonical_model_id AND cm.status = 'active'
    JOIN LATERAL (
        SELECT o.id AS offer_id, aliases.id AS offer_channel_alias_id, aliases.internal_alias,
               price.price_kind
        FROM offer_channel_aliases aliases
        JOIN offer_versions o ON o.id = aliases.offer_version_id
        JOIN official_price_versions price ON price.id = o.official_price_version_id
        WHERE aliases.merchant_channel_id = h.merchant_channel_id AND o.status = 'active'
        ORDER BY o.id DESC LIMIT 1
    ) route ON true
    WHERE h.state IN ('cooling', 'recovering') AND h.next_probe_at <= $1
    ORDER BY h.next_probe_at, h.merchant_channel_id
    LIMIT $2
    FOR UPDATE OF h SKIP LOCKED
), claimed AS (
    UPDATE channel_health h
    SET state = 'recovering', next_probe_at = $1 + interval '30 seconds', updated_at = now()
    FROM due
    WHERE h.merchant_channel_id = due.merchant_channel_id
    RETURNING h.merchant_channel_id
)
SELECT d.offer_channel_alias_id, d.offer_id, d.merchant_channel_id, d.canonical_model_id, d.newapi_channel_id,
       d.internal_alias, d.price_kind, d.state, d.consecutive_failures,
       d.cooldown_level
FROM due d
JOIN claimed c USING (merchant_channel_id)
`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("领取探活任务：%w", err)
	}
	defer rows.Close()
	probes := make([]HealthProbe, 0)
	for rows.Next() {
		var probe HealthProbe
		if err := rows.Scan(
			&probe.OfferChannelAliasID, &probe.OfferID, &probe.MerchantChannelID, &probe.CanonicalModelID,
			&probe.NewAPIChannelID, &probe.InternalAlias, &probe.PriceKind, &probe.State,
			&probe.ConsecutiveFailures, &probe.CooldownLevel,
		); err != nil {
			return nil, fmt.Errorf("解析探活任务：%w", err)
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
    (merchant_channel_id, offer_channel_alias_id, source, outcome,
     status_code, first_byte_ms, failure_summary)
VALUES ($1, $2, 'probe', $3, NULLIF($4, 0), $5, NULLIF($6, ''))
`, result.Probe.MerchantChannelID, result.Probe.OfferChannelAliasID,
		outcome, result.StatusCode, result.FirstByteMS, result.Summary); err != nil {
		return fmt.Errorf("记录探活事件：%w", err)
	}
	validObservation := result.Class != healthFailureIgnored
	success := result.Class == healthFailureNone
	if _, err := tx.ExecContext(ctx, `
UPDATE channel_health
SET state = $2, consecutive_failures = $3, cooldown_level = $4,
	    next_probe_at = $5::timestamptz,
	    cooldown_until = CASE WHEN $2 = 'cooling' THEN $5::timestamptz ELSE NULL END,
	    observations = observations + CASE WHEN $6 THEN 1 ELSE 0 END,
	    successes = successes + CASE WHEN $7 THEN 1 ELSE 0 END,
	    availability_ppm = CASE
	        WHEN observations + CASE WHEN $6 THEN 1 ELSE 0 END = 0 THEN availability_ppm
	        ELSE ((successes + CASE WHEN $7 THEN 1 ELSE 0 END) * 1000000 /
	             (observations + CASE WHEN $6 THEN 1 ELSE 0 END))::integer END,
	    ttft_p95_ms = CASE WHEN $6 AND $8 > 0 THEN $8 ELSE ttft_p95_ms END,
	    last_success_at = CASE WHEN $7 THEN now() ELSE last_success_at END,
	    last_failure_at = CASE WHEN $9 IN ('temporary', 'hard') THEN now() ELSE last_failure_at END,
	    last_failure_class = CASE WHEN $9 IN ('temporary', 'hard') THEN $9 ELSE last_failure_class END,
	    last_status_code = NULLIF($10, 0),
	    updated_at = now()
WHERE merchant_channel_id = $1
`, result.Probe.MerchantChannelID, result.State.State, result.State.ConsecutiveFailures,
		result.State.CooldownLevel, timeOrNil(result.State.NextProbeAt), validObservation,
		success, result.FirstByteMS, string(result.Class), result.StatusCode); err != nil {
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
WITH usage_stats AS (
    SELECT source_offer.merchant_profile_id, source_offer.canonical_model_id,
           count(*) FILTER (WHERE sources.source_type = 2 AND sources.status = 'settled') AS call_count,
           COALESCE(sum(
               CASE WHEN sources.source_type = 2 AND sources.status = 'settled' THEN
                   CASE
                       WHEN (CASE WHEN sources.other->>'prompt_tokens' ~ '^[0-9]+$' THEN (sources.other->>'prompt_tokens')::bigint ELSE 0 END)
                          + (CASE WHEN sources.other->>'completion_tokens' ~ '^[0-9]+$' THEN (sources.other->>'completion_tokens')::bigint ELSE 0 END) > 0
                       THEN (CASE WHEN sources.other->>'prompt_tokens' ~ '^[0-9]+$' THEN (sources.other->>'prompt_tokens')::bigint ELSE 0 END)
                          + (CASE WHEN sources.other->>'completion_tokens' ~ '^[0-9]+$' THEN (sources.other->>'completion_tokens')::bigint ELSE 0 END)
                       WHEN sources.other->>'total_tokens' ~ '^[0-9]+$' THEN (sources.other->>'total_tokens')::bigint
                       ELSE 0
                   END
               ELSE 0 END
           ), 0) AS token_usage,
           max(sources.source_created_at) FILTER (WHERE sources.source_type = 2 AND sources.status = 'settled') AS last_called_at
    FROM settlement_sources sources
    JOIN offer_versions source_offer ON source_offer.id = sources.offer_version_id
    GROUP BY source_offer.merchant_profile_id, source_offer.canonical_model_id
), metric_stats AS (
    SELECT channel.merchant_profile_id, snapshot.canonical_model_id,
           sum(snapshot.prompt_tokens) AS prompt_tokens,
           sum(snapshot.cache_tokens) AS cache_tokens,
           max(snapshot.metrics_updated_at) AS metrics_updated_at
    FROM route_metric_snapshots snapshot
    JOIN merchant_channels channel ON channel.id = snapshot.merchant_channel_id
    WHERE channel.status = 'active'
    GROUP BY channel.merchant_profile_id, snapshot.canonical_model_id
)
SELECT o.id, mp.id, mp.display_name, cm.public_name,
       o.multiplier_bps, o.official_price_version_id,
       count(*) FILTER (WHERE h.state IN ('healthy', 'degraded')),
       CASE
           WHEN bool_or(h.state = 'healthy') THEN 'healthy'
           WHEN bool_or(h.state = 'degraded') THEN 'degraded'
           WHEN bool_or(h.state = 'recovering') THEN 'recovering'
           WHEN bool_or(h.state = 'cooling') THEN 'cooling'
           ELSE 'disabled'
       END,
       COALESCE(max(h.availability_ppm) FILTER (WHERE h.state IN ('healthy', 'degraded')), 0),
       COALESCE(max(h.ttft_p95_ms) FILTER (WHERE h.state IN ('healthy', 'degraded')), 0),
	       COALESCE(stats.call_count, 0), COALESCE(stats.token_usage, 0), stats.last_called_at,
	       CASE WHEN COALESCE(metric.prompt_tokens, 0) = 0 THEN 0
	            ELSE LEAST(1000000, (metric.cache_tokens * 1000000 / metric.prompt_tokens)::integer) END,
	       COALESCE(metric.prompt_tokens, 0), COALESCE(metric.prompt_tokens, 0) >= 1000000,
	       metric.metrics_updated_at
FROM offer_versions o
JOIN canonical_models cm ON cm.id = o.canonical_model_id
JOIN offer_channel_aliases aliases ON aliases.offer_version_id = o.id
JOIN merchant_channels mc ON mc.id = aliases.merchant_channel_id
JOIN merchant_profiles mp ON mp.id = mc.merchant_profile_id
JOIN channel_health h ON h.merchant_channel_id = mc.id
	LEFT JOIN usage_stats stats ON stats.merchant_profile_id = o.merchant_profile_id
	    AND stats.canonical_model_id = o.canonical_model_id
	LEFT JOIN metric_stats metric ON metric.merchant_profile_id = o.merchant_profile_id
	    AND metric.canonical_model_id = o.canonical_model_id
WHERE o.status = 'active' AND mc.status = 'active' AND mp.status = 'active'
  AND cm.status = 'active'
  AND ($1 = '' OR cm.public_name = $1)
	GROUP BY o.id, mp.id, mp.display_name, cm.public_name,
	         stats.call_count, stats.token_usage, stats.last_called_at,
	         metric.prompt_tokens, metric.cache_tokens, metric.metrics_updated_at
HAVING count(*) FILTER (WHERE h.state IN ('healthy', 'degraded')) > 0
ORDER BY cm.public_name, o.multiplier_bps, o.id
`, strings.TrimSpace(publicModel))
	if err != nil {
		return nil, fmt.Errorf("查询市场报价：%w", err)
	}
	defer rows.Close()
	offers := make([]MarketOffer, 0)
	for rows.Next() {
		var offer MarketOffer
		var priceID int64
		var metricsUpdatedAt sql.NullTime
		if err := rows.Scan(
			&offer.OfferID, &offer.MerchantID, &offer.MerchantName, &offer.Model,
			&offer.MultiplierBPS, &priceID, &offer.ChannelCount,
			&offer.HealthState, &offer.AvailabilityPPM, &offer.TTFTP95MS,
			&offer.CallCount, &offer.TokenUsage, &offer.LastCalledAt,
			&offer.CacheHitPPM, &offer.CacheSampleInputTokens, &offer.CacheWeightReady, &metricsUpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("解析市场报价：%w", err)
		}
		if metricsUpdatedAt.Valid {
			value := metricsUpdatedAt.Time.UTC()
			offer.MetricsUpdatedAt = &value
		}
		offer.OfficialPrice, err = loadOfficialPrice(ctx, s.database, priceID)
		if err != nil {
			return nil, fmt.Errorf("读取市场报价价格：%w", err)
		}
		if err := applyMarketPrice(&offer); err != nil {
			return nil, fmt.Errorf("换算市场报价：%w", err)
		}
		offers = append(offers, offer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历市场报价：%w", err)
	}
	return offers, nil
}

func applyMarketPrice(offer *MarketOffer) error {
	if offer.OfficialPrice.PriceKind == "image" {
		_, prices, err := buildImageExpression(offer.OfficialPrice.ImagePrices, offer.MultiplierBPS)
		offer.ImagePrices = prices
		return err
	}
	_, tiers, err := buildTieredExpression(offer.OfficialPrice, offer.MultiplierBPS)
	offer.PriceTiers = tiers
	return err
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
UPDATE channel_health
SET state = 'healthy', consecutive_failures = 0, cooldown_level = 0,
    next_probe_at = NULL, cooldown_until = NULL, updated_at = now()
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
	price = normalizeOfficialPrice(price)
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return OfficialPrice{}, fmt.Errorf("开始官方价格事务：%w", err)
	}
	defer tx.Rollback()
	var currentPriceKind string
	err = tx.QueryRowContext(ctx, `
SELECT current.price_kind
FROM canonical_models model
JOIN official_price_versions current ON current.id = model.current_official_price_version_id
WHERE model.id = $1
FOR UPDATE OF model, current
`, modelID).Scan(&currentPriceKind)
	if errors.Is(err, sql.ErrNoRows) {
		return OfficialPrice{}, ErrRoutingNotFound
	}
	if err != nil {
		return OfficialPrice{}, fmt.Errorf("锁定标准模型：%w", err)
	}
	if price.PriceKind != currentPriceKind || validateAbsolutePrice(price) != nil {
		return OfficialPrice{}, ErrRoutingInvalidState
	}
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(max(version), 0) + 1 FROM official_price_versions WHERE canonical_model_id = $1
`, modelID).Scan(&price.Version); err != nil {
		return OfficialPrice{}, fmt.Errorf("生成官方价格版本：%w", err)
	}
	price, err = insertOfficialPriceVersion(ctx, tx, modelID, price.Version, adminID, price)
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
SELECT cm.id, cm.public_name, cm.provider, cm.status, cm.created_at,
       cm.current_official_price_version_id
FROM canonical_models cm
WHERE cm.current_official_price_version_id IS NOT NULL
ORDER BY cm.public_name
`)
	if err != nil {
		return nil, fmt.Errorf("查询标准模型：%w", err)
	}
	defer rows.Close()
	models := make([]CanonicalModel, 0)
	for rows.Next() {
		var model CanonicalModel
		var priceID int64
		if err := rows.Scan(
			&model.ID, &model.PublicName, &model.Provider, &model.Status,
			&model.CreatedAt, &priceID,
		); err != nil {
			return nil, fmt.Errorf("解析标准模型：%w", err)
		}
		model.Price, err = loadOfficialPrice(ctx, s.database, priceID)
		if err != nil {
			return nil, fmt.Errorf("读取标准模型价格：%w", err)
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
WITH latest AS (
    SELECT DISTINCT ON (merchant_profile_id, canonical_model_id) id
    FROM offer_versions
    ORDER BY merchant_profile_id, canonical_model_id, version DESC
), usage_rows AS (
    SELECT source_offer.merchant_profile_id, source_offer.canonical_model_id,
           sources.source_type, sources.status, sources.source_created_at,
           CASE WHEN other->>'prompt_tokens' ~ '^[0-9]+$' THEN (other->>'prompt_tokens')::bigint ELSE 0 END AS prompt_tokens,
           CASE WHEN other->>'completion_tokens' ~ '^[0-9]+$' THEN (other->>'completion_tokens')::bigint ELSE 0 END AS completion_tokens,
           CASE WHEN other->>'total_tokens' ~ '^[0-9]+$' THEN (other->>'total_tokens')::bigint ELSE 0 END AS total_tokens
    FROM settlement_sources sources
    JOIN offer_versions source_offer ON source_offer.id = sources.offer_version_id
), usage_stats AS (
    SELECT merchant_profile_id, canonical_model_id,
           count(*) FILTER (WHERE source_type = 2 AND status = 'settled') AS call_count,
           COALESCE(sum(
               CASE WHEN source_type = 2 AND status = 'settled' THEN
                   CASE WHEN prompt_tokens + completion_tokens > 0
                        THEN prompt_tokens + completion_tokens
                        ELSE total_tokens END
               ELSE 0 END
           ), 0) AS token_usage,
           max(source_created_at) FILTER (WHERE source_type = 2 AND status = 'settled') AS last_called_at
    FROM usage_rows
    GROUP BY merchant_profile_id, canonical_model_id
)
SELECT o.id, cm.public_name, o.multiplier_bps, o.status,
       COALESCE(o.sync_error, ''), o.official_price_version_id,
       COALESCE(stats.call_count, 0), COALESCE(stats.token_usage, 0), stats.last_called_at,
       count(*) FILTER (WHERE mc.status = 'active' AND h.state IN ('healthy', 'degraded'))
           OVER (PARTITION BY o.id),
	       mc.id, mc.channel_name, COALESCE(h.state, 'healthy'),
	       COALESCE(h.availability_ppm, 0), COALESCE(h.ttft_p95_ms, 0),
	       COALESCE(metric.cache_hit_ppm, 0), COALESCE(metric.prompt_tokens, 0),
	       COALESCE(metric.cache_weight_ready, false), metric.metrics_updated_at
FROM offer_versions o
JOIN latest ON latest.id = o.id
JOIN offer_channel_aliases aliases ON aliases.offer_version_id = o.id
JOIN merchant_channels mc ON mc.id = aliases.merchant_channel_id
JOIN merchant_profiles mp ON mp.id = mc.merchant_profile_id
JOIN canonical_models cm ON cm.id = o.canonical_model_id
	LEFT JOIN channel_health h ON h.merchant_channel_id = mc.id
	LEFT JOIN route_metric_snapshots metric ON metric.merchant_channel_id = mc.id
	    AND metric.canonical_model_id = o.canonical_model_id
LEFT JOIN usage_stats stats ON stats.merchant_profile_id = o.merchant_profile_id
                           AND stats.canonical_model_id = o.canonical_model_id
WHERE mp.newapi_user_id = $1 AND o.status <> 'deleted'
ORDER BY o.id DESC, mc.id
`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询商户报价：%w", err)
	}
	defer rows.Close()
	offers := make([]MerchantOfferView, 0)
	indexes := make(map[int64]int)
	for rows.Next() {
		var offer MerchantOfferView
		var priceID int64
		var lastCalledAt sql.NullTime
		var metricsUpdatedAt sql.NullTime
		var channel OfferChannelView
		if err := rows.Scan(
			&offer.OfferID, &offer.Model, &offer.MultiplierBPS,
			&offer.OfferStatus, &offer.SyncError, &priceID, &offer.CallCount, &offer.TokenUsage, &lastCalledAt, &offer.ChannelCount,
			&channel.MerchantChannelID, &channel.ChannelName, &channel.HealthState,
			&channel.AvailabilityPPM, &channel.TTFTP95MS,
			&channel.CacheHitPPM, &channel.CacheSampleInputTokens, &channel.CacheWeightReady, &metricsUpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("解析商户报价：%w", err)
		}
		if lastCalledAt.Valid {
			value := lastCalledAt.Time.UTC()
			offer.LastCalledAt = &value
		}
		if metricsUpdatedAt.Valid {
			value := metricsUpdatedAt.Time.UTC()
			channel.MetricsUpdatedAt = &value
		}
		if index, exists := indexes[offer.OfferID]; exists {
			offers[index].Channels = append(offers[index].Channels, channel)
			continue
		}
		offer.OfficialPrice, err = loadOfficialPrice(ctx, s.database, priceID)
		if err != nil {
			return nil, fmt.Errorf("读取商户报价价格：%w", err)
		}
		offer.Channels = []OfferChannelView{channel}
		offer.HealthState = channel.HealthState
		offer.AvailabilityPPM = channel.AvailabilityPPM
		offer.TTFTP95MS = channel.TTFTP95MS
		indexes[offer.OfferID] = len(offers)
		offers = append(offers, offer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历商户报价：%w", err)
	}
	for index := range offers {
		applyMerchantOfferCacheMetrics(&offers[index])
		if offers[index].OfficialPrice.PriceKind == "image" {
			_, prices, err := buildImageExpression(offers[index].OfficialPrice.ImagePrices, offers[index].MultiplierBPS)
			if err != nil {
				return nil, fmt.Errorf("换算商户图片报价：%w", err)
			}
			offers[index].ImagePrices = prices
			continue
		}
		_, tiers, err := buildTieredExpression(offers[index].OfficialPrice, offers[index].MultiplierBPS)
		if err != nil {
			return nil, fmt.Errorf("换算商户报价：%w", err)
		}
		offers[index].PriceTiers = tiers
	}
	return offers, nil
}

func applyMerchantOfferCacheMetrics(offer *MerchantOfferView) {
	var cacheTokens int64
	for _, channel := range offer.Channels {
		offer.CacheSampleInputTokens += channel.CacheSampleInputTokens
		cacheTokens += int64(channel.CacheHitPPM) * channel.CacheSampleInputTokens
		if channel.MetricsUpdatedAt != nil && (offer.MetricsUpdatedAt == nil || channel.MetricsUpdatedAt.After(*offer.MetricsUpdatedAt)) {
			value := *channel.MetricsUpdatedAt
			offer.MetricsUpdatedAt = &value
		}
	}
	if offer.CacheSampleInputTokens > 0 {
		offer.CacheHitPPM = int(cacheTokens / offer.CacheSampleInputTokens)
	}
	offer.CacheWeightReady = offer.CacheSampleInputTokens >= cacheWeightMinimumInputTokens
}

func assignNullableTierPrices(tier *PriceTier, maximum, output, cacheRead, cacheWrite sql.NullInt64) {
	tier.MaximumInputTokens = nullInt64Pointer(maximum)
	tier.OutputPriceMicros = nullInt64Pointer(output)
	tier.CacheReadPriceMicros = nullInt64Pointer(cacheRead)
	tier.CacheWritePriceMicros = nullInt64Pointer(cacheWrite)
}

func (s *SQLRoutingStore) ImagePriceItems(ctx context.Context, modelID int64) ([]ImagePriceItem, error) {
	var priceID int64
	err := s.database.QueryRowContext(ctx, `
SELECT price.id
FROM canonical_models model
JOIN official_price_versions price ON price.id = model.current_official_price_version_id
WHERE model.id = $1 AND model.status = 'active' AND price.price_kind = 'image'
`, modelID).Scan(&priceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRoutingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("读取图片模型：%w", err)
	}
	price, err := loadOfficialPrice(ctx, s.database, priceID)
	if err != nil {
		return nil, fmt.Errorf("读取图片价格矩阵：%w", err)
	}
	return price.ImagePrices, nil
}

func (s *SQLRoutingStore) DeleteCanonicalModel(ctx context.Context, modelID int64) (string, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("开始删除模型事务：%w", err)
	}
	defer tx.Rollback()
	var referenced bool
	err = tx.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM merchant_channel_tests WHERE canonical_model_id = $1
    UNION ALL SELECT 1 FROM merchant_channel_reviews WHERE canonical_model_id = $1
    UNION ALL SELECT 1 FROM merchant_channels WHERE canonical_model_id = $1
    UNION ALL SELECT 1 FROM merchant_channel_model_bindings WHERE canonical_model_id = $1
    UNION ALL SELECT 1 FROM offer_versions WHERE canonical_model_id = $1
)
FROM canonical_models
WHERE id = $1
FOR UPDATE
`, modelID).Scan(&referenced)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrRoutingNotFound
	}
	if err != nil {
		return "", fmt.Errorf("锁定待删除模型：%w", err)
	}
	result := "archived"
	if referenced {
		if _, err := tx.ExecContext(ctx, `UPDATE canonical_models SET status = 'archived', updated_at = now() WHERE id = $1`, modelID); err != nil {
			return "", fmt.Errorf("归档模型：%w", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE offer_versions SET status = 'inactive', retired_at = now() WHERE canonical_model_id = $1 AND status IN ('active', 'syncing')`, modelID); err != nil {
			return "", fmt.Errorf("下架模型报价：%w", err)
		}
	} else {
		result = "deleted"
		if _, err := tx.ExecContext(ctx, `UPDATE canonical_models SET current_official_price_version_id = NULL WHERE id = $1`, modelID); err != nil {
			return "", fmt.Errorf("解除模型价格版本：%w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM official_price_versions WHERE canonical_model_id = $1`, modelID); err != nil {
			return "", fmt.Errorf("删除模型价格：%w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM canonical_models WHERE id = $1`, modelID); err != nil {
			return "", fmt.Errorf("删除模型：%w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("提交删除模型事务：%w", err)
	}
	return result, nil
}

func (s *SQLRoutingStore) PublicModelNames(ctx context.Context, aliases []string) (map[string]string, error) {
	result := make(map[string]string)
	if len(aliases) == 0 {
		return result, nil
	}
	unique := make([]string, 0, len(aliases))
	seen := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		if alias == "" {
			continue
		}
		if _, exists := seen[alias]; exists {
			continue
		}
		seen[alias] = struct{}{}
		unique = append(unique, alias)
	}
	if len(unique) == 0 {
		return result, nil
	}
	placeholders := make([]string, len(unique))
	arguments := make([]any, len(unique))
	for index, alias := range unique {
		placeholders[index] = fmt.Sprintf("$%d", index+1)
		arguments[index] = alias
	}
	rows, err := s.database.QueryContext(ctx, `
SELECT aliases.internal_alias, cm.public_name
FROM offer_channel_aliases aliases
JOIN offer_versions o ON o.id = aliases.offer_version_id
JOIN canonical_models cm ON cm.id = o.canonical_model_id
WHERE aliases.internal_alias IN (`+strings.Join(placeholders, ",")+`)
`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("查询内部别名公开模型：%w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var alias, model string
		if err := rows.Scan(&alias, &model); err != nil {
			return nil, fmt.Errorf("解析内部别名公开模型：%w", err)
		}
		result[alias] = model
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历内部别名公开模型：%w", err)
	}
	return result, nil
}

func timeOrNil(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
