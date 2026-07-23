package modelseed

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentCatalogMatchesPinnedNewAPISource(t *testing.T) {
	sourceRoot := os.Getenv("NEW_API_RC21_SOURCE_DIR")
	if sourceRoot == "" {
		t.Skip("未设置 New API rc.21 源码目录")
	}
	head, err := exec.Command("git", "-C", sourceRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("读取 New API 源码提交：%v", err)
	}
	if strings.TrimSpace(string(head)) != pinnedNewAPICommit {
		t.Fatalf("New API 源码提交=%s，期望 %s", strings.TrimSpace(string(head)), pinnedNewAPICommit)
	}
	providerFiles := map[string]string{
		"OpenAI":    "relay/channel/openai/constant.go",
		"Anthropic": "relay/channel/claude/constants.go",
		"Gemini":    "relay/channel/gemini/constant.go",
		"DeepSeek":  "relay/channel/deepseek/constants.go",
		"Qwen":      "relay/channel/ali/constants.go",
		"Moonshot":  "relay/channel/moonshot/constants.go",
		"Minimax":   "relay/channel/minimax/constants.go",
	}
	contents := make(map[string]string, len(providerFiles))
	for provider, relativePath := range providerFiles {
		body, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(relativePath)))
		if err != nil {
			t.Fatalf("读取 %s 适配器清单：%v", provider, err)
		}
		contents[provider] = string(body)
	}
	catalog, err := CurrentCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range catalog.Models {
		if listedByDefault(model.Provider, model.PublicName) && !strings.Contains(contents[model.Provider], `"`+model.PublicName+`"`) {
			t.Errorf("New API rc.21 的 %s 默认清单缺少模型 %s", model.Provider, model.PublicName)
		}
	}
}

func listedByDefault(provider string, model string) bool {
	genericModels := map[string]bool{
		"gpt-5.6-sol": true, "gpt-5.6-terra": true, "gpt-5.6-luna": true,
		"gpt-5.5": true, "gpt-image-2": true, "kimi-k3": true, "MiniMax-M3": true,
	}
	return !genericModels[model]
}
