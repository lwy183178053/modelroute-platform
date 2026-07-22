package httpapi

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/migrations"
)

func TestSQLAnnouncementStoreVisibilityAndReadIdempotency(t *testing.T) {
	databaseURL := os.Getenv("MODELROUTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("未设置 PostgreSQL 集成测试地址")
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	ctx := context.Background()
	if err := migrations.Up(ctx, database); err != nil {
		t.Fatalf("执行迁移：%v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE announcement_reads, announcements RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 22, 16, 0, 0, 0, time.UTC)
	if _, err := database.ExecContext(ctx, `
INSERT INTO announcements (title, body, audience_mask, pinned, publish_at, expires_at, status, created_by_newapi_user_id, updated_by_newapi_user_id)
VALUES
 ('普通用户公告', '用户正文', 1, false, $1::timestamptz - interval '1 hour', NULL, 'published', 1, 1),
 ('商户公告', '商户正文', 2, true, $1::timestamptz - interval '1 hour', NULL, 'published', 1, 1),
 ('定时公告', '定时正文', 2, false, $1::timestamptz + interval '1 hour', NULL, 'scheduled', 1, 1),
 ('过期公告', '过期正文', 2, false, $1::timestamptz - interval '2 hour', $1::timestamptz - interval '1 minute', 'published', 1, 1)
`, now); err != nil {
		t.Fatal(err)
	}
	store := NewSQLAnnouncementStore(database)
	feed, err := store.Visible(ctx, 7, announcementAudienceMerchant, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) != 1 || feed.Items[0].Title != "商户公告" || feed.UnreadCount != 1 {
		t.Fatalf("商户公告列表=%+v", feed)
	}
	if err := store.MarkRead(ctx, feed.Items[0].ID, 7, announcementAudienceMerchant, now); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkRead(ctx, feed.Items[0].ID, 7, announcementAudienceMerchant, now); err != nil {
		t.Fatalf("重复已读应幂等：%v", err)
	}
	feed, err = store.Visible(ctx, 7, announcementAudienceMerchant, now)
	if err != nil || feed.UnreadCount != 0 || !feed.Items[0].IsRead {
		t.Fatalf("已读后公告列表=%+v 错误=%v", feed, err)
	}
}
