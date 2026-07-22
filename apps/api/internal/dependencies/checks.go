package dependencies

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PingerFunc func(context.Context) error

func (fn PingerFunc) Ping(ctx context.Context) error {
	return fn(ctx)
}

type HTTPPinger struct {
	client *http.Client
	url    string
}

func NewHTTPPinger(url string, timeout time.Duration) *HTTPPinger {
	return &HTTPPinger{
		client: &http.Client{Timeout: timeout},
		url:    url,
	}
}

func (p *HTTPPinger) Ping(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return fmt.Errorf("创建 New API 健康请求：%w", err)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("请求 New API 健康接口：%w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("New API 健康接口状态码为 %d", response.StatusCode)
	}
	var payload struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return fmt.Errorf("解析 New API 健康响应：%w", err)
	}
	if !payload.Success {
		return fmt.Errorf("New API 报告未就绪")
	}
	return nil
}
