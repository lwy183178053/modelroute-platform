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
			AdapterType: adapter.name, Capabilities: []string{request.Model.Price.PriceKind}, LatencyMS: latency,
		}, nil
	}
	return ChannelTestOutcome{}, fmt.Errorf("全部适配器检测失败：%w", errors.Join(attempts...))
}

func (d *NewAPIChannelDetector) TestExisting(ctx context.Context, channelID int, model CanonicalModel) (ChannelTestOutcome, error) {
	if d == nil || d.admin == nil || channelID <= 0 || model.ID <= 0 {
		return ChannelTestOutcome{}, errors.New("正式渠道模型检测参数无效")
	}
	started := time.Now()
	_, err := d.admin.Do(ctx, http.MethodGet, channelModelTestPath(channelID, model), nil)
	latency := int(time.Since(started).Milliseconds())
	if latency < 1 {
		latency = 1
	}
	if err != nil {
		return ChannelTestOutcome{LatencyMS: latency}, err
	}
	return ChannelTestOutcome{Capabilities: []string{model.Price.PriceKind}, LatencyMS: latency}, nil
}

func channelModelTestPath(channelID int, model CanonicalModel) string {
	testQuery := url.Values{"model": {model.PublicName}}
	if model.Price.PriceKind == "image" {
		testQuery.Set("endpoint_type", "image-generation")
	}
	return fmt.Sprintf("/api/channel/test/%d?%s", channelID, testQuery.Encode())
}

func (d *NewAPIChannelDetector) createTemporaryChannel(ctx context.Context, baseURL, apiKey string, adapter channelAdapter, models []string) (int, error) {
	name, err := temporaryChannelName()
	if err != nil {
		return 0, err
	}
	if _, err := d.admin.Do(ctx, http.MethodPost, "/api/channel/", map[string]any{
		"mode": "single",
		"channel": map[string]any{
			"type": adapter.channelType, "key": apiKey, "name": name,
			"base_url": newAPIChannelBaseURL(adapter.channelType, baseURL), "models": strings.Join(models, ","),
			"group": "default", "status": 1, "test_model": models[0], "tag": "modelroute-channel-test",
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
	return result.Items[0].ID, nil
}

func (d *NewAPIChannelDetector) testAdapter(ctx context.Context, request ChannelTestRequest, adapter channelAdapter) (int, error) {
	channelID, err := d.createTemporaryChannel(ctx, request.BaseURL, request.APIKey, adapter, []string{request.Model.PublicName})
	if err != nil {
		return 0, err
	}
	started := time.Now()
	_, testErr := d.admin.Do(ctx, http.MethodGet, channelModelTestPath(channelID, request.Model), nil)
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
