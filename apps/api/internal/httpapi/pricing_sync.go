package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

var newAPIPriceOptionKeys = []string{
	"billing_setting.billing_mode",
	"billing_setting.billing_expr",
}

type aliasPrice struct {
	Alias         string
	Protocol      string
	MultiplierBPS int
	Price         OfficialPrice
}

type ChannelPriceRoute struct {
	MerchantChannelID int64
	NewAPIChannelID   int
	PublicModels      []string
	Aliases           map[string]string
}

type PriceSyncRepository interface {
	AliasPrices(context.Context) ([]aliasPrice, error)
	OfferChannelRoutes(context.Context, int64) ([]ChannelPriceRoute, error)
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

type channelPriceUpdate struct {
	newAPIChannelID int
	models          string
	mapping         string
	originalModels  string
	originalMapping string
}

func NewPriceSynchronizer(repository PriceSyncRepository, admin AdminNewAPI) *PriceSynchronizer {
	return &PriceSynchronizer{repository: repository, admin: admin}
}

func (s *PriceSynchronizer) SyncOffer(ctx context.Context, offerID int64) error {
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
	routes, err := s.repository.OfferChannelRoutes(ctx, offerID)
	if err != nil {
		return fmt.Errorf("读取报价渠道映射：%w", err)
	}
	if len(routes) == 0 {
		return fmt.Errorf("报价没有可同步渠道")
	}
	channelUpdates := make([]channelPriceUpdate, 0, len(routes))
	for _, route := range routes {
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
		originalModels, originalMapping, err := decodeChannelConfiguration(channelData)
		if err != nil {
			return err
		}
		channelUpdates = append(channelUpdates, channelPriceUpdate{
			newAPIChannelID: route.NewAPIChannelID, models: strings.Join(models, ","), mapping: string(encodedMapping),
			originalModels: originalModels, originalMapping: originalMapping,
		})
	}

	attemptedOptions := make([]string, 0, len(newAPIPriceOptionKeys))
	attemptedChannels := make([]channelPriceUpdate, 0, len(channelUpdates))
	for _, key := range newAPIPriceOptionKeys {
		attemptedOptions = append(attemptedOptions, key)
		if _, err := s.admin.Do(ctx, http.MethodPut, "/api/option/", map[string]any{"key": key, "value": updates[key]}); err != nil {
			return s.syncRollbackError(ctx, fmt.Errorf("同步 New API %s：%w", key, err), existing, attemptedOptions, attemptedChannels)
		}
	}
	for _, update := range channelUpdates {
		attemptedChannels = append(attemptedChannels, update)
		if _, err := s.admin.Do(ctx, http.MethodPut, "/api/channel/", map[string]any{
			"id": update.newAPIChannelID, "models": update.models, "model_mapping": update.mapping,
		}); err != nil {
			return s.syncRollbackError(ctx, fmt.Errorf("同步 New API 渠道模型：%w", err), existing, attemptedOptions, attemptedChannels)
		}
	}
	if err := s.repository.ActivateOffer(ctx, offerID); err != nil {
		return s.syncRollbackError(ctx, fmt.Errorf("激活报价：%w", err), existing, attemptedOptions, attemptedChannels)
	}
	return nil
}

func (s *PriceSynchronizer) syncRollbackError(ctx context.Context, syncErr error, existing map[string]string, optionKeys []string, channels []channelPriceUpdate) error {
	rollbackErrors := []error{syncErr}
	for index := len(channels) - 1; index >= 0; index-- {
		channel := channels[index]
		if _, err := s.admin.Do(ctx, http.MethodPut, "/api/channel/", map[string]any{
			"id": channel.newAPIChannelID, "models": channel.originalModels, "model_mapping": channel.originalMapping,
		}); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("回滚 New API 渠道 %d：%w", channel.newAPIChannelID, err))
		}
	}
	for index := len(optionKeys) - 1; index >= 0; index-- {
		key := optionKeys[index]
		if _, err := s.admin.Do(ctx, http.MethodPut, "/api/option/", map[string]any{"key": key, "value": existing[key]}); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("回滚 New API %s：%w", key, err))
		}
	}
	return errors.Join(rollbackErrors...)
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

func decodeChannelConfiguration(data []byte) (string, string, error) {
	var channel struct {
		Models       string  `json:"models"`
		ModelMapping *string `json:"model_mapping"`
	}
	if err := json.Unmarshal(data, &channel); err != nil {
		return "", "", fmt.Errorf("解析 New API 渠道快照：%w", err)
	}
	mapping := ""
	if channel.ModelMapping != nil {
		mapping = *channel.ModelMapping
	}
	return channel.Models, mapping, nil
}

func buildAliasPriceOptions(existing map[string]string, offers []aliasPrice) (map[string]string, error) {
	values := make(map[string]map[string]string, len(newAPIPriceOptionKeys))
	for _, key := range newAPIPriceOptionKeys {
		values[key] = make(map[string]string)
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
		if offer.Alias == "" || offer.MultiplierBPS < 0 || offer.MultiplierBPS > 50_000 {
			return nil, fmt.Errorf("报价别名或倍率无效")
		}
		var expression string
		var err error
		if offer.Protocol == "image_generation" {
			expression, _, err = buildImageExpression(offer.Price.ImagePrices, offer.MultiplierBPS)
		} else {
			expression, _, err = buildTieredExpression(offer.Price, offer.MultiplierBPS, offer.Protocol)
		}
		if err != nil {
			return nil, fmt.Errorf("报价 %s 的绝对价格无效：%w", offer.Alias, err)
		}
		values["billing_setting.billing_mode"][offer.Alias] = "tiered_expr"
		values["billing_setting.billing_expr"][offer.Alias] = expression
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
