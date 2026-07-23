package httpapi

import (
	"net/url"
	"testing"
)

func TestPaymentAmountUsesIntegerArithmetic(t *testing.T) {
	tests := []struct {
		name      string
		cnyCents  int64
		wantQuota int64
	}{
		{name: "一元", cnyCents: 100, wantQuota: 500_000},
		{name: "一分", cnyCents: 1, wantQuota: 5_000},
		{name: "任意整数分", cnyCents: 12_345, wantQuota: 61_725_000},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			quota, err := paymentQuota(test.cnyCents)
			if err != nil {
				t.Fatalf("计算支付金额失败：%v", err)
			}
			if quota != test.wantQuota {
				t.Fatalf("人民币分=%d，额度=%d", test.cnyCents, quota)
			}
		})
	}
}

func TestPaymentAmountRejectsInvalidOrOverflowingValues(t *testing.T) {
	for _, input := range []int64{0, -1, 1 << 62} {
		if _, err := paymentQuota(input); err == nil {
			t.Fatalf("应拒绝 amount_cny_cents=%d", input)
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
