package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
)

func TestReadCoreQuotaPerUnitUsesStatusValue(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/status" {
			t.Fatalf("核心状态请求=%s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"quota_per_unit": 750_000},
		})
	}))
	defer upstream.Close()

	unit, err := ReadCoreQuotaPerUnit(context.Background(), newapi.NewClient(upstream.URL, time.Second))
	if err != nil {
		t.Fatalf("读取核心额度单位失败：%v", err)
	}
	if unit != 750_000 {
		t.Fatalf("额度单位=%d，期望 750000", unit)
	}
}

func TestReadCoreQuotaPerUnitRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"0", "500000.5", "500001"} {
		t.Run(value, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"data":{"quota_per_unit":` + value + `}}`))
			}))
			defer upstream.Close()
			_, err := ReadCoreQuotaPerUnit(context.Background(), newapi.NewClient(upstream.URL, time.Second))
			if err == nil || !strings.Contains(err.Error(), "额度单位") {
				t.Fatalf("值 %s 的错误=%v", value, err)
			}
		})
	}
}
