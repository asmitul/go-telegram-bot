package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryCache 内存缓存实现（用于测试）
type MemoryCache struct {
	mu    sync.RWMutex
	store map[string]*cacheItem
}

type cacheItem struct {
	value      string
	expiration time.Time
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		store: make(map[string]*cacheItem),
	}

	// 启动清理 goroutine
	go mc.cleanup()

	return mc
}

// cleanup 清理过期缓存
func (m *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, item := range m.store {
			if !item.expiration.IsZero() && now.After(item.expiration) {
				delete(m.store, key)
			}
		}
		m.mu.Unlock()
	}
}

// Get 获取缓存值
func (m *MemoryCache) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.store[key]
	if !exists {
		return "", &ErrCacheMiss{Key: key}
	}

	// 检查是否过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		return "", &ErrCacheMiss{Key: key}
	}

	return item.value, nil
}

// Set 设置缓存值
func (m *MemoryCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	m.store[key] = &cacheItem{
		value:      value,
		expiration: exp,
	}

	return nil
}

// Delete 删除缓存
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.store, key)
	return nil
}

// Exists 检查缓存是否存在
func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.store[key]
	if !exists {
		return false, nil
	}

	// 检查是否过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		return false, nil
	}

	return true, nil
}

// SetNX 如果不存在则设置
func (m *MemoryCache) SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, exists := m.store[key]

	// 检查是否存在且未过期
	if exists {
		if item.expiration.IsZero() || time.Now().Before(item.expiration) {
			return false, nil
		}
	}

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	m.store[key] = &cacheItem{
		value:      value,
		expiration: exp,
	}

	return true, nil
}

// GetMulti 批量获取
func (m *MemoryCache) GetMulti(ctx context.Context, keys []string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	now := time.Now()

	for _, key := range keys {
		item, exists := m.store[key]
		if !exists {
			continue
		}

		// 检查是否过期
		if !item.expiration.IsZero() && now.After(item.expiration) {
			continue
		}

		result[key] = item.value
	}

	return result, nil
}

// SetMulti 批量设置
func (m *MemoryCache) SetMulti(ctx context.Context, items map[string]string, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	for key, value := range items {
		m.store[key] = &cacheItem{
			value:      value,
			expiration: exp,
		}
	}

	return nil
}

// DeleteMulti 批量删除
func (m *MemoryCache) DeleteMulti(ctx context.Context, keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.store, key)
	}

	return nil
}

// TTL 获取剩余过期时间
func (m *MemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, exists := m.store[key]
	if !exists {
		return 0, &ErrCacheMiss{Key: key}
	}

	if item.expiration.IsZero() {
		return 0, nil // 没有过期时间
	}

	ttl := time.Until(item.expiration)
	if ttl < 0 {
		return 0, &ErrCacheMiss{Key: key}
	}

	return ttl, nil
}

// Expire 设置过期时间
func (m *MemoryCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, exists := m.store[key]
	if !exists {
		return &ErrCacheMiss{Key: key}
	}

	// 检查是否已过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		return &ErrCacheMiss{Key: key}
	}

	if expiration > 0 {
		item.expiration = time.Now().Add(expiration)
	} else {
		item.expiration = time.Time{}
	}

	return nil
}

// Close 关闭连接（内存缓存无需关闭）
func (m *MemoryCache) Close() error {
	return nil
}

// Clear 清空所有缓存（仅用于测试）
func (m *MemoryCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store = make(map[string]*cacheItem)
}
