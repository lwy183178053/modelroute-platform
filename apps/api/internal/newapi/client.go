package newapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const maxResponseBytes = 4 << 20

type clientIPContextKey struct{}

func WithClientIP(ctx context.Context, value string) context.Context {
	parsed := net.ParseIP(strings.TrimSpace(value))
	if parsed == nil {
		return ctx
	}
	return context.WithValue(ctx, clientIPContextKey{}, parsed.String())
}

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Role         int    `json:"role"`
	Status       int    `json:"status"`
	Group        string `json:"group"`
	Email        string `json:"email,omitempty"`
	Remark       string `json:"remark,omitempty"`
	Quota        int    `json:"quota,omitempty"`
	UsedQuota    int    `json:"used_quota,omitempty"`
	RequestCount int    `json:"request_count,omitempty"`
}

type Auth struct {
	Cookie string
	UserID int
}

type AuthenticationResult struct {
	User             User
	Cookie           string
	RequireTwoFactor bool
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Authenticate(ctx context.Context, path string, body any, cookie string) (AuthenticationResult, error) {
	data, cookies, err := c.request(ctx, http.MethodPost, path, body, &Auth{Cookie: cookie})
	if err != nil {
		return AuthenticationResult{}, err
	}
	result := AuthenticationResult{Cookie: cookie}
	for _, responseCookie := range cookies {
		if responseCookie.Name == "session" {
			result.Cookie = responseCookie.Name + "=" + responseCookie.Value
			break
		}
	}
	var loginData struct {
		User
		RequireTwoFactor bool `json:"require_2fa"`
	}
	if len(data) != 0 && string(data) != "null" {
		if err := json.Unmarshal(data, &loginData); err != nil {
			return AuthenticationResult{}, fmt.Errorf("解析 New API 登录响应：%w", err)
		}
	}
	result.User = loginData.User
	result.RequireTwoFactor = loginData.RequireTwoFactor
	if result.Cookie == "" {
		return AuthenticationResult{}, fmt.Errorf("New API 登录响应缺少会话 Cookie")
	}
	return result, nil
}

func (c *Client) Do(ctx context.Context, method string, path string, body any, auth *Auth) (json.RawMessage, error) {
	data, _, err := c.request(ctx, method, path, body, auth)
	return data, err
}

func (c *Client) request(ctx context.Context, method string, path string, body any, auth *Auth) (json.RawMessage, []*http.Cookie, error) {
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("编码 New API 请求：%w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, requestBody)
	if err != nil {
		return nil, nil, fmt.Errorf("创建 New API 请求：%w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Language", "zh-CN")
	if clientIP, ok := ctx.Value(clientIPContextKey{}).(string); ok && clientIP != "" {
		request.Header.Set("X-Forwarded-For", clientIP)
		request.Header.Set("X-Real-IP", clientIP)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if auth != nil {
		if auth.Cookie != "" {
			request.Header.Set("Cookie", auth.Cookie)
		}
		if auth.UserID > 0 {
			request.Header.Set("New-Api-User", fmt.Sprintf("%d", auth.UserID))
		}
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, nil, fmt.Errorf("请求 New API：%w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nil, nil, fmt.Errorf("读取 New API 响应：%w", err)
	}
	if len(responseBody) > maxResponseBytes {
		return nil, nil, fmt.Errorf("New API 响应超过限制")
	}
	var envelope struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return nil, nil, fmt.Errorf("解析 New API 响应：%w", err)
	}
	if response.StatusCode >= http.StatusBadRequest || !envelope.Success {
		message := strings.TrimSpace(envelope.Message)
		if message == "" {
			message = http.StatusText(response.StatusCode)
		}
		return nil, response.Cookies(), &APIError{StatusCode: response.StatusCode, Message: message}
	}
	if len(envelope.Data) == 0 {
		envelope.Data = json.RawMessage("null")
	}
	return envelope.Data, response.Cookies(), nil
}
