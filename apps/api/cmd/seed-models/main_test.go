package main

import (
	"testing"
	"time"
)

func TestAdministratorFromBodyAcceptsConfiguredSingleAdministrator(t *testing.T) {
	user, err := administratorFromBody(
		[]byte(`{"id":19,"username":"3180615598@qq.com","role":100,"status":1}`),
		"3180615598@qq.com",
	)
	if err != nil {
		t.Fatalf("解析唯一管理员：%v", err)
	}
	if user.ID != 19 {
		t.Fatalf("管理员编号=%d", user.ID)
	}
}

func TestLoadSeedConfigUsesOnlySeedDependencies(t *testing.T) {
	t.Setenv("PLATFORM_DATABASE_URL", "postgres://modelroute:test@postgres/modelroute")
	t.Setenv("NEW_API_BASE_URL", "http://new-api:3000/")
	t.Setenv("NEW_API_ADMIN_USERNAME", "3180615598@qq.com")
	t.Setenv("NEW_API_ADMIN_PASSWORD", "server-only-secret")
	t.Setenv("NEW_API_REQUEST_TIMEOUT", "45s")

	cfg, err := loadSeedConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.newAPIBaseURL != "http://new-api:3000" || cfg.requestTimeout != 45*time.Second {
		t.Fatalf("种子配置=%+v", cfg)
	}
}

func TestAdministratorFromBodyRejectsOtherOrInactiveAccounts(t *testing.T) {
	tests := []string{
		`{"id":19,"username":"other@example.com","role":100,"status":1}`,
		`{"id":19,"username":"3180615598@qq.com","role":1,"status":1}`,
		`{"id":19,"username":"3180615598@qq.com","role":100,"status":2}`,
	}
	for _, body := range tests {
		if _, err := administratorFromBody([]byte(body), "3180615598@qq.com"); err == nil {
			t.Fatalf("无效管理员应被拒绝：%s", body)
		}
	}
}
