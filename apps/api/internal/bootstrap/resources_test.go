package bootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/config"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/httpapi"
	"github.com/redis/go-redis/v9"
)

type adminAccountStub struct {
	body []byte
}

func (s adminAccountStub) Do(context.Context, string, string, any) ([]byte, error) {
	return s.body, nil
}

type platformMerchantStoreStub struct {
	userID       int
	publicOrigin string
}

func (s *platformMerchantStoreStub) EnsurePlatformMerchant(_ context.Context, userID int, publicOrigin string) (httpapi.MerchantIdentity, error) {
	s.userID = userID
	s.publicOrigin = publicOrigin
	return httpapi.MerchantIdentity{MerchantID: 31, UserID: userID, DisplayName: "ModelRoute 官方", Status: "active", IsMerchant: true, IsPlatformOwned: true}, nil
}

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
		EncryptionKey:        []byte("0123456789abcdef0123456789abcdef"),
		NewAPIAdminUsername:  "modelroute",
		NewAPIAdminPassword:  "server-only-secret",
	}
	services, err := buildHTTPServices(database, redisClient, cfg)
	if err != nil {
		t.Fatalf("构建 HTTP 依赖失败：%v", err)
	}
	if services.Sessions == nil || services.NewAPI == nil || services.Audit == nil || services.Merchants == nil || services.Credentials == nil {
		t.Fatalf("HTTP 服务依赖未完整注入：%+v", services)
	}
	if services.AdminNewAPI == nil || services.Routing == nil || services.OfferSync == nil || services.Health == nil || services.Payments == nil || services.PaymentCredit == nil || services.Settlement == nil || services.SettlementSync == nil {
		t.Fatalf("路由服务依赖未完整注入：%+v", services)
	}
	if services.Cookie.Name != "modelroute_session" || services.Cookie.PublicOrigin != "https://aggregate.lwylink.xyz" || services.Cookie.MaxAge != 12*time.Hour || !services.Cookie.Secure {
		t.Fatalf("Cookie 配置不符合预期：%+v", services.Cookie)
	}
}

func TestEnsureAdministratorMerchantUsesConfiguredSingleAdministrator(t *testing.T) {
	user, err := json.Marshal(map[string]any{
		"id": 1, "username": "3180615598@qq.com", "role": 100, "status": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	store := &platformMerchantStoreStub{}
	err = ensureAdministratorMerchant(
		context.Background(),
		adminAccountStub{body: user},
		store,
		"3180615598@qq.com",
		"https://aggregate.lwylink.xyz",
	)
	if err != nil {
		t.Fatalf("引导管理员直属商户失败：%v", err)
	}
	if store.userID != 1 || store.publicOrigin != "https://aggregate.lwylink.xyz" {
		t.Fatalf("平台商户引导参数 user=%d origin=%s", store.userID, store.publicOrigin)
	}
}
