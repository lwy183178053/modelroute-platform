package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/bootstrap"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/config"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/httpapi"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败：%v", err)
	}

	startupContext, cancelStartup := context.WithTimeout(context.Background(), 30*time.Second)
	resources, err := bootstrap.Open(startupContext, cfg)
	cancelStartup()
	if err != nil {
		log.Fatalf("初始化运行资源失败：%v", err)
	}
	defer resources.Close()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewServerWithServices(version, resources.Checks, resources.Services),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("ModelRoute API %s 监听 %s", version, cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP 服务异常：%v", err)
		}
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownSignal

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("HTTP 服务关闭异常：%v", err)
	}
}
