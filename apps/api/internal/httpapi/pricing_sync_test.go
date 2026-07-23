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
	channels    []ChannelPriceRoute
	activatedID int64
	activate    bool
}

func (s *priceSyncRepositoryStub) AliasPrices(context.Context) ([]aliasPrice, error) {
	return s.prices, nil
}

func (s *priceSyncRepositoryStub) OfferChannelRoutes(context.Context, int64) ([]ChannelPriceRoute, error) {
	return s.channels, nil
}

func (s *priceSyncRepositoryStub) FinalizeOffer(_ context.Context, offerID int64, activate bool) error {
	s.activatedID = offerID
	s.activate = activate
	return nil
}

type adminNewAPIStub struct {
	calls          []string
	updates        map[string]string
	failKey        string
	failedKey      bool
	failChannel    bool
	failedChannel  bool
	channelModels  string
	channelMapping string
}

func (s *adminNewAPIStub) Do(_ context.Context, method string, path string, body any) ([]byte, error) {
	s.calls = append(s.calls, method+" "+path)
	switch method + " " + path {
	case "GET /api/option/":
		return []byte(`[{"key":"billing_setting.billing_mode","value":"{\"existing\":\"tiered_expr\"}"},{"key":"billing_setting.billing_expr","value":"{\"existing\":\"v1:tier(\\\"tier_1\\\", p * 2)\"}"}]`), nil
	case "GET /api/channel/41":
		models := s.channelModels
		mapping := s.channelMapping
		if models == "" {
			models = "legacy"
		}
		if mapping == "" {
			mapping = `{"legacy":"legacy-upstream"}`
		}
		return []byte(fmt.Sprintf(`{"id":41,"models":%q,"model_mapping":%q}`, models, mapping)), nil
	case "PUT /api/option/":
		update := body.(map[string]any)
		if strings.TrimSpace(fmt.Sprint(update["key"])) == "" || strings.TrimSpace(fmt.Sprint(update["value"])) == "" {
			return nil, fmt.Errorf("价格配置为空")
		}
		if s.updates == nil {
			s.updates = make(map[string]string)
		}
		if fmt.Sprint(update["key"]) == s.failKey && !s.failedKey {
			s.failedKey = true
			return nil, fmt.Errorf("模拟配置写入失败")
		}
		s.updates[fmt.Sprint(update["key"])] = fmt.Sprint(update["value"])
		return []byte(`null`), nil
	case "PUT /api/channel/":
		update := body.(map[string]any)
		s.channelModels = update["models"].(string)
		s.channelMapping = update["model_mapping"].(string)
		var mapping map[string]string
		if err := json.Unmarshal([]byte(s.channelMapping), &mapping); err != nil {
			return nil, err
		}
		if s.failChannel && !s.failedChannel {
			s.failedChannel = true
			return nil, fmt.Errorf("模拟渠道写入失败")
		}
		return []byte(`null`), nil
	default:
		return nil, fmt.Errorf("未知调用 %s %s", method, path)
	}
}

