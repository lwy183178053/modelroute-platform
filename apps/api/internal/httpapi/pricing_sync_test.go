package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type priceSyncRepositoryStub struct {
	prices      []aliasPrice
	channel     ChannelPriceRoute
	activatedID int64
}

func (s *priceSyncRepositoryStub) AliasPrices(context.Context) ([]aliasPrice, error) {
	return s.prices, nil
}

func (s *priceSyncRepositoryStub) ChannelPriceRoute(context.Context, int64) (ChannelPriceRoute, error) {
	return s.channel, nil
}

func (s *priceSyncRepositoryStub) ActivateOffer(_ context.Context, offerID int64) error {
	s.activatedID = offerID
	return nil
}

type adminNewAPIStub struct {
	calls   []string
	updates map[string]string
	failKey string
}

func (s *adminNewAPIStub) Do(_ context.Context, method string, path string, body any) ([]byte, error) {
	s.calls = append(s.calls, method+" "+path)
	switch method + " " + path {
	case "GET /api/option/":
		return []byte(`[{"key":"billing_setting.billing_mode","value":"{\"existing\":\"tiered_expr\"}"},{"key":"billing_setting.billing_expr","value":"{\"existing\":\"v1:tier(\\\"tier_1\\\", p * 2)\"}"}]`), nil
	case "GET /api/channel/41":
		return []byte(`{"id":41,"models":"legacy","model_mapping":"{\"legacy\":\"legacy-upstream\"}"}`), nil
	case "PUT /api/option/":
		update := body.(map[string]any)
		if strings.TrimSpace(fmt.Sprint(update["key"])) == "" || strings.TrimSpace(fmt.Sprint(update["value"])) == "" {
			return nil, fmt.Errorf("价格配置为空")
		}
		if s.updates == nil {
			s.updates = make(map[string]string)
		}
		s.updates[fmt.Sprint(update["key"])] = fmt.Sprint(update["value"])
		if fmt.Sprint(update["key"]) == s.failKey {
			return nil, fmt.Errorf("模拟配置写入失败")
		}
		return []byte(`null`), nil
	case "PUT /api/channel/":
		update := body.(map[string]any)
		models := update["models"].(string)
		if models != "gpt-5.5,legacy,mkt-ov-42" {
			return nil, fmt.Errorf("渠道模型 = %q", models)
		}
		var mapping map[string]string
		if err := json.Unmarshal([]byte(update["model_mapping"].(string)), &mapping); err != nil {
			return nil, err
		}
		if mapping["legacy"] != "legacy-upstream" || mapping["mkt-ov-42"] != "gpt-5.5" {
			return nil, fmt.Errorf("渠道映射 = %v", mapping)
		}
		return []byte(`null`), nil
	default:
		return nil, fmt.Errorf("未知调用 %s %s", method, path)
	}
}

func TestBuildAliasPriceOptionsPreservesExistingAndWritesTieredExpression(t *testing.T) {
	output := int64(15_000_000)
	cacheRead := int64(300_000)
	cacheWrite5m := int64(3_750_000)
	cacheWrite1h := int64(6_000_000)
	existing := map[string]string{
		"billing_setting.billing_mode": `{"existing":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"existing":"v1:tier(\"tier_1\", p * 2)"}`,
	}
	offers := []aliasPrice{{
		Alias: "mkt-ov-42", Protocol: "responses", MultiplierBPS: 9_000,
		Price: OfficialPrice{
			Currency: "USD", USDExchangeRateMicros: 1_000_000, SourceURL: "https://example.com/pricing",
			Tiers: []PriceTier{{
				MinimumInputTokens: 0, InputPriceMicros: 3_000_000, OutputPriceMicros: &output,
				CacheReadPriceMicros: &cacheRead, CacheWrite5mPriceMicros: &cacheWrite5m, CacheWrite1hPriceMicros: &cacheWrite1h,
			}},
		},
	}}
	options, err := buildAliasPriceOptions(existing, offers)
	if err != nil {
		t.Fatalf("生成价格配置失败：%v", err)
	}
	assertJSONString(t, options["billing_setting.billing_mode"], "existing", "tiered_expr")
	assertJSONString(t, options["billing_setting.billing_mode"], "mkt-ov-42", "tiered_expr")
	assertJSONString(t, options["billing_setting.billing_expr"], "existing", `v1:tier("tier_1", p * 2)`)
	assertJSONString(t, options["billing_setting.billing_expr"], "mkt-ov-42", `v1:tier("tier_1", p * 2.7 + c * 13.5 + cr * 0.27 + cc * 3.375 + cc1h * 5.4)`)
}

