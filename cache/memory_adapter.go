package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryAdapter implements Cache using an in-memory map (thread-safe, for tests).
type MemoryAdapter struct {
	mu    sync.RWMutex
	items map[string]memoryItem
	lists map[string][]string
}

type memoryItem struct {
	value      string
	expiration int64
}

// NewMemoryAdapter creates a new in-memory cache adapter.
func NewMemoryAdapter() *MemoryAdapter {
	return &MemoryAdapter{
		items: make(map[string]memoryItem),
		lists: make(map[string][]string),
	}
}

func (m *MemoryAdapter) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.items[key]
	if !ok || (item.expiration > 0 && time.Now().UnixNano() > item.expiration) {
		return "", fmt.Errorf("key not found: %s", key)
	}

	return item.value, nil
}

func (m *MemoryAdapter) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	m.items[key] = memoryItem{
		value:      fmt.Sprint(value),
		expiration: expiration,
	}

	return nil
}

func (m *MemoryAdapter) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}

func (m *MemoryAdapter) Exists(ctx context.Context, key string) (bool, error) {
	_, err := m.Get(ctx, key)
	return err == nil, nil
}

func (m *MemoryAdapter) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (any, error)) (string, error) {
	val, err := m.Get(ctx, key)
	if err == nil {
		return val, nil
	}

	res, err := fn()
	if err != nil {
		return "", err
	}

	err = m.Set(ctx, key, res, ttl)
	if err != nil {
		return "", err
	}

	return fmt.Sprint(res), nil
}

func (m *MemoryAdapter) Flush(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]memoryItem)
	m.lists = make(map[string][]string)
	return nil
}

func (m *MemoryAdapter) HSet(ctx context.Context, key string, values map[string]any) error {
	for k, v := range values {
		if err := m.Set(ctx, key+":"+k, v, 0); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemoryAdapter) HGet(ctx context.Context, key, field string) (string, error) {
	return m.Get(ctx, key+":"+field)
}

func (m *MemoryAdapter) HGetAll(_ context.Context, key string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	prefix := key + ":"
	for k, v := range m.items {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			field := k[len(prefix):]
			if v.expiration == 0 || time.Now().UnixNano() <= v.expiration {
				result[field] = v.value
			}
		}
	}
	return result, nil
}

func (m *MemoryAdapter) LPush(_ context.Context, key string, values ...any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range values {
		m.lists[key] = append([]string{fmt.Sprint(v)}, m.lists[key]...)
	}
	return nil
}

func (m *MemoryAdapter) RPop(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.lists[key]
	if len(list) == 0 {
		return "", fmt.Errorf("list empty: %s", key)
	}
	val := list[len(list)-1]
	m.lists[key] = list[:len(list)-1]
	return val, nil
}

func (m *MemoryAdapter) Publish(_ context.Context, _ string, _ any) error {
	return nil // No-op in memory adapter
}

func (m *MemoryAdapter) Subscribe(_ context.Context, _ string) (<-chan string, error) {
	return make(chan string), nil // No-op in memory adapter
}

// SetModel caches data for a model using its CacheKey.
func (m *MemoryAdapter) SetModel(ctx context.Context, model CacheKeyer, data any, ttl time.Duration) error {
	return m.Set(ctx, model.CacheKey(), data, ttl)
}

// GetModel retrieves cached data for a model using its CacheKey.
func (m *MemoryAdapter) GetModel(ctx context.Context, model CacheKeyer) (string, error) {
	return m.Get(ctx, model.CacheKey())
}
