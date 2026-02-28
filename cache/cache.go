package cache

import (
	"context"
	"time"
)

// CacheKeyer is implemented by models to generate cache keys.
type CacheKeyer interface {
	CacheKey() string
}

// Cache defines the full caching interface for Gails.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (any, error)) (string, error)
	Flush(ctx context.Context) error
	HSet(ctx context.Context, key string, values map[string]any) error
	HGet(ctx context.Context, key, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	LPush(ctx context.Context, key string, values ...any) error
	RPop(ctx context.Context, key string) (string, error)
	Publish(ctx context.Context, channel string, message any) error
	Subscribe(ctx context.Context, channel string) (<-chan string, error)
	// Model-level cache helpers
	SetModel(ctx context.Context, model CacheKeyer, data any, ttl time.Duration) error
	GetModel(ctx context.Context, model CacheKeyer) (string, error)
}
