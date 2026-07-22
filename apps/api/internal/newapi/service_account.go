package newapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
)

const probeTokenName = "ModelRoute 平台探活"

type ProbeToken struct {
	ID  int
	Key string
}

type ServiceAccount struct {
	client   *Client
	username string
	password string
	mu       sync.Mutex
	auth     Auth
	probeMu  sync.Mutex
	probe    *ProbeToken
}

func (s *ServiceAccount) ProbeToken(ctx context.Context) (ProbeToken, error) {
	s.probeMu.Lock()
	defer s.probeMu.Unlock()
	if s.probe != nil {
		return *s.probe, nil
	}
	items, err := s.probeTokenItems(ctx)
	if err != nil {
		return ProbeToken{}, err
	}
	if len(items) == 0 {
		if _, err := s.Do(ctx, http.MethodPost, "/api/token/", map[string]any{
			"name": probeTokenName, "expired_time": -1, "unlimited_quota": true,
			"model_limits_enabled": false,
		}); err != nil {
			return ProbeToken{}, fmt.Errorf("创建 New API 探活令牌：%w", err)
		}
		items, err = s.probeTokenItems(ctx)
		if err != nil {
			return ProbeToken{}, err
		}
	}
	if len(items) == 0 {
		return ProbeToken{}, fmt.Errorf("New API 未返回探活令牌编号")
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID > items[j].ID })
	data, err := s.Do(ctx, http.MethodPost, fmt.Sprintf("/api/token/%d/key", items[0].ID), nil)
	if err != nil {
		return ProbeToken{}, fmt.Errorf("读取 New API 探活令牌：%w", err)
	}
	var revealed struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(data, &revealed); err != nil || revealed.Key == "" {
		return ProbeToken{}, fmt.Errorf("New API 探活令牌响应无效")
	}
	probe := ProbeToken{ID: items[0].ID, Key: revealed.Key}
	s.probe = &probe
	return probe, nil
}

func (s *ServiceAccount) probeTokenItems(ctx context.Context) ([]struct {
	ID int `json:"id"`
}, error) {
	data, err := s.Do(ctx, http.MethodGet, "/api/token/?p=1&page_size=100", nil)
	if err != nil {
		return nil, fmt.Errorf("查询 New API 探活令牌：%w", err)
	}
	var result struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析 New API 探活令牌：%w", err)
	}
	items := make([]struct {
		ID int `json:"id"`
	}, 0)
	for _, item := range result.Items {
		if item.Name == probeTokenName && item.ID > 0 {
			items = append(items, struct {
				ID int `json:"id"`
			}{ID: item.ID})
		}
	}
	return items, nil
}

func NewServiceAccount(client *Client, username string, password string) *ServiceAccount {
	return &ServiceAccount{client: client, username: username, password: password}
}

func (s *ServiceAccount) Do(ctx context.Context, method string, path string, body any) (jsonData []byte, err error) {
	auth, err := s.authentication(ctx, false)
	if err != nil {
		return nil, err
	}
	data, err := s.client.Do(ctx, method, path, body, &auth)
	if apiError, ok := err.(*APIError); !ok || apiError.StatusCode != http.StatusUnauthorized {
		return data, err
	}
	auth, err = s.authentication(ctx, true)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, method, path, body, &auth)
}

func (s *ServiceAccount) authentication(ctx context.Context, force bool) (Auth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !force && s.auth.Cookie != "" && s.auth.UserID > 0 {
		return s.auth, nil
	}
	result, err := s.client.Authenticate(ctx, "/api/user/login", map[string]string{
		"username": s.username,
		"password": s.password,
	}, "")
	if err != nil {
		return Auth{}, fmt.Errorf("New API 服务账号登录失败：%w", err)
	}
	if result.User.Status != 1 || result.User.Role < 10 {
		return Auth{}, fmt.Errorf("New API 服务账号状态或角色无效")
	}
	s.auth = Auth{Cookie: result.Cookie, UserID: result.User.ID}
	return s.auth, nil
}
