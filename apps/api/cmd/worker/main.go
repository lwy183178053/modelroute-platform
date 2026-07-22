package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/bootstrap"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/config"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败：%v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	startupContext, cancelStartup := context.WithTimeout(ctx, 30*time.Second)
	resources, err := bootstrap.Open(startupContext, cfg)
	cancelStartup()
	if err != nil {
		log.Fatalf("初始化运行资源失败：%v", err)
	}
	defer resources.Close()

	log.Printf("ModelRoute Worker %s 已启动", version)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkContext, cancelCheck := context.WithTimeout(ctx, cfg.DependencyTimeout)
			errors := 0
			for _, dependency := range []interface{ Ping(context.Context) error }{
				resources.Checks.Database,
				resources.Checks.Redis,
				resources.Checks.NewAPI,
			} {
				if err := dependency.Ping(checkContext); err != nil {
					errors++
				}
			}
			cancelCheck()
			if errors > 0 {
				log.Printf("依赖检查异常数量：%d", errors)
			}
		}
	}
}
