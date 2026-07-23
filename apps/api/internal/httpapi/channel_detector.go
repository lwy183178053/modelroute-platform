package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type channelAdapter struct {
	name        string
	channelType int
}

type NewAPIChannelDetector struct {
	admin AdminNewAPI
}

func NewNewAPIChannelDetector(admin AdminNewAPI) *NewAPIChannelDetector {
	return &NewAPIChannelDetector{admin: admin}
}

func (d *NewAPIChannelDetector) Test(ctx context.Context, request ChannelTestRequest) (ChannelTestOutcome, error) {
	if d == nil || d.admin == nil || request.Model.ID <= 0 || strings.TrimSpace(request.APIKey) == "" {
		return ChannelTestOutcome{}, errors.New("渠道检测参数无效")
	}
	var attempts []error
	for _, adapter := range adaptersForProvider(request.Model.Provider) {
		latency, err := d.testAdapter(ctx, request, adapter)
		if err != nil {
			attempts = append(attempts, fmt.Errorf("%s: %w", adapter.name, err))
			continue
		}
		return ChannelTestOutcome{
			AdapterType: adapter.name, Capabilities: []string{request.Model.Protocol}, LatencyMS: latency,
		}, nil
	}
	return ChannelTestOutcome{}, fmt.Errorf("全部适配器检测失败：%w", errors.Join(attempts...))
}

func (d *NewAPIChannelDetector) testAdapter(ctx context.Context, request ChannelTestRequest, adapter channelAdapter) (int, error) {
	name, err := temporaryChannelName()
	if err != nil {
		return 0, err
	}
	baseURL := request.BaseURL
	if adapter.channelType == 1 {
		baseURL = strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/v1")
	}
	if _, err := d.admin.Do(ctx, http.MethodPost, "/api/channel/", map[string]any{
		"mode": "single",
		"channel": map[string]any{
			"type": adapter.channelType, "key": request.APIKey, "name": name,
			"base_url": baseURL, "models": request.Model.PublicName,
			"group": "default", "status": 1, "test_model": request.Model.PublicName,
			"tag": "modelroute-channel-test",
		},
	}); err != nil {
		return 0, fmt.Errorf("创建临时渠道：%w", err)
	}
	data, err := d.admin.Do(ctx, http.MethodGet, "/api/channel/search?"+url.Values{
		"keyword": {name}, "p": {"1"}, "page_size": {"20"}, "id_sort": {"true"},
	}.Encode(), nil)
	if err != nil {
		return 0, fmt.Errorf("查找临时渠道：%w", err)
	}
	var result struct {
		Items []struct {
			ID int `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil || len(result.Items) == 0 || result.Items[0].ID <= 0 {
		return 0, errors.New("临时渠道编号缺失")
	}
	channelID := result.Items[0].ID
	started := time.Now()
	_, testErr := d.admin.Do(ctx, http.MethodGet, fmt.Sprintf("/api/channel/test/%d?%s", channelID, url.Values{
		"model": {request.Model.PublicName},
	}.Encode()), nil)
	latency := int(time.Since(started).Milliseconds())
	if latency < 1 {
		latency = 1
	}
	_, cleanupErr := d.admin.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/channel/%d", channelID), nil)
	if testErr != nil {
		return 0, errors.Join(fmt.Errorf("发送 hi：%w", testErr), cleanupErr)
	}
	if cleanupErr != nil {
		return 0, fmt.Errorf("清理临时渠道：%w", cleanupErr)
	}
	return latency, nil
}

func adaptersForProvider(provider string) []channelAdapter {
	native := map[string]channelAdapter{
		"OpenAI":    {name: "openai_compatible", channelType: 1},
		"Anthropic": {name: "anthropic", channelType: 14},
		"Gemini":    {name: "gemini", channelType: 24},
		"DeepSeek":  {name: "deepseek", channelType: 43},
		"Qwen":      {name: "qwen", channelType: 17},
		"Moonshot":  {name: "moonshot", channelType: 25},
		"Minimax":   {name: "minimax", channelType: 35},
	}
	adapter, exists := native[strings.TrimSpace(provider)]
	if !exists || adapter.channelType == 1 {
		return []channelAdapter{{name: "openai_compatible", channelType: 1}}
	}
	return []channelAdapter{adapter, {name: "openai_compatible", channelType: 1}}
}

func temporaryChannelName() (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("生成临时渠道名称：%w", err)
	}
	return "ModelRoute-Test-" + hex.EncodeToString(random), nil
}
