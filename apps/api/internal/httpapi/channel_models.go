package httpapi

import (
	"errors"
	"strings"
)

var supportedChannelProviders = map[string]string{
	"openai":    "OpenAI",
	"anthropic": "Anthropic",
	"gemini":    "Gemini",
	"deepseek":  "DeepSeek",
	"qwen":      "Qwen",
	"moonshot":  "Moonshot",
}

// SyncedChannelModel is the platform view of one model returned by an upstream channel.
type SyncedChannelModel struct {
	UpstreamModelName string `json:"upstream_model_name"`
	CanonicalModelID  int64  `json:"canonical_model_id,omitempty"`
	Model             string `json:"model,omitempty"`
	Provider          string `json:"provider,omitempty"`
	MatchStatus       string `json:"match_status"`
}

func matchUpstreamModels(upstream []string, canonical []CanonicalModel) []SyncedChannelModel {
	byName := make(map[string]CanonicalModel, len(canonical))
	for _, model := range canonical {
		name := normalizeUpstreamModelName(model.PublicName)
		if name == "" || model.Status != "active" {
			continue
		}
		byName[name] = model
	}
	seen := make(map[string]struct{}, len(upstream))
	result := make([]SyncedChannelModel, 0, len(upstream))
	for _, raw := range upstream {
		displayName := strings.TrimSpace(raw)
		name := normalizeUpstreamModelName(displayName)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		item := SyncedChannelModel{UpstreamModelName: displayName, MatchStatus: "unmatched"}
		if model, ok := byName[name]; ok {
			item.CanonicalModelID = model.ID
			item.Model = model.PublicName
			item.Provider = model.Provider
			item.MatchStatus = "matched"
		}
		result = append(result, item)
	}
	return result
}

func matchUpstreamModelsForProvider(upstream []string, canonical []CanonicalModel, provider string) []SyncedChannelModel {
	provider, err := normalizeChannelProvider(provider)
	if err != nil {
		return nil
	}
	selected := make([]CanonicalModel, 0, len(canonical))
	knownOtherProviders := make(map[string]struct{}, len(canonical))
	for _, model := range canonical {
		name := normalizeUpstreamModelName(model.PublicName)
		if name == "" || model.Status != "active" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(model.Provider), provider) {
			selected = append(selected, model)
			continue
		}
		knownOtherProviders[name] = struct{}{}
	}

	matched := matchUpstreamModels(upstream, selected)
	result := make([]SyncedChannelModel, 0, len(matched))
	for _, item := range matched {
		if item.MatchStatus != "matched" {
			if _, belongsElsewhere := knownOtherProviders[normalizeUpstreamModelName(item.UpstreamModelName)]; belongsElsewhere {
				continue
			}
			item.Provider = provider
		}
		result = append(result, item)
	}
	return result
}

func normalizeChannelProvider(raw string) (string, error) {
	provider, ok := supportedChannelProviders[strings.ToLower(strings.TrimSpace(raw))]
	if !ok {
		return "", errors.New("渠道厂商只支持 OpenAI、Anthropic、Gemini、DeepSeek、Qwen 或 Moonshot")
	}
	return provider, nil
}

func channelTypesForProvider(provider string) []int {
	provider, err := normalizeChannelProvider(provider)
	if err != nil {
		return nil
	}
	switch provider {
	case "Anthropic":
		return []int{14, 1}
	case "Gemini":
		return []int{24, 1}
	default:
		return []int{1}
	}
}

func normalizeUpstreamModelName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func newAPIChannelBaseURL(channelType int, baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	suffix := ""
	switch channelType {
	case 1, 14:
		suffix = "/v1"
	case 24:
		suffix = "/v1beta"
	}
	return strings.TrimSuffix(baseURL, suffix)
}
