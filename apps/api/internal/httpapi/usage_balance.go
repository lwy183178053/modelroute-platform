package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

func (s *server) usageBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeOpenAIError(w, http.StatusUnauthorized, "invalid_api_key", "缺少有效的 Bearer 密钥")
		return
	}
	resolver, ok := s.services.Routing.(TokenAccountResolver)
	if !ok || s.services.Credentials == nil || s.services.AdminNewAPI == nil || s.services.QuotaPerUnit <= 0 {
		writeOpenAIError(w, http.StatusServiceUnavailable, "balance_unavailable", "账户余额服务暂时不可用")
		return
	}
	userID, err := resolver.TokenAccountID(r.Context(), s.services.Credentials.Fingerprint([]byte(token)))
	if errors.Is(err, ErrTokenNotFound) {
		writeOpenAIError(w, http.StatusUnauthorized, "invalid_api_key", "密钥不存在或已停用")
		return
	}
	if err != nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "balance_unavailable", "账户余额服务暂时不可用")
		return
	}
	data, err := s.services.AdminNewAPI.Do(r.Context(), http.MethodGet, fmt.Sprintf("/api/user/%d", userID), nil)
	if err != nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "balance_unavailable", "账户余额服务暂时不可用")
		return
	}
	var user newapi.User
	if err := json.Unmarshal(data, &user); err != nil || user.ID != userID {
		writeOpenAIError(w, http.StatusServiceUnavailable, "balance_unavailable", "账户余额服务暂时不可用")
		return
	}
	if user.Status != 1 {
		writeOpenAIError(w, http.StatusForbidden, "account_inactive", "账号当前不可用")
		return
	}
	quota := int64(user.Quota)
	balance := float64(quota) / float64(s.services.QuotaPerUnit)
	writeJSON(w, http.StatusOK, map[string]any{
		"isValid":         true,
		"is_active":       true,
		"remaining":       balance,
		"balance":         balance,
		"unit":            "CNY",
		"remaining_quota": quota,
		"quota_per_unit":  s.services.QuotaPerUnit,
	})
}
