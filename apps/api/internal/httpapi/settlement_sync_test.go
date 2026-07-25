package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

type settlementLogRepositoryStub struct {
	lastSynced   int64
	retried      bool
	applied      []SettlementLog
	completedAt  int64
	failure      string
	applyResults []SettlementApplyResult
}

func (s *settlementLogRepositoryStub) LastSettlementSync(context.Context) (int64, error) {
	return s.lastSynced, nil
}

func (s *settlementLogRepositoryStub) RetrySettlementReconciliation(context.Context, int) (int, error) {
	s.retried = true
	return 0, nil
}

func (s *settlementLogRepositoryStub) ApplySettlementLog(_ context.Context, item SettlementLog) (SettlementApplyResult, error) {
	s.applied = append(s.applied, item)
	if len(s.applyResults) >= len(s.applied) {
		return s.applyResults[len(s.applied)-1], nil
	}
	return SettlementApplyResult{Status: "settled"}, nil
}

func (s *settlementLogRepositoryStub) CompleteSettlementSync(_ context.Context, through int64, _ time.Time) error {
	s.completedAt = through
	return nil
}

func (s *settlementLogRepositoryStub) RecordSettlementSyncError(_ context.Context, message string) error {
	s.failure = message
	return nil
}

type settlementAdminStub struct {
	do func(context.Context, string, string, any) ([]byte, error)
}

func (s settlementAdminStub) Do(ctx context.Context, method string, path string, body any) ([]byte, error) {
	return s.do(ctx, method, path, body)
}

func TestSettlementSyncUsesOverlapPaginationAndFinalQuota(t *testing.T) {
	repository := &settlementLogRepositoryStub{lastSynced: 1_784_692_800}
	requests := make([]url.Values, 0)
	admin := settlementAdminStub{do: func(_ context.Context, method string, path string, _ any) ([]byte, error) {
		if method != http.MethodGet {
			t.Fatalf("New API 方法=%s", method)
		}
		parsed, err := url.Parse(path)
		if err != nil || parsed.Path != "/api/log/" {
			t.Fatalf("New API 日志路径=%q err=%v", path, err)
		}
		query := parsed.Query()
		requests = append(requests, query)
		logType, _ := strconv.Atoi(query.Get("type"))
		page, _ := strconv.Atoi(query.Get("p"))
		if logType == 2 && page == 1 {
			items := make([]map[string]any, 100)
			for index := range items {
				items[index] = map[string]any{
					"id": 200 - index, "user_id": 7, "created_at": 1_784_693_000,
					"type": 2, "model_name": "mkt-ov-42", "quota": 37,
					"channel": 41, "token_id": 9, "request_id": "req-cache",
					"other": `{"cache_tokens":1200}`,
				}
			}
			return json.Marshal(map[string]any{"page": 1, "page_size": 100, "total": 101, "items": items})
		}
		if logType == 2 && page == 2 {
			return json.Marshal(map[string]any{"page": 2, "page_size": 100, "total": 101, "items": []map[string]any{{
				"id": 100, "user_id": 7, "created_at": 1_784_693_001, "type": 2,
				"model_name": "mkt-ov-42", "quota": 19, "channel": 41, "other": `{}`,
			}}})
		}
		return json.Marshal(map[string]any{"page": 1, "page_size": 100, "total": 0, "items": []any{}})
	}}
	runner := NewSettlementSyncRunner(repository, admin)
	now := time.Unix(1_784_696_400, 0).UTC()
	if err := runner.RunCycle(context.Background(), now); err != nil {
		t.Fatalf("同步周期失败：%v", err)
	}
	if !repository.retried || len(repository.applied) != 101 {
		t.Fatalf("对账重试=%v，应用日志=%d", repository.retried, len(repository.applied))
	}
	if repository.applied[0].Quota != 37 || repository.applied[0].CacheTokens != 1200 {
		t.Fatalf("缓存日志没有保留 New API 最终额度：%+v", repository.applied[0])
	}
	if repository.completedAt != now.Add(-5*time.Second).Unix() || repository.failure != "" {
		t.Fatalf("完成水位=%d，错误=%q", repository.completedAt, repository.failure)
	}
	if len(requests) != 3 {
		t.Fatalf("分页请求数量=%d", len(requests))
	}
	for _, query := range requests {
		if query.Get("start_timestamp") != "1784692500" || query.Get("end_timestamp") != "1784696395" || query.Get("page_size") != "100" {
			t.Fatalf("重叠窗口查询=%v", query)
		}
	}
}

func TestSettlementSyncDoesNotAdvanceWatermarkAfterFetchFailure(t *testing.T) {
	repository := &settlementLogRepositoryStub{lastSynced: 1_784_692_800}
	runner := NewSettlementSyncRunner(repository, settlementAdminStub{do: func(context.Context, string, string, any) ([]byte, error) {
		return nil, errors.New("upstream unavailable")
	}})
	if err := runner.RunCycle(context.Background(), time.Unix(1_784_696_400, 0)); err == nil {
		t.Fatal("日志拉取失败时同步周期必须失败")
	}
	if repository.completedAt != 0 || repository.failure == "" {
		t.Fatalf("失败后水位=%d，错误=%q", repository.completedAt, repository.failure)
	}
}

func TestSettlementSyncSkipsFailedConsumptionLog(t *testing.T) {
	repository := &settlementLogRepositoryStub{lastSynced: 1_784_692_800}
	runner := NewSettlementSyncRunner(repository, settlementAdminStub{do: func(_ context.Context, _ string, path string, _ any) ([]byte, error) {
		query, _ := url.Parse(path)
		if query.Query().Get("type") == "2" {
			return json.Marshal(map[string]any{"page": 1, "page_size": 100, "total": 2, "items": []map[string]any{
				{"id": 262, "user_id": 7, "created_at": 1_784_693_000, "type": 2, "model_name": "gpt-5.6", "quota": 0, "channel": 41},
				{"id": 263, "user_id": 7, "created_at": 1_784_693_001, "type": 2, "model_name": "gpt-5.6", "quota": 37, "channel": 41},
			}})
		}
		return json.Marshal(map[string]any{"page": 1, "page_size": 100, "total": 0, "items": []any{}})
	}})
	if err := runner.RunCycle(context.Background(), time.Unix(1_784_696_400, 0)); err != nil {
		t.Fatalf("失败消费日志不应阻塞同步：%v", err)
	}
	if len(repository.applied) != 1 || repository.applied[0].ID != 263 || repository.failure != "" {
		t.Fatalf("跳过失败日志后的同步结果=%+v 错误=%q", repository.applied, repository.failure)
	}
}
