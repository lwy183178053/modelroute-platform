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
	Protocol   string
	AdminID    int
	Price      OfficialPrice
}

type CanonicalModel struct {
	ID          int64         `json:"id"`
	PublicName  string        `json:"public_name"`
	Provider    string        `json:"provider"`
	Protocol    string        `json:"protocol"`
	Status      string        `json:"status"`
	Price       OfficialPrice `json:"price"`
	CreatedAt   time.Time     `json:"created_at,omitempty"`
	AdapterHint string        `json:"adapter_hint"`
}

type RoutingRepository interface {
	CreateCanonicalModel(context.Context, CreateCanonicalModelCommand) (CanonicalModel, error)
	CreateOffer(context.Context, CreateOfferCommand) (OfferVersion, error)
	RetireOffer(context.Context, int, int64) error
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
	DeleteCanonicalModel(context.Context, int64) (string, error)
	ImagePriceItems(context.Context, int64) ([]ImagePriceItem, error)
}

type MarketOffer struct {
	OfferID         int64            `json:"offer_id"`
	MerchantID      int64            `json:"merchant_id"`
	MerchantName    string           `json:"merchant_name"`
	Model           string           `json:"model"`
	Protocol        string           `json:"protocol"`
	MultiplierBPS   int              `json:"multiplier_bps"`
	PriceTiers      []OfferPriceTier `json:"price_tiers"`
	ImagePrices     []ImagePriceItem `json:"image_prices,omitempty"`
	ChannelCount    int              `json:"channel_count"`
	HealthState     string           `json:"health_state"`
	AvailabilityPPM int              `json:"availability_ppm"`
	TTFTP95MS       int              `json:"ttft_p95_ms"`
	InternalAlias   string           `json:"-"`
	OfficialPrice   OfficialPrice    `json:"-"`
}

type MerchantOfferView struct {
	OfferID         int64              `json:"offer_id"`
	Model           string             `json:"model"`
	Protocol        string             `json:"protocol"`
	MultiplierBPS   int                `json:"multiplier_bps"`
	ChannelCount    int                `json:"channel_count"`
	CallCount       int64              `json:"call_count"`
	TokenUsage      int64              `json:"token_usage"`
	LastCalledAt    *time.Time         `json:"last_called_at,omitempty"`
	PriceTiers      []OfferPriceTier   `json:"price_tiers"`
	ImagePrices     []ImagePriceItem   `json:"image_prices,omitempty"`
	Channels        []OfferChannelView `json:"channels"`
	OfferStatus     string             `json:"offer_status"`
	HealthState     string             `json:"health_state"`
	AvailabilityPPM int                `json:"availability_ppm"`
	TTFTP95MS       int                `json:"ttft_p95_ms"`
	SyncError       string             `json:"sync_error,omitempty"`
	InternalAlias   string             `json:"-"`
	OfficialPrice   OfficialPrice      `json:"-"`
}

type OfferChannelView struct {
	MerchantChannelID int64  `json:"merchant_channel_id"`
	ChannelName       string `json:"channel_name"`
	HealthState       string `json:"health_state"`
	AvailabilityPPM   int    `json:"availability_ppm"`
	TTFTP95MS         int    `json:"ttft_p95_ms"`
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
	RequestID           string
	TokenBindingID      int64
	CanonicalModelID    int64
	OfferChannelAliasID int64
	Mode                string
	Priority            string
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
	SyncOffer(context.Context, int64) error
}

