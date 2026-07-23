package httpapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type UsageRecordQuery struct {
	Admin                bool
	ConsumerUserID       int
	MerchantUserID       int
	FilterMerchantUserID int
	ChannelID            int64
	ModelName            string
	StartTimestamp       int64
	EndTimestamp         int64
	Page                 int
	PageSize             int
}

type UsageRecord struct {
	ID                  int64     `json:"id"`
	CreatedAt           time.Time `json:"created_at"`
	ConsumerUserID      int       `json:"user_id"`
	ModelName           string    `json:"model_name"`
	TokenID             int       `json:"token_id"`
	PromptTokens        int64     `json:"prompt_tokens"`
	CompletionTokens    int64     `json:"completion_tokens"`
	CacheTokens         int64     `json:"cache_tokens"`
	GrossQuota          int64     `json:"quota"`
	MerchantIncomeQuota int64     `json:"merchant_income_quota"`
	MerchantUserID      int       `json:"merchant_user_id"`
	MerchantName        string    `json:"merchant_name"`
	ChannelID           int64     `json:"channel_id"`
	ChannelName         string    `json:"channel_name"`
	RequestID           string    `json:"request_id"`
}

type UsageMerchantOption struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
}

type UsageRecordFilters struct {
	Models    []string              `json:"models"`
	Merchants []UsageMerchantOption `json:"merchants"`
}

type UsageRecordPage struct {
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int                `json:"total"`
	Items    []UsageRecord      `json:"items"`
	Filters  UsageRecordFilters `json:"-"`
}

type UsageRecordReader interface {
	ListUsageRecords(context.Context, UsageRecordQuery) (UsageRecordPage, error)
}

