package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type announcementRepositoryStub struct {
	AnnouncementRepository
	visible func(context.Context, int, int, time.Time) (AnnouncementFeed, error)
	read    func(context.Context, int64, int, int, time.Time) error
	create  func(context.Context, AnnouncementCommand) (Announcement, error)
}

func (s *announcementRepositoryStub) Visible(ctx context.Context, userID int, audienceMask int, now time.Time) (AnnouncementFeed, error) {
	return s.visible(ctx, userID, audienceMask, now)
}

func (s *announcementRepositoryStub) MarkRead(ctx context.Context, announcementID int64, userID int, audienceMask int, now time.Time) error {
	return s.read(ctx, announcementID, userID, audienceMask, now)
}

func (s *announcementRepositoryStub) Create(ctx context.Context, command AnnouncementCommand) (Announcement, error) {
	return s.create(ctx, command)
}

func TestAnnouncementFeedUsesExclusiveMerchantAudience(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.Merchants = &merchantRepositoryStub{identity: func(_ context.Context, userID int) (MerchantIdentity, error) {
		return MerchantIdentity{MerchantID: 31, UserID: userID, Status: "active", IsMerchant: true}, nil
	}}
	services.Announcements = &announcementRepositoryStub{visible: func(_ context.Context, userID int, audienceMask int, _ time.Time) (AnnouncementFeed, error) {
		if userID != 7 || audienceMask != announcementAudienceMerchant {
			t.Fatalf("公告受众 user=%d mask=%d", userID, audienceMask)
		}
		return AnnouncementFeed{Items: []Announcement{{ID: 9, Title: "商户公告", Body: "渠道结算安排"}}, UnreadCount: 1}, nil
	}}
	token := createPlatformSession(t, services, 7, roleCommon)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"unread_count":1`) || !strings.Contains(response.Body.String(), "商户公告") {
		t.Fatalf("公告响应=%d %s", response.Code, response.Body.String())
	}
}

func TestAdministratorAnnouncementAudienceDoesNotFallThroughToMerchant(t *testing.T) {
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.Merchants = &merchantRepositoryStub{identity: func(context.Context, int) (MerchantIdentity, error) {
		t.Fatal("管理员公告受众不应查询商户身份")
		return MerchantIdentity{}, nil
	}}
	services.Announcements = &announcementRepositoryStub{visible: func(_ context.Context, _ int, audienceMask int, _ time.Time) (AnnouncementFeed, error) {
		if audienceMask != announcementAudienceAdmin {
			t.Fatalf("管理员公告受众=%d", audienceMask)
		}
		return AnnouncementFeed{}, nil
	}}
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	request := httptest.NewRequest(http.MethodGet, "/api/announcements", nil)
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("公告响应=%d %s", response.Code, response.Body.String())
	}
}

func TestAdministratorCreatesScheduledAnnouncementWithAudienceMask(t *testing.T) {
	future := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	services, _, _ := newHTTPTestServices(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	services.Announcements = &announcementRepositoryStub{create: func(_ context.Context, command AnnouncementCommand) (Announcement, error) {
		if command.AudienceMask != announcementAudienceUser|announcementAudienceMerchant || command.Status != "scheduled" || command.PublishAt == nil || !command.PublishAt.Equal(future) {
			t.Fatalf("公告创建参数=%+v", command)
		}
		return Announcement{ID: 12, Title: command.Title, Body: command.Body, Status: command.Status}, nil
	}}
	token := createPlatformSession(t, services, 1, 100)
	handler := NewServerWithServices("test", Dependencies{}, services)
	body := `{"title":"维护安排","body":"今晚升级","audiences":["user","merchant"],"pinned":true,"publish_at":"` + future.Format(time.RFC3339) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/admin/announcements", strings.NewReader(body))
	request.Header.Set("Origin", "https://aggregate.lwylink.xyz")
	request.AddCookie(&http.Cookie{Name: "modelroute_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"id":12`) {
		t.Fatalf("公告创建响应=%d %s", response.Code, response.Body.String())
	}
}
