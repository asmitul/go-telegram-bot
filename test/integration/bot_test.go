//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"telegram-bot/internal/adapter/repository/mongodb"
	"telegram-bot/internal/commands/ping"
	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	testDB     *mongo.Database
	testClient *mongo.Client
)

// TestMain 测试入口
func TestMain(m *testing.M) {
	// 设置测试环境
	setup()

	// 运行测试
	code := m.Run()

	// 清理
	teardown()

	os.Exit(code)
}

// setup 初始化测试环境
func setup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 连接到测试数据库
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	var err error
	testClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		panic(err)
	}

	// 使用测试数据库
	testDB = testClient.Database("test_telegram_bot")
}

// teardown 清理测试环境
func teardown() {
	if testDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 删除测试数据库
		testDB.Drop(ctx)
	}

	if testClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		testClient.Disconnect(ctx)
	}
}

// TestUserRepository_Integration 用户仓储集成测试
func TestUserRepository_Integration(t *testing.T) {
	// 清理测试数据
	ctx := context.Background()
	testDB.Collection("users").Drop(ctx)

	// 创建仓储
	repo := mongodb.NewUserRepository(testDB)

	// 测试保存用户
	t.Run("Save and FindByID", func(t *testing.T) {
		u := user.NewUser(12345, "testuser", "Test", "User")
		u.SetPermission(123456, user.PermissionAdmin)

		err := repo.Save(u)
		require.NoError(t, err)

		// 查找用户
		found, err := repo.FindByID(12345)
		require.NoError(t, err)
		assert.Equal(t, u.ID, found.ID)
		assert.Equal(t, u.Username, found.Username)
		assert.Equal(t, user.PermissionAdmin, found.GetPermission(123456))
	})

	// 测试更新用户
	t.Run("Update", func(t *testing.T) {
		u, err := repo.FindByID(12345)
		require.NoError(t, err)

		u.SetPermission(123456, user.PermissionSuperAdmin)
		err = repo.Update(u)
		require.NoError(t, err)

		// 验证更新
		updated, err := repo.FindByID(12345)
		require.NoError(t, err)
		assert.Equal(t, user.PermissionSuperAdmin, updated.GetPermission(123456))
	})

	// 测试根据用户名查找
	t.Run("FindByUsername", func(t *testing.T) {
		found, err := repo.FindByUsername("testuser")
		require.NoError(t, err)
		assert.Equal(t, int64(12345), found.ID)
	})

	// 测试删除用户
	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(12345)
		require.NoError(t, err)

		// 验证已删除
		_, err = repo.FindByID(12345)
		assert.Error(t, err)
	})
}

// TestGroupRepository_Integration 群组仓储集成测试
func TestGroupRepository_Integration(t *testing.T) {
	// 清理测试数据
	ctx := context.Background()
	testDB.Collection("groups").Drop(ctx)

	// 创建仓储
	repo := mongodb.NewGroupRepository(testDB)

	// 测试保存群组
	t.Run("Save and FindByID", func(t *testing.T) {
		g := group.NewGroup(123456, "Test Group", "supergroup")
		g.EnableCommand("ping", 12345)

		err := repo.Save(g)
		require.NoError(t, err)

		// 查找群组
		found, err := repo.FindByID(123456)
		require.NoError(t, err)
		assert.Equal(t, g.ID, found.ID)
		assert.Equal(t, g.Title, found.Title)
		assert.True(t, found.IsCommandEnabled("ping"))
	})

	// 测试命令配置
	t.Run("Command Configuration", func(t *testing.T) {
		g, err := repo.FindByID(123456)
		require.NoError(t, err)

		// 禁用命令
		g.DisableCommand("ping", 12345)
		err = repo.Update(g)
		require.NoError(t, err)

		// 验证
		updated, err := repo.FindByID(123456)
		require.NoError(t, err)
		assert.False(t, updated.IsCommandEnabled("ping"))
	})
}

// TestPingCommand_Integration Ping 命令集成测试
func TestPingCommand_Integration(t *testing.T) {
	// 清理测试数据
	ctx := context.Background()
	testDB.Collection("groups").Drop(ctx)

	// 初始化
	groupRepo := mongodb.NewGroupRepository(testDB)
	handler := ping.NewHandler(groupRepo)

	// 创建测试群组
	g := group.NewGroup(123456, "Test Group", "supergroup")
	err := groupRepo.Save(g)
	require.NoError(t, err)

	t.Run("Command is enabled by default", func(t *testing.T) {
		assert.True(t, handler.IsEnabled(123456))
	})

	t.Run("Handle command", func(t *testing.T) {
		cmdCtx := &command.Context{
			Ctx:       context.Background(),
			UserID:    12345,
			GroupID:   123456,
			MessageID: 1,
			Text:      "/ping",
			Args:      []string{},
		}

		err := handler.Handle(cmdCtx)
		assert.NoError(t, err)
	})

	t.Run("Disable and check", func(t *testing.T) {
		g, err := groupRepo.FindByID(123456)
		require.NoError(t, err)

		g.DisableCommand("ping", 12345)
		err = groupRepo.Update(g)
		require.NoError(t, err)

		assert.False(t, handler.IsEnabled(123456))
	})
}

// TestPermissionWorkflow_Integration 权限流程集成测试
func TestPermissionWorkflow_Integration(t *testing.T) {
	// 清理测试数据
	ctx := context.Background()
	testDB.Collection("users").Drop(ctx)
	testDB.Collection("groups").Drop(ctx)

	// 初始化
	userRepo := mongodb.NewUserRepository(testDB)
	groupRepo := mongodb.NewGroupRepository(testDB)

	// 创建测试用户和群组
	admin := user.NewUser(11111, "admin", "Admin", "User")
	admin.SetPermission(123456, user.PermissionSuperAdmin)
	require.NoError(t, userRepo.Save(admin))

	normalUser := user.NewUser(22222, "user", "Normal", "User")
	require.NoError(t, userRepo.Save(normalUser))

	g := group.NewGroup(123456, "Test Group", "supergroup")
	require.NoError(t, groupRepo.Save(g))

	t.Run("Admin can manage commands", func(t *testing.T) {
		// 管理员禁用命令
		g, err := groupRepo.FindByID(123456)
		require.NoError(t, err)

		g.DisableCommand("ban", admin.ID)
		err = groupRepo.Update(g)
		require.NoError(t, err)

		// 验证
		updated, err := groupRepo.FindByID(123456)
		require.NoError(t, err)
		assert.False(t, updated.IsCommandEnabled("ban"))

		config := updated.GetCommandConfig("ban")
		assert.Equal(t, admin.ID, config.UpdatedBy)
	})

	t.Run("Check user permissions", func(t *testing.T) {
		// 检查管理员权限
		assert.True(t, admin.IsSuperAdmin(123456))
		assert.True(t, admin.HasPermission(123456, user.PermissionAdmin))

		// 检查普通用户权限
		assert.False(t, normalUser.IsSuperAdmin(123456))
		assert.False(t, normalUser.HasPermission(123456, user.PermissionAdmin))
		assert.True(t, normalUser.HasPermission(123456, user.PermissionUser))
	})
}
