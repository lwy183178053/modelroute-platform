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
	t.Setenv("PUBLIC_ORIGIN", "https://aggregate.lwylink.xyz/")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("DEPENDENCY_TIMEOUT", "3s")
	t.Setenv("NEW_API_REQUEST_TIMEOUT", "30s")
	t.Setenv("SESSION_COOKIE_NAME", "modelroute_session")
	t.Setenv("SESSION_TTL", "12h")
	t.Setenv("SESSION_COOKIE_SECURE", "true")

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
	if cfg.PublicOrigin != "https://aggregate.lwylink.xyz" {
		t.Fatalf("PublicOrigin = %q", cfg.PublicOrigin)
	}
	if cfg.NewAPIRequestTimeout != 30*time.Second || cfg.SessionTTL != 12*time.Hour {
		t.Fatalf("超时配置不符合预期：NewAPI=%s Session=%s", cfg.NewAPIRequestTimeout, cfg.SessionTTL)
	}
	if cfg.SessionCookieName != "modelroute_session" || !cfg.SessionCookieSecure {
		t.Fatalf("Cookie 配置不符合预期：%+v", cfg)
	}
}

func TestLoadRejectsNonHTTPSPublicOriginWhenCookieIsSecure(t *testing.T) {
	t.Setenv("PLATFORM_DATABASE_URL", "postgres://platform:test@postgres/platform")
	t.Setenv("REDIS_URL", "redis://:test@redis:6379/1")
	t.Setenv("NEW_API_BASE_URL", "http://new-api:3000")
	t.Setenv("PUBLIC_ORIGIN", "http://aggregate.lwylink.xyz")
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	if _, err := Load(); err == nil {
		t.Fatal("安全 Cookie 配合 HTTP 公网来源时应返回错误")
	}
}
