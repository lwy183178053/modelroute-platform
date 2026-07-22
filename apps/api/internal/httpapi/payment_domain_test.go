package httpapi

import (
	"net/url"
	"testing"
)

func TestPaymentAmountUsesIntegerArithmetic(t *testing.T) {
	tests := []struct {
		name      string
		usdCents  int64
		rate      int64
		wantCNY   int64
		wantQuota int64
	}{
		{name: "整美元", usdCents: 10_000, rate: 720, wantCNY: 72_000, wantQuota: 50_000_000},
		{name: "分币四舍五入", usdCents: 101, rate: 723, wantCNY: 730, wantQuota: 505_000},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cnyCents, quota, err := paymentAmounts(test.usdCents, test.rate)
			if err != nil {
				t.Fatalf("计算支付金额失败：%v", err)
			}
			if cnyCents != test.wantCNY || quota != test.wantQuota {
				t.Fatalf("人民币分=%d，额度=%d", cnyCents, quota)
			}
		})
	}
}

func TestPaymentAmountRejectsInvalidOrOverflowingValues(t *testing.T) {
	for _, input := range []struct{ usdCents, rate int64 }{
		{0, 720}, {-1, 720}, {100, 0}, {1 << 62, 720},
	} {
		if _, _, err := paymentAmounts(input.usdCents, input.rate); err == nil {
			t.Fatalf("应拒绝 usd_cents=%d rate=%d", input.usdCents, input.rate)
		}
	}
}

func TestParseGatewayMoneyWithoutFloatingPoint(t *testing.T) {
	for input, want := range map[string]int64{
		"0.01":  1,
		"10":    1000,
		"10.5":  1050,
		"10.50": 1050,
	} {
		got, err := parseGatewayMoney(input)
		if err != nil || got != want {
			t.Fatalf("解析 %q 得到 %d，错误=%v", input, got, err)
		}
	}
	for _, input := range []string{"", "0", "-1.00", "1.001", "1e2", "abc"} {
		if _, err := parseGatewayMoney(input); err == nil {
			t.Fatalf("应拒绝金额 %q", input)
		}
	}
}

func TestEasyPaySignSortsAndExcludesProtocolFields(t *testing.T) {
	values := url.Values{
		"type":         {"alipay"},
		"pid":          {"1001"},
		"out_trade_no": {"MR202607220001"},
		"money":        {"72.00"},
		"name":         {"ModelRoute 额度充值"},
		"empty":        {""},
		"sign":         {"ignored"},
		"sign_type":    {"MD5"},
	}
	canonical := easyPayCanonical(values)
	wantCanonical := "money=72.00&name=ModelRoute 额度充值&out_trade_no=MR202607220001&pid=1001&type=alipay"
	if canonical != wantCanonical {
		t.Fatalf("签名原文=%q", canonical)
	}
	if got, want := easyPaySign(values, "secret"), "341e07dcb3f12fdbc4014c58139a8787"; got != want {
		t.Fatalf("签名=%q，期望=%q", got, want)
	}
	if !verifyEasyPaySign(values, "secret", "341E07DCB3F12FDBC4014C58139A8787") {
		t.Fatal("验签应兼容大写十六进制签名")
	}
}
