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

	// 清理不活跃的用户数据（超过180天未活跃）
	usersDeleted, err := j.cleanupInactiveUsers(ctx)
	if err != nil {
		j.logger.Error("Failed to cleanup inactive users", "error", err)
		// 不返回错误，继续执行
	}

	j.logger.Info("Cleanup expired data completed",
		"users_deleted", usersDeleted,
	)

	return nil
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
