package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/boj/redistore"
)

// RedisSessionStore wraps redistore for Redis-backed session storage.
type RedisSessionStore struct {
	*redistore.RediStore
}

// NewRedisSessionStore creates a new Redis-backed session store.
func NewRedisSessionStore(host string, port int, password string, secret string) (*RedisSessionStore, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	store, err := redistore.NewRediStore(10, "tcp", addr, "", password, []byte(secret))
	if err != nil {
		return nil, err
	}
	return &RedisSessionStore{store}, nil
}

// Fragment caching helpers

// GetFragment retrieves a cached fragment by key.
func GetFragment(ctx context.Context, key string) (string, error) {
	if Redis == nil {
		return "", fmt.Errorf("redis not initialized")
	}
	return Redis.Get(ctx, "fragment:"+key).Result()
}

// SetFragment stores a fragment in cache with a TTL.
func SetFragment(ctx context.Context, key string, value string, ttl time.Duration) error {
	if Redis == nil {
		return fmt.Errorf("redis not initialized")
	}
	return Redis.Set(ctx, "fragment:"+key, value, ttl).Err()
}
