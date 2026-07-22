package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

type paymentRepositoryStub struct {
	PaymentRepository
	currentSetting  func(context.Context) (PaymentSetting, error)
	createSetting   func(context.Context, CreatePaymentSettingCommand) (PaymentSetting, error)
	createOrder     func(context.Context, CreatePaymentOrderCommand) (PaymentOrder, error)
	callbackContext func(context.Context, string) (PaymentCallbackContext, error)
	acceptCallback  func(context.Context, AcceptPaymentCallbackCommand) (PaymentOrder, bool, error)
	ordersForUser   func(context.Context, int, int, int) ([]PaymentOrder, int, error)
	orderForUser    func(context.Context, int, string) (PaymentOrder, error)
	listOrders      func(context.Context, int, int, string) ([]PaymentOrder, int, error)
	retryCredit     func(context.Context, int64, bool) error
	confirmCredit   func(context.Context, int64, time.Time) error
}

func (s *paymentRepositoryStub) CurrentPaymentSetting(ctx context.Context) (PaymentSetting, error) {
	return s.currentSetting(ctx)
}

func (s *paymentRepositoryStub) CreatePaymentSetting(ctx context.Context, command CreatePaymentSettingCommand) (PaymentSetting, error) {
	return s.createSetting(ctx, command)
}

func (s *paymentRepositoryStub) CreatePaymentOrder(ctx context.Context, command CreatePaymentOrderCommand) (PaymentOrder, error) {
	return s.createOrder(ctx, command)
}

func (s *paymentRepositoryStub) PaymentCallbackContext(ctx context.Context, orderNo string) (PaymentCallbackContext, error) {
	return s.callbackContext(ctx, orderNo)
}

func (s *paymentRepositoryStub) AcceptPaymentCallback(ctx context.Context, command AcceptPaymentCallbackCommand) (PaymentOrder, bool, error) {
	return s.acceptCallback(ctx, command)
}

func (s *paymentRepositoryStub) PaymentOrdersForUser(ctx context.Context, userID int, page int, pageSize int) ([]PaymentOrder, int, error) {
	return s.ordersForUser(ctx, userID, page, pageSize)
}

func (s *paymentRepositoryStub) PaymentOrderForUser(ctx context.Context, userID int, orderNo string) (PaymentOrder, error) {
	return s.orderForUser(ctx, userID, orderNo)
}

func (s *paymentRepositoryStub) ListPaymentOrders(ctx context.Context, page int, pageSize int, status string) ([]PaymentOrder, int, error) {
	return s.listOrders(ctx, page, pageSize, status)
}

func (s *paymentRepositoryStub) RetryPaymentCredit(ctx context.Context, orderID int64, allowUnknown bool) error {
	return s.retryCredit(ctx, orderID, allowUnknown)
}

func (s *paymentRepositoryStub) ConfirmPaymentCredit(ctx context.Context, orderID int64, confirmedAt time.Time) error {
	return s.confirmCredit(ctx, orderID, confirmedAt)
}

func TestAdminCreatesEncryptedImmutablePaymentSetting(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	box := testCredentialBox(t)
	secret := "payment-secret"
	repository := &paymentRepositoryStub{}
	repository.currentSetting = func(context.Context) (PaymentSetting, error) { return PaymentSetting{}, ErrPaymentNotConfigured }
	repository.createSetting = func(_ context.Context, command CreatePaymentSettingCommand) (PaymentSetting, error) {
		if command.AdminUserID != 1 || command.GatewayURL != "https://zpayz.cn" || command.MerchantPID != "1001" {
			t.Fatalf("支付配置参数=%+v", command)
		}
		if strings.Contains(string(command.EncryptedKey), secret) || len(command.EncryptedKey) == 0 || len(command.KeyFingerprint) != 64 {
			t.Fatal("易支付 KEY 未加密或缺少指纹")
		}
		return PaymentSetting{
			ID: 11, Version: 1, GatewayURL: command.GatewayURL, MerchantPID: command.MerchantPID,
			EncryptedKey: command.EncryptedKey, KeyFingerprint: command.KeyFingerprint,
			PaymentMethods: command.PaymentMethods, MinimumUSDCents: command.MinimumUSDCents,
			MaximumUSDCents: command.MaximumUSDCents, ExchangeRateCNYCentsPerUSD: command.ExchangeRateCNYCentsPerUSD,
			Enabled: command.Enabled, CreatedAt: time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC),
		}, nil
	}
	services.Payments = repository
	services.Credentials = box
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPut, "/api/admin/payment-settings", strings.NewReader(`{
		"gateway_url":"https://zpayz.cn/","merchant_pid":"1001","merchant_key":"payment-secret",
		"enabled":true,"payment_methods":["alipay","wxpay"],"minimum_usd_cents":1000,
		"maximum_usd_cents":2000000,"exchange_rate_cny_cents_per_usd":720
	}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("状态码=%d，响应=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), secret) || strings.Contains(response.Body.String(), "encrypted_key") {
		t.Fatalf("支付配置响应泄露 KEY：%s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"key_configured":true`) || !strings.Contains(response.Body.String(), `"version":1`) {
		t.Fatalf("支付配置响应=%s", response.Body.String())
	}
}

