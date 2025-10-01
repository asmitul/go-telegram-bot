package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"telegram-bot/internal/domain/group"
)

func TestGroupCache(t *testing.T) {
	cache := NewMemoryCache()
	log := &MockLogger{}
	groupCache := NewGroupCache(cache, log, 30*time.Minute)
	ctx := context.Background()

	t.Run("get/set group", func(t *testing.T) {
		cache.Clear()

		g := group.NewGroup(-100, "Test Group", "supergroup")
		g.EnableCommand("ping", 123)
		g.Settings = map[string]interface{}{
			"welcome": "Hello!",
			"max_warnings": 3,
		}

		// Set group
		err := groupCache.SetGroup(ctx, g)
		assert.NoError(t, err)

		// Get group
		retrieved, err := groupCache.GetGroup(ctx, -100)
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, int64(-100), retrieved.ID)
		assert.Equal(t, "Test Group", retrieved.Title)
		assert.Equal(t, "supergroup", retrieved.Type)
	})

	t.Run("get group cache miss", func(t *testing.T) {
		cache.Clear()

		_, err := groupCache.GetGroup(ctx, -999)
		assert.Error(t, err)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("delete group", func(t *testing.T) {
		cache.Clear()

		g := group.NewGroup(-200, "Group 2", "group")
		groupCache.SetGroup(ctx, g)

		err := groupCache.DeleteGroup(ctx, -200)
		assert.NoError(t, err)

		_, err = groupCache.GetGroup(ctx, -200)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("get/set command enabled", func(t *testing.T) {
		cache.Clear()

		// Set command enabled
		err := groupCache.SetCommandEnabled(ctx, -100, "ping", true)
		assert.NoError(t, err)

		// Get command enabled
		enabled, err := groupCache.GetCommandEnabled(ctx, -100, "ping")
		assert.NoError(t, err)
		assert.True(t, enabled)

		// Set command disabled
		err = groupCache.SetCommandEnabled(ctx, -100, "ban", false)
		assert.NoError(t, err)

		// Get command disabled
		enabled, err = groupCache.GetCommandEnabled(ctx, -100, "ban")
		assert.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("get command cache miss", func(t *testing.T) {
		cache.Clear()

		_, err := groupCache.GetCommandEnabled(ctx, -999, "nonexistent")
		assert.Error(t, err)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("delete command", func(t *testing.T) {
		cache.Clear()

		groupCache.SetCommandEnabled(ctx, -300, "test", true)

		err := groupCache.DeleteCommand(ctx, -300, "test")
		assert.NoError(t, err)

		_, err = groupCache.GetCommandEnabled(ctx, -300, "test")
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("get/set setting", func(t *testing.T) {
		cache.Clear()

		// Set string setting
		err := groupCache.SetSetting(ctx, -100, "welcome", "Welcome to the group!")
		assert.NoError(t, err)

		// Get string setting
		val, err := groupCache.GetSetting(ctx, -100, "welcome")
		assert.NoError(t, err)
		assert.Equal(t, "Welcome to the group!", val)

		// Set numeric setting
		err = groupCache.SetSetting(ctx, -100, "max_warnings", 5)
		assert.NoError(t, err)

		// Get numeric setting
		val, err = groupCache.GetSetting(ctx, -100, "max_warnings")
		assert.NoError(t, err)
		// JSON unmarshals numbers as float64
		assert.Equal(t, float64(5), val)
	})

	t.Run("get setting cache miss", func(t *testing.T) {
		cache.Clear()

		_, err := groupCache.GetSetting(ctx, -999, "nonexistent")
		assert.Error(t, err)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("delete setting", func(t *testing.T) {
		cache.Clear()

		groupCache.SetSetting(ctx, -400, "test_setting", "test_value")

		err := groupCache.DeleteSetting(ctx, -400, "test_setting")
		assert.NoError(t, err)

		_, err = groupCache.GetSetting(ctx, -400, "test_setting")
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("invalidate group", func(t *testing.T) {
		cache.Clear()

		g := group.NewGroup(-500, "Invalidate Group", "supergroup")
		groupCache.SetGroup(ctx, g)

		err := groupCache.InvalidateGroup(ctx, -500)
		assert.NoError(t, err)

		_, err = groupCache.GetGroup(ctx, -500)
		assert.True(t, IsCacheMiss(err))
	})

	t.Run("warmup groups", func(t *testing.T) {
		cache.Clear()

		groups := []*group.Group{
			group.NewGroup(-1, "Group 1", "supergroup"),
			group.NewGroup(-2, "Group 2", "group"),
			group.NewGroup(-3, "Group 3", "supergroup"),
		}

		err := groupCache.WarmupGroups(ctx, groups)
		assert.NoError(t, err)

		// Verify all groups are cached
		for _, g := range groups {
			retrieved, err := groupCache.GetGroup(ctx, g.ID)
			assert.NoError(t, err)
			assert.Equal(t, g.ID, retrieved.ID)
			assert.Equal(t, g.Title, retrieved.Title)
			assert.Equal(t, g.Type, retrieved.Type)
		}
	})

	t.Run("warmup empty groups", func(t *testing.T) {
		err := groupCache.WarmupGroups(ctx, []*group.Group{})
		assert.NoError(t, err)
	})

	t.Run("warmup commands", func(t *testing.T) {
		cache.Clear()

		commands := map[string]bool{
			"ping": true,
			"help": true,
			"ban":  false,
			"mute": false,
		}

		err := groupCache.WarmupCommands(ctx, -100, commands)
		assert.NoError(t, err)

		// Verify all commands are cached
		for cmdName, expectedEnabled := range commands {
			enabled, err := groupCache.GetCommandEnabled(ctx, -100, cmdName)
			assert.NoError(t, err)
			assert.Equal(t, expectedEnabled, enabled)
		}
	})

	t.Run("warmup empty commands", func(t *testing.T) {
		err := groupCache.WarmupCommands(ctx, -100, map[string]bool{})
		assert.NoError(t, err)
	})

	t.Run("custom ttl", func(t *testing.T) {
		customCache := NewGroupCache(cache, log, 10*time.Minute)
		assert.Equal(t, 10*time.Minute, customCache.defaultTTL)
	})

	t.Run("default ttl", func(t *testing.T) {
		defaultCache := NewGroupCache(cache, log, 0)
		assert.Equal(t, 30*time.Minute, defaultCache.defaultTTL)
	})

	t.Run("group with commands and settings", func(t *testing.T) {
		cache.Clear()

		g := group.NewGroup(-600, "Full Group", "supergroup")
		g.EnableCommand("ping", 1)
		g.EnableCommand("help", 1)
		g.DisableCommand("ban", 2)
		g.Settings = map[string]interface{}{
			"welcome":      "Welcome!",
			"max_warnings": 3,
			"auto_ban":     true,
		}

		err := groupCache.SetGroup(ctx, g)
		assert.NoError(t, err)

		retrieved, err := groupCache.GetGroup(ctx, -600)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(retrieved.Commands))
		assert.Equal(t, 3, len(retrieved.Settings))
		assert.True(t, retrieved.IsCommandEnabled("ping"))
		assert.True(t, retrieved.IsCommandEnabled("help"))
		assert.False(t, retrieved.IsCommandEnabled("ban"))
	})
}

func TestGroupCacheKeys(t *testing.T) {
	t.Run("group cache key", func(t *testing.T) {
		key := groupCacheKey(-100)
		assert.Equal(t, "group:-100", key)
	})

	t.Run("group command cache key", func(t *testing.T) {
		key := groupCommandCacheKey(-100, "ping")
		assert.Equal(t, "group:-100:command:ping", key)
	})

	t.Run("group setting cache key", func(t *testing.T) {
		key := groupSettingCacheKey(-100, "welcome")
		assert.Equal(t, "group:-100:setting:welcome", key)
	})
}
