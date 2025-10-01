package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestMemoryCache tests memory cache implementation
func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	t.Run("basic get/set", func(t *testing.T) {
		err := cache.Set(ctx, "key1", "value1", 1*time.Minute)
		assert.NoError(t, err)

		val, err := cache.Get(ctx, "key1")
		assert.NoError(t, err)
		assert.Equal(t, "value1", val)
	})

	t.Run("cache miss", func(t *testing.T) {
		_, err := cache.Get(ctx, "nonexistent")
		assert.Error(t, err)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("delete", func(t *testing.T) {
		cache.Set(ctx, "key2", "value2", 1*time.Minute)
		err := cache.Delete(ctx, "key2")
		assert.NoError(t, err)

		_, err = cache.Get(ctx, "key2")
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("exists", func(t *testing.T) {
		cache.Set(ctx, "key3", "value3", 1*time.Minute)

		exists, err := cache.Exists(ctx, "key3")
		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = cache.Exists(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("setnx", func(t *testing.T) {
		cache.Clear()

		// First set should succeed
		ok, err := cache.SetNX(ctx, "key4", "value4", 1*time.Minute)
		assert.NoError(t, err)
		assert.True(t, ok)

		// Second set should fail (key exists)
		ok, err = cache.SetNX(ctx, "key4", "value4-new", 1*time.Minute)
		assert.NoError(t, err)
		assert.False(t, ok)

		// Value should be unchanged
		val, _ := cache.Get(ctx, "key4")
		assert.Equal(t, "value4", val)
	})

	t.Run("expiration", func(t *testing.T) {
		cache.Clear()

		err := cache.Set(ctx, "key5", "value5", 100*time.Millisecond)
		assert.NoError(t, err)

		// Should exist immediately
		val, err := cache.Get(ctx, "key5")
		assert.NoError(t, err)
		assert.Equal(t, "value5", val)

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Should be expired
		_, err = cache.Get(ctx, "key5")
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("multi get/set", func(t *testing.T) {
		cache.Clear()

		items := map[string]string{
			"multi1": "value1",
			"multi2": "value2",
			"multi3": "value3",
		}

		err := cache.SetMulti(ctx, items, 1*time.Minute)
		assert.NoError(t, err)

		result, err := cache.GetMulti(ctx, []string{"multi1", "multi2", "multi3", "nonexistent"})
		assert.NoError(t, err)
		assert.Equal(t, 3, len(result))
		assert.Equal(t, "value1", result["multi1"])
		assert.Equal(t, "value2", result["multi2"])
		assert.Equal(t, "value3", result["multi3"])
	})

	t.Run("multi delete", func(t *testing.T) {
		cache.Clear()

		cache.Set(ctx, "del1", "value1", 1*time.Minute)
		cache.Set(ctx, "del2", "value2", 1*time.Minute)
		cache.Set(ctx, "del3", "value3", 1*time.Minute)

		err := cache.DeleteMulti(ctx, []string{"del1", "del2"})
		assert.NoError(t, err)

		_, err = cache.Get(ctx, "del1")
		assert.True(t, IsCacheMiss(err))

		_, err = cache.Get(ctx, "del2")
		assert.True(t, IsCacheMiss(err))

		val, err := cache.Get(ctx, "del3")
		assert.NoError(t, err)
		assert.Equal(t, "value3", val)
	})

	t.Run("ttl", func(t *testing.T) {
		cache.Clear()

		cache.Set(ctx, "ttl1", "value1", 5*time.Second)

		ttl, err := cache.TTL(ctx, "ttl1")
		assert.NoError(t, err)
		assert.Greater(t, ttl, 4*time.Second)
		assert.LessOrEqual(t, ttl, 5*time.Second)

		// Non-existent key
		_, err = cache.TTL(ctx, "nonexistent")
		assert.True(t, IsCacheMiss(err))

		// No expiration
		cache.Set(ctx, "ttl2", "value2", 0)
		ttl, err = cache.TTL(ctx, "ttl2")
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), ttl)
	})

	t.Run("expire", func(t *testing.T) {
		cache.Clear()

		cache.Set(ctx, "exp1", "value1", 10*time.Second)

		err := cache.Expire(ctx, "exp1", 100*time.Millisecond)
		assert.NoError(t, err)

		// Should exist immediately
		val, err := cache.Get(ctx, "exp1")
		assert.NoError(t, err)
		assert.Equal(t, "value1", val)

		// Wait for new expiration
		time.Sleep(150 * time.Millisecond)

		_, err = cache.Get(ctx, "exp1")
		assert.True(t, IsCacheMiss(err))

		// Non-existent key
		err = cache.Expire(ctx, "nonexistent", 1*time.Second)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("close", func(t *testing.T) {
		err := cache.Close()
		assert.NoError(t, err)
	})
}

// TestErrCacheMiss tests cache miss error
func TestErrCacheMiss(t *testing.T) {
	t.Run("error message", func(t *testing.T) {
		err := &ErrCacheMiss{Key: "test_key"}
		assert.Equal(t, "cache miss: test_key", err.Error())
	})

	t.Run("is cache miss", func(t *testing.T) {
		err := &ErrCacheMiss{Key: "test"}
		assert.True(t, IsCacheMiss(err))

		err2 := assert.AnError
		assert.False(t, IsCacheMiss(err2))

		assert.False(t, IsCacheMiss(nil))
	})
}

// Benchmark tests
func BenchmarkMemoryCache_Set(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(ctx, "key", "value", 1*time.Minute)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	cache.Set(ctx, "key", "value", 1*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, "key")
	}
}

func BenchmarkMemoryCache_SetMulti(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
		"key5": "value5",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.SetMulti(ctx, items, 1*time.Minute)
	}
}

func BenchmarkMemoryCache_GetMulti(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
		"key5": "value5",
	}
	cache.SetMulti(ctx, items, 1*time.Minute)
	keys := []string{"key1", "key2", "key3", "key4", "key5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetMulti(ctx, keys)
	}
}
