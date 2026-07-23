package httpapi

import "strings"

// SyncedChannelModel is the platform view of one model returned by an upstream channel.
type SyncedChannelModel struct {
	UpstreamModelName string `json:"upstream_model_name"`
	CanonicalModelID  int64  `json:"canonical_model_id,omitempty"`
	Model             string `json:"model,omitempty"`
	Provider          string `json:"provider,omitempty"`
	Protocol          string `json:"protocol,omitempty"`
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
			item.Protocol = model.Protocol
			item.MatchStatus = "matched"
		}
		result = append(result, item)
	}
	return result
}

func normalizeUpstreamModelName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func channelTypeForName(name string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "openai", "openai_compatible", "openai-compatible":
		return 1, true
	case "anthropic":
		return 14, true
	case "gemini":
		return 24, true
	default:
		return 0, false
	}
}
