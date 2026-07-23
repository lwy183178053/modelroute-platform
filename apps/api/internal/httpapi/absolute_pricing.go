package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/expr-lang/expr"
)

const (
	priceMicrosScale = int64(1_000_000)
	discountBPSScale = int64(10_000)
)

var supportedModelProviders = map[string]struct{}{
	"OpenAI": {}, "Anthropic": {}, "Gemini": {}, "DeepSeek": {},
	"Qwen": {}, "Moonshot": {}, "Minimax": {},
}

func normalizeOfficialPrice(price OfficialPrice) OfficialPrice {
	price.PriceKind = "text"
	if len(price.ImagePrices) > 0 {
		price.PriceKind = "image"
	}
	for index := range price.Tiers {
		price.Tiers[index].TierOrder = index + 1
	}
	for index := range price.ImagePrices {
		price.ImagePrices[index].Quality = strings.TrimSpace(price.ImagePrices[index].Quality)
		price.ImagePrices[index].Size = strings.TrimSpace(price.ImagePrices[index].Size)
	}
	return price
}

type PriceTier struct {
	ID                    int64  `json:"id,omitempty"`
	TierOrder             int    `json:"tier_order,omitempty"`
	MinimumInputTokens    int64  `json:"minimum_input_tokens"`
	MaximumInputTokens    *int64 `json:"maximum_input_tokens"`
	InputPriceMicros      int64  `json:"input_price_micros"`
	OutputPriceMicros     *int64 `json:"output_price_micros"`
	CacheReadPriceMicros  *int64 `json:"cache_read_price_micros"`
	CacheWritePriceMicros *int64 `json:"cache_write_price_micros"`
}

type OfferPriceTier struct {
	TierOrder             int    `json:"tier_order"`
	MinimumInputTokens    int64  `json:"minimum_input_tokens"`
	MaximumInputTokens    *int64 `json:"maximum_input_tokens"`
	InputPriceMicros      int64  `json:"input_price_micros"`
	OutputPriceMicros     *int64 `json:"output_price_micros"`
	CacheReadPriceMicros  *int64 `json:"cache_read_price_micros"`
	CacheWritePriceMicros *int64 `json:"cache_write_price_micros"`
}

type ImagePriceItem struct {
	Quality     string `json:"quality"`
	Size        string `json:"size"`
	PriceMicros int64  `json:"price_micros"`
}

func roundMultiplyDivide(value int64, multiplier int64, divisor int64) (int64, error) {
	if value < 0 || multiplier < 0 || divisor <= 0 {
		return 0, errors.New("整数换算参数无效")
	}
	product := new(big.Int).Mul(big.NewInt(value), big.NewInt(multiplier))
	product.Add(product, big.NewInt(divisor/2))
	product.Quo(product, big.NewInt(divisor))
	if !product.IsInt64() {
		return 0, errors.New("整数换算结果溢出")
	}
	return product.Int64(), nil
}

func validateOfficialPrice(price OfficialPrice, protocol string) error {
	if protocol == "image_generation" {
		if len(price.Tiers) != 0 {
			return errors.New("图片模型不得配置 Token 价格阶梯")
		}
		_, _, err := buildImageExpression(price.ImagePrices, 10_000)
		return err
	}
	if protocol != "chat_completions" && protocol != "responses" {
		return errors.New("模型协议不在平台开放范围")
	}
	if len(price.ImagePrices) != 0 {
		return errors.New("文本模型不得配置图片价格矩阵")
	}
	if len(price.Tiers) == 0 {
		return errors.New("至少需要一个价格阶梯")
	}
	for index, tier := range price.Tiers {
		if tier.MinimumInputTokens < 0 || (index == 0 && tier.MinimumInputTokens != 0) {
			return errors.New("首个价格阶梯必须从 0 Token 开始")
		}
		if tier.InputPriceMicros <= 0 {
			return errors.New("输入价格必须大于零")
		}
		if tier.OutputPriceMicros == nil || *tier.OutputPriceMicros <= 0 {
			return errors.New("文本生成模型必须填写大于零的输出价格")
		}
		for _, optional := range []*int64{tier.CacheReadPriceMicros, tier.CacheWritePriceMicros} {
			if optional != nil && *optional < 0 {
				return errors.New("缓存价格不得为负数")
			}
		}
		last := index == len(price.Tiers)-1
		if last {
			if tier.MaximumInputTokens != nil {
				return errors.New("最后一个价格阶梯不得设置上限")
			}
			continue
		}
		if tier.MaximumInputTokens == nil || *tier.MaximumInputTokens < tier.MinimumInputTokens {
			return errors.New("非末档价格阶梯必须设置有效上限")
		}
		nextMinimum := price.Tiers[index+1].MinimumInputTokens
		if *tier.MaximumInputTokens == math.MaxInt64 || nextMinimum != *tier.MaximumInputTokens+1 {
			return errors.New("价格阶梯存在重复、交叉或断档")
		}
	}
	return nil
}

