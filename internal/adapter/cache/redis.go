package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"telegram-bot/pkg/logger"
)

// RedisConfig Redis 配置
type RedisConfig struct {
	// Addr Redis 地址 (host:port)
	Addr string
	// Password Redis 密码
	Password string
	// DB Redis 数据库索引
	DB int
	// PoolSize 连接池大小
	PoolSize int
	// MinIdleConns 最小空闲连接数
	MinIdleConns int
	// MaxRetries 最大重试次数
	MaxRetries int
	// DialTimeout 连接超时
	DialTimeout time.Duration
	// ReadTimeout 读取超时
	ReadTimeout time.Duration
	// WriteTimeout 写入超时
	WriteTimeout time.Duration
}

// DefaultRedisConfig 默认 Redis 配置
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// RedisCache Redis 缓存实现
type RedisCache struct {
	client *redis.Client
	logger logger.Logger
}

// NewRedisCache 创建 Redis 缓存
func NewRedisCache(config *RedisConfig, log logger.Logger) (*RedisCache, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Info("Redis cache connected successfully",
		"addr", config.Addr,
		"db", config.DB,
	)

	return &RedisCache{
		client: client,
		logger: log,
	}, nil
}

// Get 获取缓存值
func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", &ErrCacheMiss{Key: key}
	}
	if err != nil {
		r.logger.Error("Failed to get cache", "key", key, "error", err)
		return "", fmt.Errorf("get cache failed: %w", err)
	}

	r.logger.Debug("Cache hit", "key", key)
	return val, nil
}

// Set 设置缓存值
func (r *RedisCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	err := r.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		r.logger.Error("Failed to set cache", "key", key, "error", err)
		return fmt.Errorf("set cache failed: %w", err)
	}

	r.logger.Debug("Cache set", "key", key, "expiration", expiration)
	return nil
}

// Delete 删除缓存
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		r.logger.Error("Failed to delete cache", "key", key, "error", err)
		return fmt.Errorf("delete cache failed: %w", err)
	}

	r.logger.Debug("Cache deleted", "key", key)
	return nil
}

// Exists 检查缓存是否存在
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		r.logger.Error("Failed to check cache existence", "key", key, "error", err)
		return false, fmt.Errorf("exists check failed: %w", err)
	}

	return n > 0, nil
}

// SetNX 如果不存在则设置（原子操作）
func (r *RedisCache) SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	ok, err := r.client.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		r.logger.Error("Failed to setnx cache", "key", key, "error", err)
		return false, fmt.Errorf("setnx failed: %w", err)
	}

	if ok {
		r.logger.Debug("Cache setnx succeeded", "key", key)
	} else {
		r.logger.Debug("Cache setnx failed (key exists)", "key", key)
	}

	return ok, nil
}

// GetMulti 批量获取
func (r *RedisCache) GetMulti(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	pipe := r.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd)

	for _, key := range keys {
		cmds[key] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	// Pipeline 执行错误（非 redis.Nil）才返回错误
	if err != nil && err != redis.Nil {
		r.logger.Error("Failed to get multi cache", "error", err)
		return nil, fmt.Errorf("get multi failed: %w", err)
	}

	result := make(map[string]string)
	for key, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			continue // 跳过不存在的 key
		}
		if err != nil {
			r.logger.Warn("Failed to get key in multi", "key", key, "error", err)
			continue
		}
		result[key] = val
	}

	r.logger.Debug("Cache multi get", "requested", len(keys), "found", len(result))
	return result, nil
}

// SetMulti 批量设置
func (r *RedisCache) SetMulti(ctx context.Context, items map[string]string, expiration time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for key, val := range items {
		pipe.Set(ctx, key, val, expiration)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		r.logger.Error("Failed to set multi cache", "error", err)
		return fmt.Errorf("set multi failed: %w", err)
	}

	r.logger.Debug("Cache multi set", "count", len(items), "expiration", expiration)
	return nil
}

// DeleteMulti 批量删除
func (r *RedisCache) DeleteMulti(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	err := r.client.Del(ctx, keys...).Err()
	if err != nil {
		r.logger.Error("Failed to delete multi cache", "error", err)
		return fmt.Errorf("delete multi failed: %w", err)
	}

	r.logger.Debug("Cache multi deleted", "count", len(keys))
	return nil
}

// TTL 获取剩余过期时间
func (r *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		r.logger.Error("Failed to get ttl", "key", key, "error", err)
		return 0, fmt.Errorf("get ttl failed: %w", err)
	}

	// -2 表示 key 不存在
	if ttl == -2*time.Second {
		return 0, &ErrCacheMiss{Key: key}
	}

	// -1 表示没有设置过期时间
	if ttl == -1*time.Second {
		return 0, nil
	}

	return ttl, nil
}

// Expire 设置过期时间
func (r *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	ok, err := r.client.Expire(ctx, key, expiration).Result()
	if err != nil {
		r.logger.Error("Failed to set expiration", "key", key, "error", err)
		return fmt.Errorf("expire failed: %w", err)
	}

	if !ok {
		return &ErrCacheMiss{Key: key}
	}

	r.logger.Debug("Cache expiration set", "key", key, "expiration", expiration)
	return nil
}

// Close 关闭连接
func (r *RedisCache) Close() error {
	err := r.client.Close()
	if err != nil {
		r.logger.Error("Failed to close redis connection", "error", err)
		return fmt.Errorf("close redis failed: %w", err)
	}

	r.logger.Info("Redis cache closed")
	return nil
}

// Ping 测试连接
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// FlushDB 清空当前数据库（慎用！仅用于测试）
func (r *RedisCache) FlushDB(ctx context.Context) error {
	err := r.client.FlushDB(ctx).Err()
	if err != nil {
		r.logger.Error("Failed to flush db", "error", err)
		return fmt.Errorf("flush db failed: %w", err)
	}

	r.logger.Warn("Redis database flushed")
	return nil
}
