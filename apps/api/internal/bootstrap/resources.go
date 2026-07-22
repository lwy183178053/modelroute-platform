package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/config"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/dependencies"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/httpapi"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/migrations"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/secretbox"
	"github.com/redis/go-redis/v9"
)

type Resources struct {
	Database *sql.DB
	Redis    *redis.Client
	Checks   httpapi.Dependencies
	Services httpapi.Services
}

func Open(ctx context.Context, cfg config.Config) (*Resources, error) {
	database, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("打开平台数据库：%w", err)
	}
	database.SetMaxOpenConns(20)
	database.SetMaxIdleConns(5)
	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("连接平台数据库：%w", err)
	}
	if err := migrations.Up(ctx, database); err != nil {
		database.Close()
		return nil, fmt.Errorf("执行平台迁移：%w", err)
	}

	redisOptions, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("解析 Redis 地址：%w", err)
	}
	redisClient := redis.NewClient(redisOptions)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisClient.Close()
		database.Close()
		return nil, fmt.Errorf("连接 Redis：%w", err)
	}

	services, err := buildHTTPServices(database, redisClient, cfg)
	if err != nil {
		redisClient.Close()
		database.Close()
		return nil, err
	}
	return &Resources{
		Database: database,
		Redis:    redisClient,
		Checks: httpapi.Dependencies{
			Database: dependencies.PingerFunc(database.PingContext),
			Redis: dependencies.PingerFunc(func(ctx context.Context) error {
				return redisClient.Ping(ctx).Err()
			}),
			NewAPI: dependencies.NewHTTPPinger(cfg.NewAPIBaseURL+"/api/status", cfg.DependencyTimeout),
		},
		Services: services,
	}, nil
}

func buildHTTPServices(database *sql.DB, redisClient *redis.Client, cfg config.Config) (httpapi.Services, error) {
	credentials, err := secretbox.New(cfg.EncryptionKey)
	if err != nil {
		return httpapi.Services{}, fmt.Errorf("初始化商户凭据加密：%w", err)
	}
	return httpapi.Services{
		Sessions:    platformauth.NewStore(redisClient, cfg.SessionTTL),
		NewAPI:      newapi.NewClient(cfg.NewAPIBaseURL, cfg.NewAPIRequestTimeout),
		Audit:       httpapi.NewSQLAuditStore(database),
		Merchants:   httpapi.NewSQLMerchantStore(database),
		Credentials: credentials,
		Cookie: httpapi.CookieConfig{
			Name:         cfg.SessionCookieName,
			Secure:       cfg.SessionCookieSecure,
			PublicOrigin: cfg.PublicOrigin,
			MaxAge:       cfg.SessionTTL,
		},
	}, nil
}

func (r *Resources) Close() {
	if r.Redis != nil {
		_ = r.Redis.Close()
	}
	if r.Database != nil {
		_ = r.Database.Close()
	}
}
