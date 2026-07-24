package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

type channelModelAdminStub struct {
	channelTypes []int
}

func (s *channelModelAdminStub) Do(_ context.Context, _ string, _ string, body any) ([]byte, error) {
	request := body.(map[string]any)
	channelType := request["type"].(int)
	s.channelTypes = append(s.channelTypes, channelType)
	if channelType == 14 {
		return nil, errors.New("native endpoint unavailable")
	}
	return []byte(`{"data":[{"id":"claude-opus-4.8"},{"id":"gpt-5.6"}]}`), nil
}

func TestMatchUpstreamModelsMarksMatchedAndUnmatchedModels(t *testing.T) {
	models := []CanonicalModel{
		{ID: 1, PublicName: "gpt-5.6", Status: "active"},
		{ID: 2, PublicName: "claude-fable-5", Status: "active"},
		{ID: 3, PublicName: "archived-model", Status: "archived"},
	}
	got := matchUpstreamModels([]string{"gpt-5.6", "vendor-only", "gpt-5.6", "archived-model"}, models)
	want := []SyncedChannelModel{
		{UpstreamModelName: "gpt-5.6", CanonicalModelID: 1, Model: "gpt-5.6", MatchStatus: "matched"},
		{UpstreamModelName: "vendor-only", MatchStatus: "unmatched"},
		{UpstreamModelName: "archived-model", MatchStatus: "unmatched"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("同步模型匹配结果=%+v，期望=%+v", got, want)
	}
}

func TestSupportedChannelProvidersRejectMiniMax(t *testing.T) {
	if _, err := normalizeChannelProvider("MiniMax"); err == nil {
		t.Fatal("MiniMax 不应作为可选渠道厂商")
	}
	for _, name := range []string{"OpenAI", "Anthropic", "Gemini", "DeepSeek", "Qwen", "Moonshot"} {
		if _, err := normalizeChannelProvider(name); err != nil {
			t.Fatalf("渠道厂商 %q 应可用：%v", name, err)
		}
	}
}

func TestMatchUpstreamModelsUsesNormalizedNamesButPreservesDisplay(t *testing.T) {
	got := matchUpstreamModels([]string{" GPT-5.6 ", "gpt-5.6"}, []CanonicalModel{{ID: 8, PublicName: "gpt-5.6", Status: "active"}})
	if len(got) != 1 || got[0].MatchStatus != "matched" || got[0].UpstreamModelName != "GPT-5.6" || got[0].CanonicalModelID != 8 {
		t.Fatalf("规范化匹配结果无效：%+v", got)
	}
}

func TestMatchUpstreamModelsForProviderExcludesOtherKnownProviders(t *testing.T) {
	models := []CanonicalModel{
		{ID: 1, PublicName: "gpt-5.6", Provider: "OpenAI", Status: "active"},
		{ID: 2, PublicName: "claude-opus-4.8", Provider: "Anthropic", Status: "active"},
	}
	got := matchUpstreamModelsForProvider(
		[]string{"gpt-5.6", "claude-opus-4.8", "openai-future"},
		models,
		"OpenAI",
	)
	want := []SyncedChannelModel{
		{UpstreamModelName: "gpt-5.6", CanonicalModelID: 1, Model: "gpt-5.6", Provider: "OpenAI", MatchStatus: "matched"},
		{UpstreamModelName: "openai-future", Provider: "OpenAI", MatchStatus: "unmatched"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("按厂商同步结果=%+v，期望=%+v", got, want)
	}
}

func TestChannelProviderDeterminesHiddenAdapterFallbackOrder(t *testing.T) {
	tests := map[string][]int{
		"OpenAI":    {1},
		"DeepSeek":  {1},
		"Qwen":      {1},
		"Moonshot":  {1},
		"Anthropic": {14, 1},
		"Gemini":    {24, 1},
	}
	for provider, want := range tests {
		canonical, err := normalizeChannelProvider(provider)
		if err != nil || canonical != provider {
			t.Fatalf("厂商 %q 规范化=%q err=%v", provider, canonical, err)
		}
		if got := channelTypesForProvider(provider); !reflect.DeepEqual(got, want) {
			t.Fatalf("厂商 %q 适配顺序=%v，期望=%v", provider, got, want)
		}
	}
	if _, err := normalizeChannelProvider("MiniMax"); err == nil {
		t.Fatal("MiniMax 不应作为渠道厂商")
	}
}

func TestFetchAndMatchChannelModelsFallsBackAndKeepsSelectedProvider(t *testing.T) {
	admin := &channelModelAdminStub{}
	s := &server{services: Services{
		AdminNewAPI: admin,
		Routing: &routingRepositoryStub{listModels: func(context.Context) ([]CanonicalModel, error) {
			return []CanonicalModel{
				{ID: 1, PublicName: "gpt-5.6", Provider: "OpenAI", Status: "active"},
				{ID: 2, PublicName: "claude-opus-4.8", Provider: "Anthropic", Status: "active"},
			}, nil
		}},
	}}
	request := httptest.NewRequest("POST", "/api/merchant/channel-models/sync", nil)
	models, channelType, err := s.fetchAndMatchChannelModels(request, platformauth.Session{}, "Anthropic", "https://api.example.com/v1", "sk-upstream-secret")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(admin.channelTypes, []int{14, 1}) {
		t.Fatalf("适配尝试顺序=%v", admin.channelTypes)
	}
	if channelType != 1 || len(models) != 1 || models[0].Model != "claude-opus-4.8" || models[0].Provider != "Anthropic" {
		t.Fatalf("回退同步结果 type=%d models=%+v", channelType, models)
	}
}

func TestExtractUpstreamModelNamesSupportsNewAPIShapes(t *testing.T) {
	data := json.RawMessage(`{"data":[{"id":"gpt-5.6"},{"id":"vendor-only"},{"id":"gpt-5.6"}]}`)
	got := extractUpstreamModelNames(data)
	if !reflect.DeepEqual(got, []string{"gpt-5.6", "vendor-only"}) {
		t.Fatalf("上游模型解析结果=%v", got)
	}
}

func TestNewAPIChannelBaseURLRemovesProviderVersionSuffix(t *testing.T) {
	tests := []struct {
		channelType int
		baseURL     string
		want        string
	}{
		{channelType: 1, baseURL: "https://api.example.com/v1", want: "https://api.example.com"},
		{channelType: 14, baseURL: "https://api.anthropic.com/v1", want: "https://api.anthropic.com"},
		{channelType: 24, baseURL: "https://generativelanguage.googleapis.com/v1beta", want: "https://generativelanguage.googleapis.com"},
		{channelType: 1, baseURL: "https://gateway.example.com/openai/v1", want: "https://gateway.example.com/openai"},
		{channelType: 1, baseURL: "https://api.example.com", want: "https://api.example.com"},
	}
	for _, test := range tests {
		if got := newAPIChannelBaseURL(test.channelType, test.baseURL); got != test.want {
			t.Errorf("渠道类型 %d 的 New API 基础地址=%q，期望=%q", test.channelType, got, test.want)
		}
	}
}
