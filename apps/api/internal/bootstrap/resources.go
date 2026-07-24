package bootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

	quotaPerUnit, err := httpapi.ReadCoreQuotaPerUnit(ctx, newapi.NewClient(cfg.NewAPIBaseURL, cfg.NewAPIRequestTimeout))
	if err != nil {
		redisClient.Close()
		database.Close()
		return nil, err
	}
	services, err := buildHTTPServices(database, redisClient, cfg, quotaPerUnit)
	if err != nil {
		redisClient.Close()
		database.Close()
		return nil, err
	}
	if err := ensureAdministratorMerchant(ctx, services.AdminNewAPI, services.Merchants, cfg.NewAPIAdminUsername, cfg.PublicOrigin); err != nil {
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

func buildHTTPServices(database *sql.DB, redisClient *redis.Client, cfg config.Config, quotaPerUnit int64) (httpapi.Services, error) {
	credentials, err := secretbox.New(cfg.EncryptionKey)
	if err != nil {
		return httpapi.Services{}, fmt.Errorf("初始化商户凭据加密：%w", err)
	}
	newAPIClient := newapi.NewClient(cfg.NewAPIBaseURL, cfg.NewAPIRequestTimeout)
	adminNewAPI := newapi.NewServiceAccount(newAPIClient, cfg.NewAPIAdminUsername, cfg.NewAPIAdminPassword)
	routing := httpapi.NewSQLRoutingStore(database)
	payments := httpapi.NewSQLPaymentStore(database)
	settlement := httpapi.NewSQLSettlementStore(database)
	services := httpapi.Services{
		Sessions:      platformauth.NewStore(redisClient, cfg.SessionTTL),
		NewAPI:        newAPIClient,
		AdminNewAPI:   adminNewAPI,
		AdminUsername: cfg.NewAPIAdminUsername,
		Merchants:     httpapi.NewSQLMerchantStore(database),
		Credentials:   credentials,
		ChannelTests:  httpapi.NewNewAPIChannelDetector(adminNewAPI),
		Routing:       routing,
		OfferSync:     httpapi.NewPriceSynchronizer(routing, adminNewAPI),
		RouteMetrics:  routing,
		Payments:      payments,
		Settlement:    settlement,
		UsageRecords:  settlement,
		UsageAliases:  routing,
		Overview:      httpapi.NewSQLAdminOverviewStore(database),
		Announcements: httpapi.NewSQLAnnouncementStore(database),
		QuotaPerUnit:  quotaPerUnit,
		Cookie: httpapi.CookieConfig{
			Name:         cfg.SessionCookieName,
			Secure:       cfg.SessionCookieSecure,
			PublicOrigin: cfg.PublicOrigin,
			MaxAge:       cfg.SessionTTL,
		},
	}
	services.Health = httpapi.NewHealthRunner(routing, adminNewAPI, newAPIClient, adminNewAPI)
	services.PaymentCredit = httpapi.NewPaymentCreditRunner(payments, adminNewAPI)
	services.SettlementSync = httpapi.NewSettlementSyncRunner(settlement, adminNewAPI)
	return services, nil
}

type administratorMerchantStore interface {
	EnsurePlatformMerchant(context.Context, int, string) (httpapi.MerchantIdentity, error)
}

func ensureAdministratorMerchant(ctx context.Context, admin httpapi.AdminNewAPI, merchants administratorMerchantStore, administratorUsername string, publicOrigin string) error {
	data, err := admin.Do(ctx, http.MethodGet, "/api/user/self", nil)
	if err != nil {
		return fmt.Errorf("读取唯一管理员账号：%w", err)
	}
	var user newapi.User
	if err := json.Unmarshal(data, &user); err != nil {
		return fmt.Errorf("解析唯一管理员账号：%w", err)
	}
	if user.ID <= 0 || user.Status != 1 || user.Role < 10 || !strings.EqualFold(strings.TrimSpace(user.Username), strings.TrimSpace(administratorUsername)) {
		return fmt.Errorf("模型核心服务账号不是配置的有效唯一管理员")
	}
	settings := map[string]any{}
	if value := strings.TrimSpace(user.Setting); value != "" {
		if err := json.Unmarshal([]byte(value), &settings); err != nil {
			return fmt.Errorf("解析唯一管理员设置：%w", err)
		}
	}
	if enabled, _ := settings["accept_unset_model_ratio_model"].(bool); !enabled {
		if _, exists := settings["notify_type"]; !exists {
			settings["notify_type"] = "email"
		}
		if _, exists := settings["quota_warning_threshold"]; !exists {
			settings["quota_warning_threshold"] = 500000
		}
		settings["accept_unset_model_ratio_model"] = true
		if _, err := admin.Do(ctx, http.MethodPut, "/api/user/setting", settings); err != nil {
			return fmt.Errorf("开启管理员未定价模型测试权限：%w", err)
		}
	}
	if _, err := merchants.EnsurePlatformMerchant(ctx, user.ID, publicOrigin); err != nil {
		return fmt.Errorf("创建管理员直属商户：%w", err)
	}
	return nil
}

func (r *Resources) Close() {
	if r.Redis != nil {
		_ = r.Redis.Close()
	}
	if r.Database != nil {
		_ = r.Database.Close()
	}
}
