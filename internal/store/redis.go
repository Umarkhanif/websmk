package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"wesmk/internal/model"
)

type Cache struct {
	client *redis.Client
}

func OpenCache(ctx context.Context, redisURL string) (*Cache, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	client := redis.NewClient(options)

	deadline := time.Now().Add(30 * time.Second)
	for {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = client.Ping(pingCtx).Err()
		cancel()
		if err == nil {
			return &Cache{client: client}, nil
		}
		if time.Now().After(deadline) {
			_ = client.Close()
			return nil, fmt.Errorf("redis not reachable: %w", err)
		}
		fmt.Fprintln(os.Stderr, "waiting for redis...")
		time.Sleep(1 * time.Second)
	}
}

func (c *Cache) Close() error { return c.client.Close() }

func (c *Cache) Ping(ctx context.Context) error { return c.client.Ping(ctx).Err() }

func sessionKey(token string) string      { return "auth:session:" + token }
func userSessionKey(userID string) string { return "auth:user_session:" + userID }
func sessionInfoKey(token string) string  { return "auth:session_info:" + token }

// CreateSession stores a new token and revokes the user's previous one, which
// enforces the single-device guard.
func (c *Cache) CreateSession(ctx context.Context, token, userID string, ttl time.Duration) error {
	oldToken, err := c.client.Get(ctx, userSessionKey(userID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	pipe := c.client.TxPipeline()
	if oldToken != "" && oldToken != token {
		pipe.Del(ctx, sessionKey(oldToken))
		pipe.Del(ctx, sessionInfoKey(oldToken))
	}
	pipe.Set(ctx, sessionKey(token), userID, ttl)
	pipe.Set(ctx, userSessionKey(userID), token, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

// SetSessionInfo stores the IP and browser that created the session. It uses
// the same TTL as the token so metadata can never outlive the session.
func (c *Cache) SetSessionInfo(ctx context.Context, token string, info model.SessionInfo, ttl time.Duration) error {
	payload, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, sessionInfoKey(token), payload, ttl).Err()
}

func (c *Cache) SessionInfo(ctx context.Context, token string) (model.SessionInfo, error) {
	var info model.SessionInfo
	payload, err := c.client.Get(ctx, sessionInfoKey(token)).Bytes()
	if errors.Is(err, redis.Nil) {
		return info, nil
	}
	if err != nil {
		return info, err
	}
	if err := json.Unmarshal(payload, &info); err != nil {
		return model.SessionInfo{}, err
	}
	return info, nil
}

// SessionTTL exposes the remaining lifetime of the current token.
func (c *Cache) SessionTTL(ctx context.Context, token string) (time.Duration, error) {
	ttl, err := c.client.TTL(ctx, sessionKey(token)).Result()
	if err != nil || ttl < 0 {
		return 0, err
	}
	return ttl, nil
}

func (c *Cache) SessionUser(ctx context.Context, token string) (string, error) {
	value, err := c.client.Get(ctx, sessionKey(token)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// RevokeUserSessions removes the active token for a user. The single-device
// guard means one key covers every device the account is signed in on.
func (c *Cache) RevokeUserSessions(ctx context.Context, userID string) error {
	token, err := c.client.Get(ctx, userSessionKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	pipe := c.client.TxPipeline()
	if token != "" {
		pipe.Del(ctx, sessionKey(token))
		pipe.Del(ctx, sessionInfoKey(token))
	}
	pipe.Del(ctx, userSessionKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

func (c *Cache) DeleteSession(ctx context.Context, token string) error {
	userID, err := c.SessionUser(ctx, token)
	if err != nil {
		return err
	}

	pipe := c.client.TxPipeline()
	pipe.Del(ctx, sessionKey(token))
	pipe.Del(ctx, sessionInfoKey(token))
	if userID != "" {
		mapped, err := c.client.Get(ctx, userSessionKey(userID)).Result()
		if err == nil && mapped == token {
			pipe.Del(ctx, userSessionKey(userID))
		}
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (c *Cache) LoginFailureCount(ctx context.Context, key string) (int, error) {
	value, err := c.client.Get(ctx, key).Int()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (c *Cache) RecordLoginFailure(ctx context.Context, key string, window time.Duration) (int, error) {
	return c.increment(ctx, key, window)
}

func (c *Cache) ClearLoginFailures(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *Cache) StoreMagicToken(ctx context.Context, tokenHash, userID string, ttl time.Duration) error {
	return c.client.Set(ctx, "auth:magic:"+tokenHash, userID, ttl).Err()
}

func (c *Cache) ConsumeMagicToken(ctx context.Context, tokenHash string) (string, error) {
	value, err := c.client.GetDel(ctx, "auth:magic:"+tokenHash).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (c *Cache) StoreMFAChallenge(ctx context.Context, tokenHash, userID string, ttl time.Duration) error {
	return c.client.Set(ctx, "auth:mfa:"+tokenHash, userID, ttl).Err()
}

func (c *Cache) MFAChallengeUser(ctx context.Context, tokenHash string) (string, error) {
	value, err := c.client.Get(ctx, "auth:mfa:"+tokenHash).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (c *Cache) DeleteMFAChallenge(ctx context.Context, tokenHash string) error {
	return c.client.Del(ctx, "auth:mfa:"+tokenHash).Err()
}

// IncrementCounter is a fixed-window counter used for AI and magic-link limits.
func (c *Cache) IncrementCounter(ctx context.Context, key string, window time.Duration) (int, error) {
	return c.increment(ctx, key, window)
}

func (c *Cache) increment(ctx context.Context, key string, window time.Duration) (int, error) {
	pipe := c.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	count, err := incr.Result()
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