func validateAbsolutePrice(price OfficialPrice, protocol string) error {
	return validateOfficialPrice(price, protocol)
}

func ValidateCanonicalModel(command CreateCanonicalModelCommand) error {
	return validateCanonicalModel(command)
}

func buildTieredExpression(price OfficialPrice, discountBPS int, protocol string) (string, []OfferPriceTier, error) {
	if discountBPS < 0 || discountBPS > 50_000 {
		return "", nil, errors.New("商户倍率必须在 0 到 50000 基点之间")
	}
	if err := validateAbsolutePrice(price, protocol); err != nil {
		return "", nil, err
	}
	discounted := make([]OfferPriceTier, 0, len(price.Tiers))
	expressions := make([]string, 0, len(price.Tiers))
	for index, tier := range price.Tiers {
		converted, err := discountTier(tier, discountBPS)
		if err != nil {
			return "", nil, fmt.Errorf("第 %d 档价格换算失败：%w", index+1, err)
		}
		converted.TierOrder = index + 1
		terms := []string{"p * " + scaledDecimal(converted.InputPriceMicros, priceMicrosScale)}
		if converted.OutputPriceMicros != nil {
			terms = append(terms, "c * "+scaledDecimal(*converted.OutputPriceMicros, priceMicrosScale))
		}
		if converted.CacheReadPriceMicros != nil {
			terms = append(terms, "cr * "+scaledDecimal(*converted.CacheReadPriceMicros, priceMicrosScale))
		}
		if converted.CacheWritePriceMicros != nil {
			price := scaledDecimal(*converted.CacheWritePriceMicros, priceMicrosScale)
			terms = append(terms, "cc * "+price, "cc1h * "+price)
		}
		expressions = append(expressions, fmt.Sprintf(`tier("tier_%d", %s)`, index+1, strings.Join(terms, " + ")))
		discounted = append(discounted, converted)
	}
	body := expressions[len(expressions)-1]
	for index := len(expressions) - 2; index >= 0; index-- {
		body = fmt.Sprintf("len <= %d ? %s : %s", *price.Tiers[index].MaximumInputTokens, expressions[index], body)
	}
	result := "v1:" + body
	if err := smokeTestTieredExpression(result); err != nil {
		return "", nil, err
	}
	return result, discounted, nil
}

func discountTier(tier PriceTier, discountBPS int) (OfferPriceTier, error) {
	result := OfferPriceTier{
		MinimumInputTokens: tier.MinimumInputTokens,
		MaximumInputTokens: tier.MaximumInputTokens,
	}
	input, err := roundMultiplyDivide(tier.InputPriceMicros, int64(discountBPS), discountBPSScale)
	if err != nil {
		return OfferPriceTier{}, err
	}
	if discountBPS != 0 && input < 1 {
		return OfferPriceTier{}, errors.New("必填输入价格折后低于 1 微单位")
	}
	result.InputPriceMicros = input
	if result.OutputPriceMicros, err = multiplyOptionalPrice(tier.OutputPriceMicros, discountBPS); err != nil {
		return OfferPriceTier{}, err
	}
	if discountBPS != 0 && tier.OutputPriceMicros != nil && *result.OutputPriceMicros < 1 {
		return OfferPriceTier{}, errors.New("必填输出价格折后低于 1 微单位")
	}
	if result.CacheReadPriceMicros, err = multiplyOptionalPrice(tier.CacheReadPriceMicros, discountBPS); err != nil {
		return OfferPriceTier{}, err
	}
	if result.CacheWritePriceMicros, err = multiplyOptionalPrice(tier.CacheWritePriceMicros, discountBPS); err != nil {
		return OfferPriceTier{}, err
	}
	return result, nil
}

func multiplyOptionalPrice(value *int64, multiplierBPS int) (*int64, error) {
	if value == nil {
		return nil, nil
	}
	converted, err := roundMultiplyDivide(*value, int64(multiplierBPS), discountBPSScale)
	if err != nil {
		return nil, err
	}
	return &converted, nil
}

