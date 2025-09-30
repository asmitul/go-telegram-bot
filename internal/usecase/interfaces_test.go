package usecase

import (
	"context"
	"testing"

	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	userUseCase "telegram-bot/internal/usecase/user"
	groupUseCase "telegram-bot/internal/usecase/group"
)

// 验证适配器实现接口
var _ UserManagement = (*userManagementAdapter)(nil)
var _ GroupCommandConfig = (*groupCommandConfigAdapter)(nil)
var _ GroupConfig = (*groupConfigAdapter)(nil)

// mockUserRepository 模拟用户仓储
type mockUserRepository struct {
	users map[int64]*user.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[int64]*user.User),
	}
}

func (m *mockUserRepository) FindByID(id int64) (*user.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepository) FindByUsername(username string) (*user.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepository) Save(u *user.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepository) Update(u *user.User) error {
	if _, ok := m.users[u.ID]; !ok {
		return user.ErrUserNotFound
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepository) Delete(id int64) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepository) FindAdminsByGroup(groupID int64) ([]*user.User, error) {
	admins := make([]*user.User, 0)
	for _, u := range m.users {
		if u.GetPermission(groupID) >= user.PermissionAdmin {
			admins = append(admins, u)
		}
	}
	return admins, nil
}

// mockGroupRepository 模拟群组仓储
type mockGroupRepository struct {
	groups map[int64]*group.Group
}

func newMockGroupRepository() *mockGroupRepository {
	return &mockGroupRepository{
		groups: make(map[int64]*group.Group),
	}
}

func (m *mockGroupRepository) FindByID(id int64) (*group.Group, error) {
	if g, ok := m.groups[id]; ok {
		return g, nil
	}
	return nil, group.ErrGroupNotFound
}

func (m *mockGroupRepository) Save(g *group.Group) error {
	m.groups[g.ID] = g
	return nil
}

func (m *mockGroupRepository) Update(g *group.Group) error {
	if _, ok := m.groups[g.ID]; !ok {
		return group.ErrGroupNotFound
	}
	m.groups[g.ID] = g
	return nil
}

func (m *mockGroupRepository) Delete(id int64) error {
	delete(m.groups, id)
	return nil
}

func (m *mockGroupRepository) FindAll() ([]*group.Group, error) {
	groups := make([]*group.Group, 0, len(m.groups))
	for _, g := range m.groups {
		groups = append(groups, g)
	}
	return groups, nil
}

func setupTestData() (*mockUserRepository, *mockGroupRepository) {
	userRepo := newMockUserRepository()
	groupRepo := newMockGroupRepository()

	// 创建测试用户
	admin := user.NewUser(1, "admin", "Admin", "User")
	admin.SetPermission(-100, user.PermissionAdmin)
	userRepo.Save(admin)

	normalUser := user.NewUser(2, "user", "Normal", "User")
	normalUser.SetPermission(-100, user.PermissionUser)
	userRepo.Save(normalUser)

	// 创建测试群组
	testGroup := group.NewGroup(-100, "Test Group", "supergroup")
	groupRepo.Save(testGroup)

	return userRepo, groupRepo
}

func TestUserManagementAdapter(t *testing.T) {
	userRepo, _ := setupTestData()

	checkPermUC := userUseCase.NewCheckPermissionUseCase(userRepo)
	manageAdminUC := userUseCase.NewManageAdminUseCase(userRepo)

	adapter := NewUserManagementAdapter(checkPermUC, manageAdminUC)

	ctx := context.Background()

	// 测试 PromoteAdmin
	t.Run("PromoteAdmin", func(t *testing.T) {
		err := adapter.PromoteAdmin(ctx, userUseCase.PromoteAdminInput{
			OperatorID: 1,
			TargetID:   2,
			GroupID:    -100,
			Permission: user.PermissionAdmin,
		})
		if err != nil {
			t.Errorf("PromoteAdmin() error = %v", err)
		}
	})

	// 测试 ListAdmins
	t.Run("ListAdmins", func(t *testing.T) {
		output, err := adapter.ListAdmins(ctx, 1, -100)
		if err != nil {
			t.Errorf("ListAdmins() error = %v", err)
		}
		if output.Total < 2 {
			t.Errorf("expected at least 2 admins, got %d", output.Total)
		}
	})

	// 测试 CheckPermission
	t.Run("CheckPermission", func(t *testing.T) {
		err := adapter.CheckPermission(ctx, 1, -100, user.PermissionAdmin)
		if err != nil {
			t.Errorf("CheckPermission() error = %v", err)
		}
	})

	// 测试 GetUserPermission
	t.Run("GetUserPermission", func(t *testing.T) {
		perm, err := adapter.GetUserPermission(ctx, 1, -100)
		if err != nil {
			t.Errorf("GetUserPermission() error = %v", err)
		}
		if perm != user.PermissionAdmin {
			t.Errorf("expected PermissionAdmin, got %v", perm)
		}
	})

	// 测试 IsAdmin
	t.Run("IsAdmin", func(t *testing.T) {
		isAdmin, err := adapter.IsAdmin(ctx, 1, -100)
		if err != nil {
			t.Errorf("IsAdmin() error = %v", err)
		}
		if !isAdmin {
			t.Error("expected user to be admin")
		}
	})
}

