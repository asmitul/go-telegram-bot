package cache

import (
	"context"
	"time"
)

// Cache 缓存接口
type Cache interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string) (string, error)
	// Set 设置缓存值
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	// Delete 删除缓存
	Delete(ctx context.Context, key string) error
	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)
	// SetNX 如果不存在则设置（原子操作）
	SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error)
	// GetMulti 批量获取
	GetMulti(ctx context.Context, keys []string) (map[string]string, error)
	// SetMulti 批量设置
	SetMulti(ctx context.Context, items map[string]string, expiration time.Duration) error
	// DeleteMulti 批量删除
	DeleteMulti(ctx context.Context, keys []string) error
	// TTL 获取剩余过期时间
	TTL(ctx context.Context, key string) (time.Duration, error)
	// Expire 设置过期时间
	Expire(ctx context.Context, key string, expiration time.Duration) error
	// Close 关闭连接
	Close() error
}

// ErrCacheMiss 缓存未命中错误
type ErrCacheMiss struct {
	Key string
}

func (e *ErrCacheMiss) Error() string {
	return "cache miss: " + e.Key
}

// IsCacheMiss 判断是否是缓存未命中错误
func IsCacheMiss(err error) bool {
	_, ok := err.(*ErrCacheMiss)
	return ok
}
