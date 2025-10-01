package scheduler

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/logger"
)

// CleanupExpiredDataJob 清理过期数据任务
type CleanupExpiredDataJob struct {
	db     *mongo.Database
	logger logger.Logger
}

// NewCleanupExpiredDataJob 创建清理过期数据任务
func NewCleanupExpiredDataJob(db *mongo.Database, log logger.Logger) *CleanupExpiredDataJob {
	return &CleanupExpiredDataJob{
		db:     db,
		logger: log,
	}
}

func (j *CleanupExpiredDataJob) Name() string {
	return "CleanupExpiredData"
}

func (j *CleanupExpiredDataJob) Schedule() string {
	return "1d" // 每天执行一次
}

func (j *CleanupExpiredDataJob) Run(ctx context.Context) error {
	j.logger.Info("Starting cleanup expired data job")

	// 清理过期的警告记录（超过90天）
	warningsDeleted, err := j.cleanupExpiredWarnings(ctx)
	if err != nil {
		j.logger.Error("Failed to cleanup expired warnings", "error", err)
		return fmt.Errorf("cleanup warnings failed: %w", err)
	}

	// 清理不活跃的用户数据（超过180天未活跃）
	usersDeleted, err := j.cleanupInactiveUsers(ctx)
	if err != nil {
		j.logger.Error("Failed to cleanup inactive users", "error", err)
		// 不返回错误，继续执行
	}

	j.logger.Info("Cleanup expired data completed",
		"warnings_deleted", warningsDeleted,
		"users_deleted", usersDeleted,
	)

	return nil
}

// cleanupExpiredWarnings 清理过期警告
func (j *CleanupExpiredDataJob) cleanupExpiredWarnings(ctx context.Context) (int64, error) {
	collection := j.db.Collection("warnings")

	// 删除90天前的警告
	cutoffTime := time.Now().Add(-90 * 24 * time.Hour)
	filter := bson.M{
		"created_at": bson.M{"$lt": cutoffTime},
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// cleanupInactiveUsers 清理不活跃用户
func (j *CleanupExpiredDataJob) cleanupInactiveUsers(ctx context.Context) (int64, error) {
	collection := j.db.Collection("users")

	// 删除180天未活跃的普通用户（非管理员）
	cutoffTime := time.Now().Add(-180 * 24 * time.Hour)
	filter := bson.M{
		"updated_at": bson.M{"$lt": cutoffTime},
		"permissions": bson.M{
			"$not": bson.M{
				"$elemMatch": bson.M{
					"level": bson.M{"$gte": 2}, // Admin 及以上不删除
				},
			},
		},
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// StatisticsReportJob 统计报告任务
type StatisticsReportJob struct {
	userRepo  user.Repository
	groupRepo group.Repository
	logger    logger.Logger
}

// NewStatisticsReportJob 创建统计报告任务
func NewStatisticsReportJob(userRepo user.Repository, groupRepo group.Repository, log logger.Logger) *StatisticsReportJob {
	return &StatisticsReportJob{
		userRepo:  userRepo,
		groupRepo: groupRepo,
		logger:    log,
	}
}

func (j *StatisticsReportJob) Name() string {
	return "StatisticsReport"
}

func (j *StatisticsReportJob) Schedule() string {
	return "1h" // 每小时执行一次
}

func (j *StatisticsReportJob) Run(ctx context.Context) error {
	j.logger.Info("Starting statistics report job")

	// 统计活跃用户数（这里简化为记录日志）
	// 在实际应用中，可以将统计数据存储到数据库或发送报告

	stats := map[string]interface{}{
		"timestamp": time.Now(),
	}

	// 这里可以添加更多统计逻辑
	// 例如：查询数据库获取用户数、群组数、消息数等

	j.logger.Info("Statistics report generated", "stats", stats)

	return nil
}

// AutoUnbanJob 自动解除临时封禁任务
type AutoUnbanJob struct {
	db     *mongo.Database
	logger logger.Logger
}

// NewAutoUnbanJob 创建自动解除封禁任务
func NewAutoUnbanJob(db *mongo.Database, log logger.Logger) *AutoUnbanJob {
	return &AutoUnbanJob{
		db:     db,
		logger: log,
	}
}

func (j *AutoUnbanJob) Name() string {
	return "AutoUnban"
}

func (j *AutoUnbanJob) Schedule() string {
	return "5m" // 每5分钟执行一次
}

func (j *AutoUnbanJob) Run(ctx context.Context) error {
	j.logger.Info("Starting auto unban job")

	// 查找需要解除封禁的用户
	// 这里需要一个 bans 集合来存储临时封禁记录
	// 格式：{ user_id, group_id, banned_until, unbanned }

	collection := j.db.Collection("bans")

	// 查找已过期且未解除的封禁记录
	filter := bson.M{
		"banned_until": bson.M{"$lte": time.Now()},
		"unbanned":     false,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to query expired bans: %w", err)
	}
	defer cursor.Close(ctx)

	var unbanCount int64
	for cursor.Next(ctx) {
		var ban struct {
			ID         interface{} `bson:"_id"`
			UserID     int64       `bson:"user_id"`
			GroupID    int64       `bson:"group_id"`
			BannedUntil time.Time  `bson:"banned_until"`
		}

		if err := cursor.Decode(&ban); err != nil {
			j.logger.Warn("Failed to decode ban record", "error", err)
			continue
		}

		// 标记为已解除
		update := bson.M{
			"$set": bson.M{
				"unbanned":   true,
				"unbanned_at": time.Now(),
			},
		}

		_, err := collection.UpdateOne(ctx, bson.M{"_id": ban.ID}, update)
		if err != nil {
			j.logger.Error("Failed to mark ban as unbanned",
				"user_id", ban.UserID,
				"group_id", ban.GroupID,
				"error", err,
			)
			continue
		}

		j.logger.Info("User auto-unbanned",
			"user_id", ban.UserID,
			"group_id", ban.GroupID,
			"banned_until", ban.BannedUntil,
		)

		unbanCount++

		// 注意：这里只更新了数据库记录
		// 实际解除 Telegram 封禁需要调用 Telegram API
		// 可以通过事件或消息队列通知 bot 执行实际的解封操作
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	j.logger.Info("Auto unban job completed", "unban_count", unbanCount)

	return nil
}

// CacheWarmupJob 缓存预热任务
type CacheWarmupJob struct {
	logger logger.Logger
	warmupFunc func(ctx context.Context) error
}

// NewCacheWarmupJob 创建缓存预热任务
func NewCacheWarmupJob(log logger.Logger, warmupFunc func(ctx context.Context) error) *CacheWarmupJob {
	return &CacheWarmupJob{
		logger:     log,
		warmupFunc: warmupFunc,
	}
}

func (j *CacheWarmupJob) Name() string {
	return "CacheWarmup"
}

func (j *CacheWarmupJob) Schedule() string {
	return "30m" // 每30分钟执行一次
}

func (j *CacheWarmupJob) Run(ctx context.Context) error {
	j.logger.Info("Starting cache warmup job")

	if j.warmupFunc != nil {
		if err := j.warmupFunc(ctx); err != nil {
			return fmt.Errorf("cache warmup failed: %w", err)
		}
	}

	j.logger.Info("Cache warmup completed")
	return nil
}
