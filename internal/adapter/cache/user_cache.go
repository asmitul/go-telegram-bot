package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/logger"
)

// UserCache 用户缓存
type UserCache struct {
	cache  Cache
	logger logger.Logger
	// 默认过期时间
	defaultTTL time.Duration
}

// NewUserCache 创建用户缓存
func NewUserCache(cache Cache, log logger.Logger, ttl time.Duration) *UserCache {
	if ttl == 0 {
		ttl = 1 * time.Hour // 默认 1 小时
	}

	return &UserCache{
		cache:      cache,
		logger:     log,
		defaultTTL: ttl,
	}
}

// userCacheKey 生成用户缓存 key
func userCacheKey(userID int64) string {
	return fmt.Sprintf("user:%d", userID)
}

// userPermissionCacheKey 生成用户权限缓存 key
func userPermissionCacheKey(userID, groupID int64) string {
	return fmt.Sprintf("user:%d:group:%d:permission", userID, groupID)
}

// GetUser 获取用户缓存
func (uc *UserCache) GetUser(ctx context.Context, userID int64) (*user.User, error) {
	key := userCacheKey(userID)
	val, err := uc.cache.Get(ctx, key)
	if err != nil {
		if IsCacheMiss(err) {
			uc.logger.Debug("User cache miss", "user_id", userID)
		} else {
			uc.logger.Error("Failed to get user from cache", "user_id", userID, "error", err)
		}
		return nil, err
	}

	var u user.User
	if err := json.Unmarshal([]byte(val), &u); err != nil {
		uc.logger.Error("Failed to unmarshal user", "user_id", userID, "error", err)
		return nil, fmt.Errorf("unmarshal user failed: %w", err)
	}

	uc.logger.Debug("User cache hit", "user_id", userID)
	return &u, nil
}

// SetUser 设置用户缓存
func (uc *UserCache) SetUser(ctx context.Context, u *user.User) error {
	key := userCacheKey(u.ID)

	data, err := json.Marshal(u)
	if err != nil {
		uc.logger.Error("Failed to marshal user", "user_id", u.ID, "error", err)
		return fmt.Errorf("marshal user failed: %w", err)
	}

	if err := uc.cache.Set(ctx, key, string(data), uc.defaultTTL); err != nil {
		uc.logger.Error("Failed to set user cache", "user_id", u.ID, "error", err)
		return err
	}

	uc.logger.Debug("User cached", "user_id", u.ID, "ttl", uc.defaultTTL)
	return nil
}

// DeleteUser 删除用户缓存
func (uc *UserCache) DeleteUser(ctx context.Context, userID int64) error {
	key := userCacheKey(userID)
	if err := uc.cache.Delete(ctx, key); err != nil {
		uc.logger.Error("Failed to delete user cache", "user_id", userID, "error", err)
		return err
	}

	uc.logger.Debug("User cache deleted", "user_id", userID)
	return nil
}

// GetPermission 获取用户权限缓存
func (uc *UserCache) GetPermission(ctx context.Context, userID, groupID int64) (user.Permission, error) {
	key := userPermissionCacheKey(userID, groupID)
	val, err := uc.cache.Get(ctx, key)
	if err != nil {
		if IsCacheMiss(err) {
			uc.logger.Debug("Permission cache miss", "user_id", userID, "group_id", groupID)
		} else {
			uc.logger.Error("Failed to get permission from cache", "user_id", userID, "group_id", groupID, "error", err)
		}
		return 0, err
	}

	var perm user.Permission
	if err := json.Unmarshal([]byte(val), &perm); err != nil {
		uc.logger.Error("Failed to unmarshal permission", "user_id", userID, "group_id", groupID, "error", err)
		return 0, fmt.Errorf("unmarshal permission failed: %w", err)
	}

	uc.logger.Debug("Permission cache hit", "user_id", userID, "group_id", groupID, "permission", perm)
	return perm, nil
}

// SetPermission 设置用户权限缓存
func (uc *UserCache) SetPermission(ctx context.Context, userID, groupID int64, perm user.Permission) error {
	key := userPermissionCacheKey(userID, groupID)

	data, err := json.Marshal(perm)
	if err != nil {
		uc.logger.Error("Failed to marshal permission", "user_id", userID, "group_id", groupID, "error", err)
		return fmt.Errorf("marshal permission failed: %w", err)
	}

	if err := uc.cache.Set(ctx, key, string(data), uc.defaultTTL); err != nil {
		uc.logger.Error("Failed to set permission cache", "user_id", userID, "group_id", groupID, "error", err)
		return err
	}

	uc.logger.Debug("Permission cached", "user_id", userID, "group_id", groupID, "permission", perm, "ttl", uc.defaultTTL)
	return nil
}

// DeletePermission 删除用户权限缓存
func (uc *UserCache) DeletePermission(ctx context.Context, userID, groupID int64) error {
	key := userPermissionCacheKey(userID, groupID)
	if err := uc.cache.Delete(ctx, key); err != nil {
		uc.logger.Error("Failed to delete permission cache", "user_id", userID, "group_id", groupID, "error", err)
		return err
	}

	uc.logger.Debug("Permission cache deleted", "user_id", userID, "group_id", groupID)
	return nil
}

// InvalidateUser 使用户的所有缓存失效（包括用户信息和所有群组的权限）
// 注意：这只删除用户基本信息缓存，权限缓存需要单独删除
func (uc *UserCache) InvalidateUser(ctx context.Context, userID int64) error {
	return uc.DeleteUser(ctx, userID)
}

// WarmupUsers 预热用户缓存（批量加载用户）
func (uc *UserCache) WarmupUsers(ctx context.Context, users []*user.User) error {
	if len(users) == 0 {
		return nil
	}

	items := make(map[string]string)
	for _, u := range users {
		key := userCacheKey(u.ID)
		data, err := json.Marshal(u)
		if err != nil {
			uc.logger.Warn("Failed to marshal user for warmup", "user_id", u.ID, "error", err)
			continue
		}
		items[key] = string(data)
	}

	if err := uc.cache.SetMulti(ctx, items, uc.defaultTTL); err != nil {
		uc.logger.Error("Failed to warmup users", "count", len(items), "error", err)
		return fmt.Errorf("warmup users failed: %w", err)
	}

	uc.logger.Info("Users warmed up", "count", len(items), "ttl", uc.defaultTTL)
	return nil
}
