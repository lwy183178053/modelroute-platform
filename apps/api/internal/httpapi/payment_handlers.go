package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var allowedPaymentMethods = map[string]bool{
	"alipay": true,
	"wxpay":  true,
	"qqpay":  true,
}

type paymentSettingView struct {
	ID                int64     `json:"id"`
	Version           int       `json:"version"`
	GatewayURL        string    `json:"gateway_url"`
	MerchantPIDMasked string    `json:"merchant_pid_masked"`
	KeyConfigured     bool      `json:"key_configured"`
	PaymentMethods    []string  `json:"payment_methods"`
	MinimumCNYCents   int64     `json:"minimum_cny_cents"`
	MaximumCNYCents   int64     `json:"maximum_cny_cents"`
	Enabled           bool      `json:"enabled"`
	NotifyURL         string    `json:"notify_url"`
	ReturnURL         string    `json:"return_url"`
	CreatedAt         time.Time `json:"created_at"`
}

type publicPaymentSettingView struct {
	Version         int      `json:"version"`
	PaymentMethods  []string `json:"payment_methods"`
	MinimumCNYCents int64    `json:"minimum_cny_cents"`
	MaximumCNYCents int64    `json:"maximum_cny_cents"`
	Enabled         bool     `json:"enabled"`
}

func (s *server) paymentSettings(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleCommon); !ok {
		return
	}
	setting, err := s.currentPaymentSetting(r)
	if err != nil {
		writePaymentReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": publicPaymentSettingView{
		Version: setting.Version, PaymentMethods: setting.PaymentMethods,
		MinimumCNYCents: setting.MinimumCNYCents, MaximumCNYCents: setting.MaximumCNYCents,
		Enabled: setting.Enabled,
	}})
}

func (s *server) paymentOrders(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok {
		return
	}
	if s.services.Payments == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "payment_unavailable", "支付服务尚未就绪")
		return
	}
	page, pageSize := pagination(r)
	items, total, err := s.services.Payments.PaymentOrdersForUser(r.Context(), session.UserID, page, pageSize)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "payment_orders_failed", "读取充值订单失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"page": page, "page_size": pageSize, "total": total, "items": items,
	}})
}

func (s *server) paymentOrder(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok {
		return
	}
	if s.services.Payments == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "payment_unavailable", "支付服务尚未就绪")
		return
	}
	order, err := s.services.Payments.PaymentOrderForUser(r.Context(), session.UserID, strings.TrimSpace(r.PathValue("order_no")))
	if err != nil {
		writePaymentReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": order})
}

func (s *server) adminPaymentSettings(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok {
		return
	}
	setting, err := s.currentPaymentSetting(r)
	if err != nil {
		writePaymentReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": s.paymentSettingView(setting)})
}

func (s *server) adminPaymentOrders(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok {
		return
	}
	if s.services.Payments == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "payment_unavailable", "支付服务尚未就绪")
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && !map[string]bool{"pending": true, "paid": true, "closed": true, "processing": true, "succeeded": true, "failed": true, "unknown": true}[status] {
		writeAPIError(w, http.StatusBadRequest, "invalid_payment_status", "订单状态筛选无效")
		return
	}
	page, pageSize := pagination(r)
	items, total, err := s.services.Payments.ListPaymentOrders(r.Context(), page, pageSize, status)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "payment_orders_failed", "读取充值订单失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"page": page, "page_size": pageSize, "total": total, "items": items,
	}})
}

func (s *server) adminRetryPaymentCredit(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	orderID, ok := paymentOrderPathID(w, r)
	if !ok {
		return
	}
	if err := s.services.Payments.RetryPaymentCredit(r.Context(), orderID, false); err != nil {
		writePaymentCreditActionError(w, err)
		return
	}
	if err := s.recordAudit(r, session, "payment.credit.retry", int(orderID), "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "入账已重新排队，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nil})
}

