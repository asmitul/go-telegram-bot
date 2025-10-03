# Repository 开发指南

## 📚 目录

- [概述](#概述)
- [Repository 模式](#repository-模式)
- [快速开始](#快速开始)
- [领域模型详解](#领域模型详解)
- [MongoDB Repository 实现](#mongodb-repository-实现)
- [索引优化](#索引优化)
- [查询优化](#查询优化)
- [测试方法](#测试方法)
- [实际场景示例](#实际场景示例)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)

---

## 概述

**Repository 模式**是一种数据访问模式，将数据访问逻辑与业务逻辑分离，提供统一的接口来操作数据。

### 为什么使用 Repository？

- ✅ **关注点分离**：业务逻辑不关心数据如何存储
- ✅ **易于测试**：可以用 Mock 替换真实数据库
- ✅ **易于切换**：可以轻松从 MongoDB 切换到 PostgreSQL
- ✅ **统一接口**：所有数据访问通过统一的接口
- ✅ **领域驱动**：围绕领域模型设计

### 项目中的 Repository

本项目包含 2 个核心 Repository：

| Repository | 领域模型 | 用途 |
|------------|---------|------|
| `UserRepository` | `User` | 用户数据管理 |
| `GroupRepository` | `Group` | 群组数据管理 |

---

## Repository 模式

### 架构层次

```
Controller/Handler
    ↓
Service/UseCase
    ↓
Repository Interface (Domain Layer)
    ↓
Repository Implementation (Infrastructure Layer)
    ↓
Database (MongoDB)
```

### 分层说明

```
internal/
├── domain/                  # 领域层
│   ├── user/
│   │   ├── user.go          # User 实体
│   │   └── repository.go    # Repository 接口（在同一文件）
│   └── group/
│       ├── group.go         # Group 实体
│       └── repository.go    # Repository 接口（在同一文件）
│
└── adapter/                 # 基础设施层
    └── repository/
        └── mongodb/
            ├── user_repository.go    # UserRepository 实现
            ├── group_repository.go   # GroupRepository 实现
            └── indexes.go            # 索引管理
```

### 依赖关系

```
Handler → Domain (Interface) ← MongoDB (Implementation)
```

**关键原则**：业务逻辑依赖接口，不依赖具体实现。

---

## 快速开始

### 步骤 1：定义领域模型

在 `internal/domain/` 下创建实体和接口。

```go
// internal/domain/myentity/myentity.go
package myentity

import "time"

// MyEntity 实体
type MyEntity struct {
    ID        int64
    Name      string
    CreatedAt time.Time
    UpdatedAt time.Time
}

// NewMyEntity 创建新实体
func NewMyEntity(id int64, name string) *MyEntity {
    now := time.Now()
    return &MyEntity{
        ID:        id,
        Name:      name,
        CreatedAt: now,
        UpdatedAt: now,
    }
}

// Repository 接口
type Repository interface {
    FindByID(id int64) (*MyEntity, error)
    Save(entity *MyEntity) error
    Update(entity *MyEntity) error
    Delete(id int64) error
}
```

### 步骤 2：实现 Repository

在 `internal/adapter/repository/mongodb/` 下创建实现。

```go
// internal/adapter/repository/mongodb/myentity_repository.go
package mongodb

import (
    "context"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "telegram-bot/internal/domain/myentity"
)

type MyEntityRepository struct {
    collection *mongo.Collection
    timeout    time.Duration
}

func NewMyEntityRepository(db *mongo.Database) *MyEntityRepository {
    return &MyEntityRepository{
        collection: db.Collection("my_entities"),
        timeout:    10 * time.Second,
    }
}

// MongoDB 文档结构
type myEntityDocument struct {
    ID        int64     `bson:"_id"`
    Name      string    `bson:"name"`
    CreatedAt time.Time `bson:"created_at"`
    UpdatedAt time.Time `bson:"updated_at"`
}

// 领域对象 → 文档
func (r *MyEntityRepository) toDocument(e *myentity.MyEntity) *myEntityDocument {
    return &myEntityDocument{
        ID:        e.ID,
        Name:      e.Name,
        CreatedAt: e.CreatedAt,
        UpdatedAt: e.UpdatedAt,
    }
}

// 文档 → 领域对象
func (r *MyEntityRepository) toDomain(doc *myEntityDocument) *myentity.MyEntity {
    return &myentity.MyEntity{
        ID:        doc.ID,
        Name:      doc.Name,
        CreatedAt: doc.CreatedAt,
        UpdatedAt: doc.UpdatedAt,
    }
}

func (r *MyEntityRepository) FindByID(id int64) (*myentity.MyEntity, error) {
    ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
    defer cancel()

    var doc myEntityDocument
    err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, myentity.ErrNotFound
        }
        return nil, err
    }

    return r.toDomain(&doc), nil
}

func (r *MyEntityRepository) Save(e *myentity.MyEntity) error {
    ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
    defer cancel()

    doc := r.toDocument(e)
    _, err := r.collection.InsertOne(ctx, doc)
    return err
}
```

### 步骤 3：注册到依赖注入

在 `cmd/bot/main.go` 中创建实例：

```go
// 初始化仓储
myEntityRepo := mongodb.NewMyEntityRepository(db)
```

---

## 领域模型详解

### 1. User 实体

**位置**：`internal/domain/user/user.go`

**核心字段**：
```go
type User struct {
    ID          int64                    // Telegram 用户 ID
    Username    string                   // 用户名
    FirstName   string                   // 名字
    LastName    string                   // 姓氏
    Permissions map[int64]Permission     // 群组ID → 权限级别
    CreatedAt   time.Time                // 创建时间
    UpdatedAt   time.Time                // 更新时间
}
```

**权限系统**：
```go
const (
    PermissionNone Permission = iota   // 0: 无权限
    PermissionUser                     // 1: 普通用户
    PermissionAdmin                    // 2: 管理员
    PermissionSuperAdmin               // 3: 超级管理员
    PermissionOwner                    // 4: 所有者
)
```

**核心方法**：
```go
// 获取权限
func (u *User) GetPermission(groupID int64) Permission

// 设置权限
func (u *User) SetPermission(groupID int64, perm Permission)

// 检查权限
func (u *User) HasPermission(groupID int64, required Permission) bool

// 检查是否为管理员
func (u *User) IsAdmin(groupID int64) bool
```

### 2. Group 实体

**位置**：`internal/domain/group/group.go`

**核心字段**：
```go
type Group struct {
    ID        int64                       // 群组 ID
    Title     string                      // 群组名称
    Type      string                      // 类型："group", "supergroup", "channel"
    Commands  map[string]*CommandConfig   // 命令配置
    Settings  map[string]interface{}      // 自定义设置
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**命令配置**：
```go
type CommandConfig struct {
    CommandName string     // 命令名称
    Enabled     bool       // 是否启用
    UpdatedAt   time.Time  // 更新时间
    UpdatedBy   int64      // 更新者 ID
}
```

**核心方法**：
```go
// 检查命令是否启用
func (g *Group) IsCommandEnabled(commandName string) bool

// 启用命令
func (g *Group) EnableCommand(commandName string, userID int64)

// 禁用命令
func (g *Group) DisableCommand(commandName string, userID int64)

// 获取/设置自定义配置
func (g *Group) GetSetting(key string) (interface{}, bool)
func (g *Group) SetSetting(key string, value interface{})
```

---

## MongoDB Repository 实现

### UserRepository 实现

**位置**：`internal/adapter/repository/mongodb/user_repository.go`

**接口实现**：
```go
type Repository interface {
    FindByID(id int64) (*User, error)           // 按 ID 查找
    FindByUsername(username string) (*User, error) // 按用户名查找
    Save(user *User) error                      // 保存新用户
    Update(user *User) error                    // 更新用户
    Delete(id int64) error                      // 删除用户
    FindAdminsByGroup(groupID int64) ([]*User, error) // 查找群组管理员
}
```

**文档结构**：
```go
type userDocument struct {
    ID          int64         `bson:"_id"`
    Username    string        `bson:"username"`
    FirstName   string        `bson:"first_name"`
    LastName    string        `bson:"last_name"`
    Permissions map[int64]int `bson:"permissions"` // 存储为 int 类型
    CreatedAt   time.Time     `bson:"created_at"`
    UpdatedAt   time.Time     `bson:"updated_at"`
}
```

**关键实现**：

```go
// 查找用户
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

// 更新用户
func (r *UserRepository) Update(u *user.User) error {
    ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
    defer cancel()

    doc := r.toDocument(u)
    filter := bson.M{"_id": u.ID}
    update := bson.M{"$set": doc}

    _, err := r.collection.UpdateOne(ctx, filter, update)
    return err
}
```

### GroupRepository 实现

**位置**：`internal/adapter/repository/mongodb/group_repository.go`

**接口实现**：
```go
type Repository interface {
    FindByID(id int64) (*Group, error)
    Save(group *Group) error
    Update(group *Group) error
    Delete(id int64) error
    FindAll() ([]*Group, error)
}
```

**文档结构**：
```go
type groupDocument struct {
    ID        int64                        `bson:"_id"`
    Title     string                       `bson:"title"`
    Type      string                       `bson:"type"`
    Commands  map[string]*commandConfigDoc `bson:"commands"`
    Settings  map[string]interface{}       `bson:"settings"`
    CreatedAt time.Time                    `bson:"created_at"`
    UpdatedAt time.Time                    `bson:"updated_at"`
}
```

---

## 索引优化

### IndexManager

**位置**：`internal/adapter/repository/mongodb/indexes.go`

**作用**：统一管理所有集合的索引。

### 用户集合索引

```go
func (im *IndexManager) ensureUserIndexes(ctx context.Context) error {
    indexes := []mongo.IndexModel{
        {
            // 主键索引（自动创建，但显式定义）
            Keys:    bson.D{{Key: "_id", Value: 1}},
            Options: options.Index().SetName("idx_user_id").SetUnique(true),
        },
        {
            // 用户名索引（用于快速查找）
            Keys:    bson.D{{Key: "username", Value: 1}},
            Options: options.Index().SetName("idx_username").SetSparse(true),
        },
        {
            // 组合索引：权限查询优化
            Keys: bson.D{
                {Key: "permissions", Value: 1},
                {Key: "updated_at", Value: -1},
            },
            Options: options.Index().SetName("idx_permissions_updated"),
        },
        {
            // 创建时间索引（用于统计和清理）
            Keys:    bson.D{{Key: "created_at", Value: -1}},
            Options: options.Index().SetName("idx_created_at"),
        },
    }

    return im.createIndexes(ctx, collection, indexes, "users")
}
```

### 群组集合索引

```go
func (im *IndexManager) ensureGroupIndexes(ctx context.Context) error {
    indexes := []mongo.IndexModel{
        {
            Keys:    bson.D{{Key: "_id", Value: 1}},
            Options: options.Index().SetName("idx_group_id").SetUnique(true),
        },
        {
            Keys:    bson.D{{Key: "title", Value: 1}},
            Options: options.Index().SetName("idx_group_title"),
        },
        {
            Keys:    bson.D{{Key: "type", Value: 1}},
            Options: options.Index().SetName("idx_group_type"),
        },
        {
            Keys:    bson.D{{Key: "updated_at", Value: -1}},
            Options: options.Index().SetName("idx_group_updated_at"),
        },
    }

    return im.createIndexes(ctx, collection, indexes, "groups")
}
```

### TTL 索引（自动清理）

```go
func (im *IndexManager) ensureWarningIndexes(ctx context.Context) error {
    indexes := []mongo.IndexModel{
        {
            // TTL 索引：90天后自动删除
            Keys: bson.D{{Key: "created_at", Value: 1}},
            Options: options.Index().
                SetName("idx_warning_ttl").
                SetExpireAfterSeconds(90 * 24 * 60 * 60),
        },
    }
    // ...
}
```

### 索引管理工具

```go
// 创建所有索引
indexManager.EnsureIndexes(ctx)

// 列出所有索引
indexes, _ := indexManager.ListIndexes(ctx)

// 获取索引统计
stats, _ := indexManager.GetIndexStats(ctx, "users")

// 删除所有索引（重建时使用）
indexManager.DropAllIndexes(ctx)
```

---

## 查询优化

### 1. 使用索引

```go
// ✅ 好：使用索引字段查询
r.collection.FindOne(ctx, bson.M{"_id": userID})
r.collection.FindOne(ctx, bson.M{"username": "john"})

// ❌ 坏：查询未索引字段
r.collection.FindOne(ctx, bson.M{"first_name": "John"})
```

### 2. 投影（Projection）

```go
// ✅ 好：只查询需要的字段
opts := options.FindOne().SetProjection(bson.M{
    "username": 1,
    "first_name": 1,
})
r.collection.FindOne(ctx, filter, opts)

// ❌ 坏：查询所有字段
r.collection.FindOne(ctx, filter)
```

### 3. 批量操作

```go
// ✅ 好：使用 BulkWrite
bulkOps := []mongo.WriteModel{
    mongo.NewInsertOneModel().SetDocument(doc1),
    mongo.NewInsertOneModel().SetDocument(doc2),
}
r.collection.BulkWrite(ctx, bulkOps)

// ❌ 坏：循环插入
for _, doc := range docs {
    r.collection.InsertOne(ctx, doc)
}
```

### 4. 分页查询

```go
func (r *UserRepository) FindPaginated(page, pageSize int) ([]*user.User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
    defer cancel()

    skip := (page - 1) * pageSize
    opts := options.Find().
        SetSkip(int64(skip)).
        SetLimit(int64(pageSize)).
        SetSort(bson.D{{Key: "created_at", Value: -1}})

    cursor, err := r.collection.Find(ctx, bson.M{}, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var users []*user.User
    for cursor.Next(ctx) {
        var doc userDocument
        if err := cursor.Decode(&doc); err != nil {
            return nil, err
        }
        users = append(users, r.toDomain(&doc))
    }

    return users, cursor.Err()
}
```

### 5. 聚合查询

```go
func (r *UserRepository) GetUserStats() (map[string]int64, error) {
    ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
    defer cancel()

    pipeline := mongo.Pipeline{
        // 按权限分组统计
        {{Key: "$group", Value: bson.D{
            {Key: "_id", Value: "$permissions"},
            {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
        }}},
    }

    cursor, err := r.collection.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    stats := make(map[string]int64)
    for cursor.Next(ctx) {
        var result bson.M
        cursor.Decode(&result)
        // 处理结果
    }

    return stats, nil
}
```

---

## 测试方法

### 1. 使用 Mock

创建 Mock Repository：

```go
// test/mocks/user_repository_mock.go
type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) FindByID(id int64) (*user.User, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) Save(u *user.User) error {
    args := m.Called(u)
    return args.Error(0)
}
```

使用 Mock 测试：

```go
func TestMyService(t *testing.T) {
    mockRepo := new(MockUserRepository)

    // 设置期望
    expectedUser := &user.User{ID: 123, Username: "test"}
    mockRepo.On("FindByID", int64(123)).Return(expectedUser, nil)

    // 测试
    service := NewService(mockRepo)
    result, err := service.GetUser(123)

    // 验证
    assert.NoError(t, err)
    assert.Equal(t, expectedUser, result)
    mockRepo.AssertExpectations(t)
}
```

### 2. 集成测试

使用真实 MongoDB 测试：

```go
//go:build integration
// +build integration

func TestUserRepository_Integration(t *testing.T) {
    // 连接测试数据库
    client, _ := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb+srv://user:pass@cluster.mongodb.net/"))
    db := client.Database("telegram_bot_test")
    defer db.Drop(context.Background())

    repo := mongodb.NewUserRepository(db)

    // 测试保存
    u := user.NewUser(123, "test", "Test", "User")
    err := repo.Save(u)
    assert.NoError(t, err)

    // 测试查找
    found, err := repo.FindByID(123)
    assert.NoError(t, err)
    assert.Equal(t, u.Username, found.Username)
}
```

---

## 实际场景示例

### 场景 1：复杂查询

```go
func (r *UserRepository) FindActiveAdmins(groupID int64, days int) ([]*user.User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
    defer cancel()

    cutoffTime := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

    filter := bson.M{
        "$and": []bson.M{
            // 权限 >= Admin
            {fmt.Sprintf("permissions.%d", groupID): bson.M{"$gte": int(user.PermissionAdmin)}},
            // 最近活跃
            {"updated_at": bson.M{"$gte": cutoffTime}},
        },
    }

    cursor, err := r.collection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var users []*user.User
    for cursor.Next(ctx) {
        var doc userDocument
        if err := cursor.Decode(&doc); err != nil {
            continue
        }
        users = append(users, r.toDomain(&doc))
    }

    return users, nil
}
```

### 场景 2：事务操作

```go
func (r *UserRepository) TransferPermission(fromUserID, toUserID int64, groupID int64) error {
    session, err := r.collection.Database().Client().StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(context.Background())

    callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
        // 1. 读取源用户
        fromUser, err := r.FindByID(fromUserID)
        if err != nil {
            return nil, err
        }

        // 2. 读取目标用户
        toUser, err := r.FindByID(toUserID)
        if err != nil {
            return nil, err
        }

        // 3. 转移权限
        perm := fromUser.GetPermission(groupID)
        toUser.SetPermission(groupID, perm)
        fromUser.SetPermission(groupID, user.PermissionUser)

        // 4. 更新两个用户
        if err := r.Update(fromUser); err != nil {
            return nil, err
        }
        if err := r.Update(toUser); err != nil {
            return nil, err
        }

        return nil, nil
    }

    _, err = session.WithTransaction(context.Background(), callback)
    return err
}
```

### 场景 3：软删除

```go
// 在 User 实体中添加字段
type User struct {
    // ... 其他字段
    DeletedAt *time.Time `bson:"deleted_at,omitempty"`
}

// 软删除
func (r *UserRepository) SoftDelete(id int64) error {
    ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
    defer cancel()

    now := time.Now()
    filter := bson.M{"_id": id}
    update := bson.M{"$set": bson.M{"deleted_at": now}}

    _, err := r.collection.UpdateOne(ctx, filter, update)
    return err
}

// 查询时过滤已删除
func (r *UserRepository) FindByID(id int64) (*user.User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
    defer cancel()

    filter := bson.M{
        "_id":        id,
        "deleted_at": nil, // 只查询未删除的
    }

    var doc userDocument
    err := r.collection.FindOne(ctx, filter).Decode(&doc)
    // ...
}
```

---

## 最佳实践

### 1. 使用超时 Context

```go
// ✅ 好
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := r.collection.FindOne(ctx, filter).Decode(&doc)

// ❌ 坏
err := r.collection.FindOne(context.Background(), filter).Decode(&doc)
```

### 2. 处理错误

```go
// ✅ 好：区分不同错误
err := r.collection.FindOne(ctx, filter).Decode(&doc)
if err != nil {
    if err == mongo.ErrNoDocuments {
        return nil, user.ErrUserNotFound // 返回领域错误
    }
    return nil, err // 返回数据库错误
}

// ❌ 坏：直接返回 MongoDB 错误
return nil, err
```

### 3. 分离领域对象和文档

```go
// ✅ 好：使用单独的文档结构
type userDocument struct {
    ID          int64         `bson:"_id"`
    Permissions map[int64]int `bson:"permissions"` // MongoDB 格式
}

// 转换函数
func (r *UserRepository) toDomain(doc *userDocument) *user.User {
    perms := make(map[int64]user.Permission)
    for groupID, perm := range doc.Permissions {
        perms[groupID] = user.Permission(perm)
    }
    return &user.User{Permissions: perms}
}
```

### 4. 索引策略

```go
// ✅ 好：创建复合索引
{
    Keys: bson.D{
        {Key: "user_id", Value: 1},
        {Key: "group_id", Value: 1},
        {Key: "created_at", Value: -1},
    },
}

// 可以优化这些查询：
// - {user_id: X}
// - {user_id: X, group_id: Y}
// - {user_id: X, group_id: Y, created_at: ...}
```

### 5. 使用 Upsert

```go
func (r *UserRepository) SaveOrUpdate(u *user.User) error {
    ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
    defer cancel()

    doc := r.toDocument(u)
    filter := bson.M{"_id": u.ID}
    update := bson.M{"$set": doc}
    opts := options.Update().SetUpsert(true)

    _, err := r.collection.UpdateOne(ctx, filter, update, opts)
    return err
}
```

### 6. 游标关闭

```go
// ✅ 好：使用 defer 确保关闭
cursor, err := r.collection.Find(ctx, filter)
if err != nil {
    return nil, err
}
defer cursor.Close(ctx)

// ❌ 坏：忘记关闭游标
cursor, _ := r.collection.Find(ctx, filter)
for cursor.Next(ctx) { /* ... */ }
// 泄漏资源
```

---

## 常见问题

### Q1：为什么要分离领域对象和文档结构？

**原因**：
- ✅ 领域对象使用 Go 类型（如 `user.Permission` 枚举）
- ✅ MongoDB 文档使用简单类型（如 `int`）
- ✅ 易于迁移数据库
- ✅ 领域逻辑不依赖存储细节

### Q2：应该在 Repository 中包含业务逻辑吗？

**不应该**。Repository 只负责数据访问，业务逻辑应该在：
- **领域模型**：`user.HasPermission()`
- **Service/UseCase 层**：复杂的业务流程

```go
// ✅ 好：Repository 只做数据访问
func (r *UserRepository) FindByID(id int64) (*user.User, error)

// ❌ 坏：Repository 包含业务逻辑
func (r *UserRepository) PromoteToAdmin(id int64) error
```

### Q3：如何测试 Repository？

**两种方式**：

1. **Mock**（单元测试）：快速，隔离
2. **真实数据库**（集成测试）：准确，慢

推荐两者结合使用。

### Q4：应该使用 ORM 吗（如 GORM）？

**本项目不使用 ORM**，原因：
- ✅ MongoDB 驱动已经很简洁
- ✅ 更好的性能
- ✅ 更灵活的查询
- ✅ 更清晰的数据流

### Q5：如何处理大量数据？

```go
// 使用游标分批处理
func (r *UserRepository) ProcessAllUsers(callback func(*user.User) error) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()

    cursor, err := r.collection.Find(ctx, bson.M{})
    if err != nil {
        return err
    }
    defer cursor.Close(ctx)

    for cursor.Next(ctx) {
        var doc userDocument
        if err := cursor.Decode(&doc); err != nil {
            continue
        }

        u := r.toDomain(&doc)
        if err := callback(u); err != nil {
            return err
        }
    }

    return cursor.Err()
}
```

---

## 附录

### 相关资源

- [User Domain 模型](../internal/domain/user/user.go)
- [Group Domain 模型](../internal/domain/group/group.go)
- [UserRepository 实现](../internal/adapter/repository/mongodb/user_repository.go)
- [GroupRepository 实现](../internal/adapter/repository/mongodb/group_repository.go)
- [索引管理器](../internal/adapter/repository/mongodb/indexes.go)

### 相关文档

- [项目快速入门](./getting-started.md)
- [中间件开发指南](./middleware-guide.md)
- [架构总览](../CLAUDE.md)

### 扩展阅读

- [MongoDB Go Driver 文档](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo)
- [Repository 模式](https://martinfowler.com/eaaCatalog/repository.html)
- [领域驱动设计](https://domainlanguage.com/ddd/)

---

**编写日期**: 2025-10-02
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team