func smokeTestTieredExpression(expression string) error {
	body := strings.TrimPrefix(expression, "v1:")
	vectors := []float64{0, 1_000, 100_000, 1_000_000}
	for _, value := range vectors {
		environment := map[string]any{
			"p": value, "c": value, "len": value, "cr": value,
			"cc": value, "cc1h": value,
			"tier": func(_ string, cost float64) float64 { return cost },
		}
		program, err := expr.Compile(body, expr.Env(environment), expr.AsFloat64())
		if err != nil {
			return fmt.Errorf("New API 计费表达式编译失败：%w", err)
		}
		output, err := expr.Run(program, environment)
		if err != nil {
			return fmt.Errorf("New API 计费表达式执行失败：%w", err)
		}
		cost, ok := output.(float64)
		if !ok || cost < 0 || math.IsNaN(cost) || math.IsInf(cost, 0) {
			return errors.New("New API 计费表达式返回无效金额")
		}
	}
	return nil
}

func buildImageExpression(items []ImagePriceItem, multiplierBPS int) (string, []ImagePriceItem, error) {
	if multiplierBPS < 0 || multiplierBPS > 50_000 {
		return "", nil, errors.New("商户倍率必须在 0 到 50000 基点之间")
	}
	if len(items) == 0 {
		return "", nil, errors.New("图片价格矩阵不得为空")
	}
	prices := make([]ImagePriceItem, len(items))
	seen := make(map[string]struct{}, len(items))
	maximumPrice := int64(0)
	for index, item := range items {
		item.Quality = strings.TrimSpace(item.Quality)
		item.Size = strings.TrimSpace(item.Size)
		if item.Quality == "" || item.Size == "" || item.PriceMicros <= 0 {
			return "", nil, errors.New("图片质量、尺寸和单张价格必须有效")
		}
		key := item.Quality + "\x00" + item.Size
		if _, exists := seen[key]; exists {
			return "", nil, errors.New("图片价格矩阵存在重复组合")
		}
		seen[key] = struct{}{}
		price, err := roundMultiplyDivide(item.PriceMicros, int64(multiplierBPS), discountBPSScale)
		if err != nil {
			return "", nil, err
		}
		if multiplierBPS != 0 && price < 1 {
			return "", nil, errors.New("图片单张价格折后低于 1 微单位")
		}
		item.PriceMicros = price
		prices[index] = item
		if price > maximumPrice {
			maximumPrice = price
		}
	}
	body := imagePriceDecimal(maximumPrice)
	for index := len(prices) - 1; index >= 0; index-- {
		quality, _ := json.Marshal(prices[index].Quality)
		size, _ := json.Marshal(prices[index].Size)
		condition := fmt.Sprintf(`param("quality") == %s && param("size") == %s`, quality, size)
		body = fmt.Sprintf("%s ? %s : %s", condition, imagePriceDecimal(prices[index].PriceMicros), body)
	}
	expression := `v1:tier("image", (param("n") == nil ? 1.0 : param("n")) * (` + body + `))`
	if err := smokeTestImageExpression(expression); err != nil {
		return "", nil, err
	}
	return expression, prices, nil
}

func smokeTestImageExpression(expression string) error {
	body := strings.TrimPrefix(expression, "v1:")
	for _, vector := range []struct {
		quality string
		size    string
		n       any
	}{
		{quality: "standard", size: "1024x1024", n: nil},
		{quality: "high", size: "1024x1024", n: 2.0},
	} {
		environment := map[string]any{
			"param": func(name string) any {
				switch name {
				case "quality":
					return vector.quality
				case "size":
					return vector.size
				case "n":
					return vector.n
				default:
					return nil
				}
			},
			"tier": func(_ string, cost float64) float64 { return cost },
		}
		program, err := expr.Compile(body, expr.Env(environment), expr.AsFloat64())
		if err != nil {
			return fmt.Errorf("New API 图片计费表达式编译失败：%w", err)
		}
		output, err := expr.Run(program, environment)
		if err != nil {
			return fmt.Errorf("New API 图片计费表达式执行失败：%w", err)
		}
		cost, ok := output.(float64)
		if !ok || cost < 0 || math.IsNaN(cost) || math.IsInf(cost, 0) {
			return errors.New("New API 图片计费表达式返回无效金额")
		}
	}
	return nil
}

func imagePriceDecimal(value int64) string {
	formatted := scaledDecimal(value, priceMicrosScale)
	if !strings.Contains(formatted, ".") {
		return formatted + ".0"
	}
	return formatted
}
