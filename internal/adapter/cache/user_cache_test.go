package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/logger"
)

func TestUserCache(t *testing.T) {
	cache := NewMemoryCache()
	log := &MockLogger{}
	userCache := NewUserCache(cache, log, 1*time.Hour)
	ctx := context.Background()

	t.Run("get/set user", func(t *testing.T) {
		cache.Clear()

		u := user.NewUser(123, "testuser", "Test", "User")
		u.SetPermission(-100, user.PermissionAdmin)

		// Set user
		err := userCache.SetUser(ctx, u)
		assert.NoError(t, err)

		// Get user
		retrieved, err := userCache.GetUser(ctx, 123)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, int64(123), retrieved.ID)
		assert.Equal(t, "testuser", retrieved.Username)
		assert.Equal(t, "Test", retrieved.FirstName)
		assert.Equal(t, "User", retrieved.LastName)
	})

	t.Run("get user cache miss", func(t *testing.T) {
		cache.Clear()

		_, err := userCache.GetUser(ctx, 999)
		assert.Error(t, err)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("delete user", func(t *testing.T) {
		cache.Clear()

		u := user.NewUser(456, "user2", "User", "Two")
		userCache.SetUser(ctx, u)

		err := userCache.DeleteUser(ctx, 456)
		assert.NoError(t, err)

		_, err = userCache.GetUser(ctx, 456)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("get/set permission", func(t *testing.T) {
		cache.Clear()

		// Set permission
		err := userCache.SetPermission(ctx, 123, -100, user.PermissionAdmin)
		assert.NoError(t, err)

		// Get permission
		perm, err := userCache.GetPermission(ctx, 123, -100)
		assert.NoError(t, err)
		assert.Equal(t, user.PermissionAdmin, perm)
	})

	t.Run("get permission cache miss", func(t *testing.T) {
		cache.Clear()

		_, err := userCache.GetPermission(ctx, 999, -999)
		assert.Error(t, err)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("delete permission", func(t *testing.T) {
		cache.Clear()

		userCache.SetPermission(ctx, 789, -200, user.PermissionSuperAdmin)

		err := userCache.DeletePermission(ctx, 789, -200)
		assert.NoError(t, err)

		_, err = userCache.GetPermission(ctx, 789, -200)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("invalidate user", func(t *testing.T) {
		cache.Clear()

		u := user.NewUser(111, "invalidate", "Test", "User")
		userCache.SetUser(ctx, u)

		err := userCache.InvalidateUser(ctx, 111)
		assert.NoError(t, err)

		_, err = userCache.GetUser(ctx, 111)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("warmup users", func(t *testing.T) {
		cache.Clear()

		users := []*user.User{
			user.NewUser(1, "user1", "User", "One"),
			user.NewUser(2, "user2", "User", "Two"),
			user.NewUser(3, "user3", "User", "Three"),
		}

		err := userCache.WarmupUsers(ctx, users)
		assert.NoError(t, err)

		// Verify all users are cached
		for _, u := range users {
			retrieved, err := userCache.GetUser(ctx, u.ID)
			assert.NoError(t, err)
			assert.Equal(t, u.ID, retrieved.ID)
			assert.Equal(t, u.Username, retrieved.Username)
		}
	})

	t.Run("warmup empty users", func(t *testing.T) {
		err := userCache.WarmupUsers(ctx, []*user.User{})
		assert.NoError(t, err)
	})

	t.Run("custom ttl", func(t *testing.T) {
		customCache := NewUserCache(cache, log, 5*time.Minute)
		assert.Equal(t, 5*time.Minute, customCache.defaultTTL)
	})

	t.Run("default ttl", func(t *testing.T) {
		defaultCache := NewUserCache(cache, log, 0)
		assert.Equal(t, 1*time.Hour, defaultCache.defaultTTL)
	})

	t.Run("user with permissions", func(t *testing.T) {
		cache.Clear()

		u := user.NewUser(555, "permuser", "Perm", "User")
		u.SetPermission(-100, user.PermissionAdmin)
		u.SetPermission(-200, user.PermissionSuperAdmin)
		u.SetPermission(-300, user.PermissionUser)

		err := userCache.SetUser(ctx, u)
		assert.NoError(t, err)

		retrieved, err := userCache.GetUser(ctx, 555)
		assert.NoError(t, err)
		assert.Equal(t, user.PermissionAdmin, retrieved.GetPermission(-100))
		assert.Equal(t, user.PermissionSuperAdmin, retrieved.GetPermission(-200))
		assert.Equal(t, user.PermissionUser, retrieved.GetPermission(-300))
	})
}

func TestUserCacheKeys(t *testing.T) {
	t.Run("user cache key", func(t *testing.T) {
		key := userCacheKey(123)
		assert.Equal(t, "user:123", key)
	})

	t.Run("user permission cache key", func(t *testing.T) {
		key := userPermissionCacheKey(123, -100)
		assert.Equal(t, "user:123:group:-100:permission", key)
	})
}

// MockLogger for testing
type MockLogger struct {
	DebugCallCount int
	InfoCallCount  int
	WarnCallCount  int
	ErrorCallCount int
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.DebugCallCount++
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.InfoCallCount++
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.WarnCallCount++
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.ErrorCallCount++
}

func (m *MockLogger) WithField(key string, value interface{}) logger.Logger {
	return m
}

func (m *MockLogger) WithFields(fields map[string]interface{}) logger.Logger {
	return m
}

func (m *MockLogger) WithContext(ctx context.Context) logger.Logger {
	return m
}

func (m *MockLogger) SetLevel(level logger.Level) {}
