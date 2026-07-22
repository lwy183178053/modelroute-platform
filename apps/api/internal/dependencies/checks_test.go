package dependencies

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAPIPingerRequiresSuccessfulStatusPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	pinger := NewHTTPPinger(server.URL+"/api/status", time.Second)
	if err := pinger.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() 返回错误：%v", err)
	}
}

func TestNewAPIPingerRejectsUnhealthyPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false}`))
	}))
	defer server.Close()

	pinger := NewHTTPPinger(server.URL+"/api/status", time.Second)
	if err := pinger.Ping(context.Background()); err == nil {
		t.Fatal("success=false 时应返回错误")
	}
}
