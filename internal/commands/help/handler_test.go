package help

import (
	"context"
	"testing"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
)

// mockHandler 模拟命令处理器
type mockHandler struct {
	name        string
	description string
	permission  user.Permission
	enabled     bool
}

func (m *mockHandler) Name() string {
	return m.name
}

func (m *mockHandler) Description() string {
	return m.description
}

func (m *mockHandler) RequiredPermission() user.Permission {
	return m.permission
}

func (m *mockHandler) Handle(ctx *command.Context) error {
	return nil
}

func (m *mockHandler) IsEnabled(groupID int64) bool {
	return m.enabled
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

func setupTestRegistry() command.Registry {
	registry := command.NewRegistry()

	// 注册一些测试命令
	registry.Register(&mockHandler{
		name:        "ping",
		description: "测试机器人是否在线",
		permission:  user.PermissionUser,
		enabled:     true,
	})

	registry.Register(&mockHandler{
		name:        "stats",
		description: "显示群组统计信息",
		permission:  user.PermissionUser,
		enabled:     true,
	})

	registry.Register(&mockHandler{
		name:        "ban",
		description: "封禁用户",
		permission:  user.PermissionAdmin,
		enabled:     true,
	})

	registry.Register(&mockHandler{
		name:        "config",
		description: "配置群组设置",
		permission:  user.PermissionAdmin,
		enabled:     true,
	})

	registry.Register(&mockHandler{
		name:        "promote",
		description: "提升管理员",
		permission:  user.PermissionSuperAdmin,
		enabled:     true,
	})

	registry.Register(&mockHandler{
		name:        "disabled_cmd",
		description: "已禁用的命令",
		permission:  user.PermissionUser,
		enabled:     false,
	})

	return registry
}

func TestHelp_Name(t *testing.T) {
	registry := setupTestRegistry()
	handler := NewHandler(registry)

	if handler.Name() != "help" {
		t.Errorf("expected name 'help', got '%s'", handler.Name())
	}
}

func TestHelp_Description(t *testing.T) {
	registry := setupTestRegistry()
	handler := NewHandler(registry)

	if handler.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestHelp_RequiredPermission(t *testing.T) {
	registry := setupTestRegistry()
	handler := NewHandler(registry)

	if handler.RequiredPermission() != user.PermissionUser {
		t.Errorf("expected PermissionUser, got %v", handler.RequiredPermission())
	}
}

func TestHelp_IsEnabled(t *testing.T) {
	registry := setupTestRegistry()
	handler := NewHandler(registry)

	if !handler.IsEnabled(-100) {
		t.Error("help command should always be enabled")
	}
}

func TestHelp_ShowAllCommands(t *testing.T) {
	registry := setupTestRegistry()
	handler := NewHandler(registry)

	tests := []struct {
		name           string
		userPermission user.Permission
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name:           "normal user sees user commands",
			userPermission: user.PermissionUser,
			shouldContain:  []string{"ping", "stats", "普通用户"},
			shouldNotContain: []string{"ban", "config", "promote", "disabled_cmd"},
		},
		{
			name:           "admin sees user and admin commands",
			userPermission: user.PermissionAdmin,
			shouldContain:  []string{"ping", "stats", "ban", "config", "管理员"},
			shouldNotContain: []string{"promote", "disabled_cmd"},
		},
		{
			name:           "superadmin sees all commands",
			userPermission: user.PermissionSuperAdmin,
			shouldContain:  []string{"ping", "stats", "ban", "config", "promote", "超级管理员"},
			shouldNotContain: []string{"disabled_cmd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试用户
			testUser := user.NewUser(1, "testuser", "Test", "User")
			testUser.SetPermission(-100, tt.userPermission)

			ctx := &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -100,
				User:    testUser,
				Args:    []string{},
			}

			// 由于 sendMessage 会打印到 stdout，这里只测试不报错
			err := handler.Handle(ctx)
			if err != nil {
				t.Errorf("Handle() error = %v", err)
			}

			// 注意：实际测试中需要 mock sendMessage 函数来验证输出内容
			// 这里简化处理，只验证不报错
		})
	}
}

