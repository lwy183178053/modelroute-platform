package httpapi

import (
	"strings"
	"testing"
)

func int64Pointer(value int64) *int64 {
	return &value
}

func TestRoundMultiplyDivideUsesIntegerHalfUp(t *testing.T) {
	tests := []struct {
		name       string
		value      int64
		multiplier int64
		divisor    int64
		want       int64
	}{
		{name: "人民币换算", value: 7_000_000, multiplier: 137_000, divisor: 1_000_000, want: 959_000},
		{name: "半数向上舍入", value: 5, multiplier: 5_000, divisor: 10_000, want: 3},
		{name: "不足半数舍弃", value: 4, multiplier: 5_000, divisor: 10_000, want: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := roundMultiplyDivide(test.value, test.multiplier, test.divisor)
			if err != nil || got != test.want {
				t.Fatalf("roundMultiplyDivide() = %d, %v，期望 %d", got, err, test.want)
			}
		})
	}
}

func TestValidateAbsolutePriceRequiresContinuousTiers(t *testing.T) {
	valid := OfficialPrice{
		Currency:              "USD",
		USDExchangeRateMicros: 1_000_000,
		SourceURL:             "https://example.com/pricing",
		Tiers: []PriceTier{
			{MinimumInputTokens: 0, MaximumInputTokens: int64Pointer(200_000), InputPriceMicros: 3_000_000, OutputPriceMicros: int64Pointer(15_000_000)},
			{MinimumInputTokens: 200_001, InputPriceMicros: 6_000_000, OutputPriceMicros: int64Pointer(22_500_000)},
		},
	}
	if err := validateAbsolutePrice(valid, "responses"); err != nil {
		t.Fatalf("连续价格阶梯被拒绝：%v", err)
	}

	tests := []struct {
		name  string
		tiers []PriceTier
	}{
		{name: "首档不从零开始", tiers: []PriceTier{{MinimumInputTokens: 1, InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}}},
		{name: "阶梯断档", tiers: []PriceTier{{MinimumInputTokens: 0, MaximumInputTokens: int64Pointer(10), InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}, {MinimumInputTokens: 12, InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}}},
		{name: "阶梯交叉", tiers: []PriceTier{{MinimumInputTokens: 0, MaximumInputTokens: int64Pointer(10), InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}, {MinimumInputTokens: 10, InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}}},
		{name: "末档仍有上限", tiers: []PriceTier{{MinimumInputTokens: 0, MaximumInputTokens: int64Pointer(10), InputPriceMicros: 1, OutputPriceMicros: int64Pointer(1)}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			price := valid
			price.Tiers = test.tiers
			if err := validateAbsolutePrice(price, "responses"); err == nil {
				t.Fatal("无效阶梯未被拒绝")
			}
		})
	}
}

func TestValidateAbsolutePriceUsesProtocolAndProviderWhitelists(t *testing.T) {
	base := OfficialPrice{
		Currency: "USD", USDExchangeRateMicros: 1_000_000,
		SourceURL: "https://example.com/pricing",
		Tiers:     []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 100_000, OutputPriceMicros: int64Pointer(200_000)}},
	}
	if err := validateCanonicalModel(CreateCanonicalModelCommand{PublicName: "model", Provider: "Unknown", Protocol: "responses", Lifecycle: "stable", Price: base}); err == nil {
		t.Fatal("未知模型厂商未被拒绝")
	}
	if err := validateCanonicalModel(CreateCanonicalModelCommand{PublicName: "model", Provider: "OpenAI", Protocol: "responses", Lifecycle: "removed", Price: base}); err == nil {
		t.Fatal("未知生命周期未被拒绝")
	}
	embeddings := base
	embeddings.Tiers = []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 100_000}}
	if err := validateCanonicalModel(CreateCanonicalModelCommand{PublicName: "embedding", Provider: "OpenAI", Protocol: "embeddings", Lifecycle: "stable", Price: embeddings}); err != nil {
		t.Fatalf("Embedding 输入单价被拒绝：%v", err)
	}
	if err := validateAbsolutePrice(embeddings, "responses"); err == nil {
		t.Fatal("文本生成模型缺少输出价格未被拒绝")
	}
}

func TestInitialModelStatusOnlyEnablesStableLifecycle(t *testing.T) {
	if got := initialModelStatus("stable"); got != "active" {
		t.Fatalf("稳定版初始状态=%q", got)
	}
	for _, lifecycle := range []string{"preview", "limited", "retiring"} {
		if got := initialModelStatus(lifecycle); got != "inactive" {
			t.Fatalf("%s 初始状态=%q", lifecycle, got)
		}
	}
}

func TestBuildTieredExpressionUsesDiscountedAbsoluteUSDPrices(t *testing.T) {
	price := OfficialPrice{
		Currency: "USD", USDExchangeRateMicros: 1_000_000, SourceURL: "https://example.com/pricing",
		Tiers: []PriceTier{
			{
				MinimumInputTokens: 0, MaximumInputTokens: int64Pointer(200_000),
				InputPriceMicros: 3_000_000, OutputPriceMicros: int64Pointer(15_000_000),
				CacheReadPriceMicros: int64Pointer(300_000), CacheWrite5mPriceMicros: int64Pointer(3_750_000), CacheWrite1hPriceMicros: int64Pointer(6_000_000),
			},
			{MinimumInputTokens: 200_001, InputPriceMicros: 6_000_000, OutputPriceMicros: int64Pointer(22_500_000)},
		},
	}
	expression, tiers, err := buildTieredExpression(price, 9_000, "responses")
	if err != nil {
		t.Fatalf("生成表达式失败：%v", err)
	}
	want := `v1:len <= 200000 ? tier("tier_1", p * 2.7 + c * 13.5 + cr * 0.27 + cc * 3.375 + cc1h * 5.4) : tier("tier_2", p * 5.4 + c * 20.25)`
	if expression != want {
		t.Fatalf("表达式=\n%s\n期望=\n%s", expression, want)
	}
	if len(tiers) != 2 || tiers[0].InputPriceMicroUSD != 2_700_000 || tiers[1].OutputPriceMicroUSD == nil || *tiers[1].OutputPriceMicroUSD != 20_250_000 {
		t.Fatalf("折后美元阶梯=%+v", tiers)
	}
	if err := smokeTestTieredExpression(expression); err != nil {
		t.Fatalf("New API 同语法冒烟失败：%v", err)
	}
}

func TestBuildTieredExpressionConvertsCNYAndRejectsSubMicroRequiredPrice(t *testing.T) {
	price := OfficialPrice{
		Currency: "CNY", USDExchangeRateMicros: 137_000, SourceURL: "https://example.com/pricing",
		Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 7_000_000}},
	}
	expression, tiers, err := buildTieredExpression(price, 10_000, "embeddings")
	if err != nil || expression != `v1:tier("tier_1", p * 0.959)` || tiers[0].InputPriceMicroUSD != 959_000 {
		t.Fatalf("人民币表达式=%q，阶梯=%+v，错误=%v", expression, tiers, err)
	}
	price.Currency = "USD"
	price.USDExchangeRateMicros = 1_000_000
	price.Tiers[0].InputPriceMicros = 1
	if _, _, err := buildTieredExpression(price, 1, "embeddings"); err == nil || !strings.Contains(err.Error(), "低于 1 微美元") {
		t.Fatalf("折后零价错误=%v", err)
	}
}