func TestAdminKeepsExistingPIDAndKeyWhenRotationFieldsAreBlank(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	box := testCredentialBox(t)
	repository := &paymentRepositoryStub{}
	repository.currentSetting = func(context.Context) (PaymentSetting, error) {
		return PaymentSetting{MerchantPID: "existing-pid", EncryptedKey: []byte("existing-cipher"), KeyFingerprint: strings64("a")}, nil
	}
	repository.createSetting = func(_ context.Context, command CreatePaymentSettingCommand) (PaymentSetting, error) {
		if command.MerchantPID != "existing-pid" || string(command.EncryptedKey) != "existing-cipher" || command.KeyFingerprint != strings64("a") {
			t.Fatalf("未沿用现有 PID/KEY：%+v", command)
		}
		return PaymentSetting{ID: 12, Version: 4, MerchantPID: command.MerchantPID, EncryptedKey: command.EncryptedKey, Enabled: true}, nil
	}
	services.Payments = repository
	services.Credentials = box
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPut, "/api/admin/payment-settings", strings.NewReader(`{
		"gateway_url":"https://zpayz.cn","merchant_pid":"","merchant_key":"","enabled":true,
		"payment_methods":["alipay"],"minimum_usd_cents":1000,"maximum_usd_cents":2000000,
		"exchange_rate_cny_cents_per_usd":720
	}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("状态码=%d，响应=%s", response.Code, response.Body.String())
	}
}

func TestUserCreatesUSDRechargeOrderAndReceivesSignedPaymentURL(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	box := testCredentialBox(t)
	encrypted, err := box.Encrypt([]byte("payment-secret"))
	if err != nil {
		t.Fatal(err)
	}
	setting := PaymentSetting{
		ID: 11, Version: 3, GatewayURL: "https://zpayz.cn", MerchantPID: "1001", EncryptedKey: encrypted,
		PaymentMethods: []string{"alipay", "wxpay"}, MinimumUSDCents: 1000, MaximumUSDCents: 2000000,
		ExchangeRateCNYCentsPerUSD: 720, Enabled: true,
	}
	repository := &paymentRepositoryStub{}
	repository.currentSetting = func(context.Context) (PaymentSetting, error) { return setting, nil }
	repository.createOrder = func(_ context.Context, command CreatePaymentOrderCommand) (PaymentOrder, error) {
		if command.UserID != 7 || command.SettingVersionID != 11 || command.USDCents != 10_000 || command.CNYCents != 72_000 || command.NewAPIQuota != 50_000_000 {
			t.Fatalf("充值订单参数=%+v", command)
		}
		return PaymentOrder{
			ID: 21, OrderNo: "MR202607220001", UserID: command.UserID, SettingVersionID: command.SettingVersionID,
			PaymentType: command.PaymentType, USDCents: command.USDCents, CNYCents: command.CNYCents,
			ExchangeRateCNYCentsPerUSD: command.ExchangeRateCNYCentsPerUSD, NewAPIQuota: command.NewAPIQuota,
			PaymentStatus: "pending", CreditStatus: "pending", ExpiresAt: command.ExpiresAt,
		}, nil
	}
	services.Payments = repository
	services.Credentials = box
	token := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodPost, "/api/payment-orders", strings.NewReader(`{"usd_cents":10000,"payment_type":"alipay"}`))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("状态码=%d，响应=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Data struct {
			Order      PaymentOrder `json:"order"`
			PaymentURL string       `json:"payment_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(payload.Data.PaymentURL)
	if err != nil || parsed.Host != "zpayz.cn" || parsed.Path != "/submit.php" {
		t.Fatalf("支付地址=%q，错误=%v", payload.Data.PaymentURL, err)
	}
	query := parsed.Query()
	if query.Get("out_trade_no") != "MR202607220001" || query.Get("money") != "720.00" || query.Get("notify_url") != "https://aggregate.lwylink.xyz/api/payments/easypay/notify" {
		t.Fatalf("支付参数=%v", query)
	}
	if !verifyEasyPaySign(query, "payment-secret", query.Get("sign")) {
		t.Fatalf("支付地址签名无效：%v", query)
	}
}

func TestEasyPayNotifyValidatesBeforeIdempotentAcceptance(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	box := testCredentialBox(t)
	encrypted, err := box.Encrypt([]byte("payment-secret"))
	if err != nil {
		t.Fatal(err)
	}
	repository := &paymentRepositoryStub{}
	repository.callbackContext = func(_ context.Context, orderNo string) (PaymentCallbackContext, error) {
		return PaymentCallbackContext{
			Order:   PaymentOrder{ID: 21, OrderNo: orderNo, PaymentType: "alipay", CNYCents: 72_000, PaymentStatus: "pending"},
			Setting: PaymentSetting{ID: 11, MerchantPID: "1001", EncryptedKey: encrypted},
		}, nil
	}
	accepted := 0
	repository.acceptCallback = func(_ context.Context, command AcceptPaymentCallbackCommand) (PaymentOrder, bool, error) {
		accepted++
		if command.OrderNo != "MR202607220001" || command.TradeNo != "EP202607220009" || len(command.CallbackKey) != 64 {
			t.Fatalf("回调认账参数=%+v", command)
		}
		return PaymentOrder{ID: 21, OrderNo: command.OrderNo, PaymentStatus: "paid", CreditStatus: "pending"}, true, nil
	}
	services.Payments = repository
	services.Credentials = box
	handler := NewServerWithServices("test", Dependencies{}, services)

	values := url.Values{
		"pid": {"1001"}, "trade_no": {"EP202607220009"}, "out_trade_no": {"MR202607220001"},
		"type": {"alipay"}, "name": {"ModelRoute 额度充值"}, "money": {"720.00"}, "trade_status": {"TRADE_SUCCESS"},
		"sign_type": {"MD5"},
	}
	values.Set("sign", "bad-signature")
	badRequest := httptest.NewRequest(http.MethodPost, "/api/payments/easypay/notify", strings.NewReader(values.Encode()))
	badRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badResponse := httptest.NewRecorder()
	handler.ServeHTTP(badResponse, badRequest)
	if badResponse.Code != http.StatusBadRequest || accepted != 0 {
		t.Fatalf("错误签名状态码=%d，认账次数=%d", badResponse.Code, accepted)
	}

	for _, mutation := range []struct {
		name  string
		field string
		value string
	}{
		{name: "PID 篡改", field: "pid", value: "other-pid"},
		{name: "金额篡改", field: "money", value: "719.99"},
		{name: "状态篡改", field: "trade_status", value: "WAIT_BUYER_PAY"},
		{name: "支付方式篡改", field: "type", value: "wxpay"},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			copyValues := make(url.Values, len(values))
			for key, list := range values {
				copyValues[key] = append([]string(nil), list...)
			}
			copyValues.Set(mutation.field, mutation.value)
			copyValues.Set("sign", easyPaySign(copyValues, "payment-secret"))
			request := httptest.NewRequest(http.MethodPost, "/api/payments/easypay/notify", strings.NewReader(copyValues.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || accepted != 0 {
				t.Fatalf("状态码=%d，认账次数=%d", response.Code, accepted)
			}
		})
	}

	values.Set("sign", easyPaySign(values, "payment-secret"))
	goodRequest := httptest.NewRequest(http.MethodGet, "/api/payments/easypay/notify?"+values.Encode(), nil)
	goodResponse := httptest.NewRecorder()
	handler.ServeHTTP(goodResponse, goodRequest)
	if goodResponse.Code != http.StatusOK || goodResponse.Body.String() != "success" || accepted != 1 {
		t.Fatalf("正确回调状态码=%d，响应=%q，认账次数=%d", goodResponse.Code, goodResponse.Body.String(), accepted)
	}
}

func TestPaymentReadAPIsRespectUserOwnershipAndAdminRole(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	box := testCredentialBox(t)
	repository := &paymentRepositoryStub{}
	repository.currentSetting = func(context.Context) (PaymentSetting, error) {
		return PaymentSetting{
			ID: 11, Version: 3, GatewayURL: "https://zpayz.cn", MerchantPID: "1234567890",
			EncryptedKey: []byte("encrypted-secret"), PaymentMethods: []string{"alipay"},
			MinimumUSDCents: 1000, MaximumUSDCents: 2_000_000, ExchangeRateCNYCentsPerUSD: 720, Enabled: true,
		}, nil
	}
	repository.ordersForUser = func(_ context.Context, userID int, page int, pageSize int) ([]PaymentOrder, int, error) {
		if userID != 7 || page != 1 || pageSize != 20 {
			t.Fatalf("用户订单查询参数 uid=%d page=%d pageSize=%d", userID, page, pageSize)
		}
		return []PaymentOrder{{ID: 21, OrderNo: "MR-OWN", UserID: 7, PaymentStatus: "paid"}}, 1, nil
	}
	repository.orderForUser = func(_ context.Context, userID int, orderNo string) (PaymentOrder, error) {
		if userID != 7 || orderNo != "MR-OWN" {
			t.Fatalf("用户订单详情参数 uid=%d order=%s", userID, orderNo)
		}
		return PaymentOrder{ID: 21, OrderNo: orderNo, UserID: userID, PaymentStatus: "paid"}, nil
	}
	repository.listOrders = func(_ context.Context, page int, pageSize int, status string) ([]PaymentOrder, int, error) {
		if page != 1 || pageSize != 20 || status != "paid" {
			t.Fatalf("管理员订单查询参数 page=%d pageSize=%d status=%s", page, pageSize, status)
		}
		return []PaymentOrder{{ID: 21, OrderNo: "MR-OWN", UserID: 7, PaymentStatus: "paid"}}, 1, nil
	}
	services.Payments = repository
	services.Credentials = box
	userToken := createPlatformSession(t, services, 7, 1)
	adminToken := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)

	tests := []struct {
		path       string
		token      string
		wantStatus int
		want       string
		forbidden  string
	}{
		{path: "/api/payment-settings", token: userToken, wantStatus: http.StatusOK, want: `"exchange_rate_cny_cents_per_usd":720`, forbidden: "1234567890"},
		{path: "/api/payment-orders", token: userToken, wantStatus: http.StatusOK, want: "MR-OWN"},
		{path: "/api/payment-orders/MR-OWN", token: userToken, wantStatus: http.StatusOK, want: "MR-OWN"},
		{path: "/api/admin/payment-settings", token: userToken, wantStatus: http.StatusForbidden},
		{path: "/api/admin/payment-settings", token: adminToken, wantStatus: http.StatusOK, want: `"merchant_pid_masked":"123****890"`, forbidden: "encrypted-secret"},
		{path: "/api/admin/payment-orders?status=paid", token: adminToken, wantStatus: http.StatusOK, want: "MR-OWN"},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: test.token})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.wantStatus || test.want != "" && !strings.Contains(response.Body.String(), test.want) || test.forbidden != "" && strings.Contains(response.Body.String(), test.forbidden) {
			t.Fatalf("GET %s 状态码=%d，响应=%s", test.path, response.Code, response.Body.String())
		}
	}
}

