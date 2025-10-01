package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"telegram-bot/pkg/logger"
)

// IndexManager 索引管理器
type IndexManager struct {
	db     *mongo.Database
	logger logger.Logger
}

// NewIndexManager 创建索引管理器
func NewIndexManager(db *mongo.Database, log logger.Logger) *IndexManager {
	return &IndexManager{
		db:     db,
		logger: log,
	}
}

// EnsureIndexes 确保所有索引存在
func (im *IndexManager) EnsureIndexes(ctx context.Context) error {
	if err := im.ensureUserIndexes(ctx); err != nil {
		return err
	}

	if err := im.ensureGroupIndexes(ctx); err != nil {
		return err
	}

	if err := im.ensureWarningIndexes(ctx); err != nil {
		return err
	}

	im.logger.Info("All indexes created successfully")
	return nil
}

// ensureUserIndexes 创建用户集合索引
func (im *IndexManager) ensureUserIndexes(ctx context.Context) error {
	collection := im.db.Collection("users")

	indexes := []mongo.IndexModel{
		{
			// 主键索引（_id 自动创建）
			Keys: bson.D{{Key: "_id", Value: 1}},
			Options: options.Index().
				SetName("idx_user_id").
				SetUnique(true),
		},
		{
			// 用户名索引（用于快速查找用户）
			Keys: bson.D{{Key: "username", Value: 1}},
			Options: options.Index().
				SetName("idx_username").
				SetSparse(true), // 允许 null 值
		},
		{
			// 组合索引：权限查询优化
			Keys: bson.D{
				{Key: "permissions", Value: 1},
				{Key: "updated_at", Value: -1},
			},
			Options: options.Index().
				SetName("idx_permissions_updated"),
		},
		{
			// 创建时间索引（用于统计和清理）
			Keys: bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().
				SetName("idx_created_at"),
		},
	}

	return im.createIndexes(ctx, collection, indexes, "users")
}

// ensureGroupIndexes 创建群组集合索引
func (im *IndexManager) ensureGroupIndexes(ctx context.Context) error {
	collection := im.db.Collection("groups")

	indexes := []mongo.IndexModel{
		{
			// 主键索引（_id 自动创建）
			Keys: bson.D{{Key: "_id", Value: 1}},
			Options: options.Index().
				SetName("idx_group_id").
				SetUnique(true),
		},
		{
			// 群组名称索引
			Keys: bson.D{{Key: "title", Value: 1}},
			Options: options.Index().
				SetName("idx_group_title"),
		},
		{
			// 群组类型索引（用于分类统计）
			Keys: bson.D{{Key: "type", Value: 1}},
			Options: options.Index().
				SetName("idx_group_type"),
		},
		{
			// 命令配置索引（用于快速查询命令状态）
			Keys: bson.D{
				{Key: "commands", Value: 1},
			},
			Options: options.Index().
				SetName("idx_commands"),
		},
		{
			// 更新时间索引
			Keys: bson.D{{Key: "updated_at", Value: -1}},
			Options: options.Index().
				SetName("idx_group_updated_at"),
		},
	}

	return im.createIndexes(ctx, collection, indexes, "groups")
}

// ensureWarningIndexes 创建警告集合索引
func (im *IndexManager) ensureWarningIndexes(ctx context.Context) error {
	collection := im.db.Collection("warnings")

	indexes := []mongo.IndexModel{
		{
			// 用户ID索引（最常用的查询条件）
			Keys: bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().
				SetName("idx_warning_user_id"),
		},
		{
			// 群组ID索引
			Keys: bson.D{{Key: "group_id", Value: 1}},
			Options: options.Index().
				SetName("idx_warning_group_id"),
		},
		{
			// 组合索引：用户+群组（最常用的组合查询）
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "group_id", Value: 1},
			},
			Options: options.Index().
				SetName("idx_warning_user_group").
				SetUnique(false),
		},
		{
			// 组合索引：用户+群组+创建时间（用于统计和清理）
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "group_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().
				SetName("idx_warning_user_group_created"),
		},
		{
			// TTL 索引：自动删除过期警告（90天后过期）
			Keys: bson.D{{Key: "created_at", Value: 1}},
			Options: options.Index().
				SetName("idx_warning_ttl").
				SetExpireAfterSeconds(90 * 24 * 60 * 60), // 90天
		},
	}

	return im.createIndexes(ctx, collection, indexes, "warnings")
}

// createIndexes 创建索引的辅助方法
func (im *IndexManager) createIndexes(ctx context.Context, collection *mongo.Collection, indexes []mongo.IndexModel, collectionName string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	im.logger.Info("Creating indexes", "collection", collectionName, "count", len(indexes))

	names, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		im.logger.Error("Failed to create indexes", "collection", collectionName, "error", err)
		return err
	}

	im.logger.Info("Indexes created successfully", "collection", collectionName, "indexes", names)
	return nil
}

// DropAllIndexes 删除所有索引（用于重建）
func (im *IndexManager) DropAllIndexes(ctx context.Context) error {
	collections := []string{"users", "groups", "warnings"}

	for _, collName := range collections {
		collection := im.db.Collection(collName)

		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		_, err := collection.Indexes().DropAll(ctx)
		if err != nil {
			im.logger.Error("Failed to drop indexes", "collection", collName, "error", err)
			return err
		}

		im.logger.Info("Indexes dropped", "collection", collName)
	}

	return nil
}

// ListIndexes 列出所有索引
func (im *IndexManager) ListIndexes(ctx context.Context) (map[string][]string, error) {
	collections := []string{"users", "groups", "warnings"}
	result := make(map[string][]string)

	for _, collName := range collections {
		collection := im.db.Collection(collName)

		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		cursor, err := collection.Indexes().List(ctx)
		if err != nil {
			im.logger.Error("Failed to list indexes", "collection", collName, "error", err)
			return nil, err
		}
		defer cursor.Close(ctx)

		var indexes []bson.M
		if err := cursor.All(ctx, &indexes); err != nil {
			im.logger.Error("Failed to decode indexes", "collection", collName, "error", err)
			return nil, err
		}

		var indexNames []string
		for _, idx := range indexes {
			if name, ok := idx["name"].(string); ok {
				indexNames = append(indexNames, name)
			}
		}

		result[collName] = indexNames
		im.logger.Info("Listed indexes", "collection", collName, "count", len(indexNames))
	}

	return result, nil
}

// GetIndexStats 获取索引统计信息
func (im *IndexManager) GetIndexStats(ctx context.Context, collectionName string) ([]bson.M, error) {
	collection := im.db.Collection(collectionName)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cursor, err := collection.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$indexStats", Value: bson.D{}}},
	})
	if err != nil {
		im.logger.Error("Failed to get index stats", "collection", collectionName, "error", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var stats []bson.M
	if err := cursor.All(ctx, &stats); err != nil {
		im.logger.Error("Failed to decode index stats", "collection", collectionName, "error", err)
		return nil, err
	}

	im.logger.Info("Retrieved index stats", "collection", collectionName, "count", len(stats))
	return stats, nil
}
