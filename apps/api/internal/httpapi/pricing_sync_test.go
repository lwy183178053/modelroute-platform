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
	calls []string
}

func (s *adminNewAPIStub) Do(_ context.Context, method string, path string, body any) ([]byte, error) {
	s.calls = append(s.calls, method+" "+path)
	switch method + " " + path {
	case "GET /api/option/":
		return []byte(`[{"key":"ModelRatio","value":"{\"existing\":2}"}]`), nil
	case "GET /api/channel/41":
		return []byte(`{"id":41,"models":"legacy","model_mapping":"{\"legacy\":\"legacy-upstream\"}"}`), nil
	case "PUT /api/option/":
		update := body.(map[string]any)
		if strings.TrimSpace(fmt.Sprint(update["key"])) == "" || strings.TrimSpace(fmt.Sprint(update["value"])) == "" {
			return nil, fmt.Errorf("价格配置为空")
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

func TestBuildAliasPriceOptionsPreservesExistingAndCopiesAllTokenDimensions(t *testing.T) {
	existing := map[string]string{
		"ModelRatio":           `{"existing-model":2}`,
		"ModelPrice":           `{"fixed-existing":0.8}`,
		"CompletionRatio":      `{"existing-model":3}`,
		"CacheRatio":           `{}`,
		"CreateCacheRatio":     `{}`,
		"ImageRatio":           `{}`,
		"AudioRatio":           `{}`,
		"AudioCompletionRatio": `{}`,
	}
	offers := []aliasPrice{{
		Alias: "mkt-ov-42", BillingMode: "token", MultiplierBPS: 4_000,
		ModelRatioMicros: 1_250_000, CompletionRatioMicros: 3_000_000,
		CacheRatioMicros: 100_000, CreateCacheRatioMicros: 1_250_000,
		ImageRatioMicros: 2_000_000, AudioRatioMicros: 1_500_000,
		AudioCompletionRatioMicros: 2_500_000,
	}}
	options, err := buildAliasPriceOptions(existing, offers)
	if err != nil {
		t.Fatalf("生成价格配置失败：%v", err)
	}
	assertJSONNumber(t, options["ModelRatio"], "existing-model", "2")
	assertJSONNumber(t, options["ModelRatio"], "mkt-ov-42", "0.5")
	assertJSONNumber(t, options["CompletionRatio"], "mkt-ov-42", "3")
	assertJSONNumber(t, options["CacheRatio"], "mkt-ov-42", "0.1")
	assertJSONNumber(t, options["CreateCacheRatio"], "mkt-ov-42", "1.25")
	assertJSONNumber(t, options["ImageRatio"], "mkt-ov-42", "2")
	assertJSONNumber(t, options["AudioRatio"], "mkt-ov-42", "1.5")
	assertJSONNumber(t, options["AudioCompletionRatio"], "mkt-ov-42", "2.5")
	assertJSONMissing(t, options["ModelPrice"], "mkt-ov-42")
}

func TestBuildAliasPriceOptionsUsesFixedPriceWithoutFloatCalculation(t *testing.T) {
	options, err := buildAliasPriceOptions(map[string]string{}, []aliasPrice{{
		Alias: "mkt-ov-fixed", BillingMode: "fixed", MultiplierBPS: 4_000,
		ModelPriceMicroUSD: 500_000, CompletionRatioMicros: 1_000_000,
		CacheRatioMicros: 1_000_000, CreateCacheRatioMicros: 1_000_000,
		ImageRatioMicros: 1_000_000, AudioRatioMicros: 1_000_000,
		AudioCompletionRatioMicros: 1_000_000,
	}})
	if err != nil {
		t.Fatalf("生成固定价格配置失败：%v", err)
	}
	assertJSONNumber(t, options["ModelPrice"], "mkt-ov-fixed", "0.2")
	assertJSONMissing(t, options["ModelRatio"], "mkt-ov-fixed")
}

func TestPriceSynchronizerUpdatesAllOptionsAndChannelBeforeActivation(t *testing.T) {
	repository := &priceSyncRepositoryStub{
		prices: []aliasPrice{{
			Alias: "mkt-ov-42", BillingMode: "token", MultiplierBPS: 4_000,
			ModelRatioMicros: 1_250_000, CompletionRatioMicros: 3_000_000,
			CacheRatioMicros: 100_000, CreateCacheRatioMicros: 1_250_000,
			ImageRatioMicros: 1_000_000, AudioRatioMicros: 1_000_000,
			AudioCompletionRatioMicros: 1_000_000,
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
