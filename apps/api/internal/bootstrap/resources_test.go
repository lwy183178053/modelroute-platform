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

type administratorSettingStub struct {
	calls   []string
	body    map[string]any
	setting string
}

func (s *administratorSettingStub) Do(_ context.Context, method string, path string, body any) ([]byte, error) {
	s.calls = append(s.calls, method+" "+path)
	if method == "GET" && path == "/api/user/self" {
		setting := s.setting
		if setting == "" {
			setting = `{"notify_type":"email","quota_warning_threshold":700000,"notification_email":"ops@example.com","record_ip_log":true}`
		}
		return json.Marshal(map[string]any{
			"id": 1, "username": "3180615598@qq.com", "role": 100, "status": 1,
			"setting": setting,
		})
	}
	if method == "PUT" && path == "/api/user/setting" {
		encoded, _ := json.Marshal(body)
		_ = json.Unmarshal(encoded, &s.body)
		return []byte(`null`), nil
	}
	return nil, nil
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
	services, err := buildHTTPServices(database, redisClient, cfg, 750_000)
	if err != nil {
		t.Fatalf("构建 HTTP 依赖失败：%v", err)
	}
	if services.Sessions == nil || services.NewAPI == nil || services.Merchants == nil || services.Credentials == nil {
		t.Fatalf("HTTP 服务依赖未完整注入：%+v", services)
	}
	if services.AdminNewAPI == nil || services.Routing == nil || services.OfferSync == nil || services.Health == nil || services.RouteMetrics == nil || services.Payments == nil || services.PaymentCredit == nil || services.Settlement == nil || services.SettlementSync == nil {
		t.Fatalf("路由服务依赖未完整注入：%+v", services)
	}
	if services.QuotaPerUnit != 750_000 {
		t.Fatalf("核心额度单位未注入：%d", services.QuotaPerUnit)
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

func TestEnsureAdministratorMerchantEnablesUnpricedModelTestsAndPreservesSettings(t *testing.T) {
	admin := &administratorSettingStub{}
	store := &platformMerchantStoreStub{}
	if err := ensureAdministratorMerchant(context.Background(), admin, store, "3180615598@qq.com", "https://aggregate.lwylink.xyz"); err != nil {
		t.Fatalf("初始化管理员测试设置失败：%v", err)
	}
	if len(admin.calls) != 2 || admin.calls[0] != "GET /api/user/self" || admin.calls[1] != "PUT /api/user/setting" {
		t.Fatalf("管理员设置调用=%v", admin.calls)
	}
	if admin.body["accept_unset_model_ratio_model"] != true || admin.body["notify_type"] != "email" || admin.body["quota_warning_threshold"] != float64(700000) || admin.body["notification_email"] != "ops@example.com" || admin.body["record_ip_log"] != true {
		t.Fatalf("管理员设置未保留：%v", admin.body)
	}
}

func TestEnsureAdministratorMerchantKeepsEnabledTestPermissionIdempotently(t *testing.T) {
	admin := &administratorSettingStub{setting: `{"notify_type":"email","accept_unset_model_ratio_model":true}`}
	store := &platformMerchantStoreStub{}
	if err := ensureAdministratorMerchant(context.Background(), admin, store, "3180615598@qq.com", "https://aggregate.lwylink.xyz"); err != nil {
		t.Fatalf("重复初始化管理员测试设置失败：%v", err)
	}
	if len(admin.calls) != 1 || admin.calls[0] != "GET /api/user/self" {
		t.Fatalf("已开启权限时不应重复写设置：%v", admin.calls)
	}
}
