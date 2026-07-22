package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

type announcementRequest struct {
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Audiences []string   `json:"audiences"`
	Pinned    bool       `json:"pinned"`
	PublishAt *time.Time `json:"publish_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func (s *server) announcements(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.announcementReady(w) {
		return
	}
	audience, ok := s.announcementAudience(w, r, session)
	if !ok {
		return
	}
	feed, err := s.services.Announcements.Visible(r.Context(), session.UserID, audience, time.Now().UTC())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "announcement_store_error", "读取公告失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": feed})
}

func (s *server) readAnnouncement(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok || !s.requireSameOrigin(w, r) || !s.announcementReady(w) {
		return
	}
	audience, ok := s.announcementAudience(w, r, session)
	if !ok {
		return
	}
	id, ok := pathInt64(w, r, "id", "公告编号无效")
	if !ok {
		return
	}
	if err := s.services.Announcements.MarkRead(r.Context(), id, session.UserID, audience, time.Now().UTC()); err != nil {
		if errors.Is(err, ErrAnnouncementNotFound) {
			writeAPIError(w, http.StatusNotFound, "announcement_not_found", "公告不存在或已下架")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "announcement_store_error", "标记公告已读失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nil})
}

func (s *server) adminAnnouncements(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok || !s.announcementReady(w) {
		return
	}
	items, err := s.services.Announcements.List(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "announcement_store_error", "读取公告管理列表失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func (s *server) createAdminAnnouncement(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.announcementReady(w) {
		return
	}
	command, ok := decodeAnnouncementCommand(w, r, session.UserID, 0)
	if !ok {
		return
	}
	item, err := s.services.Announcements.Create(r.Context(), command)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "announcement_store_error", "创建公告失败")
		return
	}
	if err := s.recordAnnouncementAudit(r, session, "announcement.create", item.ID); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "公告已创建，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": item})
}

func (s *server) updateAdminAnnouncement(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.announcementReady(w) {
		return
	}
	id, ok := pathInt64(w, r, "id", "公告编号无效")
	if !ok {
		return
	}
	command, ok := decodeAnnouncementCommand(w, r, session.UserID, id)
	if !ok {
		return
	}
	item, err := s.services.Announcements.Update(r.Context(), command)
	if err != nil {
		s.writeAnnouncementError(w, err, "更新公告失败")
		return
	}
	if err := s.recordAnnouncementAudit(r, session, "announcement.update", item.ID); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "公告已更新，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": item})
}

func (s *server) publishAdminAnnouncement(w http.ResponseWriter, r *http.Request) {
	s.changeAnnouncementStatus(w, r, true)
}

func (s *server) withdrawAdminAnnouncement(w http.ResponseWriter, r *http.Request) {
	s.changeAnnouncementStatus(w, r, false)
}

func (s *server) changeAnnouncementStatus(w http.ResponseWriter, r *http.Request, publish bool) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.announcementReady(w) {
		return
	}
	id, ok := pathInt64(w, r, "id", "公告编号无效")
	if !ok {
		return
	}
	var item Announcement
	var err error
	action := "withdraw"
	if publish {
		action = "publish"
		item, err = s.services.Announcements.Publish(r.Context(), id, session.UserID, time.Now().UTC())
	} else {
		item, err = s.services.Announcements.Withdraw(r.Context(), id, session.UserID)
	}
	if err != nil {
		s.writeAnnouncementError(w, err, "更新公告状态失败")
		return
	}
	if err := s.recordAnnouncementAudit(r, session, "announcement."+action, item.ID); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "audit_failed", "公告状态已更新，但审计记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": item})
}

func decodeAnnouncementCommand(w http.ResponseWriter, r *http.Request, actorUserID int, id int64) (AnnouncementCommand, bool) {
	var request announcementRequest
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "公告内容格式无效")
		return AnnouncementCommand{}, false
	}
	title := strings.TrimSpace(request.Title)
	body := strings.TrimSpace(request.Body)
	mask, valid := announcementAudienceMask(request.Audiences)
	now := time.Now().UTC()
	if title == "" || len([]rune(title)) > 120 || body == "" || len([]rune(body)) > 10000 || !valid {
		writeAPIError(w, http.StatusBadRequest, "invalid_announcement", "请填写有效的公告标题、正文和受众")
		return AnnouncementCommand{}, false
	}
	status := "draft"
	if request.PublishAt != nil {
		published := request.PublishAt.UTC()
		request.PublishAt = &published
		if !published.After(now) {
			writeAPIError(w, http.StatusBadRequest, "invalid_publish_time", "定时发布时间必须晚于当前时间")
			return AnnouncementCommand{}, false
		}
		status = "scheduled"
	}
	if request.ExpiresAt != nil {
		expires := request.ExpiresAt.UTC()
		request.ExpiresAt = &expires
		start := now
		if request.PublishAt != nil {
			start = *request.PublishAt
		}
		if !expires.After(start) {
			writeAPIError(w, http.StatusBadRequest, "invalid_expiry_time", "下架时间必须晚于发布时间")
			return AnnouncementCommand{}, false
		}
	}
	return AnnouncementCommand{
		ID: id, Title: title, Body: body, AudienceMask: mask, Pinned: request.Pinned,
		PublishAt: request.PublishAt, ExpiresAt: request.ExpiresAt, Status: status, ActorUserID: actorUserID,
	}, true
}

func announcementAudienceMask(values []string) (int, bool) {
	mask := 0
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "user":
			mask |= announcementAudienceUser
		case "merchant":
			mask |= announcementAudienceMerchant
		case "admin":
			mask |= announcementAudienceAdmin
		default:
			return 0, false
		}
	}
	return mask, mask != 0
}

func (s *server) announcementAudience(w http.ResponseWriter, r *http.Request, session platformauth.Session) (int, bool) {
	if session.Role >= roleAdmin {
		return announcementAudienceAdmin, true
	}
	if s.services.Merchants == nil {
		return announcementAudienceUser, true
	}
	identity, err := s.services.Merchants.Identity(r.Context(), session.UserID)
	switch {
	case err == nil && identity.IsMerchant && identity.Status == "active":
		return announcementAudienceMerchant, true
	case errors.Is(err, ErrMerchantNotFound), err == nil:
		return announcementAudienceUser, true
	default:
		writeAPIError(w, http.StatusInternalServerError, "merchant_store_error", "读取公告受众失败")
		return 0, false
	}
}

func (s *server) announcementReady(w http.ResponseWriter) bool {
	if s.services.Announcements == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "service_not_ready", "公告服务正在初始化")
		return false
	}
	return true
}

func (s *server) writeAnnouncementError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrAnnouncementNotFound):
		writeAPIError(w, http.StatusNotFound, "announcement_not_found", "公告不存在")
	case errors.Is(err, ErrAnnouncementConflict):
		writeAPIError(w, http.StatusConflict, "announcement_state_conflict", "公告状态已变化，请刷新后重试")
	default:
		writeAPIError(w, http.StatusInternalServerError, "announcement_store_error", fallback)
	}
}

func (s *server) recordAnnouncementAudit(r *http.Request, actor platformauth.Session, action string, announcementID int64) error {
	if s.services.Audit == nil {
		return nil
	}
	return s.services.Audit.Record(r.Context(), AuditEntry{
		ActorUserID: actor.UserID,
		ActorRole:   strconv.Itoa(actor.Role),
		Action:      action,
		ObjectType:  "announcement",
		ObjectID:    strconv.FormatInt(announcementID, 10),
		Result:      "success",
		RequestID:   r.Header.Get("X-Request-ID"),
		SourceIP:    clientIP(r),
	})
}
