package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const settlementPageSize = 100

type SettlementApplyResult struct {
	SourceID int64  `json:"source_id"`
	Status   string `json:"status"`
}

type SettlementLogRepository interface {
	LastSettlementSync(context.Context) (int64, error)
	RetrySettlementReconciliation(context.Context, int) (int, error)
	ApplySettlementLog(context.Context, SettlementLog) (SettlementApplyResult, error)
	CompleteSettlementSync(context.Context, int64, time.Time) error
	RecordSettlementSyncError(context.Context, string) error
}

type SettlementSyncRunner struct {
	repository SettlementLogRepository
	admin      AdminNewAPI
}

func NewSettlementSyncRunner(repository SettlementLogRepository, admin AdminNewAPI) *SettlementSyncRunner {
	return &SettlementSyncRunner{repository: repository, admin: admin}
}

func (r *SettlementSyncRunner) RunCycle(ctx context.Context, now time.Time) error {
	if r.repository == nil || r.admin == nil {
		return errors.New("结算同步服务未初始化")
	}
	lastSynced, err := r.repository.LastSettlementSync(ctx)
	if err != nil {
		return r.fail(ctx, fmt.Errorf("读取结算同步水位：%w", err))
	}
	if _, err := r.repository.RetrySettlementReconciliation(ctx, 100); err != nil {
		return r.fail(ctx, fmt.Errorf("重试结算对账项：%w", err))
	}
	start, end := settlementWindow(lastSynced, now)
	for _, logType := range []int{2, 6} {
		if err := r.fetchType(ctx, logType, start, end); err != nil {
			return r.fail(ctx, err)
		}
	}
	if err := r.repository.CompleteSettlementSync(ctx, end, now.UTC()); err != nil {
		return r.fail(ctx, fmt.Errorf("推进结算同步水位：%w", err))
	}
	return nil
}

func (r *SettlementSyncRunner) fetchType(ctx context.Context, logType int, start int64, end int64) error {
	for pageNumber := 1; ; pageNumber++ {
		query := url.Values{}
		query.Set("p", strconv.Itoa(pageNumber))
		query.Set("page_size", strconv.Itoa(settlementPageSize))
		query.Set("type", strconv.Itoa(logType))
		query.Set("start_timestamp", strconv.FormatInt(start, 10))
		query.Set("end_timestamp", strconv.FormatInt(end, 10))
		data, err := r.admin.Do(ctx, http.MethodGet, "/api/log/?"+query.Encode(), nil)
		if err != nil {
			return fmt.Errorf("拉取 New API 日志 type=%d page=%d：%w", logType, pageNumber, err)
		}
		page, err := decodeSettlementLogPage(data)
		if err != nil {
			return fmt.Errorf("解析 New API 日志 type=%d page=%d：%w", logType, pageNumber, err)
		}
		for _, item := range page.Items {
			if item.ID <= 0 || item.UserID <= 0 || item.Type != logType || item.ModelName == "" || item.ChannelID <= 0 || item.Quota <= 0 {
				return fmt.Errorf("New API 日志字段无效：id=%d type=%d", item.ID, item.Type)
			}
			if _, err := r.repository.ApplySettlementLog(ctx, item); err != nil {
				return fmt.Errorf("应用 New API 日志 id=%d：%w", item.ID, err)
			}
		}
		if pageNumber*settlementPageSize >= page.Total {
			return nil
		}
		if len(page.Items) == 0 {
			return fmt.Errorf("New API 日志分页提前结束：type=%d page=%d total=%d", logType, pageNumber, page.Total)
		}
	}
}

func (r *SettlementSyncRunner) fail(ctx context.Context, err error) error {
	message := err.Error()
	if len(message) > 500 {
		message = message[:500]
	}
	_ = r.repository.RecordSettlementSyncError(ctx, message)
	return err
}