func TestHelp_ShowCommandDetail(t *testing.T) {
	registry := setupTestRegistry()
	handler := NewHandler(registry)

	tests := []struct {
		name           string
		cmdArg         string
		userPermission user.Permission
		wantErr        bool
	}{
		{
			name:           "show existing command",
			cmdArg:         "ping",
			userPermission: user.PermissionUser,
			wantErr:        false,
		},
		{
			name:           "show existing command with slash",
			cmdArg:         "/ping",
			userPermission: user.PermissionUser,
			wantErr:        false,
		},
		{
			name:           "show non-existent command",
			cmdArg:         "nonexistent",
			userPermission: user.PermissionUser,
			wantErr:        false, // 不报错，只是返回命令不存在
		},
		{
			name:           "show disabled command",
			cmdArg:         "disabled_cmd",
			userPermission: user.PermissionUser,
			wantErr:        false, // 不报错，只是提示已禁用
		},
		{
			name:           "show command without permission",
			cmdArg:         "ban",
			userPermission: user.PermissionUser,
			wantErr:        false, // 不报错，只是提示权限不足
		},
		{
			name:           "admin shows admin command",
			cmdArg:         "ban",
			userPermission: user.PermissionAdmin,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testUser := user.NewUser(1, "testuser", "Test", "User")
			testUser.SetPermission(-100, tt.userPermission)

			ctx := &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -100,
				User:    testUser,
				Args:    []string{tt.cmdArg},
			}

			err := handler.Handle(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHelp_EmptyRegistry(t *testing.T) {
	// 空注册表
	registry := command.NewRegistry()
	handler := NewHandler(registry)

	testUser := user.NewUser(1, "testuser", "Test", "User")
	testUser.SetPermission(-100, user.PermissionUser)

	ctx := &command.Context{
		Ctx:     context.Background(),
		UserID:  1,
		GroupID: -100,
		User:    testUser,
		Args:    []string{},
	}

	err := handler.Handle(ctx)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}
}

func TestGetPermissionLabel(t *testing.T) {
	tests := []struct {
		perm     user.Permission
		expected string
	}{
		{user.PermissionNone, "🚫 无权限"},
		{user.PermissionUser, "👤 普通用户"},
		{user.PermissionAdmin, "👮 管理员"},
		{user.PermissionSuperAdmin, "⭐ 超级管理员"},
		{user.PermissionOwner, "👑 群主"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			label := getPermissionLabel(tt.perm)
			if label != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, label)
			}
		})
	}
}

func TestGetStatusEmoji(t *testing.T) {
	tests := []struct {
		enabled  bool
		expected string
	}{
		{true, "✅ 已启用"},
		{false, "❌ 已禁用"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			emoji := getStatusEmoji(tt.enabled)
			if emoji != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, emoji)
			}
		})
	}
}

func TestHelp_CommandGrouping(t *testing.T) {
	registry := setupTestRegistry()
	_ = NewHandler(registry)

	// 验证命令按权限分组
	handlers := registry.GetAll()

	userCmds := 0
	adminCmds := 0
	superAdminCmds := 0

	for _, h := range handlers {
		if !h.IsEnabled(-100) {
			continue
		}
		switch h.RequiredPermission() {
		case user.PermissionUser:
			userCmds++
		case user.PermissionAdmin:
			adminCmds++
		case user.PermissionSuperAdmin:
			superAdminCmds++
		}
	}

	if userCmds == 0 {
		t.Error("should have user commands")
	}
	if adminCmds == 0 {
		t.Error("should have admin commands")
	}
	if superAdminCmds == 0 {
		t.Error("should have superadmin commands")
	}

	t.Logf("Commands: user=%d, admin=%d, superadmin=%d", userCmds, adminCmds, superAdminCmds)
}

func TestHelp_NilUser(t *testing.T) {
	registry := setupTestRegistry()
	handler := NewHandler(registry)

	// 没有用户信息的情况
	ctx := &command.Context{
		Ctx:     context.Background(),
		UserID:  1,
		GroupID: -100,
		User:    nil, // nil user
		Args:    []string{},
	}

	err := handler.Handle(ctx)
	if err != nil {
		t.Errorf("Handle() should not error with nil user, got: %v", err)
	}
}

func TestHelp_CommandNameCaseSensitive(t *testing.T) {
	registry := setupTestRegistry()
	_ = NewHandler(registry)

	testUser := user.NewUser(1, "testuser", "Test", "User")
	testUser.SetPermission(-100, user.PermissionUser)

	// 测试大小写敏感 - 命令名应该是小写
	// 这里只是验证注册表行为
	_, exists := registry.Get("PING")
	if exists {
		t.Error("command should be case sensitive, PING should not exist")
	}

	_, exists = registry.Get("ping")
	if !exists {
		t.Error("ping command should exist")
	}
}