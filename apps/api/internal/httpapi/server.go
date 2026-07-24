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
	mux.HandleFunc("GET /api/announcements", s.announcements)
	mux.HandleFunc("POST /api/announcements/{id}/read", s.readAnnouncement)
	mux.HandleFunc("GET /api/keys", s.keys)
	mux.HandleFunc("POST /api/keys", s.createKey)
	mux.HandleFunc("PUT /api/keys/{id}", s.updateKey)
	mux.HandleFunc("POST /api/keys/{id}/status", s.updateKeyStatus)
	mux.HandleFunc("DELETE /api/keys/{id}", s.deleteKey)
	mux.HandleFunc("PUT /api/keys/{id}/routing-policies", s.replaceKeyRoutingPolicies)
	mux.HandleFunc("GET /api/usage", s.usage)
	mux.HandleFunc("POST /api/redemptions/redeem", s.redeemCode)
	mux.HandleFunc("GET /api/admin/users", s.adminUsers)
	mux.HandleFunc("GET /api/admin/users/{id}", s.adminUser)
	mux.HandleFunc("POST /api/admin/users/{id}/status", s.adminUserStatus)
	mux.HandleFunc("POST /api/admin/users/{id}/quota", s.adminUserQuota)
	mux.HandleFunc("POST /api/admin/users/{id}/password", s.adminUserPassword)
	mux.HandleFunc("GET /api/admin/channel-logs", s.adminChannelLogs)
	mux.HandleFunc("GET /api/admin/overview", s.adminOverview)
	mux.HandleFunc("GET /api/admin/announcements", s.adminAnnouncements)
	mux.HandleFunc("POST /api/admin/announcements", s.createAdminAnnouncement)
	mux.HandleFunc("PUT /api/admin/announcements/{id}", s.updateAdminAnnouncement)
	mux.HandleFunc("POST /api/admin/announcements/{id}/publish", s.publishAdminAnnouncement)
	mux.HandleFunc("POST /api/admin/announcements/{id}/withdraw", s.withdrawAdminAnnouncement)
	mux.HandleFunc("GET /api/admin/platform-settings", s.adminPlatformSettings)
	mux.HandleFunc("PUT /api/admin/platform-settings", s.updateAdminPlatformSettings)
	mux.HandleFunc("GET /api/admin/redemptions", s.adminRedemptions)
	mux.HandleFunc("POST /api/admin/redemptions", s.createAdminRedemption)
	mux.HandleFunc("PUT /api/admin/redemptions/{id}", s.updateAdminRedemption)
	mux.HandleFunc("POST /api/admin/redemptions/{id}/status", s.updateAdminRedemptionStatus)
	mux.HandleFunc("DELETE /api/admin/redemptions/invalid", s.deleteInvalidAdminRedemptions)
	mux.HandleFunc("DELETE /api/admin/redemptions/{id}", s.deleteAdminRedemption)
	mux.HandleFunc("GET /api/merchant/application", s.merchantApplication)
	mux.HandleFunc("POST /api/merchant/application", s.createMerchantApplication)
	mux.HandleFunc("GET /api/merchant/profile", s.merchantProfile)
	mux.HandleFunc("PUT /api/merchant/profile", s.updateMerchantProfile)
	mux.HandleFunc("POST /api/merchant/channel-tests", s.createMerchantChannelTest)
	mux.HandleFunc("POST /api/merchant/channel-models/sync", s.syncMerchantChannelModels)
	mux.HandleFunc("GET /api/merchant/channels", s.merchantChannels)
	mux.HandleFunc("POST /api/merchant/channels", s.createMerchantChannel)
	mux.HandleFunc("DELETE /api/merchant/channel-reviews/{id}", s.deleteMerchantChannelReview)
	mux.HandleFunc("POST /api/merchant/channels/{id}/models/sync", s.syncExistingMerchantChannelModels)
	mux.HandleFunc("POST /api/merchant/channels/{id}/status", s.updateMerchantChannelStatus)
	mux.HandleFunc("DELETE /api/merchant/channels/{id}", s.deleteMerchantChannel)
	mux.HandleFunc("GET /api/admin/merchant-applications", s.adminMerchantApplications)
	mux.HandleFunc("GET /api/admin/merchant-applications/{id}", s.adminMerchantApplication)
	mux.HandleFunc("POST /api/admin/merchant-applications/{id}/approve", s.adminApproveApplication)
	mux.HandleFunc("POST /api/admin/merchant-applications/{id}/reject", s.adminRejectApplication)
	mux.HandleFunc("POST /api/admin/users/{id}/merchant-status", s.adminUserMerchantStatus)
	mux.HandleFunc("GET /api/admin/channel-reviews", s.adminChannelReviews)
	mux.HandleFunc("POST /api/admin/channel-reviews/{id}/verify", s.adminVerifyReview)
	mux.HandleFunc("POST /api/admin/channel-reviews/{id}/model-tests", s.adminTestReviewModel)
	mux.HandleFunc("POST /api/admin/channel-reviews/{id}/approve", s.adminApproveReview)
	mux.HandleFunc("POST /api/admin/channel-reviews/{id}/reject", s.adminRejectReview)
	mux.HandleFunc("GET /api/admin/channels", s.adminMerchantChannels)
	mux.HandleFunc("POST /api/admin/channels/{id}/status", s.adminUpdateChannelStatus)
	mux.HandleFunc("DELETE /api/admin/channels/{id}", s.adminDeleteChannel)
	mux.HandleFunc("POST /api/admin/models", s.createCanonicalModel)
	mux.HandleFunc("POST /api/admin/models/detect", s.detectCanonicalModel)
	mux.HandleFunc("GET /api/admin/models", s.adminModels)
	mux.HandleFunc("POST /api/admin/models/{id}/prices", s.publishOfficialPrice)
	mux.HandleFunc("DELETE /api/admin/models/{id}", s.deleteCanonicalModel)
	mux.HandleFunc("GET /api/models", s.models)
	mux.HandleFunc("PUT /api/merchant/offers/{model_id}", s.createMerchantOffer)
	mux.HandleFunc("POST /api/merchant/offers/{model_id}/status", s.updateMerchantOfferStatus)
	mux.HandleFunc("DELETE /api/merchant/offers/{model_id}", s.deleteMerchantOffer)
	mux.HandleFunc("GET /api/merchant/offers", s.merchantOffers)
	mux.HandleFunc("GET /api/merchant/balance", s.merchantBalance)
	mux.HandleFunc("GET /api/merchant/usage", s.merchantUsage)
	mux.HandleFunc("GET /api/merchant/withdrawals", s.merchantWithdrawals)
	mux.HandleFunc("POST /api/merchant/withdrawals", s.createMerchantWithdrawal)
	mux.HandleFunc("GET /api/routing-policies", s.routingPolicies)
	mux.HandleFunc("POST /api/routing-policies", s.createRoutingPolicy)
	mux.HandleFunc("POST /api/routing-policies/preview", s.previewRoutingPolicy)
	mux.HandleFunc("GET /api/routing-policies/{id}", s.routingPolicy)
	mux.HandleFunc("PUT /api/routing-policies/{id}", s.updateRoutingPolicy)
	mux.HandleFunc("DELETE /api/routing-policies/{id}", s.deleteRoutingPolicy)
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
	mux.HandleFunc("POST /api/payment-orders", s.createPaymentOrder)
	mux.HandleFunc("GET /api/payment-orders/{order_no}", s.paymentOrder)
	mux.HandleFunc("POST /api/payments/easypay/notify", s.easyPayNotify)
	mux.HandleFunc("GET /api/payments/easypay/notify", s.easyPayNotify)
	mux.HandleFunc("POST /v1/chat/completions", s.modelProxy)
	mux.HandleFunc("POST /v1/responses", s.modelProxy)
	mux.HandleFunc("POST /v1/images/generations", s.modelProxy)
	mux.HandleFunc("GET /v1/models", s.modelListProxy)
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
		"data": map[string]any{
			"version":        s.version,
			"quota_per_unit": s.services.QuotaPerUnit,
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
