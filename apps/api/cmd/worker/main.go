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
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	runHealthCycle(ctx, resources)
	runPaymentCreditCycle(ctx, resources)
	runSettlementCycle(ctx, resources)
	ticks := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ticks++
			runHealthCycle(ctx, resources)
			runPaymentCreditCycle(ctx, resources)
			runSettlementCycle(ctx, resources)
			if ticks%3 != 0 {
				continue
			}
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

func runSettlementCycle(ctx context.Context, resources *bootstrap.Resources) {
	if resources.Services.SettlementSync == nil {
		return
	}
	cycleContext, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := resources.Services.SettlementSync.RunCycle(cycleContext, time.Now().UTC()); err != nil {
		log.Printf("结算同步周期异常：%v", err)
	}
}

func runPaymentCreditCycle(ctx context.Context, resources *bootstrap.Resources) {
	if resources.Services.PaymentCredit == nil {
		return
	}
	cycleContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := resources.Services.PaymentCredit.RunCycle(cycleContext, time.Now().UTC()); err != nil {
		log.Printf("充值入账周期异常：%v", err)
	}
}

func runHealthCycle(ctx context.Context, resources *bootstrap.Resources) {
	if resources.Services.Health == nil {
		return
	}
	cycleContext, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := resources.Services.Health.RunCycle(cycleContext, time.Now().UTC()); err != nil {
		log.Printf("渠道探活异常：%v", err)
	}
}
