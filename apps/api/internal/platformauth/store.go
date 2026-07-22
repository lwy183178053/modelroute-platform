package platformauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrSessionNotFound = errors.New("平台会话不存在")

type Session struct {
	UserID           int    `json:"user_id"`
	Username         string `json:"username"`
	DisplayName      string `json:"display_name,omitempty"`
	Role             int    `json:"role"`
	Status           int    `json:"status"`
	Group            string `json:"group,omitempty"`
	UpstreamCookie   string `json:"upstream_cookie"`
	PendingTwoFactor bool   `json:"pending_two_factor,omitempty"`
}

type Store struct {
	redis *redis.Client
	ttl   time.Duration
}

func NewStore(client *redis.Client, ttl time.Duration) *Store {
	return &Store{redis: client, ttl: ttl}
}

func (s *Store) Create(ctx context.Context, session Session) (string, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("生成平台会话：%w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(random[:])
	if err := s.write(ctx, token, session, 0); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) Get(ctx context.Context, token string) (Session, error) {
	value, err := s.redis.Get(ctx, sessionKey(token)).Bytes()
	if errors.Is(err, redis.Nil) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("读取平台会话：%w", err)
	}
	var session Session
	if err := json.Unmarshal(value, &session); err != nil {
		return Session{}, fmt.Errorf("解析平台会话：%w", err)
	}
	return session, nil
}

func (s *Store) Save(ctx context.Context, token string, session Session) error {
	previous, err := s.Get(ctx, token)
	if err != nil && !errors.Is(err, ErrSessionNotFound) {
		return err
	}
	return s.write(ctx, token, session, previous.UserID)
}

func (s *Store) Delete(ctx context.Context, token string) error {
	session, err := s.Get(ctx, token)
	if errors.Is(err, ErrSessionNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	pipe := s.redis.TxPipeline()
	pipe.Del(ctx, sessionKey(token))
	if session.UserID > 0 {
		pipe.SRem(ctx, userIndexKey(session.UserID), sessionDigest(token))
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID int) error {
	digests, err := s.redis.SMembers(ctx, userIndexKey(userID)).Result()
	if err != nil {
		return fmt.Errorf("读取用户会话索引：%w", err)
	}
	keys := make([]string, 0, len(digests)+1)
	for _, digest := range digests {
		keys = append(keys, "platform:session:"+digest)
	}
	keys = append(keys, userIndexKey(userID))
	if err := s.redis.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("清理用户会话：%w", err)
	}
	return nil
}

func (s *Store) write(ctx context.Context, token string, session Session, previousUserID int) error {
	encoded, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("编码平台会话：%w", err)
	}
	digest := sessionDigest(token)
	pipe := s.redis.TxPipeline()
	pipe.Set(ctx, "platform:session:"+digest, encoded, s.ttl)
	if previousUserID > 0 && previousUserID != session.UserID {
		pipe.SRem(ctx, userIndexKey(previousUserID), digest)
	}
	if session.UserID > 0 {
		indexKey := userIndexKey(session.UserID)
		pipe.SAdd(ctx, indexKey, digest)
		pipe.Expire(ctx, indexKey, s.ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("保存平台会话：%w", err)
	}
	return nil
}

func sessionKey(token string) string {
	return "platform:session:" + sessionDigest(token)
}

func sessionDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func userIndexKey(userID int) string {
	return fmt.Sprintf("platform:user_sessions:%d", userID)
}
