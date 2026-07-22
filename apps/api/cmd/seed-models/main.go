package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/migrations"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/modelseed"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

var version = "dev"

type seedConfig struct {
	databaseURL      string
	newAPIBaseURL    string
	adminUsername    string
	adminPassword    string
	requestTimeout   time.Duration
	operationTimeout time.Duration
}

func main() {
	cfg, err := loadSeedConfig()
	if err != nil {
		log.Fatalf("加载模型种子配置失败：%v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.operationTimeout)
	defer cancel()

	database, err := sql.Open("pgx", cfg.databaseURL)
	if err != nil {
		log.Fatalf("打开平台数据库失败：%v", err)
	}
	defer database.Close()
	database.SetMaxOpenConns(4)
	database.SetMaxIdleConns(1)
	if err := database.PingContext(ctx); err != nil {
		log.Fatalf("连接平台数据库失败：%v", err)
	}
	if err := migrations.Up(ctx, database); err != nil {
		log.Fatalf("执行平台迁移失败：%v", err)
	}

	client := newapi.NewClient(cfg.newAPIBaseURL, cfg.requestTimeout)
	administrator := newapi.NewServiceAccount(client, cfg.adminUsername, cfg.adminPassword)
	body, err := administrator.Do(ctx, http.MethodGet, "/api/user/self", nil)
	if err != nil {
		log.Fatalf("读取唯一管理员失败：%v", err)
	}
	user, err := administratorFromBody(body, cfg.adminUsername)
	if err != nil {
		log.Fatalf("验证唯一管理员失败：%v", err)
	}

	catalog, err := modelseed.CurrentCatalog()
	if err != nil {
		log.Fatalf("读取官方模型种子失败：%v", err)
	}
	result, err := modelseed.Apply(ctx, modelseed.NewSQLRepository(database), catalog, user.ID)
	if err != nil {
		log.Fatalf("应用官方模型种子失败：%v", err)
	}
	log.Printf("ModelRoute 模型种子 %s 完成：新增 %d，调价 %d，未变化 %d，程序版本 %s",
		catalog.Version, result.Created, result.Published, result.Unchanged, version)
}

func loadSeedConfig() (seedConfig, error) {
	cfg := seedConfig{
		databaseURL:      strings.TrimSpace(os.Getenv("PLATFORM_DATABASE_URL")),
		newAPIBaseURL:    strings.TrimRight(strings.TrimSpace(os.Getenv("NEW_API_BASE_URL")), "/"),
		adminUsername:    strings.TrimSpace(os.Getenv("NEW_API_ADMIN_USERNAME")),
		adminPassword:    strings.TrimSpace(os.Getenv("NEW_API_ADMIN_PASSWORD")),
		requestTimeout:   30 * time.Second,
		operationTimeout: 2 * time.Minute,
	}
	for name, value := range map[string]string{
		"PLATFORM_DATABASE_URL":  cfg.databaseURL,
		"NEW_API_BASE_URL":       cfg.newAPIBaseURL,
		"NEW_API_ADMIN_USERNAME": cfg.adminUsername,
		"NEW_API_ADMIN_PASSWORD": cfg.adminPassword,
	} {
		if value == "" {
			return seedConfig{}, fmt.Errorf("缺少环境变量 %s", name)
		}
	}
	if raw := strings.TrimSpace(os.Getenv("NEW_API_REQUEST_TIMEOUT")); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			return seedConfig{}, fmt.Errorf("NEW_API_REQUEST_TIMEOUT 必须是正数时长")
		}
		cfg.requestTimeout = value
	}
	return cfg, nil
}

func administratorFromBody(body []byte, expectedUsername string) (newapi.User, error) {
	var user newapi.User
	if err := json.Unmarshal(body, &user); err != nil {
		return newapi.User{}, fmt.Errorf("解析管理员账号：%w", err)
	}
	if user.ID <= 0 || user.Status != 1 || user.Role < 10 || !strings.EqualFold(strings.TrimSpace(user.Username), strings.TrimSpace(expectedUsername)) {
		return newapi.User{}, fmt.Errorf("New API 服务账号不是配置的有效唯一管理员")
	}
	return user, nil
}
