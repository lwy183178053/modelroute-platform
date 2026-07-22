package httpapi

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSQLAuditStoreRecordsAndListsEntries(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建模拟数据库失败：%v", err)
	}
	defer database.Close()
	store := NewSQLAuditStore(database)
	entry := AuditEntry{
		ActorUserID: 1,
		ActorRole:   "100",
		Action:      "user.disable",
		ObjectType:  "user",
		ObjectID:    "7",
		Result:      "success",
		RequestID:   "req-1",
		SourceIP:    "127.0.0.1",
	}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO platform_audit_logs")).
		WithArgs(1, "100", "user.disable", "user", "7", "success", "", "req-1", "127.0.0.1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := store.Record(context.Background(), entry); err != nil {
		t.Fatalf("写审计失败：%v", err)
	}

	createdAt := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM platform_audit_logs")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, actor_newapi_user_id, actor_role, action, object_type,")).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "actor_newapi_user_id", "actor_role", "action", "object_type", "object_id",
			"result", "reason", "request_id", "source_ip", "created_at",
		}).AddRow(3, 1, "100", "user.disable", "user", "7", "success", "", "req-1", "127.0.0.1", createdAt))
	entries, total, err := store.List(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("读取审计失败：%v", err)
	}
	if total != 1 || len(entries) != 1 || entries[0].ID != 3 || entries[0].CreatedAt != createdAt {
		t.Fatalf("审计列表不符合预期：total=%d entries=%+v", total, entries)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 预期未满足：%v", err)
	}
}