func (s *server) adminResolvePaymentCredit(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	orderID, ok := paymentOrderPathID(w, r)
	if !ok {
		return
	}
	var request struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_credit_resolution", "核对处置参数无效")
		return
	}
	request.Action = strings.TrimSpace(request.Action)
	request.Reason = strings.TrimSpace(request.Reason)
	if (request.Action != "retry" && request.Action != "credited") || len(request.Reason) < 3 || len(request.Reason) > 500 {
		writeAPIError(w, http.StatusBadRequest, "invalid_credit_resolution", "核对处置操作或原因无效")
		return
	}
	var err error
	if request.Action == "retry" {
		err = s.services.Payments.RetryPaymentCredit(r.Context(), orderID, true)
	} else {
		err = s.services.Payments.ConfirmPaymentCredit(r.Context(), orderID, time.Now().UTC())
	}
	if err != nil {
		writePaymentCreditActionError(w, err)
		return
	}
	if err := s.recordAudit(r, session, "payment.credit.resolve."+request.Action, int(orderID), "success", request.Reason); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "核对处置已完成，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nil})
}

func paymentOrderPathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || value <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_payment_order_id", "充值订单编号无效")
		return 0, false
	}
	return value, true
}

func writePaymentCreditActionError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrPaymentConflict) || errors.Is(err, ErrPaymentNotFound) {
		writeAPIError(w, http.StatusConflict, "payment_credit_state_conflict", "当前入账状态不允许此操作")
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "payment_credit_action_failed", "更新入账状态失败")
}

func (s *server) currentPaymentSetting(r *http.Request) (PaymentSetting, error) {
	if s.services.Payments == nil {
		return PaymentSetting{}, errors.New("支付服务未就绪")
	}
	return s.services.Payments.CurrentPaymentSetting(r.Context())
}

func writePaymentReadError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrPaymentNotFound) || errors.Is(err, ErrPaymentNotConfigured) {
		writeAPIError(w, http.StatusNotFound, "payment_not_found", "支付配置或订单不存在")
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "payment_read_failed", "读取支付数据失败")
}

