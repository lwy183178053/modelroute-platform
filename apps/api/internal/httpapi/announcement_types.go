package httpapi

import (
	"context"
	"errors"
	"time"
)

const (
	announcementAudienceUser     = 1
	announcementAudienceMerchant = 2
	announcementAudienceAdmin    = 4
)

var (
	ErrAnnouncementNotFound = errors.New("announcement not found")
	ErrAnnouncementConflict = errors.New("announcement state conflict")
)

type Announcement struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	AudienceMask int        `json:"-"`
	Audiences    []string   `json:"audiences"`
	Pinned       bool       `json:"pinned"`
	PublishAt    *time.Time `json:"publish_at,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Status       string     `json:"status"`
	IsRead       bool       `json:"is_read"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type AnnouncementFeed struct {
	Items       []Announcement `json:"items"`
	UnreadCount int            `json:"unread_count"`
}

type AnnouncementCommand struct {
	ID           int64
	Title        string
	Body         string
	AudienceMask int
	Pinned       bool
	PublishAt    *time.Time
	ExpiresAt    *time.Time
	Status       string
	ActorUserID  int
}

type AnnouncementRepository interface {
	Visible(context.Context, int, int, time.Time) (AnnouncementFeed, error)
	MarkRead(context.Context, int64, int, int, time.Time) error
	List(context.Context) ([]Announcement, error)
	Create(context.Context, AnnouncementCommand) (Announcement, error)
	Update(context.Context, AnnouncementCommand) (Announcement, error)
	Publish(context.Context, int64, int, time.Time) (Announcement, error)
	Withdraw(context.Context, int64, int) (Announcement, error)
}

func announcementAudiences(mask int) []string {
	items := make([]string, 0, 3)
	if mask&announcementAudienceUser != 0 {
		items = append(items, "user")
	}
	if mask&announcementAudienceMerchant != 0 {
		items = append(items, "merchant")
	}
	if mask&announcementAudienceAdmin != 0 {
		items = append(items, "admin")
	}
	return items
}
