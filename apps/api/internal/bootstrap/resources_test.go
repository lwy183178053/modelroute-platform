package bootstrap

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/config"
	"github.com/redis/go-redis/v9"
)

func TestBuildHTTPServicesUsesSessionAndOriginConfig(t *testing.T) {
	database, err := sql.Open("pgx", "postgres://unused:unused@localhost/unused")
	if err != nil {
		t.Fatalf("创建数据库句柄失败：%v", err)
	}
	defer database.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer redisClient.Close()
	cfg := config.Config{
		NewAPIBaseURL:        "http://new-api:3000",
		NewAPIRequestTimeout: 30 * time.Second,
		PublicOrigin:         "https://aggregate.lwylink.xyz",
		SessionCookieName:    "modelroute_session",
		SessionCookieSecure:  true,
		SessionTTL:           12 * time.Hour,
	}
	services := buildHTTPServices(database, redisClient, cfg)
	if services.Sessions == nil || services.NewAPI == nil || services.Audit == nil {
		t.Fatalf("HTTP 服务依赖未完整注入：%+v", services)
	}
	if services.Cookie.Name != "modelroute_session" || services.Cookie.PublicOrigin != "https://aggregate.lwylink.xyz" || services.Cookie.MaxAge != 12*time.Hour || !services.Cookie.Secure {
		t.Fatalf("Cookie 配置不符合预期：%+v", services.Cookie)
	}
}