func TestGroupCommandConfigAdapter(t *testing.T) {
	userRepo, groupRepo := setupTestData()

	configureCommandUC := groupUseCase.NewConfigureCommandUseCase(groupRepo, userRepo)
	adapter := NewGroupCommandConfigAdapter(configureCommandUC)

	ctx := context.Background()

	// 测试 EnableCommand
	t.Run("EnableCommand", func(t *testing.T) {
		err := adapter.EnableCommand(ctx, groupUseCase.EnableCommandInput{
			OperatorID:  1,
			GroupID:     -100,
			CommandName: "/test",
		})
		if err != nil {
			t.Errorf("EnableCommand() error = %v", err)
		}
	})

	// 测试 GetCommandStatus
	t.Run("GetCommandStatus", func(t *testing.T) {
		output, err := adapter.GetCommandStatus(ctx, groupUseCase.GetCommandStatusInput{
			OperatorID:  1,
			GroupID:     -100,
			CommandName: "/test",
		})
		if err != nil {
			t.Errorf("GetCommandStatus() error = %v", err)
		}
		if !output.Enabled {
			t.Error("expected command to be enabled")
		}
	})

	// 测试 ListCommands
	t.Run("ListCommands", func(t *testing.T) {
		output, err := adapter.ListCommands(ctx, 1, -100)
		if err != nil {
			t.Errorf("ListCommands() error = %v", err)
		}
		if output.Total == 0 {
			t.Error("expected at least 1 command")
		}
	})
}

func TestGroupConfigAdapter(t *testing.T) {
	userRepo, groupRepo := setupTestData()

	getConfigUC := groupUseCase.NewGetConfigUseCase(groupRepo, userRepo)
	adapter := NewGroupConfigAdapter(getConfigUC)

	ctx := context.Background()

	// 测试 GetGroupConfig
	t.Run("GetGroupConfig", func(t *testing.T) {
		output, err := adapter.GetGroupConfig(ctx, 1, -100)
		if err != nil {
			t.Errorf("GetGroupConfig() error = %v", err)
		}
		if output.GroupID != -100 {
			t.Errorf("expected group id -100, got %d", output.GroupID)
		}
	})

	// 测试 SetGroupSetting
	t.Run("SetGroupSetting", func(t *testing.T) {
		err := adapter.SetGroupSetting(ctx, 1, -100, "test_key", "test_value")
		if err != nil {
			t.Errorf("SetGroupSetting() error = %v", err)
		}
	})

	// 测试 GetGroupSetting
	t.Run("GetGroupSetting", func(t *testing.T) {
		value, err := adapter.GetGroupSetting(ctx, 1, -100, "test_key")
		if err != nil {
			t.Errorf("GetGroupSetting() error = %v", err)
		}
		if value != "test_value" {
			t.Errorf("expected 'test_value', got %v", value)
		}
	})

	// 测试 UpdateGroupSettings
	t.Run("UpdateGroupSettings", func(t *testing.T) {
		err := adapter.UpdateGroupSettings(ctx, groupUseCase.UpdateGroupSettingsInput{
			OperatorID: 1,
			GroupID:    -100,
			Settings: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
		})
		if err != nil {
			t.Errorf("UpdateGroupSettings() error = %v", err)
		}
	})
}

func TestNewUseCases(t *testing.T) {
	userRepo, groupRepo := setupTestData()

	// 创建所有用例
	checkPermUC := userUseCase.NewCheckPermissionUseCase(userRepo)
	manageAdminUC := userUseCase.NewManageAdminUseCase(userRepo)
	configureCommandUC := groupUseCase.NewConfigureCommandUseCase(groupRepo, userRepo)
	getConfigUC := groupUseCase.NewGetConfigUseCase(groupRepo, userRepo)

	// 创建适配器
	userManagement := NewUserManagementAdapter(checkPermUC, manageAdminUC)
	groupCommandConfig := NewGroupCommandConfigAdapter(configureCommandUC)
	groupConfig := NewGroupConfigAdapter(getConfigUC)

	// 创建用例聚合
	useCases := NewUseCases(userManagement, groupCommandConfig, groupConfig)

	if useCases == nil {
		t.Fatal("NewUseCases() returned nil")
	}

	if useCases.UserManagement == nil {
		t.Error("UserManagement is nil")
	}

	if useCases.GroupCommandConfig == nil {
		t.Error("GroupCommandConfig is nil")
	}

	if useCases.GroupConfig == nil {
		t.Error("GroupConfig is nil")
	}
}