func TestBuildAliasPriceOptionsPreservesExistingAndWritesTieredExpression(t *testing.T) {
	output := int64(15_000_000)
	cacheRead := int64(300_000)
	cacheWrite := int64(3_750_000)
	existing := map[string]string{
		"billing_setting.billing_mode": `{"existing":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"existing":"v1:tier(\"tier_1\", p * 2)"}`,
	}
	offers := []aliasPrice{{
		Alias: "mkt-ov-42", PriceKind: "text", MultiplierBPS: 9_000,
		Price: OfficialPrice{Tiers: []PriceTier{{
			MinimumInputTokens: 0, InputPriceMicros: 3_000_000, OutputPriceMicros: &output,
			CacheReadPriceMicros: &cacheRead, CacheWritePriceMicros: &cacheWrite,
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
	assertJSONString(t, options["billing_setting.billing_expr"], "mkt-ov-42", `v1:tier("tier_1", p * 2.7 + c * 13.5 + cr * 0.27 + cc * 3.375 + cc1h * 3.375)`)
}

func TestBuildAliasPriceOptionsUsesImageMatrix(t *testing.T) {
	options, err := buildAliasPriceOptions(map[string]string{}, []aliasPrice{{
		Alias: "mkt-ov-image", PriceKind: "image", MultiplierBPS: 10_000,
		Price: OfficialPrice{ImagePrices: []ImagePriceItem{{Quality: "standard", Size: "1024x1024", PriceMicros: 40_000}}},
	}})
	if err != nil {
		t.Fatalf("生成图片价格配置失败：%v", err)
	}
	var expressions map[string]string
	if err := json.Unmarshal([]byte(options["billing_setting.billing_expr"]), &expressions); err != nil || !strings.Contains(expressions["mkt-ov-image"], `param("n")`) {
		t.Fatalf("图片价格表达式=%v，错误=%v", expressions, err)
	}
}

func TestPriceSynchronizerUpdatesAllOptionsAndChannelBeforeActivation(t *testing.T) {
	repository := &priceSyncRepositoryStub{
		prices: []aliasPrice{{
			Alias: "mkt-ov-42", PriceKind: "text", MultiplierBPS: 10_000,
			Price: OfficialPrice{Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 1_000_000, OutputPriceMicros: int64Pointer(2_000_000)}}},
		}},
		channels: []ChannelPriceRoute{{MerchantChannelID: 51, NewAPIChannelID: 41, PublicModels: []string{"gpt-5.5"}, Aliases: map[string]string{"mkt-ov-42": "gpt-5.5"}}},
	}
	admin := &adminNewAPIStub{}
	synchronizer := NewPriceSynchronizer(repository, admin)
	if err := synchronizer.SyncOffer(context.Background(), 42, true); err != nil {
		t.Fatalf("同步报价失败：%v", err)
	}
	if repository.activatedID != 42 || !repository.activate {
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
	if admin.channelModels != "gpt-5.5,legacy,mkt-ov-42" {
		t.Fatalf("渠道模型=%q", admin.channelModels)
	}
	var mapping map[string]string
	if err := json.Unmarshal([]byte(admin.channelMapping), &mapping); err != nil || mapping["legacy"] != "legacy-upstream" || mapping["mkt-ov-42"] != "gpt-5.5" {
		t.Fatalf("渠道映射=%v，错误=%v", mapping, err)
	}
}

func TestPriceSynchronizerDoesNotActivateWhenExpressionUpdateFails(t *testing.T) {
	repository := &priceSyncRepositoryStub{
		prices: []aliasPrice{{
			Alias: "mkt-ov-42", PriceKind: "text", MultiplierBPS: 10_000,
			Price: OfficialPrice{Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 1_000_000, OutputPriceMicros: int64Pointer(2_000_000)}}},
		}},
		channels: []ChannelPriceRoute{{MerchantChannelID: 51, NewAPIChannelID: 41}},
	}
	admin := &adminNewAPIStub{failKey: "billing_setting.billing_expr"}
	err := NewPriceSynchronizer(repository, admin).SyncOffer(context.Background(), 42, true)
	if err == nil || repository.activatedID != 0 {
		t.Fatalf("同步错误=%v，激活报价=%d", err, repository.activatedID)
	}
	for _, call := range admin.calls {
		if call == "PUT /api/channel/" {
			t.Fatalf("表达式失败后仍更新了渠道：%v", admin.calls)
		}
	}
	if admin.updates["billing_setting.billing_mode"] != `{"existing":"tiered_expr"}` || admin.updates["billing_setting.billing_expr"] != `{"existing":"v1:tier(\"tier_1\", p * 2)"}` {
		t.Fatalf("失败后价格配置未回滚：%v", admin.updates)
	}
}

func TestPriceSynchronizerRestoresOptionsAndChannelWhenChannelUpdateFails(t *testing.T) {
	repository := &priceSyncRepositoryStub{
		prices: []aliasPrice{{
			Alias: "mkt-ov-42", PriceKind: "text", MultiplierBPS: 10_000,
			Price: OfficialPrice{Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 1_000_000, OutputPriceMicros: int64Pointer(2_000_000)}}},
		}},
		channels: []ChannelPriceRoute{{MerchantChannelID: 51, NewAPIChannelID: 41, PublicModels: []string{"gpt-5.5"}, Aliases: map[string]string{"mkt-ov-42": "gpt-5.5"}}},
	}
	admin := &adminNewAPIStub{failChannel: true}
	err := NewPriceSynchronizer(repository, admin).SyncOffer(context.Background(), 42, true)
	if err == nil || repository.activatedID != 0 {
		t.Fatalf("同步错误=%v，激活报价=%d", err, repository.activatedID)
	}
	if admin.channelModels != "legacy" || admin.channelMapping != `{"legacy":"legacy-upstream"}` {
		t.Fatalf("失败后渠道未回滚：models=%q mapping=%q", admin.channelModels, admin.channelMapping)
	}
	if admin.updates["billing_setting.billing_mode"] != `{"existing":"tiered_expr"}` || admin.updates["billing_setting.billing_expr"] != `{"existing":"v1:tier(\"tier_1\", p * 2)"}` {
		t.Fatalf("失败后价格配置未回滚：%v", admin.updates)
	}
}

func TestPriceSynchronizerCanFinalizeUpdatedOfferAsInactive(t *testing.T) {
	repository := &priceSyncRepositoryStub{
		prices: []aliasPrice{{
			Alias: "mkt-ov-43", PriceKind: "text", MultiplierBPS: 9_000,
			Price: OfficialPrice{Tiers: []PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 1_000_000, OutputPriceMicros: int64Pointer(2_000_000)}}},
		}},
		channels: []ChannelPriceRoute{{MerchantChannelID: 51, NewAPIChannelID: 41, PublicModels: []string{"gpt-5.5"}, Aliases: map[string]string{"mkt-ov-43": "gpt-5.5"}}},
	}
	if err := NewPriceSynchronizer(repository, &adminNewAPIStub{}).SyncOffer(context.Background(), 43, false); err != nil {
		t.Fatalf("同步下架模型新价格失败：%v", err)
	}
	if repository.activatedID != 43 || repository.activate {
		t.Fatalf("下架模型同步结束状态错误：id=%d activate=%v", repository.activatedID, repository.activate)
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
