package httpapi

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const newAPIQuotaPerCNYCent int64 = 5000

func paymentQuota(cnyCents int64) (int64, error) {
	if cnyCents <= 0 {
		return 0, errors.New("充值金额必须为正数")
	}
	if cnyCents > math.MaxInt64/newAPIQuotaPerCNYCent {
		return 0, errors.New("New API 额度超出范围")
	}
	return cnyCents * newAPIQuotaPerCNYCent, nil
}

func parseGatewayMoney(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") {
		return 0, errors.New("支付金额格式无效")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || (len(parts) == 2 && (len(parts[1]) == 0 || len(parts[1]) > 2)) {
		return 0, errors.New("支付金额格式无效")
	}
	for _, part := range parts {
		for _, char := range part {
			if char < '0' || char > '9' {
				return 0, errors.New("支付金额格式无效")
			}
		}
	}
	yuan, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || yuan > math.MaxInt64/100 {
		return 0, errors.New("支付金额超出范围")
	}
	fraction := int64(0)
	if len(parts) == 2 {
		fraction, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, errors.New("支付金额格式无效")
		}
		if len(parts[1]) == 1 {
			fraction *= 10
		}
	}
	cents := yuan*100 + fraction
	if cents <= 0 {
		return 0, errors.New("支付金额必须为正数")
	}
	return cents, nil
}

func easyPayCanonical(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == "sign" || key == "sign_type" || values.Get(key) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	return strings.Join(parts, "&")
}

func easyPaySign(values url.Values, key string) string {
	digest := md5.Sum([]byte(easyPayCanonical(values) + key))
	return hex.EncodeToString(digest[:])
}

func verifyEasyPaySign(values url.Values, key string, signature string) bool {
	expected, err := hex.DecodeString(easyPaySign(values, key))
	if err != nil {
		return false
	}
	provided, err := hex.DecodeString(strings.ToLower(strings.TrimSpace(signature)))
	if err != nil || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare(expected, provided) == 1
}