func (s *server) adminCreatePaymentSetting(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	if s.services.Payments == nil || s.services.Credentials == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "payment_unavailable", "支付服务尚未就绪")
		return
	}
	var request struct {
		GatewayURL      string   `json:"gateway_url"`
		MerchantPID     string   `json:"merchant_pid"`
		MerchantKey     string   `json:"merchant_key"`
		Enabled         bool     `json:"enabled"`
		PaymentMethods  []string `json:"payment_methods"`
		MinimumCNYCents int64    `json:"minimum_cny_cents"`
		MaximumCNYCents int64    `json:"maximum_cny_cents"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_payment_setting", "支付配置参数无效")
		return
	}
	gatewayURL, err := normalizePaymentGatewayURL(request.GatewayURL)
	methods, methodsOK := normalizePaymentMethods(request.PaymentMethods)
	pid := strings.TrimSpace(request.MerchantPID)
	merchantKey := strings.TrimSpace(request.MerchantKey)
	var current PaymentSetting
	if pid == "" || merchantKey == "" {
		var currentErr error
		current, currentErr = s.services.Payments.CurrentPaymentSetting(r.Context())
		if currentErr != nil {
			writeAPIError(w, http.StatusBadRequest, "payment_credentials_required", "首次配置必须填写商户编号和商户密钥")
			return
		}
		if pid == "" {
			pid = current.MerchantPID
		}
	}
	if err != nil || pid == "" || len(pid) > 128 || !methodsOK || request.MinimumCNYCents <= 0 ||
		request.MaximumCNYCents < request.MinimumCNYCents {
		writeAPIError(w, http.StatusBadRequest, "invalid_payment_setting", "支付配置参数无效")
		return
	}

	var encryptedKey []byte
	var fingerprint string
	if merchantKey == "" {
		encryptedKey = current.EncryptedKey
		fingerprint = current.KeyFingerprint
	} else {
		encryptedKey, err = s.services.Credentials.Encrypt([]byte(merchantKey))
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "payment_key_encrypt_failed", "保存支付密钥失败")
			return
		}
		fingerprint = s.services.Credentials.Fingerprint([]byte(merchantKey))
	}
	setting, err := s.services.Payments.CreatePaymentSetting(r.Context(), CreatePaymentSettingCommand{
		AdminUserID: session.UserID, GatewayURL: gatewayURL, MerchantPID: pid, EncryptedKey: encryptedKey,
		KeyFingerprint: fingerprint, PaymentMethods: methods, MinimumCNYCents: request.MinimumCNYCents,
		MaximumCNYCents: request.MaximumCNYCents,
		Enabled:         request.Enabled,
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "payment_setting_save_failed", "保存支付配置失败")
		return
	}
	if err := s.recordAudit(r, session, "payment.setting.create", int(setting.ID), "success", ""); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "支付配置已保存，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": s.paymentSettingView(setting)})
}

func (s *server) createPaymentOrder(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) {
		return
	}
	if s.services.Payments == nil || s.services.Credentials == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "payment_unavailable", "支付服务尚未就绪")
		return
	}
	var request struct {
		AmountCNYCents int64  `json:"amount_cny_cents"`
		PaymentType    string `json:"payment_type"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_payment_order", "充值订单参数无效")
		return
	}
	setting, err := s.services.Payments.CurrentPaymentSetting(r.Context())
	if err != nil || !setting.Enabled {
		writeAPIError(w, http.StatusServiceUnavailable, "payment_disabled", "充值通道当前未启用")
		return
	}
	paymentType := strings.TrimSpace(request.PaymentType)
	if request.AmountCNYCents < setting.MinimumCNYCents || request.AmountCNYCents > setting.MaximumCNYCents || !containsString(setting.PaymentMethods, paymentType) {
		writeAPIError(w, http.StatusBadRequest, "invalid_payment_order", "充值金额或支付方式不在允许范围")
		return
	}
	quota, err := paymentQuota(request.AmountCNYCents)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_payment_order", "充值金额超出范围")
		return
	}
	orderNo, err := newPaymentOrderNo(time.Now().UTC())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "payment_order_create_failed", "创建充值订单失败")
		return
	}
	order, err := s.services.Payments.CreatePaymentOrder(r.Context(), CreatePaymentOrderCommand{
		OrderNo: orderNo, UserID: session.UserID, SettingVersionID: setting.ID, PaymentType: paymentType,
		AmountCNYCents: request.AmountCNYCents, NewAPIQuota: quota,
		ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "payment_order_create_failed", "创建充值订单失败")
		return
	}
	key, err := s.services.Credentials.Decrypt(setting.EncryptedKey)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "payment_key_decrypt_failed", "读取支付配置失败")
		return
	}
	paymentURL := s.easyPayPaymentURL(setting, order, string(key))
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": map[string]any{
		"order": order, "payment_url": paymentURL,
	}})
}

