package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

type Pinger interface {
	Ping(context.Context) error
}

type Dependencies struct {
	Database Pinger
	Redis    Pinger
	NewAPI   Pinger
}

type server struct {
	version      string
	dependencies Dependencies
	services     Services
	readiness    readinessCache
}

const readinessCacheDuration = time.Second

type readinessCache struct {
	mu               sync.Mutex
	checkedAt        time.Time
	failedDependency string
}

func NewServer(version string, dependencies Dependencies) http.Handler {
	return NewServerWithServices(version, dependencies, Services{})
}

func NewServerWithServices(version string, dependencies Dependencies, services Services) http.Handler {
	s := &server{version: version, dependencies: dependencies, services: services}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health/live", s.live)
	mux.HandleFunc("GET /api/health/ready", s.ready)
	mux.HandleFunc("GET /api/health", s.ready)
	mux.HandleFunc("GET /api/version", s.versionInfo)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/login/2fa", s.loginTwoFactor)
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/email-verification", s.sendEmailVerification)
	mux.HandleFunc("POST /api/auth/password-reset/request", s.requestPasswordReset)
	mux.HandleFunc("POST /api/auth/password-reset/confirm", s.confirmPasswordReset)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/me", s.me)
	mux.HandleFunc("GET /api/keys", s.keys)
	mux.HandleFunc("POST /api/keys", s.createKey)
	mux.HandleFunc("PUT /api/keys/{id}", s.updateKey)
	mux.HandleFunc("POST /api/keys/{id}/status", s.updateKeyStatus)
	mux.HandleFunc("POST /api/keys/{id}/reveal", s.revealKey)
	mux.HandleFunc("DELETE /api/keys/{id}", s.deleteKey)
	mux.HandleFunc("GET /api/usage", s.usage)
	mux.HandleFunc("POST /api/redemptions/redeem", s.redeemCode)
	mux.HandleFunc("GET /api/admin/users", s.adminUsers)
	mux.HandleFunc("GET /api/admin/users/{id}", s.adminUser)
	mux.HandleFunc("POST /api/admin/users/{id}/status", s.adminUserStatus)
	mux.HandleFunc("POST /api/admin/users/{id}/quota", s.adminUserQuota)
	mux.HandleFunc("POST /api/admin/users/{id}/password", s.adminUserPassword)
	mux.HandleFunc("GET /api/admin/audit-logs", s.adminAuditLogs)
	mux.HandleFunc("GET /api/admin/usage", s.adminUsage)
	mux.HandleFunc("GET /api/admin/overview", s.adminOverview)
	mux.HandleFunc("GET /api/admin/platform-settings", s.adminPlatformSettings)
	mux.HandleFunc("PUT /api/admin/platform-settings", s.updateAdminPlatformSettings)
	mux.HandleFunc("GET /api/admin/groups", s.adminGroups)
	mux.HandleFunc("PUT /api/admin/groups", s.updateAdminGroups)
	mux.HandleFunc("GET /api/admin/redemptions", s.adminRedemptions)
	mux.HandleFunc("POST /api/admin/redemptions", s.createAdminRedemption)
	mux.HandleFunc("PUT /api/admin/redemptions/{id}", s.updateAdminRedemption)
	mux.HandleFunc("POST /api/admin/redemptions/{id}/status", s.updateAdminRedemptionStatus)
	mux.HandleFunc("DELETE /api/admin/redemptions/invalid", s.deleteInvalidAdminRedemptions)
	mux.HandleFunc("DELETE /api/admin/redemptions/{id}", s.deleteAdminRedemption)
	mux.HandleFunc("GET /api/admin/system-tasks", s.adminSystemTasks)
	mux.HandleFunc("GET /api/admin/system-tasks/{task_id}", s.adminSystemTask)
	mux.HandleFunc("POST /api/admin/system-tasks/log-cleanup", s.createAdminLogCleanupTask)
	mux.HandleFunc("GET /api/merchant/application", s.merchantApplication)
	mux.HandleFunc("POST /api/merchant/application", s.createMerchantApplication)
	mux.HandleFunc("GET /api/merchant/channels", s.merchantChannels)
	mux.HandleFunc("POST /api/merchant/channels", s.createMerchantChannel)
	mux.HandleFunc("GET /api/admin/merchant-applications", s.adminMerchantApplications)
	mux.HandleFunc("GET /api/admin/merchant-applications/{id}", s.adminMerchantApplication)
	mux.HandleFunc("POST /api/admin/merchant-applications/{id}/verify", s.adminVerifyApplication)
	mux.HandleFunc("POST /api/admin/merchant-applications/{id}/approve", s.adminApproveApplication)
	mux.HandleFunc("POST /api/admin/merchant-applications/{id}/reject", s.adminRejectApplication)
	mux.HandleFunc("GET /api/admin/merchants", s.adminMerchants)
	mux.HandleFunc("POST /api/admin/merchants/{id}/status", s.adminMerchantStatus)
	mux.HandleFunc("GET /api/admin/channel-reviews", s.adminChannelReviews)
	mux.HandleFunc("POST /api/admin/channel-reviews/{id}/verify", s.adminVerifyReview)
	mux.HandleFunc("POST /api/admin/channel-reviews/{id}/approve", s.adminApproveReview)
	mux.HandleFunc("POST /api/admin/channel-reviews/{id}/reject", s.adminRejectReview)
	mux.HandleFunc("GET /api/admin/channels", s.adminMerchantChannels)
	mux.HandleFunc("GET /api/admin/channels/{id}/runtime", s.adminChannelRuntime)
	mux.HandleFunc("POST /api/admin/channels/{id}/test", s.adminTestChannel)
	mux.HandleFunc("POST /api/admin/channels/{id}/balance/refresh", s.adminRefreshChannelBalance)
	mux.HandleFunc("POST /api/admin/models", s.createCanonicalModel)
	mux.HandleFunc("GET /api/admin/models", s.adminModels)
	mux.HandleFunc("POST /api/admin/models/{id}/prices", s.publishOfficialPrice)
	mux.HandleFunc("GET /api/models", s.models)
	mux.HandleFunc("POST /api/merchant/offers", s.createMerchantOffer)
	mux.HandleFunc("GET /api/merchant/offers", s.merchantOffers)
	mux.HandleFunc("GET /api/merchant/health", s.merchantHealth)
	mux.HandleFunc("GET /api/merchant/balance", s.merchantBalance)
	mux.HandleFunc("GET /api/merchant/ledger", s.merchantLedger)
	mux.HandleFunc("GET /api/merchant/withdrawals", s.merchantWithdrawals)
	mux.HandleFunc("POST /api/merchant/withdrawals", s.createMerchantWithdrawal)
	mux.HandleFunc("GET /api/routing-policies/{id}", s.routingPolicy)
	mux.HandleFunc("PUT /api/routing-policies/{id}", s.updateRoutingPolicy)
	mux.HandleFunc("GET /api/offers", s.marketOffers)
	mux.HandleFunc("POST /api/admin/channels/{id}/health/reset", s.adminResetChannelHealth)
	mux.HandleFunc("GET /api/payment-settings", s.paymentSettings)
	mux.HandleFunc("GET /api/payment-orders", s.paymentOrders)
	mux.HandleFunc("PUT /api/admin/payment-settings", s.adminCreatePaymentSetting)
	mux.HandleFunc("GET /api/admin/payment-settings", s.adminPaymentSettings)
	mux.HandleFunc("GET /api/admin/payment-orders", s.adminPaymentOrders)
	mux.HandleFunc("POST /api/admin/payment-orders/{id}/credit/retry", s.adminRetryPaymentCredit)
	mux.HandleFunc("POST /api/admin/payment-orders/{id}/credit/resolve", s.adminResolvePaymentCredit)
	mux.HandleFunc("GET /api/admin/withdrawals", s.adminWithdrawals)
	mux.HandleFunc("POST /api/admin/withdrawals/{id}/approve", s.adminApproveWithdrawal)
	mux.HandleFunc("POST /api/admin/withdrawals/{id}/reject", s.adminRejectWithdrawal)
	mux.HandleFunc("POST /api/admin/withdrawals/{id}/paid", s.adminPayWithdrawal)
	mux.HandleFunc("GET /api/admin/settlement-reconciliation", s.adminSettlementReconciliation)
	mux.HandleFunc("POST /api/admin/settlement-reconciliation/{id}/ignore", s.adminIgnoreSettlementReconciliation)
	mux.HandleFunc("POST /api/payment-orders", s.createPaymentOrder)
	mux.HandleFunc("GET /api/payment-orders/{order_no}", s.paymentOrder)
	mux.HandleFunc("POST /api/payments/easypay/notify", s.easyPayNotify)
	mux.HandleFunc("GET /api/payments/easypay/notify", s.easyPayNotify)
	mux.HandleFunc("POST /v1/chat/completions", s.modelProxy)
	mux.HandleFunc("POST /v1/responses", s.modelProxy)
	mux.HandleFunc("POST /v1/embeddings", s.modelProxy)
	return requestID(clientMetadata(mux))
}

func clientMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := newapi.WithClientIP(r.Context(), clientIP(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *server) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]string{
			"service": "modelroute-platform",
			"status":  "ok",
			"version": s.version,
		},
	})
}

func (s *server) ready(w http.ResponseWriter, r *http.Request) {
	failedDependency := s.readinessFailure(r.Context())
	if failedDependency != "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"success": false,
			"error": map[string]string{
				"code":    "dependency_unavailable",
				"message": failedDependency + " 连接异常",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    map[string]string{"status": "ready"},
	})
}

func (s *server) readinessFailure(ctx context.Context) string {
	s.readiness.mu.Lock()
	defer s.readiness.mu.Unlock()

	if !s.readiness.checkedAt.IsZero() && time.Since(s.readiness.checkedAt) < readinessCacheDuration {
		return s.readiness.failedDependency
	}

	checks := []struct {
		name   string
		pinger Pinger
	}{
		{name: "database", pinger: s.dependencies.Database},
		{name: "redis", pinger: s.dependencies.Redis},
		{name: "new_api", pinger: s.dependencies.NewAPI},
	}

	failedDependency := ""
	for _, check := range checks {
		if check.pinger == nil || check.pinger.Ping(ctx) != nil {
			failedDependency = check.name
			break
		}
	}

	s.readiness.checkedAt = time.Now()
	s.readiness.failedDependency = failedDependency
	return failedDependency
}

func (s *server) versionInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    map[string]string{"version": s.version},
	})
}

func modelRouteDisabled(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]any{
		"error": map[string]any{
			"message": "模型路由正在部署中",
			"type":    "server_error",
			"param":   nil,
			"code":    "service_not_ready",
		},
	})
}

type requestIDContextKey struct{}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			var value [12]byte
			if _, err := rand.Read(value[:]); err == nil {
				requestID = hex.EncodeToString(value[:])
			}
		}
		if requestID != "" {
			w.Header().Set("X-Request-ID", requestID)
			r = r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, requestID))
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
