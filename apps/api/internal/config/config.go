package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	RedisURL          string
	NewAPIBaseURL     string
	DependencyTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:          valueOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:       strings.TrimSpace(os.Getenv("PLATFORM_DATABASE_URL")),
		RedisURL:          strings.TrimSpace(os.Getenv("REDIS_URL")),
		NewAPIBaseURL:     strings.TrimRight(strings.TrimSpace(os.Getenv("NEW_API_BASE_URL")), "/"),
		DependencyTimeout: 2 * time.Second,
	}

	if raw := strings.TrimSpace(os.Getenv("DEPENDENCY_TIMEOUT")); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("DEPENDENCY_TIMEOUT 必须是正数时长")
		}
		cfg.DependencyTimeout = timeout
	}

	required := map[string]string{
		"PLATFORM_DATABASE_URL": cfg.DatabaseURL,
		"REDIS_URL":             cfg.RedisURL,
		"NEW_API_BASE_URL":      cfg.NewAPIBaseURL,
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

	return cfg, nil
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
