package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shaurya/gails/config"
)

// RedisAdapter implements Cache using Redis.
type RedisAdapter struct {
	Client *redis.Client
}

// Redis is the global Redis client, exposed for advanced usage.
var Redis *redis.Client

// NewRedisAdapter creates a new Redis-backed cache adapter.
func NewRedisAdapter(cfg config.RedisConfig) (*RedisAdapter, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("[Gails] ERROR: Invalid Redis URL %s — %v", cfg.URL, err)
	}

	opts.PoolSize = cfg.Pool
	opts.DB = cfg.DB

	client := redis.NewClient(opts)

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("[Gails] ERROR: Cannot connect to Redis at %s — %v", cfg.URL, err)
	}

	Redis = client
	return &RedisAdapter{Client: client}, nil
}

// MustNewRedisAdapter creates a Redis adapter or panics.
func MustNewRedisAdapter(cfg config.RedisConfig) *RedisAdapter {
	adapter, err := NewRedisAdapter(cfg)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	return adapter
}

func (r *RedisAdapter) Get(ctx context.Context, key string) (string, error) {
	val, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func (r *RedisAdapter) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return r.Client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisAdapter) Delete(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}

func (r *RedisAdapter) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.Client.Exists(ctx, key).Result()
	return n > 0, err
}

func (r *RedisAdapter) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (any, error)) (string, error) {
	val, err := r.Get(ctx, key)
	if err == nil {
		return val, nil
	}

	res, err := fn()
	if err != nil {
		return "", err
	}

	err = r.Set(ctx, key, res, ttl)
	if err != nil {
		return "", err
	}

	return fmt.Sprint(res), nil
}

func (r *RedisAdapter) Flush(ctx context.Context) error {
	return r.Client.FlushDB(ctx).Err()
}

func (r *RedisAdapter) HSet(ctx context.Context, key string, values map[string]any) error {
	return r.Client.HSet(ctx, key, values).Err()
}

func (r *RedisAdapter) HGet(ctx context.Context, key, field string) (string, error) {
	return r.Client.HGet(ctx, key, field).Result()
}

func (r *RedisAdapter) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.Client.HGetAll(ctx, key).Result()
}

func (r *RedisAdapter) LPush(ctx context.Context, key string, values ...any) error {
	return r.Client.LPush(ctx, key, values...).Err()
}

func (r *RedisAdapter) RPop(ctx context.Context, key string) (string, error) {
	return r.Client.RPop(ctx, key).Result()
}

func (r *RedisAdapter) Publish(ctx context.Context, channel string, message any) error {
	return r.Client.Publish(ctx, channel, message).Err()
}

func (r *RedisAdapter) Subscribe(ctx context.Context, channel string) (<-chan string, error) {
	pubsub := r.Client.Subscribe(ctx, channel)
	ch := make(chan string)
	go func() {
		defer close(ch)
		for msg := range pubsub.Channel() {
			ch <- msg.Payload
		}
	}()
	return ch, nil
}

// SetModel caches data for a model using its CacheKey.
func (r *RedisAdapter) SetModel(ctx context.Context, model CacheKeyer, data any, ttl time.Duration) error {
	return r.Set(ctx, model.CacheKey(), data, ttl)
}

// GetModel retrieves cached data for a model using its CacheKey.
func (r *RedisAdapter) GetModel(ctx context.Context, model CacheKeyer) (string, error) {
	return r.Get(ctx, model.CacheKey())
}
