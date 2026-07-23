package httpapi

import (
	"encoding/json"
	"reflect"
	"testing"
)

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

func TestSupportedChannelTypeRejectsMiniMax(t *testing.T) {
	if _, ok := channelTypeForName("minimax"); ok {
		t.Fatal("MiniMax 不应作为可选渠道类型")
	}
	for _, name := range []string{"openai_compatible", "anthropic", "gemini"} {
		if _, ok := channelTypeForName(name); !ok {
			t.Fatalf("渠道类型 %q 应可用", name)
		}
	}
}

func TestMatchUpstreamModelsUsesNormalizedNamesButPreservesDisplay(t *testing.T) {
	got := matchUpstreamModels([]string{" GPT-5.6 ", "gpt-5.6"}, []CanonicalModel{{ID: 8, PublicName: "gpt-5.6", Status: "active"}})
	if len(got) != 1 || got[0].MatchStatus != "matched" || got[0].UpstreamModelName != "GPT-5.6" || got[0].CanonicalModelID != 8 {
		t.Fatalf("规范化匹配结果无效：%+v", got)
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
