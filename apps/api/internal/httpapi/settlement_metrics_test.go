package httpapi

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRecordSettlementRouteMetricUsesFinalPromptAndCacheTokens(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	tx, err := database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, 7, 23, 10, 36, 0, 0, time.UTC)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO route_metric_buckets")).
		WithArgs(int64(81), createdAt.Truncate(time.Hour), int64(2_400), int64(1_200)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := recordSettlementRouteMetric(context.Background(), tx, 81, SettlementLog{
		CreatedAt: createdAt.Unix(), PromptTokens: 2_400, CacheTokens: 1_200,
	}); err != nil {
		t.Fatalf("写入消费缓存指标失败：%v", err)
	}
	_ = tx.Rollback()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
