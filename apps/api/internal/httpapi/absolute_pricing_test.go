package httpapi

import (
	"strings"
	"testing"
)

func int64Pointer(value int64) *int64 { return &value }

func TestValidateTextPriceRequiresContinuousTiers(t *testing.T) {
	valid := OfficialPrice{Tiers: []PriceTier{
		{MinimumInputTokens: 0, MaximumInputTokens: int64Pointer(200_000), InputPriceMicros: 3_000_000, OutputPriceMicros: int64Pointer(15_000_000)},
		{MinimumInputTokens: 200_001, InputPriceMicros: 6_000_000, OutputPriceMicros: int64Pointer(22_500_000)},
	}}
	if err := validateOfficialPrice(valid); err != nil {
		t.Fatalf("连续价格阶梯被拒绝：%v", err)
	}
	for _, test := range []struct {
		name  string
		tiers []PriceTier
	}{
		{name: "首档不从零开始", tiers: []PriceTier{{MinimumInputTokens: 1, InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}}},
		{name: "阶梯断档", tiers: []PriceTier{{MinimumInputTokens: 0, MaximumInputTokens: int64Pointer(10), InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}, {MinimumInputTokens: 12, InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}}},
		{name: "阶梯交叉", tiers: []PriceTier{{MinimumInputTokens: 0, MaximumInputTokens: int64Pointer(10), InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}, {MinimumInputTokens: 10, InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}}},
		{name: "末档仍有上限", tiers: []PriceTier{{MinimumInputTokens: 0, MaximumInputTokens: int64Pointer(10), InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			price := valid
			price.Tiers = test.tiers
	if err := validateOfficialPrice(price); err == nil {
				t.Fatal("无效阶梯未被拒绝")
			}
		})
	}
}

func TestValidateCanonicalModelInfersTextOrImageFromPrice(t *testing.T) {
	price := OfficialPrice{Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 100_000, OutputPriceMicros: int64Pointer(200_000)}}}
	if err := validateCanonicalModel(CreateCanonicalModelCommand{PublicName: "model", Provider: "Unknown", Price: price}); err == nil {
		t.Fatal("未知模型厂商未被拒绝")
	}
	if err := validateCanonicalModel(CreateCanonicalModelCommand{PublicName: "model", Provider: "OpenAI", Price: price}); err != nil {
		t.Fatalf("无协议文本模型被拒绝：%v", err)
	}
	image := OfficialPrice{ImagePrices: []ImagePriceItem{{Quality: "standard", Size: "1024x1024", PriceMicros: 40_000}}}
	if err := validateCanonicalModel(CreateCanonicalModelCommand{PublicName: "gpt-image-2", Provider: "OpenAI", Price: image}); err != nil {
		t.Fatalf("图片模型被拒绝：%v", err)
	}
}

func TestBuildTieredExpressionUsesNumericPricesAndCache(t *testing.T) {
	price := OfficialPrice{Tiers: []PriceTier{
		{
			MinimumInputTokens: 0, MaximumInputTokens: int64Pointer(200_000),
			InputPriceMicros: 3_000_000, OutputPriceMicros: int64Pointer(15_000_000),
			CacheReadPriceMicros: int64Pointer(300_000), CacheWritePriceMicros: int64Pointer(3_750_000),
		},
		{MinimumInputTokens: 200_001, InputPriceMicros: 6_000_000, OutputPriceMicros: int64Pointer(22_500_000)},
	}}
	expression, tiers, err := buildTieredExpression(price, 9_000)
	if err != nil {
		t.Fatalf("生成表达式失败：%v", err)
	}
	want := `v1:len <= 200000 ? tier("tier_1", p * 2.7 + c * 13.5 + cr * 0.27 + cc * 3.375 + cc1h * 3.375) : tier("tier_2", p * 5.4 + c * 20.25)`
	if expression != want {
		t.Fatalf("表达式=\n%s\n期望=\n%s", expression, want)
	}
	if len(tiers) != 2 || tiers[0].InputPriceMicros != 2_700_000 || tiers[1].OutputPriceMicros == nil || *tiers[1].OutputPriceMicros != 20_250_000 {
		t.Fatalf("折后价格阶梯=%+v", tiers)
	}
}

func TestBuildTieredExpressionAcceptsZeroAndFiveTimesMultiplier(t *testing.T) {
	output := int64(2_000_000)
	price := OfficialPrice{Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 1_000_000, OutputPriceMicros: &output}}}
	zero, tiers, err := buildTieredExpression(price, 0)
	if err != nil || zero != `v1:tier("tier_1", p * 0 + c * 0)` || tiers[0].InputPriceMicros != 0 {
		t.Fatalf("零倍率表达式=%q，阶梯=%+v，错误=%v", zero, tiers, err)
	}
	five, _, err := buildTieredExpression(price, 50_000)
	if err != nil || five != `v1:tier("tier_1", p * 5 + c * 10)` {
		t.Fatalf("五倍表达式=%q，错误=%v", five, err)
	}
}

func TestBuildImageExpressionUsesQualitySizeAndCount(t *testing.T) {
	items := []ImagePriceItem{
		{Quality: "standard", Size: "1024x1024", PriceMicros: 40_000},
		{Quality: "high", Size: "1024x1024", PriceMicros: 80_000},
	}
	expression, prices, err := buildImageExpression(items, 10_000)
	if err != nil {
		t.Fatalf("生成图片价格表达式失败：%v", err)
	}
	for _, fragment := range []string{`param("quality")`, `param("size")`, `param("n")`, "0.04", "0.08"} {
		if !strings.Contains(expression, fragment) {
			t.Fatalf("图片表达式缺少 %s：%s", fragment, expression)
		}
	}
	if len(prices) != 2 || prices[0].PriceMicros != 40_000 || prices[1].PriceMicros != 80_000 {
		t.Fatalf("图片折后价格=%+v", prices)
	}
}

func TestBuildImageExpressionValidatesMatrixAndMultiplier(t *testing.T) {
	valid := []ImagePriceItem{{Quality: "standard", Size: "1024x1024", PriceMicros: 40_000}}
	zero, prices, err := buildImageExpression(valid, 0)
	if err != nil || prices[0].PriceMicros != 0 || !strings.Contains(zero, "0.0") {
		t.Fatalf("图片零倍率表达式=%q，价格=%+v，错误=%v", zero, prices, err)
	}
	for _, test := range []struct {
		name       string
		items      []ImagePriceItem
		multiplier int
	}{
		{name: "空矩阵", multiplier: 10_000},
		{name: "重复组合", items: append(valid, valid...), multiplier: 10_000},
		{name: "零单价", items: []ImagePriceItem{{Quality: "standard", Size: "1024x1024"}}, multiplier: 10_000},
		{name: "超过五倍", items: valid, multiplier: 50_001},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := buildImageExpression(test.items, test.multiplier); err == nil {
				t.Fatal("无效图片价格未被拒绝")
			}
		})
	}
}
