package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr             string
	DatabaseURL          string
	RedisURL             string
	NewAPIBaseURL        string
	DependencyTimeout    time.Duration
	NewAPIRequestTimeout time.Duration
	PublicOrigin         string
	SessionCookieName    string
	SessionCookieSecure  bool
	SessionTTL           time.Duration
	EncryptionKey        []byte
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:             valueOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:          strings.TrimSpace(os.Getenv("PLATFORM_DATABASE_URL")),
		RedisURL:             strings.TrimSpace(os.Getenv("REDIS_URL")),
		NewAPIBaseURL:        strings.TrimRight(strings.TrimSpace(os.Getenv("NEW_API_BASE_URL")), "/"),
		DependencyTimeout:    2 * time.Second,
		NewAPIRequestTimeout: 30 * time.Second,
		PublicOrigin:         strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_ORIGIN")), "/"),
		SessionCookieName:    valueOrDefault("SESSION_COOKIE_NAME", "modelroute_session"),
		SessionCookieSecure:  true,
		SessionTTL:           24 * time.Hour,
	}

	if raw := strings.TrimSpace(os.Getenv("DEPENDENCY_TIMEOUT")); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("DEPENDENCY_TIMEOUT 必须是正数时长")
		}
		cfg.DependencyTimeout = timeout
	}
	if err := loadPositiveDuration("NEW_API_REQUEST_TIMEOUT", &cfg.NewAPIRequestTimeout); err != nil {
		return Config{}, err
	}
	if err := loadPositiveDuration("SESSION_TTL", &cfg.SessionTTL); err != nil {
		return Config{}, err
	}
	if raw := strings.TrimSpace(os.Getenv("SESSION_COOKIE_SECURE")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("SESSION_COOKIE_SECURE 必须是布尔值")
		}
		cfg.SessionCookieSecure = value
	}

	required := map[string]string{
		"PLATFORM_DATABASE_URL": cfg.DatabaseURL,
		"REDIS_URL":             cfg.RedisURL,
		"NEW_API_BASE_URL":      cfg.NewAPIBaseURL,
		"PUBLIC_ORIGIN":         cfg.PublicOrigin,
	}
	for name, value := range required {
		if value == "" {
			return Config{}, fmt.Errorf("缺少环境变量 %s", name)
		}
	}

	for name, raw := range map[string]string{
		"PLATFORM_DATABASE_URL": cfg.DatabaseURL,
		"REDIS_URL":             cfg.RedisURL,
		"NEW_API_BASE_URL":      cfg.NewAPIBaseURL,
	} {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return Config{}, fmt.Errorf("环境变量 %s 不是有效地址", name)
		}
	}
	publicOrigin, err := url.Parse(cfg.PublicOrigin)
	if err != nil || publicOrigin.Scheme == "" || publicOrigin.Host == "" || publicOrigin.Path != "" {
		return Config{}, fmt.Errorf("环境变量 PUBLIC_ORIGIN 不是有效来源地址")
	}
	if cfg.SessionCookieSecure && publicOrigin.Scheme != "https" {
		return Config{}, fmt.Errorf("安全会话 Cookie 要求 PUBLIC_ORIGIN 使用 HTTPS")
	}
	if strings.ContainsAny(cfg.SessionCookieName, " ;,=\t\r\n") {
		return Config{}, fmt.Errorf("SESSION_COOKIE_NAME 格式无效")
	}
	encryptionKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(os.Getenv("PLATFORM_ENCRYPTION_KEY")))
	if err != nil || len(encryptionKey) != 32 {
		return Config{}, fmt.Errorf("PLATFORM_ENCRYPTION_KEY 必须是 Base64 编码的 32 字节密钥")
	}
	cfg.EncryptionKey = encryptionKey

	return cfg, nil
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func loadPositiveDuration(name string, target *time.Duration) error {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fmt.Errorf("%s 必须是正数时长", name)
	}
	*target = value
	return nil
}
