package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

const priceScale = int64(1_000_000)

var newAPIPriceOptionKeys = []string{
	"ModelRatio",
	"ModelPrice",
	"CompletionRatio",
	"CacheRatio",
	"CreateCacheRatio",
	"ImageRatio",
	"AudioRatio",
	"AudioCompletionRatio",
}

type aliasPrice struct {
	Alias                      string
	BillingMode                string
	MultiplierBPS              int
	ModelRatioMicros           int64
	ModelPriceMicroUSD         int64
	CompletionRatioMicros      int64
	CacheRatioMicros           int64
	CreateCacheRatioMicros     int64
	ImageRatioMicros           int64
	AudioRatioMicros           int64
	AudioCompletionRatioMicros int64
}

type ChannelPriceRoute struct {
	NewAPIChannelID int
	PublicModels    []string
	Aliases         map[string]string
}

type PriceSyncRepository interface {
	AliasPrices(context.Context) ([]aliasPrice, error)
	ChannelPriceRoute(context.Context, int64) (ChannelPriceRoute, error)
	ActivateOffer(context.Context, int64) error
}

type AdminNewAPI interface {
	Do(context.Context, string, string, any) ([]byte, error)
}

type PriceSynchronizer struct {
	repository PriceSyncRepository
	admin      AdminNewAPI
	mu         sync.Mutex
}

func NewPriceSynchronizer(repository PriceSyncRepository, admin AdminNewAPI) *PriceSynchronizer {
	return &PriceSynchronizer{repository: repository, admin: admin}
}

func (s *PriceSynchronizer) SyncOffer(ctx context.Context, offerID int64, merchantChannelID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	prices, err := s.repository.AliasPrices(ctx)
	if err != nil {
		return fmt.Errorf("读取报价价格：%w", err)
	}
	optionData, err := s.admin.Do(ctx, http.MethodGet, "/api/option/", nil)
	if err != nil {
		return fmt.Errorf("读取 New API 价格配置：%w", err)
	}
	existing, err := decodeNewAPIOptions(optionData)
	if err != nil {
		return err
	}
	updates, err := buildAliasPriceOptions(existing, prices)
	if err != nil {
		return err
	}
	for _, key := range newAPIPriceOptionKeys {
		if _, err := s.admin.Do(ctx, http.MethodPut, "/api/option/", map[string]any{"key": key, "value": updates[key]}); err != nil {
			return fmt.Errorf("同步 New API %s：%w", key, err)
		}
	}

	route, err := s.repository.ChannelPriceRoute(ctx, merchantChannelID)
	if err != nil {
		return fmt.Errorf("读取渠道报价映射：%w", err)
	}
	channelData, err := s.admin.Do(ctx, http.MethodGet, fmt.Sprintf("/api/channel/%d", route.NewAPIChannelID), nil)
	if err != nil {
		return fmt.Errorf("读取 New API 渠道：%w", err)
	}
	models, mapping, err := mergeChannelRoute(channelData, route)
	if err != nil {
		return err
	}
	encodedMapping, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("编码渠道模型映射：%w", err)
	}
	if _, err := s.admin.Do(ctx, http.MethodPut, "/api/channel/", map[string]any{
		"id": route.NewAPIChannelID, "models": strings.Join(models, ","), "model_mapping": string(encodedMapping),
	}); err != nil {
		return fmt.Errorf("同步 New API 渠道模型：%w", err)
	}
	if err := s.repository.ActivateOffer(ctx, offerID); err != nil {
		return fmt.Errorf("激活报价：%w", err)
	}
	return nil
}

func decodeNewAPIOptions(data []byte) (map[string]string, error) {
	var items []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("解析 New API 价格配置：%w", err)
	}
	result := make(map[string]string, len(items))
	for _, item := range items {
		result[item.Key] = item.Value
	}
	return result, nil
}

