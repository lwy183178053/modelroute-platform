package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type channelDetectorAdminStub struct {
	calls        []string
	createdTypes []int
	searchCount  int
}

func (s *channelDetectorAdminStub) Do(_ context.Context, method string, path string, body any) ([]byte, error) {
	s.calls = append(s.calls, method+" "+strings.Split(path, "?")[0])
	switch {
	case method == http.MethodPost && path == "/api/channel/":
		encoded, _ := json.Marshal(body)
		var request struct {
			Channel struct {
				Type   int    `json:"type"`
				Models string `json:"models"`
			} `json:"channel"`
		}
		_ = json.Unmarshal(encoded, &request)
		if request.Channel.Models != "claude-fable-5" {
			return nil, fmt.Errorf("模型=%q", request.Channel.Models)
		}
		s.createdTypes = append(s.createdTypes, request.Channel.Type)
		return []byte(`null`), nil
	case method == http.MethodGet && strings.HasPrefix(path, "/api/channel/search?"):
		s.searchCount++
		return []byte(fmt.Sprintf(`{"items":[{"id":%d}]}`, 40+s.searchCount)), nil
	case method == http.MethodGet && strings.HasPrefix(path, "/api/channel/test/41"):
		return nil, errors.New("原生适配器发送 hi 失败")
	case method == http.MethodGet && strings.HasPrefix(path, "/api/channel/test/42"):
		return []byte(`null`), nil
	case method == http.MethodDelete && (path == "/api/channel/41" || path == "/api/channel/42"):
		return []byte(`null`), nil
	default:
		return nil, fmt.Errorf("未知调用 %s %s", method, path)
	}
}

func TestNewAPIChannelDetectorFallsBackAndAlwaysDeletesTemporaryChannels(t *testing.T) {
	admin := &channelDetectorAdminStub{}
	detector := NewNewAPIChannelDetector(admin)
	outcome, err := detector.Test(context.Background(), ChannelTestRequest{
		Model:   CanonicalModel{ID: 61, PublicName: "claude-fable-5", Provider: "Anthropic", Protocol: "responses"},
		BaseURL: "https://api.merchant.example", APIKey: "sk-upstream-secret",
	})
	if err != nil {
		t.Fatalf("检测失败：%v", err)
	}
	if outcome.AdapterType != "openai_compatible" || len(outcome.Capabilities) != 1 || outcome.Capabilities[0] != "responses" {
		t.Fatalf("检测结果=%+v", outcome)
	}
	if fmt.Sprint(admin.createdTypes) != "[14 1]" {
		t.Fatalf("适配器尝试顺序=%v", admin.createdTypes)
	}
	wantCalls := "POST /api/channel/|GET /api/channel/search|GET /api/channel/test/41|DELETE /api/channel/41|POST /api/channel/|GET /api/channel/search|GET /api/channel/test/42|DELETE /api/channel/42"
	if strings.Join(admin.calls, "|") != wantCalls {
		t.Fatalf("临时渠道调用=%v", admin.calls)
	}
}
