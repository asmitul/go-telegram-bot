package mongodb

import (
	"context"
	"telegram-bot/internal/domain/group"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// GroupRepository MongoDB 群组仓储实现
type GroupRepository struct {
	collection *mongo.Collection
	timeout    time.Duration
}

// NewGroupRepository 创建 MongoDB 群组仓储
func NewGroupRepository(db *mongo.Database) *GroupRepository {
	return &GroupRepository{
		collection: db.Collection("groups"),
		timeout:    10 * time.Second,
	}
}

// groupDocument MongoDB 文档结构
type groupDocument struct {
	ID        int64                        `bson:"_id"`
	Title     string                       `bson:"title"`
	Type      string                       `bson:"type"`
	Commands  map[string]*commandConfigDoc `bson:"commands"`
	Settings  map[string]interface{}       `bson:"settings"`
	CreatedAt time.Time                    `bson:"created_at"`
	UpdatedAt time.Time                    `bson:"updated_at"`
}

// commandConfigDoc 命令配置文档
type commandConfigDoc struct {
	CommandName string    `bson:"command_name"`
	Enabled     bool      `bson:"enabled"`
	UpdatedAt   time.Time `bson:"updated_at"`
	UpdatedBy   int64     `bson:"updated_by"`
}

// toDocument 将领域对象转换为文档
func (r *GroupRepository) toDocument(g *group.Group) *groupDocument {
	commands := make(map[string]*commandConfigDoc)
	for name, config := range g.Commands {
		commands[name] = &commandConfigDoc{
			CommandName: config.CommandName,
			Enabled:     config.Enabled,
			UpdatedAt:   config.UpdatedAt,
			UpdatedBy:   config.UpdatedBy,
		}
	}

	return &groupDocument{
		ID:        g.ID,
		Title:     g.Title,
		Type:      g.Type,
		Commands:  commands,
		Settings:  g.Settings,
		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}
}

// toDomain 将文档转换为领域对象
func (r *GroupRepository) toDomain(doc *groupDocument) *group.Group {
	commands := make(map[string]*group.CommandConfig)
	for name, config := range doc.Commands {
		commands[name] = &group.CommandConfig{
			CommandName: config.CommandName,
			Enabled:     config.Enabled,
			UpdatedAt:   config.UpdatedAt,
			UpdatedBy:   config.UpdatedBy,
		}
	}

	return &group.Group{
		ID:        doc.ID,
		Title:     doc.Title,
		Type:      doc.Type,
		Commands:  commands,
		Settings:  doc.Settings,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
}

// FindByID 根据 ID 查找群组
func (r *GroupRepository) FindByID(ctx context.Context, id int64) (*group.Group, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var doc groupDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, group.ErrGroupNotFound
		}
		return nil, err
	}

	return r.toDomain(&doc), nil
}

// Save 保存群组
func (r *GroupRepository) Save(ctx context.Context, g *group.Group) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	doc := r.toDocument(g)
	_, err := r.collection.InsertOne(ctx, doc)
	return err
}

// Update 更新群组
func (r *GroupRepository) Update(ctx context.Context, g *group.Group) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	doc := r.toDocument(g)
	filter := bson.M{"_id": g.ID}
	update := bson.M{"$set": doc}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return group.ErrGroupNotFound
	}

	return nil
}

// Delete 删除群组
func (r *GroupRepository) Delete(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// FindAll 查找所有群组
func (r *GroupRepository) FindAll(ctx context.Context) ([]*group.Group, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var groups []*group.Group
	for cursor.Next(ctx) {
		var doc groupDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		groups = append(groups, r.toDomain(&doc))
	}

	return groups, cursor.Err()
}

// EnsureIndexes 确保索引存在
func (r *GroupRepository) EnsureIndexes(ctx context.Context) error {
	// 创建索引以提高查询性能
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "title", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
	})
	return err
}
