package modelseed

import (
	"strings"
	"testing"
)

func TestParseCatalogRejectsUnknownFields(t *testing.T) {
	_, err := ParseCatalog([]byte(`{
		"version":"2026-07-23-v2",
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

func TestParseCatalogRejectsNonOfficialEvidenceURL(t *testing.T) {
	body := strings.Replace(validCatalogJSON("gpt-5.4", "OpenAI"), "https://developers.openai.com/api/docs/models/gpt-5.4", "https://example.test/pricing", 1)
	_, err := ParseCatalog([]byte(body))
	if err == nil || !strings.Contains(err.Error(), "官方域名") {
		t.Fatalf("非官方核价来源应被拒绝，错误=%v", err)
	}
}

func TestCurrentCatalogCoversPinnedProvidersAndPriceKinds(t *testing.T) {
	catalog, err := CurrentCatalog()
	if err != nil {
		t.Fatalf("读取当前种子：%v", err)
	}
	if catalog.Version != "2026-07-23-v2" || catalog.Digest == "" {
		t.Fatalf("种子版本或摘要无效：%+v", catalog)
	}
	if len(catalog.Models) != 21 {
		t.Fatalf("当前种子模型数=%d，期望 21", len(catalog.Models))
	}
	providers := map[string]bool{}
	priceKinds := map[string]bool{}
	models := map[string]bool{}
	for _, model := range catalog.Models {
		providers[model.Provider] = true
		priceKinds[comparablePrice(model.Price).PriceKind] = true
		models[model.PublicName] = true
	}
	for _, provider := range []string{"OpenAI", "Anthropic", "Gemini", "DeepSeek", "Qwen", "Moonshot"} {
		if !providers[provider] {
			t.Fatalf("当前种子缺少厂商 %s", provider)
		}
	}
	if providers["Minimax"] {
		t.Fatal("当前种子不应包含 MiniMax")
	}
	for _, priceKind := range []string{"text", "image"} {
		if !priceKinds[priceKind] {
			t.Fatalf("当前种子缺少价格类型 %s", priceKind)
		}
	}
	for _, name := range []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna", "gpt-5.5", "gpt-5.4", "gpt-image-1", "gpt-image-1.5", "gpt-image-2", "claude-opus-4-8", "claude-opus-4-7", "claude-opus-4-6", "claude-sonnet-5", "kimi-k3"} {
		if !models[name] {
			t.Fatalf("当前种子缺少已核验模型 %s", name)
		}
	}
	for _, name := range []string{"gpt-5.4-pro", "claude-fable-5", "claude-sonnet-4-6", "claude-haiku-4-5-20251001"} {
		if models[name] {
			t.Fatalf("缺少 API 单价证据的模型不应自动发布：%s", name)
		}
	}
}

func TestCurrentCatalogLocksTierBoundariesAndNumericOneToOnePricing(t *testing.T) {
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
	kimi := models["kimi-k3"]
	if len(kimi.Price.Tiers) != 1 || kimi.Price.Tiers[0].InputPriceMicros != 20_000_000 || kimi.Price.Tiers[0].OutputPriceMicros == nil || *kimi.Price.Tiers[0].OutputPriceMicros != 100_000_000 {
		t.Fatalf("Kimi K3 未按官方数字一比一保存：%+v", kimi.Price)
	}
	image := models["gpt-image-2"]
	if len(image.Price.ImagePrices) != 9 || image.Price.ImagePrices[0].PriceMicros <= 0 {
		t.Fatalf("GPT Image 2 价格矩阵=%+v", image.Price.ImagePrices)
	}
}

func validCatalogJSON(modelName string, provider string) string {
	return `{
		"version":"2026-07-23-v2",
		"new_api_version":"v1.0.0-rc.21",
		"new_api_source_commit":"bde9b2f44887d34ec54799ae191d50f97914359e",
		"models":[{
			"public_name":"` + modelName + `",
			"provider":"` + provider + `",
			"evidence_url":"https://developers.openai.com/api/docs/models/gpt-5.4",
			"price":{
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