func mergeChannelRoute(data []byte, route ChannelPriceRoute) ([]string, map[string]string, error) {
	var channel struct {
		Models       string  `json:"models"`
		ModelMapping *string `json:"model_mapping"`
	}
	if err := json.Unmarshal(data, &channel); err != nil {
		return nil, nil, fmt.Errorf("解析 New API 渠道：%w", err)
	}
	modelSet := make(map[string]struct{})
	for _, model := range append(strings.Split(channel.Models, ","), route.PublicModels...) {
		if model = strings.TrimSpace(model); model != "" {
			modelSet[model] = struct{}{}
		}
	}
	mapping := make(map[string]string)
	if channel.ModelMapping != nil && strings.TrimSpace(*channel.ModelMapping) != "" {
		if err := json.Unmarshal([]byte(*channel.ModelMapping), &mapping); err != nil {
			return nil, nil, fmt.Errorf("解析 New API 渠道模型映射：%w", err)
		}
	}
	for alias, publicModel := range route.Aliases {
		modelSet[alias] = struct{}{}
		mapping[alias] = publicModel
	}
	models := make([]string, 0, len(modelSet))
	for model := range modelSet {
		models = append(models, model)
	}
	sort.Strings(models)
	return models, mapping, nil
}

func buildAliasPriceOptions(existing map[string]string, offers []aliasPrice) (map[string]string, error) {
	values := make(map[string]map[string]json.RawMessage, len(newAPIPriceOptionKeys))
	for _, key := range newAPIPriceOptionKeys {
		values[key] = make(map[string]json.RawMessage)
		raw := strings.TrimSpace(existing[key])
		if raw == "" {
			continue
		}
		parsed := values[key]
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			return nil, fmt.Errorf("New API %s 配置不是有效对象：%w", key, err)
		}
		values[key] = parsed
	}

	for _, offer := range offers {
		if offer.Alias == "" || offer.MultiplierBPS < 1 || offer.MultiplierBPS > 10_000 {
			return nil, fmt.Errorf("报价别名或倍率无效")
		}
		delete(values["ModelRatio"], offer.Alias)
		delete(values["ModelPrice"], offer.Alias)
		switch offer.BillingMode {
		case "token":
			if offer.ModelRatioMicros <= 0 {
				return nil, fmt.Errorf("Token 计费缺少主倍率")
			}
			values["ModelRatio"][offer.Alias] = decimalJSON(applyBasisPoints(offer.ModelRatioMicros, offer.MultiplierBPS))
		case "fixed":
			if offer.ModelPriceMicroUSD <= 0 {
				return nil, fmt.Errorf("固定计费缺少价格")
			}
			values["ModelPrice"][offer.Alias] = decimalJSON(applyBasisPoints(offer.ModelPriceMicroUSD, offer.MultiplierBPS))
		default:
			return nil, fmt.Errorf("不支持的计费模式 %q", offer.BillingMode)
		}
		for key, value := range map[string]int64{
			"CompletionRatio":      offer.CompletionRatioMicros,
			"CacheRatio":           offer.CacheRatioMicros,
			"CreateCacheRatio":     offer.CreateCacheRatioMicros,
			"ImageRatio":           offer.ImageRatioMicros,
			"AudioRatio":           offer.AudioRatioMicros,
			"AudioCompletionRatio": offer.AudioCompletionRatioMicros,
		} {
			if value < 0 {
				return nil, fmt.Errorf("报价 %s 的 %s 不能为负数", offer.Alias, key)
			}
			values[key][offer.Alias] = decimalJSON(value)
		}
	}

	result := make(map[string]string, len(values))
	for _, key := range newAPIPriceOptionKeys {
		encoded, err := json.Marshal(values[key])
		if err != nil {
			return nil, fmt.Errorf("编码 New API %s 配置：%w", key, err)
		}
		result[key] = string(encoded)
	}
	return result, nil
}

func decimalJSON(value int64) json.RawMessage {
	return json.RawMessage(scaledDecimal(value, priceScale))
}
