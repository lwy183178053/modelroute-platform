package httpapi

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

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

func initialModelStatus(lifecycle string) string {
	if lifecycle == "stable" {
		return "active"
	}
	return "inactive"
}

func normalizeOfficialPrice(price OfficialPrice) OfficialPrice {
	price.BillingMode = "token"
	price.PricingScheme = "absolute"
	price.Currency = strings.TrimSpace(strings.ToUpper(price.Currency))
	price.SourceURL = strings.TrimSpace(price.SourceURL)
	if price.CheckedAt.IsZero() {
		price.CheckedAt = time.Now().UTC()
	}
	for index := range price.Tiers {
		price.Tiers[index].TierOrder = index + 1
	}
	return price
}

type PriceTier struct {
	ID                      int64  `json:"id,omitempty"`
	TierOrder               int    `json:"tier_order,omitempty"`
	MinimumInputTokens      int64  `json:"minimum_input_tokens"`
	MaximumInputTokens      *int64 `json:"maximum_input_tokens"`
	InputPriceMicros        int64  `json:"input_price_micros"`
	OutputPriceMicros       *int64 `json:"output_price_micros"`
	CacheReadPriceMicros    *int64 `json:"cache_read_price_micros"`
	CacheWrite5mPriceMicros *int64 `json:"cache_write_5m_price_micros"`
	CacheWrite1hPriceMicros *int64 `json:"cache_write_1h_price_micros"`
}

type OfferPriceTier struct {
	TierOrder                 int    `json:"tier_order"`
	MinimumInputTokens        int64  `json:"minimum_input_tokens"`
	MaximumInputTokens        *int64 `json:"maximum_input_tokens"`
	InputPriceMicroUSD        int64  `json:"input_price_microusd"`
	OutputPriceMicroUSD       *int64 `json:"output_price_microusd"`
	CacheReadPriceMicroUSD    *int64 `json:"cache_read_price_microusd"`
	CacheWrite5mPriceMicroUSD *int64 `json:"cache_write_5m_price_microusd"`
	CacheWrite1hPriceMicroUSD *int64 `json:"cache_write_1h_price_microusd"`
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

func validateAbsolutePrice(price OfficialPrice, protocol string) error {
	if price.Currency != "USD" && price.Currency != "CNY" {
		return errors.New("原币币种必须是 USD 或 CNY")
	}
	if price.USDExchangeRateMicros <= 0 {
		return errors.New("美元兑原币汇率必须大于零")
	}
	if price.Currency == "USD" && price.USDExchangeRateMicros != priceMicrosScale {
		return errors.New("美元价格的锁定汇率必须为 1")
	}
	if strings.TrimSpace(price.SourceURL) == "" || len(price.SourceURL) > 2048 {
		return errors.New("必须填写有效的核价来源")
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
		if protocol == "embeddings" {
			if tier.OutputPriceMicros != nil {
				return errors.New("Embeddings 只接受输入价格")
			}
		} else if tier.OutputPriceMicros == nil || *tier.OutputPriceMicros <= 0 {
			return errors.New("文本生成模型必须填写大于零的输出价格")
		}
		for _, optional := range []*int64{tier.CacheReadPriceMicros, tier.CacheWrite5mPriceMicros, tier.CacheWrite1hPriceMicros} {
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

func ValidateCanonicalModel(command CreateCanonicalModelCommand) error {
	return validateCanonicalModel(command)
}

func buildTieredExpression(price OfficialPrice, discountBPS int, protocol string) (string, []OfferPriceTier, error) {
	if discountBPS < 1 || discountBPS > int(discountBPSScale) {
		return "", nil, errors.New("商户折扣必须在 1 到 10000 基点之间")
	}
	if err := validateAbsolutePrice(price, protocol); err != nil {
		return "", nil, err
	}
	discounted := make([]OfferPriceTier, 0, len(price.Tiers))
	expressions := make([]string, 0, len(price.Tiers))
	for index, tier := range price.Tiers {
		converted, err := discountTier(tier, price.USDExchangeRateMicros, discountBPS)
		if err != nil {
			return "", nil, fmt.Errorf("第 %d 档价格换算失败：%w", index+1, err)
		}
		converted.TierOrder = index + 1
		terms := []string{"p * " + scaledDecimal(converted.InputPriceMicroUSD, priceMicrosScale)}
		if converted.OutputPriceMicroUSD != nil {
			terms = append(terms, "c * "+scaledDecimal(*converted.OutputPriceMicroUSD, priceMicrosScale))
		}
		if converted.CacheReadPriceMicroUSD != nil {
			terms = append(terms, "cr * "+scaledDecimal(*converted.CacheReadPriceMicroUSD, priceMicrosScale))
		}
		if converted.CacheWrite5mPriceMicroUSD != nil {
			terms = append(terms, "cc * "+scaledDecimal(*converted.CacheWrite5mPriceMicroUSD, priceMicrosScale))
		}
		if converted.CacheWrite1hPriceMicroUSD != nil {
			terms = append(terms, "cc1h * "+scaledDecimal(*converted.CacheWrite1hPriceMicroUSD, priceMicrosScale))
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

func discountTier(tier PriceTier, exchangeRateMicros int64, discountBPS int) (OfferPriceTier, error) {
	result := OfferPriceTier{
		MinimumInputTokens: tier.MinimumInputTokens,
		MaximumInputTokens: tier.MaximumInputTokens,
	}
	input, err := convertAndDiscountPrice(tier.InputPriceMicros, exchangeRateMicros, discountBPS)
	if err != nil {
		return OfferPriceTier{}, err
	}
	if input < 1 {
		return OfferPriceTier{}, errors.New("必填输入价格折后低于 1 微美元")
	}
	result.InputPriceMicroUSD = input
	if result.OutputPriceMicroUSD, err = convertOptionalPrice(tier.OutputPriceMicros, exchangeRateMicros, discountBPS); err != nil {
		return OfferPriceTier{}, err
	}
	if tier.OutputPriceMicros != nil && *result.OutputPriceMicroUSD < 1 {
		return OfferPriceTier{}, errors.New("必填输出价格折后低于 1 微美元")
	}
	if result.CacheReadPriceMicroUSD, err = convertOptionalPrice(tier.CacheReadPriceMicros, exchangeRateMicros, discountBPS); err != nil {
		return OfferPriceTier{}, err
	}
	if result.CacheWrite5mPriceMicroUSD, err = convertOptionalPrice(tier.CacheWrite5mPriceMicros, exchangeRateMicros, discountBPS); err != nil {
		return OfferPriceTier{}, err
	}
	if result.CacheWrite1hPriceMicroUSD, err = convertOptionalPrice(tier.CacheWrite1hPriceMicros, exchangeRateMicros, discountBPS); err != nil {
		return OfferPriceTier{}, err
	}
	return result, nil
}

func convertAndDiscountPrice(value int64, exchangeRateMicros int64, discountBPS int) (int64, error) {
	usd, err := roundMultiplyDivide(value, exchangeRateMicros, priceMicrosScale)
	if err != nil {
		return 0, err
	}
	return roundMultiplyDivide(usd, int64(discountBPS), discountBPSScale)
}

func convertOptionalPrice(value *int64, exchangeRateMicros int64, discountBPS int) (*int64, error) {
	if value == nil {
		return nil, nil
	}
	converted, err := convertAndDiscountPrice(*value, exchangeRateMicros, discountBPS)
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
