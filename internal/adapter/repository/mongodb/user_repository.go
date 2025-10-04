package mongodb

import (
	"context"
	"telegram-bot/internal/domain/user"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserRepository MongoDB 用户仓储实现
type UserRepository struct {
	collection *mongo.Collection
	timeout    time.Duration
}

// NewUserRepository 创建 MongoDB 用户仓储
func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		collection: db.Collection("users"),
		timeout:    10 * time.Second,
	}
}

// userDocument MongoDB 文档结构
type userDocument struct {
	ID          int64         `bson:"_id"`
	Username    string        `bson:"username"`
	FirstName   string        `bson:"first_name"`
	LastName    string        `bson:"last_name"`
	Permissions map[int64]int `bson:"permissions"` // groupID -> permission level
	CreatedAt   time.Time     `bson:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at"`
}

// toDocument 将领域对象转换为文档
func (r *UserRepository) toDocument(u *user.User) *userDocument {
	perms := make(map[int64]int)
	for groupID, perm := range u.Permissions {
		perms[groupID] = int(perm)
	}

	return &userDocument{
		ID:          u.ID,
		Username:    u.Username,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Permissions: perms,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// toDomain 将文档转换为领域对象
func (r *UserRepository) toDomain(doc *userDocument) *user.User {
	perms := make(map[int64]user.Permission)
	for groupID, perm := range doc.Permissions {
		perms[groupID] = user.Permission(perm)
	}

	return &user.User{
		ID:          doc.ID,
		Username:    doc.Username,
		FirstName:   doc.FirstName,
		LastName:    doc.LastName,
		Permissions: perms,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}

// FindByID 根据 ID 查找用户
func (r *UserRepository) FindByID(id int64) (*user.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	var doc userDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}

	return r.toDomain(&doc), nil
}

// FindByUsername 根据用户名查找用户
func (r *UserRepository) FindByUsername(username string) (*user.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	var doc userDocument
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}

	return r.toDomain(&doc), nil
}

// Save 保存用户
func (r *UserRepository) Save(u *user.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	doc := r.toDocument(u)
	_, err := r.collection.InsertOne(ctx, doc)
	return err
}

// Update 更新用户
func (r *UserRepository) Update(u *user.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	doc := r.toDocument(u)
	filter := bson.M{"_id": u.ID}
	update := bson.M{"$set": doc}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// Delete 删除用户
func (r *UserRepository) Delete(id int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// FindAdminsByGroup 查找群组的所有管理员
func (r *UserRepository) FindAdminsByGroup(groupID int64) ([]*user.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	// 获取所有用户，然后在应用层过滤
	// 这比使用复杂的 MongoDB 查询更简单可靠
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var admins []*user.User
	for cursor.Next(ctx) {
		var doc userDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}

		// 转换为领域对象
		u := r.toDomain(&doc)

		// 检查该用户在指定群组中是否有管理员权限
		if u.GetPermission(groupID) >= user.PermissionAdmin {
			admins = append(admins, u)
		}
	}

	return admins, cursor.Err()
}
