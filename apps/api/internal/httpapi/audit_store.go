package httpapi

import (
	"context"
	"database/sql"
	"fmt"
)

type SQLAuditStore struct {
	database *sql.DB
}

func NewSQLAuditStore(database *sql.DB) *SQLAuditStore {
	return &SQLAuditStore{database: database}
}

func (s *SQLAuditStore) Record(ctx context.Context, entry AuditEntry) error {
	_, err := s.database.ExecContext(ctx, `
INSERT INTO platform_audit_logs
    (actor_newapi_user_id, actor_role, action, object_type, object_id, result, reason, request_id, source_ip)
VALUES
    ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, '')::inet)
`, entry.ActorUserID, entry.ActorRole, entry.Action, entry.ObjectType, entry.ObjectID, entry.Result, entry.Reason, entry.RequestID, entry.SourceIP)
	if err != nil {
		return fmt.Errorf("写入平台审计：%w", err)
	}
	return nil
}

func (s *SQLAuditStore) List(ctx context.Context, page int, pageSize int) ([]AuditEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var total int
	if err := s.database.QueryRowContext(ctx, "SELECT count(*) FROM platform_audit_logs").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计平台审计：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, `
SELECT id, actor_newapi_user_id, actor_role, action, object_type,
       object_id, result, reason, request_id, host(source_ip), created_at
FROM platform_audit_logs
ORDER BY id DESC
LIMIT $1 OFFSET $2
`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询平台审计：%w", err)
	}
	defer rows.Close()
	entries := make([]AuditEntry, 0)
	for rows.Next() {
		var entry AuditEntry
		var actorID sql.NullInt64
		var objectID, reason, requestID, sourceIP sql.NullString
		if err := rows.Scan(
			&entry.ID, &actorID, &entry.ActorRole, &entry.Action, &entry.ObjectType,
			&objectID, &entry.Result, &reason, &requestID, &sourceIP, &entry.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("解析平台审计：%w", err)
		}
		if actorID.Valid {
			entry.ActorUserID = int(actorID.Int64)
		}
		entry.ObjectID = objectID.String
		entry.Reason = reason.String
		entry.RequestID = requestID.String
		entry.SourceIP = sourceIP.String
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历平台审计：%w", err)
	}
	return entries, total, nil
}
