package httpapi

import (
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"time"
)

const (
	settlementOverlap       = 5 * time.Minute
	settlementSafetyDelay   = 5 * time.Second
	settlementFirstLookback = 24 * time.Hour
)

type SettlementLog struct {
	ID               int64           `json:"id"`
	UserID           int             `json:"user_id"`
	CreatedAt        int64           `json:"created_at"`
	Type             int             `json:"type"`
	ModelName        string          `json:"model_name"`
	Quota            int64           `json:"quota"`
	ChannelID        int             `json:"channel"`
	TokenID          int             `json:"token_id"`
	RequestID        string          `json:"request_id"`
	OtherText        string          `json:"other"`
	PromptTokens     int64           `json:"prompt_tokens"`
	CompletionTokens int64           `json:"completion_tokens"`
	TotalTokens      int64           `json:"total_tokens"`
	Other            json.RawMessage `json:"-"`
	TaskID           string          `json:"-"`
	CacheTokens      int64           `json:"-"`
}

type settlementLogPage struct {
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int             `json:"total"`
	Items    []SettlementLog `json:"items"`
}

func splitSettlementQuota(quota int64) (merchant int64, platform int64) {
	if quota <= 0 {
		return 0, 0
	}
	platform = quota / 10
	return quota - platform, platform
}

func withdrawalBreakdown(amountQuota int64, quotaPerUnit int64, withdrawalTaxBPS int64) (taxQuota int64, netPayoutQuota int64, ok bool) {
	if quotaPerUnit <= 0 || withdrawalTaxBPS < 0 || withdrawalTaxBPS > 10_000 || amountQuota < 50*quotaPerUnit || amountQuota%quotaPerUnit != 0 {
		return 0, 0, false
	}
	taxQuota = amountQuota * withdrawalTaxBPS / 10_000
	return taxQuota, amountQuota - taxQuota, true
}

func refundSplitDelta(gross, merchantTotal, platformTotal, refundedGross, refundGross int64) (int64, int64, error) {
	if gross <= 0 || merchantTotal < 0 || platformTotal < 0 || merchantTotal+platformTotal != gross ||
		refundedGross < 0 || refundGross <= 0 || refundedGross > gross-refundGross {
		return 0, 0, errors.New("退款额度超出原消费")
	}
	previousPlatform := proportionalFloor(platformTotal, refundedGross, gross)
	nextPlatform := proportionalFloor(platformTotal, refundedGross+refundGross, gross)
	platform := nextPlatform - previousPlatform
	return refundGross - platform, platform, nil
}

func proportionalFloor(value, numerator, denominator int64) int64 {
	product := new(big.Int).Mul(big.NewInt(value), big.NewInt(numerator))
	return product.Div(product, big.NewInt(denominator)).Int64()
}

func decodeSettlementLogPage(data json.RawMessage) (settlementLogPage, error) {
	var page settlementLogPage
	if err := json.Unmarshal(data, &page); err != nil {
		return settlementLogPage{}, err
	}
	for index := range page.Items {
		item := &page.Items[index]
		item.Other = json.RawMessage(item.OtherText)
		if len(item.Other) == 0 || !json.Valid(item.Other) {
			item.Other = json.RawMessage(`{}`)
		}
		var metadata struct {
			TaskID           string `json:"task_id"`
			CacheTokens      int64  `json:"cache_tokens"`
			PromptTokens     int64  `json:"prompt_tokens"`
			CompletionTokens int64  `json:"completion_tokens"`
			TotalTokens      int64  `json:"total_tokens"`
		}
		_ = json.Unmarshal(item.Other, &metadata)
		item.TaskID = metadata.TaskID
		item.CacheTokens = metadata.CacheTokens
		item.Other = mergeSettlementTokenMetadata(item.Other, item.PromptTokens, item.CompletionTokens, item.TotalTokens, item.CacheTokens)
	}
	return page, nil
}

func mergeSettlementTokenMetadata(raw json.RawMessage, prompt, completion, total, cache int64) json.RawMessage {
	metadata := map[string]any{}
	if len(raw) > 0 && json.Valid(raw) {
		_ = json.Unmarshal(raw, &metadata)
	}
	for key, value := range map[string]int64{
		"prompt_tokens": prompt, "completion_tokens": completion, "total_tokens": total, "cache_tokens": cache,
	} {
		if value > 0 {
			if _, exists := metadata[key]; !exists {
				metadata[key] = strconv.FormatInt(value, 10)
			}
		}
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return encoded
}

func settlementWindow(lastSyncedThrough int64, now time.Time) (int64, int64) {
	end := now.UTC().Add(-settlementSafetyDelay).Unix()
	if lastSyncedThrough <= 0 {
		return now.UTC().Add(-settlementFirstLookback).Unix(), end
	}
	start := lastSyncedThrough - int64(settlementOverlap/time.Second)
	if start > end {
		start = end
	}
	return start, end
}