type OfferVersion struct {
	ID                     int64               `json:"id"`
	CanonicalModelID       int64               `json:"canonical_model_id"`
	OfficialPriceVersionID int64               `json:"official_price_version_id"`
	PreviousOfferID        int64               `json:"previous_offer_id,omitempty"`
	MultiplierBPS          int                 `json:"multiplier_bps"`
	Status                 string              `json:"status"`
	PublicModel            string              `json:"model"`
	Protocol               string              `json:"protocol"`
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
	model := CanonicalModel{PublicName: command.PublicName, Provider: command.Provider, Protocol: command.Protocol, AdapterHint: "auto", Status: "active", Price: command.Price}
	err = tx.QueryRowContext(ctx, `
INSERT INTO canonical_models (public_name, provider, protocol, adapter_hint, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, status, created_at
`, command.PublicName, command.Provider, command.Protocol, model.AdapterHint, model.Status).Scan(&model.ID, &model.Status, &model.CreatedAt)
	if err != nil {
		return CanonicalModel{}, fmt.Errorf("创建标准模型：%w", err)
	}
	model.Price, err = insertOfficialPriceVersion(ctx, tx, model.ID, 1, command.AdminID, command.Protocol, command.Price)
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

func insertOfficialPriceVersion(ctx context.Context, tx *sql.Tx, modelID int64, version int, adminID int, protocol string, price OfficialPrice) (OfficialPrice, error) {
	price = normalizeOfficialPrice(price)
	if err := validateOfficialPrice(price, protocol); err != nil {
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
SELECT mp.id, cm.current_official_price_version_id, cm.public_name, cm.protocol,
	   COALESCE((SELECT id FROM offer_versions old
	             WHERE old.merchant_profile_id = mp.id AND old.canonical_model_id = cm.id AND old.status = 'active'
	             ORDER BY old.id DESC LIMIT 1), 0),
	   COALESCE((SELECT max(version) + 1 FROM offer_versions versions
	             WHERE versions.merchant_profile_id = mp.id AND versions.canonical_model_id = cm.id), 1)
FROM merchant_profiles mp
JOIN canonical_models cm ON cm.id = $2
WHERE mp.newapi_user_id = $1 AND mp.status = 'active'
  AND cm.status = 'active' AND cm.current_official_price_version_id IS NOT NULL
FOR UPDATE OF mp, cm
`, command.UserID, command.CanonicalModelID).Scan(
		&offer.MerchantID, &offer.OfficialPriceVersionID, &offer.PublicModel,
		&offer.Protocol, &offer.PreviousOfferID, &offer.Version,
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

func (s *SQLRoutingStore) ActivateOffer(ctx context.Context, offerID int64) error {
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
	result, err := tx.ExecContext(ctx, `UPDATE offer_versions SET status = 'active', activated_at = now(), sync_error = NULL WHERE id = $1 AND status = 'syncing'`, offerID)
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
SELECT aliases.internal_alias, cm.protocol, o.multiplier_bps, o.official_price_version_id
FROM offer_channel_aliases aliases
JOIN offer_versions o ON o.id = aliases.offer_version_id
JOIN canonical_models cm ON cm.id = o.canonical_model_id
WHERE o.status <> 'sync_failed'
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
		if err := rows.Scan(&price.Alias, &price.Protocol, &price.MultiplierBPS, &priceID); err != nil {
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
SELECT o.id, aliases.id, mp.id, mc.id, mc.newapi_channel_id, cm.id, aliases.internal_alias,
       cm.public_name, cm.protocol, o.multiplier_bps, h.state,
       h.availability_ppm, h.ttft_p95_ms
FROM routing_policies p
JOIN canonical_models cm ON cm.id = $2
JOIN offer_versions o ON o.canonical_model_id = cm.id AND o.status = 'active'
JOIN offer_channel_aliases aliases ON aliases.offer_version_id = o.id
JOIN merchant_channels mc ON mc.id = aliases.merchant_channel_id AND mc.status = 'active'
JOIN merchant_profiles mp ON mp.id = mc.merchant_profile_id AND mp.status = 'active'
JOIN channel_health h ON h.merchant_channel_id = mc.id
WHERE p.id = $1
  AND (mc.canonical_model_id = $2 OR EXISTS (
      SELECT 1 FROM merchant_channel_model_bindings binding
      WHERE binding.merchant_channel_id = mc.id
        AND binding.canonical_model_id = $2 AND binding.match_status = 'matched'
  ))
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
			&candidate.OfferID, &candidate.OfferChannelAliasID, &candidate.MerchantID, &candidate.MerchantChannelID,
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
    (request_id, token_binding_id, canonical_model_id, offer_channel_alias_id, mode, priority)
VALUES ($1, $2, $3, NULLIF($4, 0), $5, $6)
`, decision.RequestID, decision.TokenBindingID, decision.CanonicalModelID, decision.OfferChannelAliasID, decision.Mode, decision.Priority)
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

func (s *SQLRoutingStore) ClaimHealthProbes(ctx context.Context, now time.Time, limit int) ([]HealthProbe, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.database.QueryContext(ctx, `
WITH due AS (
    SELECT h.merchant_channel_id, mc.canonical_model_id, route.offer_id,
           route.offer_channel_alias_id, mc.newapi_channel_id, route.internal_alias,
           cm.protocol, h.state, h.consecutive_failures, h.cooldown_level
    FROM channel_health h
    JOIN merchant_channels mc ON mc.id = h.merchant_channel_id AND mc.status = 'active'
    JOIN canonical_models cm ON cm.id = mc.canonical_model_id AND cm.status = 'active'
    JOIN LATERAL (
        SELECT o.id AS offer_id, aliases.id AS offer_channel_alias_id, aliases.internal_alias
        FROM offer_channel_aliases aliases
        JOIN offer_versions o ON o.id = aliases.offer_version_id
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
       d.internal_alias, d.protocol, d.state, d.consecutive_failures,
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
			&probe.NewAPIChannelID, &probe.InternalAlias, &probe.Protocol, &probe.State,
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
SELECT o.id, mp.id, mp.display_name, cm.public_name, cm.protocol,
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
       COALESCE(max(h.ttft_p95_ms) FILTER (WHERE h.state IN ('healthy', 'degraded')), 0)
FROM offer_versions o
JOIN canonical_models cm ON cm.id = o.canonical_model_id
JOIN offer_channel_aliases aliases ON aliases.offer_version_id = o.id
JOIN merchant_channels mc ON mc.id = aliases.merchant_channel_id
JOIN merchant_profiles mp ON mp.id = mc.merchant_profile_id
JOIN channel_health h ON h.merchant_channel_id = mc.id
WHERE o.status = 'active' AND mc.status = 'active' AND mp.status = 'active'
  AND cm.status = 'active'
  AND ($1 = '' OR cm.public_name = $1)
GROUP BY o.id, mp.id, mp.display_name, cm.public_name, cm.protocol
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
		if err := rows.Scan(
			&offer.OfferID, &offer.MerchantID, &offer.MerchantName, &offer.Model,
			&offer.Protocol, &offer.MultiplierBPS, &priceID, &offer.ChannelCount,
			&offer.HealthState, &offer.AvailabilityPPM, &offer.TTFTP95MS,
		); err != nil {
			return nil, fmt.Errorf("解析市场报价：%w", err)
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
	if offer.Protocol == "image_generation" {
		_, prices, err := buildImageExpression(offer.OfficialPrice.ImagePrices, offer.MultiplierBPS)
		offer.ImagePrices = prices
		return err
	}
	_, tiers, err := buildTieredExpression(offer.OfficialPrice, offer.MultiplierBPS, offer.Protocol)
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
	var protocol string
	err = tx.QueryRowContext(ctx, `SELECT protocol FROM canonical_models WHERE id = $1 FOR UPDATE`, modelID).Scan(&protocol)
	if errors.Is(err, sql.ErrNoRows) {
		return OfficialPrice{}, ErrRoutingNotFound
	}
	if err != nil {
		return OfficialPrice{}, fmt.Errorf("锁定标准模型：%w", err)
	}
	if err := validateAbsolutePrice(price, protocol); err != nil {
		return OfficialPrice{}, ErrRoutingInvalidState
	}
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(max(version), 0) + 1 FROM official_price_versions WHERE canonical_model_id = $1
`, modelID).Scan(&price.Version); err != nil {
		return OfficialPrice{}, fmt.Errorf("生成官方价格版本：%w", err)
	}
	price, err = insertOfficialPriceVersion(ctx, tx, modelID, price.Version, adminID, protocol, price)
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
       cm.adapter_hint, cm.current_official_price_version_id
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
			&model.ID, &model.PublicName, &model.Provider, &model.Protocol, &model.Status,
			&model.CreatedAt, &model.AdapterHint, &priceID,
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
SELECT o.id, cm.public_name, cm.protocol, o.multiplier_bps, o.status,
       COALESCE(o.sync_error, ''), o.official_price_version_id,
       COALESCE(stats.call_count, 0), COALESCE(stats.token_usage, 0), stats.last_called_at,
       count(*) FILTER (WHERE mc.status = 'active' AND h.state IN ('healthy', 'degraded'))
           OVER (PARTITION BY o.id),
       mc.id, mc.channel_name, COALESCE(h.state, 'healthy'),
       COALESCE(h.availability_ppm, 0), COALESCE(h.ttft_p95_ms, 0)
FROM offer_versions o
JOIN latest ON latest.id = o.id
JOIN offer_channel_aliases aliases ON aliases.offer_version_id = o.id
JOIN merchant_channels mc ON mc.id = aliases.merchant_channel_id
JOIN merchant_profiles mp ON mp.id = mc.merchant_profile_id
JOIN canonical_models cm ON cm.id = o.canonical_model_id
LEFT JOIN channel_health h ON h.merchant_channel_id = mc.id
LEFT JOIN usage_stats stats ON stats.merchant_profile_id = o.merchant_profile_id
                           AND stats.canonical_model_id = o.canonical_model_id
WHERE mp.newapi_user_id = $1
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
		var channel OfferChannelView
		if err := rows.Scan(
			&offer.OfferID, &offer.Model, &offer.Protocol, &offer.MultiplierBPS,
			&offer.OfferStatus, &offer.SyncError, &priceID, &offer.CallCount, &offer.TokenUsage, &lastCalledAt, &offer.ChannelCount,
			&channel.MerchantChannelID, &channel.ChannelName, &channel.HealthState,
			&channel.AvailabilityPPM, &channel.TTFTP95MS,
		); err != nil {
			return nil, fmt.Errorf("解析商户报价：%w", err)
		}
		if lastCalledAt.Valid {
			value := lastCalledAt.Time.UTC()
			offer.LastCalledAt = &value
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
		if offers[index].Protocol == "image_generation" {
			_, prices, err := buildImageExpression(offers[index].OfficialPrice.ImagePrices, offers[index].MultiplierBPS)
			if err != nil {
				return nil, fmt.Errorf("换算商户图片报价：%w", err)
			}
			offers[index].ImagePrices = prices
			continue
		}
		_, tiers, err := buildTieredExpression(offers[index].OfficialPrice, offers[index].MultiplierBPS, offers[index].Protocol)
		if err != nil {
			return nil, fmt.Errorf("换算商户报价：%w", err)
		}
		offers[index].PriceTiers = tiers
	}
	return offers, nil
}

func assignNullableTierPrices(tier *PriceTier, maximum, output, cacheRead, cacheWrite sql.NullInt64) {
	tier.MaximumInputTokens = nullInt64Pointer(maximum)
	tier.OutputPriceMicros = nullInt64Pointer(output)
	tier.CacheReadPriceMicros = nullInt64Pointer(cacheRead)
	tier.CacheWritePriceMicros = nullInt64Pointer(cacheWrite)
}

func (s *SQLRoutingStore) ImagePriceItems(ctx context.Context, modelID int64) ([]ImagePriceItem, error) {
	var priceID int64
	var protocol string
	err := s.database.QueryRowContext(ctx, `
SELECT protocol, current_official_price_version_id
FROM canonical_models
WHERE id = $1 AND status = 'active'
`, modelID).Scan(&protocol, &priceID)
	if errors.Is(err, sql.ErrNoRows) || protocol != "image_generation" {
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
