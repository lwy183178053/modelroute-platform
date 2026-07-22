package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SQLAnnouncementStore struct {
	database *sql.DB
}

func NewSQLAnnouncementStore(database *sql.DB) *SQLAnnouncementStore {
	return &SQLAnnouncementStore{database: database}
}

func (s *SQLAnnouncementStore) Visible(ctx context.Context, userID int, audienceMask int, now time.Time) (AnnouncementFeed, error) {
	rows, err := s.database.QueryContext(ctx, `
SELECT a.id, a.title, a.body, a.audience_mask, a.pinned, a.publish_at, a.expires_at,
       a.status, (r.announcement_id IS NOT NULL), a.created_at, a.updated_at
FROM announcements a
LEFT JOIN announcement_reads r
  ON r.announcement_id = a.id AND r.newapi_user_id = $1
WHERE (a.audience_mask & $2) <> 0
  AND a.status IN ('scheduled', 'published')
  AND a.publish_at <= $3
  AND (a.expires_at IS NULL OR a.expires_at > $3)
ORDER BY a.pinned DESC, a.publish_at DESC, a.id DESC
`, userID, audienceMask, now)
	if err != nil {
		return AnnouncementFeed{}, fmt.Errorf("查询有效公告：%w", err)
	}
	defer rows.Close()
	feed := AnnouncementFeed{Items: make([]Announcement, 0)}
	for rows.Next() {
		item, err := scanAnnouncement(rows)
		if err != nil {
			return AnnouncementFeed{}, err
		}
		if !item.IsRead {
			feed.UnreadCount++
		}
		feed.Items = append(feed.Items, item)
	}
	if err := rows.Err(); err != nil {
		return AnnouncementFeed{}, fmt.Errorf("遍历有效公告：%w", err)
	}
	return feed, nil
}

func (s *SQLAnnouncementStore) MarkRead(ctx context.Context, announcementID int64, userID int, audienceMask int, now time.Time) error {
	var returnedID int64
	err := s.database.QueryRowContext(ctx, `
WITH visible AS (
    SELECT id
    FROM announcements
    WHERE id = $1
      AND (audience_mask & $3) <> 0
      AND status IN ('scheduled', 'published')
      AND publish_at <= $4
      AND (expires_at IS NULL OR expires_at > $4)
)
INSERT INTO announcement_reads (announcement_id, newapi_user_id)
SELECT id, $2 FROM visible
ON CONFLICT (announcement_id, newapi_user_id)
DO UPDATE SET read_at = announcement_reads.read_at
RETURNING announcement_id
`, announcementID, userID, audienceMask, now).Scan(&returnedID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAnnouncementNotFound
	}
	if err != nil {
		return fmt.Errorf("标记公告已读：%w", err)
	}
	return nil
}

func (s *SQLAnnouncementStore) List(ctx context.Context) ([]Announcement, error) {
	rows, err := s.database.QueryContext(ctx, `
SELECT id, title, body, audience_mask, pinned, publish_at, expires_at,
       status, false, created_at, updated_at
FROM announcements
ORDER BY pinned DESC, created_at DESC, id DESC
`)
	if err != nil {
		return nil, fmt.Errorf("查询公告：%w", err)
	}
	defer rows.Close()
	items := make([]Announcement, 0)
	for rows.Next() {
		item, err := scanAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历公告：%w", err)
	}
	return items, nil
}

func (s *SQLAnnouncementStore) Create(ctx context.Context, command AnnouncementCommand) (Announcement, error) {
	return scanAnnouncement(s.database.QueryRowContext(ctx, `
INSERT INTO announcements
    (title, body, audience_mask, pinned, publish_at, expires_at, status,
     created_by_newapi_user_id, updated_by_newapi_user_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
RETURNING id, title, body, audience_mask, pinned, publish_at, expires_at,
          status, false, created_at, updated_at
`, command.Title, command.Body, command.AudienceMask, command.Pinned, command.PublishAt,
		command.ExpiresAt, command.Status, command.ActorUserID))
}

func (s *SQLAnnouncementStore) Update(ctx context.Context, command AnnouncementCommand) (Announcement, error) {
	item, err := scanAnnouncement(s.database.QueryRowContext(ctx, `
UPDATE announcements
SET title = $1, body = $2, audience_mask = $3, pinned = $4,
    publish_at = $5, expires_at = $6, status = $7,
    updated_by_newapi_user_id = $8, updated_at = now()
WHERE id = $9 AND status IN ('draft', 'scheduled')
RETURNING id, title, body, audience_mask, pinned, publish_at, expires_at,
          status, false, created_at, updated_at
`, command.Title, command.Body, command.AudienceMask, command.Pinned, command.PublishAt,
		command.ExpiresAt, command.Status, command.ActorUserID, command.ID))
	if errors.Is(err, ErrAnnouncementNotFound) {
		return Announcement{}, ErrAnnouncementConflict
	}
	return item, err
}

func (s *SQLAnnouncementStore) Publish(ctx context.Context, id int64, actorUserID int, now time.Time) (Announcement, error) {
	item, err := scanAnnouncement(s.database.QueryRowContext(ctx, `
UPDATE announcements
SET status = CASE WHEN publish_at > $2 THEN 'scheduled' ELSE 'published' END,
    publish_at = CASE WHEN publish_at IS NULL OR publish_at <= $2 THEN $2 ELSE publish_at END,
    updated_by_newapi_user_id = $3, updated_at = now()
WHERE id = $1 AND status IN ('draft', 'scheduled')
RETURNING id, title, body, audience_mask, pinned, publish_at, expires_at,
          status, false, created_at, updated_at
`, id, now, actorUserID))
	if errors.Is(err, ErrAnnouncementNotFound) {
		return Announcement{}, ErrAnnouncementConflict
	}
	return item, err
}

func (s *SQLAnnouncementStore) Withdraw(ctx context.Context, id int64, actorUserID int) (Announcement, error) {
	item, err := scanAnnouncement(s.database.QueryRowContext(ctx, `
UPDATE announcements
SET status = 'withdrawn', updated_by_newapi_user_id = $2, updated_at = now()
WHERE id = $1 AND status IN ('scheduled', 'published')
RETURNING id, title, body, audience_mask, pinned, publish_at, expires_at,
          status, false, created_at, updated_at
`, id, actorUserID))
	if errors.Is(err, ErrAnnouncementNotFound) {
		return Announcement{}, ErrAnnouncementConflict
	}
	return item, err
}

type announcementScanner interface {
	Scan(...any) error
}

func scanAnnouncement(scanner announcementScanner) (Announcement, error) {
	var item Announcement
	var publishAt, expiresAt sql.NullTime
	if err := scanner.Scan(
		&item.ID, &item.Title, &item.Body, &item.AudienceMask, &item.Pinned,
		&publishAt, &expiresAt, &item.Status, &item.IsRead, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Announcement{}, ErrAnnouncementNotFound
		}
		return Announcement{}, fmt.Errorf("解析公告：%w", err)
	}
	if publishAt.Valid {
		value := publishAt.Time.UTC()
		item.PublishAt = &value
	}
	if expiresAt.Valid {
		value := expiresAt.Time.UTC()
		item.ExpiresAt = &value
	}
	item.Audiences = announcementAudiences(item.AudienceMask)
	return item, nil
}