func TestAdminRetriesFailedCreditAndResolvesUnknownCredit(t *testing.T) {
	services, _, audit := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	repository := &paymentRepositoryStub{}
	retryCalls := make([]string, 0)
	repository.retryCredit = func(_ context.Context, orderID int64, allowUnknown bool) error {
		retryCalls = append(retryCalls, strconv.FormatInt(orderID, 10)+":"+strconv.FormatBool(allowUnknown))
		return nil
	}
	confirmed := int64(0)
	repository.confirmCredit = func(_ context.Context, orderID int64, _ time.Time) error {
		confirmed = orderID
		return nil
	}
	services.Payments = repository
	adminToken := createPlatformSession(t, services, 1, 100)
	userToken := createPlatformSession(t, services, 7, 1)
	handler := NewServerWithServices("test", Dependencies{}, services)

	for _, test := range []struct {
		path       string
		body       string
		token      string
		wantStatus int
	}{
		{path: "/api/admin/payment-orders/21/credit/retry", body: `{}`, token: userToken, wantStatus: http.StatusForbidden},
		{path: "/api/admin/payment-orders/21/credit/retry", body: `{}`, token: adminToken, wantStatus: http.StatusOK},
		{path: "/api/admin/payment-orders/22/credit/resolve", body: `{"action":"retry","reason":"已核对 New API 审计，确认此前没有加款"}`, token: adminToken, wantStatus: http.StatusOK},
		{path: "/api/admin/payment-orders/23/credit/resolve", body: `{"action":"credited","reason":"已核对 New API 余额，确认此前已经加款"}`, token: adminToken, wantStatus: http.StatusOK},
	} {
		request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
		request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: test.token})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.wantStatus {
			t.Fatalf("POST %s 状态码=%d，响应=%s", test.path, response.Code, response.Body.String())
		}
	}
	if strings.Join(retryCalls, "|") != "21:false|22:true" || confirmed != 23 {
		t.Fatalf("重试=%v，确认=%d", retryCalls, confirmed)
	}
	if len(audit.entries) != 3 || audit.entries[0].Action != "payment.credit.retry" || audit.entries[1].Action != "payment.credit.resolve.retry" || audit.entries[2].Action != "payment.credit.resolve.credited" {
		t.Fatalf("支付审计=%+v", audit.entries)
	}
}
