package modelseed

import (
	"strings"
	"testing"
)

func TestParseCatalogRejectsUnknownFields(t *testing.T) {
	_, err := ParseCatalog([]byte(`{
		"version":"2026-07-23-v1",
		"new_api_version":"v1.0.0-rc.21",
		"new_api_source_commit":"bde9b2f44887d34ec54799ae191d50f97914359e",
		"models":[],
		"unexpected":true
	}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("未知字段应被拒绝，错误=%v", err)
	}
}

func TestParseCatalogRejectsModelOutsidePinnedNewAPISupport(t *testing.T) {
	_, err := ParseCatalog([]byte(validCatalogJSON("future-model-not-in-rc21", "OpenAI")))
	if err == nil || !strings.Contains(err.Error(), "New API rc.21") {
		t.Fatalf("未进入固定支持清单的模型应被拒绝，错误=%v", err)
	}
}

func TestCurrentCatalogCoversPinnedProvidersAndProtocols(t *testing.T) {
	catalog, err := CurrentCatalog()
	if err != nil {
		t.Fatalf("读取当前种子：%v", err)
	}
	if catalog.Version != "2026-07-23-v1" || catalog.Digest == "" {
		t.Fatalf("种子版本或摘要无效：%+v", catalog)
	}
	if len(catalog.Models) != 21 {
		t.Fatalf("当前种子模型数=%d，期望 21", len(catalog.Models))
	}
	providers := map[string]bool{}
	protocols := map[string]bool{}
	lifecycles := map[string]int{}
	for _, model := range catalog.Models {
		providers[model.Provider] = true
		protocols[model.Protocol] = true
		lifecycles[model.Lifecycle]++
	}
	for _, provider := range []string{"OpenAI", "Anthropic", "Gemini", "DeepSeek", "Qwen", "Moonshot", "Minimax"} {
		if !providers[provider] {
			t.Fatalf("当前种子缺少厂商 %s", provider)
		}
	}
	for _, protocol := range []string{"chat_completions", "responses", "embeddings"} {
		if !protocols[protocol] {
			t.Fatalf("当前种子缺少协议 %s", protocol)
		}
	}
	if lifecycles["stable"] != 19 || lifecycles["preview"] != 2 || len(lifecycles) != 2 {
		t.Fatalf("当前种子生命周期分布=%+v，期望 stable=19 preview=2", lifecycles)
	}
}

func TestCurrentCatalogLocksTierBoundariesAndCNYRate(t *testing.T) {
	catalog, err := CurrentCatalog()
	if err != nil {
		t.Fatal(err)
	}
	models := make(map[string]SeedModel, len(catalog.Models))
	for _, model := range catalog.Models {
		models[model.PublicName] = model
	}
	gemini := models["gemini-2.5-pro"]
	if len(gemini.Price.Tiers) != 2 || gemini.Price.Tiers[0].MaximumInputTokens == nil || *gemini.Price.Tiers[0].MaximumInputTokens != 200000 || gemini.Price.Tiers[1].MinimumInputTokens != 200001 {
		t.Fatalf("Gemini 200K 阶梯=%+v", gemini.Price.Tiers)
	}
	openAI := models["gpt-5.4"]
	if len(openAI.Price.Tiers) != 2 || openAI.Price.Tiers[0].MaximumInputTokens == nil || *openAI.Price.Tiers[0].MaximumInputTokens != 272000 {
		t.Fatalf("OpenAI 272K 阶梯=%+v", openAI.Price.Tiers)
	}
	for _, name := range []string{"kimi-k2.5", "MiniMax-M2.7", "MiniMax-M2.7-highspeed"} {
		model := models[name]
		if model.Price.Currency != "CNY" || model.Price.USDExchangeRateMicros != 147650 {
			t.Fatalf("人民币模型 %s 的锁定汇率=%+v", name, model.Price)
		}
	}
}

func validCatalogJSON(modelName string, provider string) string {
	return `{
		"version":"2026-07-23-v1",
		"new_api_version":"v1.0.0-rc.21",
		"new_api_source_commit":"bde9b2f44887d34ec54799ae191d50f97914359e",
		"models":[{
			"public_name":"` + modelName + `",
			"provider":"` + provider + `",
			"protocol":"responses",
			"lifecycle":"stable",
			"price":{
				"currency":"USD",
				"usd_exchange_rate_micros":1000000,
				"source_url":"https://example.test/pricing",
				"checked_at":"2026-07-23T00:00:00Z",
				"tiers":[{
					"minimum_input_tokens":0,
					"maximum_input_tokens":null,
					"input_price_micros":2500000,
					"output_price_micros":15000000
				}]
			}
		}]
	}`
}
