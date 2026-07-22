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
	"github.com/redis/go-redis/v9"
)

type Resources struct {
	Database *sql.DB
	Redis    *redis.Client
	Checks   httpapi.Dependencies
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