func (s *SQLSettlementStore) ListUsageRecords(ctx context.Context, query UsageRecordQuery) (UsageRecordPage, error) {
	if !query.Admin && query.ConsumerUserID <= 0 && query.MerchantUserID <= 0 {
		return UsageRecordPage{}, errors.New("调用记录查询缺少用户范围")
	}
	if query.Admin && (query.ConsumerUserID > 0 || query.MerchantUserID > 0) || query.ConsumerUserID > 0 && query.MerchantUserID > 0 {
		return UsageRecordPage{}, errors.New("调用记录查询范围冲突")
	}
	query.Page, query.PageSize = usagePagination(query.Page, query.PageSize)
	query.ModelName = strings.TrimSpace(query.ModelName)
	merchantUserID := query.MerchantUserID
	if query.FilterMerchantUserID > 0 {
		merchantUserID = query.FilterMerchantUserID
	}
	startTime, endTime := usageRecordTimeArguments(query)
	arguments := []any{
		query.ConsumerUserID, merchantUserID, query.ModelName,
		startTime, endTime, query.ChannelID,
	}
	const scope = `
FROM settlement_sources source
JOIN merchant_profiles merchant ON merchant.id = source.merchant_profile_id
JOIN offer_versions offer ON offer.id = source.offer_version_id
JOIN canonical_models model ON model.id = offer.canonical_model_id
JOIN offer_channel_aliases alias ON alias.id = source.offer_channel_alias_id
JOIN merchant_channels channel ON channel.id = alias.merchant_channel_id
WHERE source.source_type = 2 AND source.status = 'settled'
  AND ($1 = 0 OR source.newapi_user_id = $1)
  AND ($2 = 0 OR merchant.newapi_user_id = $2)
  AND ($3 = '' OR model.public_name = $3)
  AND ($4::timestamptz IS NULL OR source.source_created_at >= $4::timestamptz)
  AND ($5::timestamptz IS NULL OR source.source_created_at <= $5::timestamptz)
  AND ($6 = 0 OR channel.id = $6)
`
	var total int
	if err := s.database.QueryRowContext(ctx, "SELECT count(*) "+scope, arguments...).Scan(&total); err != nil {
		return UsageRecordPage{}, fmt.Errorf("统计调用记录：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, `
SELECT source.newapi_log_id, source.source_created_at, source.newapi_user_id, model.public_name,
       source.newapi_token_id,
       CASE WHEN source.other->>'prompt_tokens' ~ '^[0-9]+$' THEN (source.other->>'prompt_tokens')::bigint ELSE 0 END,
       CASE WHEN source.other->>'completion_tokens' ~ '^[0-9]+$' THEN (source.other->>'completion_tokens')::bigint ELSE 0 END,
       CASE WHEN source.other->>'cache_tokens' ~ '^[0-9]+$' THEN (source.other->>'cache_tokens')::bigint ELSE 0 END,
       source.gross_quota, source.merchant_quota,
       merchant.newapi_user_id, merchant.display_name,
       channel.id, channel.channel_name, COALESCE(source.request_id, '')
`+scope+`
ORDER BY source.source_created_at DESC, source.newapi_log_id DESC
LIMIT $7 OFFSET $8
`, append(arguments, query.PageSize, (query.Page-1)*query.PageSize)...)
	if err != nil {
		return UsageRecordPage{}, fmt.Errorf("查询调用记录：%w", err)
	}
	defer rows.Close()
	items := make([]UsageRecord, 0, query.PageSize)
	for rows.Next() {
		var item UsageRecord
		if err := rows.Scan(
			&item.ID, &item.CreatedAt, &item.ConsumerUserID, &item.ModelName, &item.TokenID,
			&item.PromptTokens, &item.CompletionTokens, &item.CacheTokens,
			&item.GrossQuota, &item.MerchantIncomeQuota,
			&item.MerchantUserID, &item.MerchantName,
			&item.ChannelID, &item.ChannelName, &item.RequestID,
		); err != nil {
			return UsageRecordPage{}, fmt.Errorf("读取调用记录：%w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return UsageRecordPage{}, fmt.Errorf("遍历调用记录：%w", err)
	}
	filters, err := s.usageRecordFilters(ctx, query)
	if err != nil {
		return UsageRecordPage{}, err
	}
	return UsageRecordPage{Page: query.Page, PageSize: query.PageSize, Total: total, Items: items, Filters: filters}, nil
}

func (s *SQLSettlementStore) usageRecordFilters(ctx context.Context, query UsageRecordQuery) (UsageRecordFilters, error) {
	filters := UsageRecordFilters{Models: []string{}, Merchants: []UsageMerchantOption{}}
	merchantUserID := query.MerchantUserID
	if query.FilterMerchantUserID > 0 {
		merchantUserID = query.FilterMerchantUserID
	}
	startTime, endTime := usageRecordTimeArguments(query)
	modelRows, err := s.database.QueryContext(ctx, `
SELECT DISTINCT model.public_name
FROM settlement_sources source
JOIN merchant_profiles merchant ON merchant.id = source.merchant_profile_id
JOIN offer_versions offer ON offer.id = source.offer_version_id
JOIN canonical_models model ON model.id = offer.canonical_model_id
JOIN offer_channel_aliases alias ON alias.id = source.offer_channel_alias_id
JOIN merchant_channels channel ON channel.id = alias.merchant_channel_id
WHERE source.source_type = 2 AND source.status = 'settled'
  AND ($1 = 0 OR source.newapi_user_id = $1)
  AND ($2 = 0 OR merchant.newapi_user_id = $2)
  AND ($3::timestamptz IS NULL OR source.source_created_at >= $3::timestamptz)
  AND ($4::timestamptz IS NULL OR source.source_created_at <= $4::timestamptz)
  AND ($5 = 0 OR channel.id = $5)
ORDER BY model.public_name
`, query.ConsumerUserID, merchantUserID, startTime, endTime, query.ChannelID)
	if err != nil {
		return UsageRecordFilters{}, fmt.Errorf("查询调用模型筛选项：%w", err)
	}
	for modelRows.Next() {
		var model string
		if err := modelRows.Scan(&model); err != nil {
			modelRows.Close()
			return UsageRecordFilters{}, fmt.Errorf("读取调用模型筛选项：%w", err)
		}
		filters.Models = append(filters.Models, model)
	}
	if err := modelRows.Close(); err != nil {
		return UsageRecordFilters{}, fmt.Errorf("关闭调用模型筛选项：%w", err)
	}
	if query.MerchantUserID > 0 {
		return filters, nil
	}
	merchantRows, err := s.database.QueryContext(ctx, `
SELECT DISTINCT merchant.newapi_user_id, merchant.display_name
FROM settlement_sources source
JOIN merchant_profiles merchant ON merchant.id = source.merchant_profile_id
WHERE source.source_type = 2 AND source.status = 'settled'
  AND ($1 = 0 OR source.newapi_user_id = $1)
  AND ($2::timestamptz IS NULL OR source.source_created_at >= $2::timestamptz)
  AND ($3::timestamptz IS NULL OR source.source_created_at <= $3::timestamptz)
  AND ($4 = 0 OR EXISTS (
      SELECT 1 FROM offer_channel_aliases selected_alias
      JOIN merchant_channels selected_channel ON selected_channel.id = selected_alias.merchant_channel_id
      WHERE selected_alias.id = source.offer_channel_alias_id AND selected_channel.id = $4
  ))
ORDER BY merchant.display_name, merchant.newapi_user_id
`, query.ConsumerUserID, startTime, endTime, query.ChannelID)
	if err != nil {
		return UsageRecordFilters{}, fmt.Errorf("查询调用商户筛选项：%w", err)
	}
	defer merchantRows.Close()
	for merchantRows.Next() {
		var option UsageMerchantOption
		if err := merchantRows.Scan(&option.UserID, &option.Name); err != nil {
			return UsageRecordFilters{}, fmt.Errorf("读取调用商户筛选项：%w", err)
		}
		filters.Merchants = append(filters.Merchants, option)
	}
	if err := merchantRows.Err(); err != nil {
		return UsageRecordFilters{}, fmt.Errorf("遍历调用商户筛选项：%w", err)
	}
	return filters, nil
}

func usagePagination(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	switch pageSize {
	case 20, 50, 200:
	default:
		pageSize = 20
	}
	return page, pageSize
}

func usageRecordTimeArguments(query UsageRecordQuery) (any, any) {
	var startTime any
	var endTime any
	if query.StartTimestamp > 0 {
		startTime = time.Unix(query.StartTimestamp, 0).UTC()
	}
	if query.EndTimestamp > 0 {
		endTime = time.Unix(query.EndTimestamp, 0).UTC()
	}
	return startTime, endTime
}
