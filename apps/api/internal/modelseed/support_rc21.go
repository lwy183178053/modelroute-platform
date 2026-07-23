package modelseed

var pinnedNewAPIModels = map[string]map[string]struct{}{
	"OpenAI": {
		"gpt-5.6-sol": {}, "gpt-5.6-terra": {}, "gpt-5.6-luna": {},
		"gpt-5.5": {}, "gpt-5.4": {},
		"gpt-image-1": {}, "gpt-image-1.5": {}, "gpt-image-2": {},
	},
	"Anthropic": {
		"claude-opus-4-8": {}, "claude-opus-4-7": {}, "claude-opus-4-6": {}, "claude-sonnet-5": {},
	},
	"Gemini": {
		"gemini-3.1-pro-preview": {}, "gemini-3-flash-preview": {},
		"gemini-2.5-pro": {}, "gemini-2.5-flash": {}, "gemini-2.5-flash-lite": {},
	},
	"DeepSeek": {
		"deepseek-v4-flash": {}, "deepseek-v4-pro": {},
	},
	"Qwen": {
		"qwen-max": {},
	},
	"Moonshot": {
		"kimi-k3": {},
	},
}

func supportedByPinnedNewAPI(provider string, model string) bool {
	models, exists := pinnedNewAPIModels[provider]
	if !exists {
		return false
	}
	_, exists = models[model]
	return exists
}