func (s *server) easyPayNotify(w http.ResponseWriter, r *http.Request) {
	if s.services.Payments == nil || s.services.Credentials == nil {
		http.Error(w, "payment unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	orderNo := strings.TrimSpace(r.Form.Get("out_trade_no"))
	tradeNo := strings.TrimSpace(r.Form.Get("trade_no"))
	if orderNo == "" || tradeNo == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	callbackContext, err := s.services.Payments.PaymentCallbackContext(r.Context(), orderNo)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}
	key, err := s.services.Credentials.Decrypt(callbackContext.Setting.EncryptedKey)
	if err != nil {
		http.Error(w, "payment configuration error", http.StatusInternalServerError)
		return
	}
	money, moneyErr := parseGatewayMoney(r.Form.Get("money"))
	if !strings.EqualFold(strings.TrimSpace(r.Form.Get("sign_type")), "MD5") ||
		!verifyEasyPaySign(r.Form, string(key), r.Form.Get("sign")) ||
		r.Form.Get("pid") != callbackContext.Setting.MerchantPID ||
		r.Form.Get("trade_status") != "TRADE_SUCCESS" ||
		r.Form.Get("type") != callbackContext.Order.PaymentType && callbackContext.Order.PaymentType != "" ||
		moneyErr != nil || money != callbackContext.Order.AmountCNYCents {
		http.Error(w, "invalid callback", http.StatusBadRequest)
		return
	}
	payload := make(map[string]string, len(r.Form))
	for name := range r.Form {
		if name != "sign" {
			payload[name] = r.Form.Get(name)
		}
	}
	digest := sha256.Sum256([]byte(easyPayCanonical(r.Form)))
	_, _, err = s.services.Payments.AcceptPaymentCallback(r.Context(), AcceptPaymentCallbackCommand{
		OrderNo: orderNo, TradeNo: tradeNo, CallbackKey: hex.EncodeToString(digest[:]), Payload: payload, PaidAt: time.Now().UTC(),
	})
	if err != nil {
		http.Error(w, "callback conflict", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("success"))
}

func (s *server) easyPayPaymentURL(setting PaymentSetting, order PaymentOrder, key string) string {
	values := url.Values{
		"pid":          {setting.MerchantPID},
		"type":         {order.PaymentType},
		"out_trade_no": {order.OrderNo},
		"notify_url":   {s.services.Cookie.PublicOrigin + "/api/payments/easypay/notify"},
		"return_url":   {s.services.Cookie.PublicOrigin + "/payments/return"},
		"name":         {"ModelRoute 额度充值"},
		"money":        {formatGatewayMoney(order.AmountCNYCents)},
	}
	values.Set("sign", easyPaySign(values, key))
	values.Set("sign_type", "MD5")
	return strings.TrimRight(setting.GatewayURL, "/") + "/submit.php?" + values.Encode()
}

func (s *server) paymentSettingView(setting PaymentSetting) paymentSettingView {
	return paymentSettingView{
		ID: setting.ID, Version: setting.Version, GatewayURL: setting.GatewayURL,
		MerchantPIDMasked: maskPaymentPID(setting.MerchantPID), KeyConfigured: len(setting.EncryptedKey) > 0,
		PaymentMethods: setting.PaymentMethods, MinimumCNYCents: setting.MinimumCNYCents,
		MaximumCNYCents: setting.MaximumCNYCents,
		Enabled:         setting.Enabled, NotifyURL: s.services.Cookie.PublicOrigin + "/api/payments/easypay/notify",
		ReturnURL: s.services.Cookie.PublicOrigin + "/payments/return", CreatedAt: setting.CreatedAt,
	}
}

func normalizePaymentGatewayURL(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("支付网关地址无效")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return strings.TrimRight(parsed.String(), "/"), nil
}

func normalizePaymentMethods(values []string) ([]string, bool) {
	result := make([]string, 0, len(values))
	seen := make(map[string]bool)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !allowedPaymentMethods[value] || seen[value] {
			return nil, false
		}
		seen[value] = true
		result = append(result, value)
	}
	return result, len(result) > 0
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func newPaymentOrderNo(now time.Time) (string, error) {
	var random [6]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "MR" + now.Format("20060102150405") + strings.ToUpper(hex.EncodeToString(random[:])), nil
}

func formatGatewayMoney(cents int64) string {
	return strconv.FormatInt(cents/100, 10) + "." + fmt.Sprintf("%02d", cents%100)
}

func maskPaymentPID(value string) string {
	if len(value) <= 6 {
		return strings.Repeat("*", len(value))
	}
	return value[:3] + strings.Repeat("*", len(value)-6) + value[len(value)-3:]
}
