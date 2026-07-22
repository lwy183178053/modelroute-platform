package platformauth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestStore(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewStore(client, time.Hour), server
}

func TestCreateStoresOnlyHashedBrowserToken(t *testing.T) {
	store, redisServer := newTestStore(t)
	token, err := store.Create(context.Background(), Session{
		UserID:         7,
		Username:       "alice",
		Role:           1,
		Status:         1,
		UpstreamCookie: "session=upstream-secret",
	})
	if err != nil {
		t.Fatalf("创建会话失败：%v", err)
	}
	if len(token) < 40 {
		t.Fatalf("浏览器令牌长度不足：%d", len(token))
	}
	for _, key := range redisServer.Keys() {
		if strings.Contains(key, token) {
			t.Fatalf("Redis 键泄露浏览器令牌：%s", key)
		}
	}

	session, err := store.Get(context.Background(), token)
	if err != nil {
		t.Fatalf("读取会话失败：%v", err)
	}
	if session.UserID != 7 || session.UpstreamCookie != "session=upstream-secret" {
		t.Fatalf("会话内容不符合预期：%+v", session)
	}
}

func TestDeleteUserSessionsInvalidatesEveryBrowserSession(t *testing.T) {
	store, _ := newTestStore(t)
	first, err := store.Create(context.Background(), Session{UserID: 7, Username: "alice"})
	if err != nil {
		t.Fatalf("创建首个会话失败：%v", err)
	}
	second, err := store.Create(context.Background(), Session{UserID: 7, Username: "alice"})
	if err != nil {
		t.Fatalf("创建第二个会话失败：%v", err)
	}

	if err := store.DeleteUserSessions(context.Background(), 7); err != nil {
		t.Fatalf("清理用户会话失败：%v", err)
	}
	for _, token := range []string{first, second} {
		if _, err := store.Get(context.Background(), token); !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("会话仍然有效：%q，错误=%v", token, err)
		}
	}
}

func TestSavePromotesPendingTwoFactorSessionIntoUserIndex(t *testing.T) {
	store, _ := newTestStore(t)
	token, err := store.Create(context.Background(), Session{
		PendingTwoFactor: true,
		UpstreamCookie:   "session=pending",
	})
	if err != nil {
		t.Fatalf("创建待验证会话失败：%v", err)
	}
	if err := store.Save(context.Background(), token, Session{
		UserID:         9,
		Username:       "bob",
		Role:           1,
		Status:         1,
		UpstreamCookie: "session=ready",
	}); err != nil {
		t.Fatalf("更新会话失败：%v", err)
	}
	if err := store.DeleteUserSessions(context.Background(), 9); err != nil {
		t.Fatalf("清理用户会话失败：%v", err)
	}
	if _, err := store.Get(context.Background(), token); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("升级后的会话未进入用户索引：%v", err)
	}
}