func TestBuildAliasPriceOptionsUsesCNYEmbeddingPriceWithoutFloatCalculation(t *testing.T) {
	options, err := buildAliasPriceOptions(map[string]string{}, []aliasPrice{{
		Alias: "mkt-ov-embedding", Protocol: "embeddings", MultiplierBPS: 10_000,
		Price: OfficialPrice{Currency: "CNY", USDExchangeRateMicros: 137_000, SourceURL: "https://example.com/pricing", Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 7_000_000}}},
	}})
	if err != nil {
		t.Fatalf("生成 Embeddings 价格配置失败：%v", err)
	}
	assertJSONString(t, options["billing_setting.billing_expr"], "mkt-ov-embedding", `v1:tier("tier_1", p * 0.959)`)
}

func TestPriceSynchronizerUpdatesAllOptionsAndChannelBeforeActivation(t *testing.T) {
	repository := &priceSyncRepositoryStub{
		prices: []aliasPrice{{
			Alias: "mkt-ov-42", Protocol: "embeddings", MultiplierBPS: 10_000,
			Price: OfficialPrice{Currency: "USD", USDExchangeRateMicros: 1_000_000, SourceURL: "https://example.com/pricing", Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 1_000_000}}},
		}},
		channel: ChannelPriceRoute{NewAPIChannelID: 41, PublicModels: []string{"gpt-5.5"}, Aliases: map[string]string{"mkt-ov-42": "gpt-5.5"}},
	}
	admin := &adminNewAPIStub{}
	synchronizer := NewPriceSynchronizer(repository, admin)
	if err := synchronizer.SyncOffer(context.Background(), 42, 51); err != nil {
		t.Fatalf("同步报价失败：%v", err)
	}
	if repository.activatedID != 42 {
		t.Fatalf("激活报价 = %d", repository.activatedID)
	}
	optionUpdates := 0
	for _, call := range admin.calls {
		if call == http.MethodPut+" /api/option/" {
			optionUpdates++
		}
	}
	if optionUpdates != len(newAPIPriceOptionKeys) {
		t.Fatalf("价格配置更新次数 = %d；调用=%v", optionUpdates, admin.calls)
	}
	if admin.calls[len(admin.calls)-1] != "PUT /api/channel/" {
		t.Fatalf("渠道映射不是激活前最后一次上游调用：%v", admin.calls)
	}
}

func TestPriceSynchronizerDoesNotActivateWhenExpressionUpdateFails(t *testing.T) {
	repository := &priceSyncRepositoryStub{
		prices: []aliasPrice{{
			Alias: "mkt-ov-42", Protocol: "embeddings", MultiplierBPS: 10_000,
			Price: OfficialPrice{Currency: "USD", USDExchangeRateMicros: 1_000_000, SourceURL: "https://example.com/pricing", Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 1_000_000}}},
		}},
		channel: ChannelPriceRoute{NewAPIChannelID: 41},
	}
	admin := &adminNewAPIStub{failKey: "billing_setting.billing_expr"}
	err := NewPriceSynchronizer(repository, admin).SyncOffer(context.Background(), 42, 51)
	if err == nil || repository.activatedID != 0 {
		t.Fatalf("同步错误=%v，激活报价=%d", err, repository.activatedID)
	}
	for _, call := range admin.calls {
		if call == "PUT /api/channel/" {
			t.Fatalf("表达式失败后仍更新了渠道：%v", admin.calls)
		}
	}
}

func assertJSONString(t *testing.T, raw string, key string, want string) {
	t.Helper()
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		t.Fatalf("解析 %s 失败：%v", raw, err)
	}
	if got := values[key]; got != want {
		t.Fatalf("%s = %q，期望 %q；配置：%s", key, got, want, raw)
	}
}

func assertJSONNumber(t *testing.T, raw string, key string, want string) {
	t.Helper()
	var values map[string]json.Number
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&values); err != nil {
		t.Fatalf("解析 %s 失败：%v", raw, err)
	}
	if got := values[key].String(); got != want {
		t.Fatalf("%s = %q，期望 %q；配置：%s", key, got, want, raw)
	}
}

func assertJSONMissing(t *testing.T, raw string, key string) {
	t.Helper()
	var values map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		t.Fatalf("解析 %s 失败：%v", raw, err)
	}
	if _, exists := values[key]; exists {
		t.Fatalf("配置不应包含 %s：%s", key, raw)
	}
}
