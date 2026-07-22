package config

import (
	"testing"
	"time"
)

func TestLoadRequiresRuntimeConnections(t *testing.T) {
	t.Setenv("PLATFORM_DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("NEW_API_BASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("缺少运行连接时应返回错误")
	}
}

func TestLoadUsesExplicitValues(t *testing.T) {
	t.Setenv("PLATFORM_DATABASE_URL", "postgres://platform:test@postgres/platform")
	t.Setenv("REDIS_URL", "redis://:test@redis:6379/1")
	t.Setenv("NEW_API_BASE_URL", "http://new-api:3000/")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("DEPENDENCY_TIMEOUT", "3s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() 返回错误：%v", err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.NewAPIBaseURL != "http://new-api:3000" {
		t.Fatalf("NewAPIBaseURL = %q", cfg.NewAPIBaseURL)
	}
	if cfg.DependencyTimeout != 3*time.Second {
		t.Fatalf("DependencyTimeout = %s", cfg.DependencyTimeout)
	}
}
