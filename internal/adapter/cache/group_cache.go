package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"telegram-bot/internal/domain/group"
	"telegram-bot/pkg/logger"
)

// GroupCache 群组缓存
type GroupCache struct {
	cache  Cache
	logger logger.Logger
	// 默认过期时间
	defaultTTL time.Duration
}

// NewGroupCache 创建群组缓存
func NewGroupCache(cache Cache, log logger.Logger, ttl time.Duration) *GroupCache {
	if ttl == 0 {
		ttl = 30 * time.Minute // 默认 30 分钟
	}

	return &GroupCache{
		cache:      cache,
		logger:     log,
		defaultTTL: ttl,
	}
}

// groupCacheKey 生成群组缓存 key
func groupCacheKey(groupID int64) string {
	return fmt.Sprintf("group:%d", groupID)
}

// groupCommandCacheKey 生成群组命令配置缓存 key
func groupCommandCacheKey(groupID int64, commandName string) string {
	return fmt.Sprintf("group:%d:command:%s", groupID, commandName)
}

// groupSettingCacheKey 生成群组设置缓存 key
func groupSettingCacheKey(groupID int64, settingKey string) string {
	return fmt.Sprintf("group:%d:setting:%s", groupID, settingKey)
}

// GetGroup 获取群组缓存
func (gc *GroupCache) GetGroup(ctx context.Context, groupID int64) (*group.Group, error) {
	key := groupCacheKey(groupID)
	val, err := gc.cache.Get(ctx, key)
	if err != nil {
		if IsCacheMiss(err) {
			gc.logger.Debug("Group cache miss", "group_id", groupID)
		} else {
			gc.logger.Error("Failed to get group from cache", "group_id", groupID, "error", err)
		}
		return nil, err
	}

	var g group.Group
	if err := json.Unmarshal([]byte(val), &g); err != nil {
		gc.logger.Error("Failed to unmarshal group", "group_id", groupID, "error", err)
		return nil, fmt.Errorf("unmarshal group failed: %w", err)
	}

	gc.logger.Debug("Group cache hit", "group_id", groupID)
	return &g, nil
}

// SetGroup 设置群组缓存
func (gc *GroupCache) SetGroup(ctx context.Context, g *group.Group) error {
	key := groupCacheKey(g.ID)

	data, err := json.Marshal(g)
	if err != nil {
		gc.logger.Error("Failed to marshal group", "group_id", g.ID, "error", err)
		return fmt.Errorf("marshal group failed: %w", err)
	}

	if err := gc.cache.Set(ctx, key, string(data), gc.defaultTTL); err != nil {
		gc.logger.Error("Failed to set group cache", "group_id", g.ID, "error", err)
		return err
	}

	gc.logger.Debug("Group cached", "group_id", g.ID, "ttl", gc.defaultTTL)
	return nil
}

// DeleteGroup 删除群组缓存
func (gc *GroupCache) DeleteGroup(ctx context.Context, groupID int64) error {
	key := groupCacheKey(groupID)
	if err := gc.cache.Delete(ctx, key); err != nil {
		gc.logger.Error("Failed to delete group cache", "group_id", groupID, "error", err)
		return err
	}

	gc.logger.Debug("Group cache deleted", "group_id", groupID)
	return nil
}

// GetCommandEnabled 获取命令启用状态缓存
func (gc *GroupCache) GetCommandEnabled(ctx context.Context, groupID int64, commandName string) (bool, error) {
	key := groupCommandCacheKey(groupID, commandName)
	val, err := gc.cache.Get(ctx, key)
	if err != nil {
		if IsCacheMiss(err) {
			gc.logger.Debug("Command cache miss", "group_id", groupID, "command", commandName)
		} else {
			gc.logger.Error("Failed to get command from cache", "group_id", groupID, "command", commandName, "error", err)
		}
		return false, err
	}

	var enabled bool
	if err := json.Unmarshal([]byte(val), &enabled); err != nil {
		gc.logger.Error("Failed to unmarshal command status", "group_id", groupID, "command", commandName, "error", err)
		return false, fmt.Errorf("unmarshal command status failed: %w", err)
	}

	gc.logger.Debug("Command cache hit", "group_id", groupID, "command", commandName, "enabled", enabled)
	return enabled, nil
}

// SetCommandEnabled 设置命令启用状态缓存
func (gc *GroupCache) SetCommandEnabled(ctx context.Context, groupID int64, commandName string, enabled bool) error {
	key := groupCommandCacheKey(groupID, commandName)

	data, err := json.Marshal(enabled)
	if err != nil {
		gc.logger.Error("Failed to marshal command status", "group_id", groupID, "command", commandName, "error", err)
		return fmt.Errorf("marshal command status failed: %w", err)
	}

	if err := gc.cache.Set(ctx, key, string(data), gc.defaultTTL); err != nil {
		gc.logger.Error("Failed to set command cache", "group_id", groupID, "command", commandName, "error", err)
		return err
	}

	gc.logger.Debug("Command cached", "group_id", groupID, "command", commandName, "enabled", enabled, "ttl", gc.defaultTTL)
	return nil
}

