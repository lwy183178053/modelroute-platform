package modelseed

var pinnedNewAPIModels = map[string]map[string]struct{}{
	"OpenAI": {
		"gpt-5.4": {}, "gpt-5.4-pro": {},
		"text-embedding-3-small": {}, "text-embedding-3-large": {},
	},
	"Anthropic": {
		"claude-opus-4-8": {}, "claude-opus-4-7": {}, "claude-opus-4-6": {},
		"claude-sonnet-4-6": {}, "claude-haiku-4-5-20251001": {},
	},
	"Gemini": {
		"gemini-3.1-pro-preview": {}, "gemini-3-flash-preview": {},
		"gemini-2.5-pro": {}, "gemini-2.5-flash": {}, "gemini-2.5-flash-lite": {},
		"gemini-embedding-001": {},
	},
	"DeepSeek": {
		"deepseek-v4-flash": {}, "deepseek-v4-pro": {},
	},
	"Qwen": {
		"qwen-max": {},
	},
	"Moonshot": {
		"kimi-k2.5": {},
	},
	"Minimax": {
		"MiniMax-M2.7": {}, "MiniMax-M2.7-highspeed": {},
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