// DeleteCommand 删除命令缓存
func (gc *GroupCache) DeleteCommand(ctx context.Context, groupID int64, commandName string) error {
	key := groupCommandCacheKey(groupID, commandName)
	if err := gc.cache.Delete(ctx, key); err != nil {
		gc.logger.Error("Failed to delete command cache", "group_id", groupID, "command", commandName, "error", err)
		return err
	}

	gc.logger.Debug("Command cache deleted", "group_id", groupID, "command", commandName)
	return nil
}

// GetSetting 获取群组设置缓存
func (gc *GroupCache) GetSetting(ctx context.Context, groupID int64, settingKey string) (interface{}, error) {
	key := groupSettingCacheKey(groupID, settingKey)
	val, err := gc.cache.Get(ctx, key)
	if err != nil {
		if IsCacheMiss(err) {
			gc.logger.Debug("Setting cache miss", "group_id", groupID, "setting", settingKey)
		} else {
			gc.logger.Error("Failed to get setting from cache", "group_id", groupID, "setting", settingKey, "error", err)
		}
		return nil, err
	}

	var setting interface{}
	if err := json.Unmarshal([]byte(val), &setting); err != nil {
		gc.logger.Error("Failed to unmarshal setting", "group_id", groupID, "setting", settingKey, "error", err)
		return nil, fmt.Errorf("unmarshal setting failed: %w", err)
	}

	gc.logger.Debug("Setting cache hit", "group_id", groupID, "setting", settingKey)
	return setting, nil
}

// SetSetting 设置群组设置缓存
func (gc *GroupCache) SetSetting(ctx context.Context, groupID int64, settingKey string, value interface{}) error {
	key := groupSettingCacheKey(groupID, settingKey)

	data, err := json.Marshal(value)
	if err != nil {
		gc.logger.Error("Failed to marshal setting", "group_id", groupID, "setting", settingKey, "error", err)
		return fmt.Errorf("marshal setting failed: %w", err)
	}

	if err := gc.cache.Set(ctx, key, string(data), gc.defaultTTL); err != nil {
		gc.logger.Error("Failed to set setting cache", "group_id", groupID, "setting", settingKey, "error", err)
		return err
	}

	gc.logger.Debug("Setting cached", "group_id", groupID, "setting", settingKey, "ttl", gc.defaultTTL)
	return nil
}

// DeleteSetting 删除群组设置缓存
func (gc *GroupCache) DeleteSetting(ctx context.Context, groupID int64, settingKey string) error {
	key := groupSettingCacheKey(groupID, settingKey)
	if err := gc.cache.Delete(ctx, key); err != nil {
		gc.logger.Error("Failed to delete setting cache", "group_id", groupID, "setting", settingKey, "error", err)
		return err
	}

	gc.logger.Debug("Setting cache deleted", "group_id", groupID, "setting", settingKey)
	return nil
}

// InvalidateGroup 使群组的所有缓存失效
func (gc *GroupCache) InvalidateGroup(ctx context.Context, groupID int64) error {
	// 注意：这里只删除群组基本信息，命令和设置缓存需要单独删除
	// 在实际使用中，可以使用 Redis 的 SCAN 或者维护一个 key 列表来删除所有相关缓存
	return gc.DeleteGroup(ctx, groupID)
}

// WarmupGroups 预热群组缓存（批量加载群组）
func (gc *GroupCache) WarmupGroups(ctx context.Context, groups []*group.Group) error {
	if len(groups) == 0 {
		return nil
	}

	items := make(map[string]string)
	for _, g := range groups {
		key := groupCacheKey(g.ID)
		data, err := json.Marshal(g)
		if err != nil {
			gc.logger.Warn("Failed to marshal group for warmup", "group_id", g.ID, "error", err)
			continue
		}
		items[key] = string(data)
	}

	if err := gc.cache.SetMulti(ctx, items, gc.defaultTTL); err != nil {
		gc.logger.Error("Failed to warmup groups", "count", len(items), "error", err)
		return fmt.Errorf("warmup groups failed: %w", err)
	}

	gc.logger.Info("Groups warmed up", "count", len(items), "ttl", gc.defaultTTL)
	return nil
}

// WarmupCommands 预热命令配置缓存
func (gc *GroupCache) WarmupCommands(ctx context.Context, groupID int64, commands map[string]bool) error {
	if len(commands) == 0 {
		return nil
	}

	items := make(map[string]string)
	for cmdName, enabled := range commands {
		key := groupCommandCacheKey(groupID, cmdName)
		data, err := json.Marshal(enabled)
		if err != nil {
			gc.logger.Warn("Failed to marshal command for warmup", "group_id", groupID, "command", cmdName, "error", err)
			continue
		}
		items[key] = string(data)
	}

	if err := gc.cache.SetMulti(ctx, items, gc.defaultTTL); err != nil {
		gc.logger.Error("Failed to warmup commands", "group_id", groupID, "count", len(items), "error", err)
		return fmt.Errorf("warmup commands failed: %w", err)
	}

	gc.logger.Info("Commands warmed up", "group_id", groupID, "count", len(items), "ttl", gc.defaultTTL)
	return nil
}
