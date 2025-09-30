package manage

import (
	"testing"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
)

// MockGroupRepository Group Repository mock
type MockGroupRepository struct {
	groups map[int64]*group.Group
}

func NewMockGroupRepository() *MockGroupRepository {
	return &MockGroupRepository{
		groups: make(map[int64]*group.Group),
	}
}

func (m *MockGroupRepository) FindByID(id int64) (*group.Group, error) {
	g, ok := m.groups[id]
	if !ok {
		return nil, group.ErrGroupNotFound
	}
	return g, nil
}

func (m *MockGroupRepository) Save(g *group.Group) error {
	m.groups[g.ID] = g
	return nil
}

func (m *MockGroupRepository) Update(g *group.Group) error {
	m.groups[g.ID] = g
	return nil
}

func (m *MockGroupRepository) Delete(id int64) error {
	delete(m.groups, id)
	return nil
}

func (m *MockGroupRepository) FindAll() ([]*group.Group, error) {
	groups := make([]*group.Group, 0, len(m.groups))
	for _, g := range m.groups {
		groups = append(groups, g)
	}
	return groups, nil
}

// MockRegistry Command Registry mock
type MockRegistry struct {
	commands map[string]command.Handler
}

func NewMockRegistry() *MockRegistry {
	return &MockRegistry{
		commands: make(map[string]command.Handler),
	}
}

func (m *MockRegistry) Register(handler command.Handler) error {
	m.commands[handler.Name()] = handler
	return nil
}

func (m *MockRegistry) Get(name string) (command.Handler, bool) {
	handler, exists := m.commands[name]
	return handler, exists
}

func (m *MockRegistry) GetAll() []command.Handler {
	handlers := make([]command.Handler, 0, len(m.commands))
	for _, handler := range m.commands {
		handlers = append(handlers, handler)
	}
	return handlers
}

func (m *MockRegistry) Unregister(name string) {
	delete(m.commands, name)
}

// MockCommand Mock command for testing
type MockCommand struct {
	name        string
	description string
	permission  user.Permission
}

func (m *MockCommand) Name() string {
	return m.name
}

func (m *MockCommand) Description() string {
	return m.description
}

func (m *MockCommand) RequiredPermission() user.Permission {
	return m.permission
}

func (m *MockCommand) Handle(ctx *command.Context) error {
	return nil
}

func (m *MockCommand) IsEnabled(groupID int64) bool {
	return true
}

// Test cases

func TestHandler_Name(t *testing.T) {
	handler := &Handler{}
	if got := handler.Name(); got != "manage" {
		t.Errorf("Name() = %v, want %v", got, "manage")
	}
}

func TestHandler_Description(t *testing.T) {
	handler := &Handler{}
	if got := handler.Description(); got == "" {
		t.Error("Description() should not be empty")
	}
}

func TestHandler_RequiredPermission(t *testing.T) {
	handler := &Handler{}
	if got := handler.RequiredPermission(); got != user.PermissionAdmin {
		t.Errorf("RequiredPermission() = %v, want %v", got, user.PermissionAdmin)
	}
}

func TestHandler_IsEnabled(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	registry := NewMockRegistry()
	handler := NewHandler(groupRepo, registry)

	tests := []struct {
		name    string
		setup   func()
		groupID int64
		want    bool
	}{
		{
			name: "default enabled",
			setup: func() {
				// no groups added
			},
			groupID: 1,
			want:    true,
		},
		{
			name: "enabled in group",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.EnableCommand("manage", 1)
				groupRepo.Save(g)
			},
			groupID: 1,
			want:    true,
		},
		{
			name: "disabled in group",
			setup: func() {
				g := group.NewGroup(2, "Test Group 2", "supergroup")
				g.DisableCommand("manage", 1)
				groupRepo.Save(g)
			},
			groupID: 2,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			if got := handler.IsEnabled(tt.groupID); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_Handle(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	registry := NewMockRegistry()
	handler := NewHandler(groupRepo, registry)

	// Register some test commands
	registry.Register(&MockCommand{name: "test", description: "Test command", permission: user.PermissionUser})
	registry.Register(&MockCommand{name: "admin", description: "Admin command", permission: user.PermissionAdmin})

	// Setup test data
	g := group.NewGroup(1, "Test Group", "supergroup")
	g.EnableCommand("test", 1)
	g.DisableCommand("admin", 1)
	groupRepo.Save(g)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "show commands list",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "list commands",
			args:    []string{"list"},
			wantErr: false,
		},
		{
			name:    "enable command",
			args:    []string{"enable", "admin"},
			wantErr: false,
		},
		{
			name:    "disable command",
			args:    []string{"disable", "test"},
			wantErr: false,
		},
		{
			name:    "show command status",
			args:    []string{"status", "test"},
			wantErr: false,
		},
		{
			name:    "unknown subcommand",
			args:    []string{"unknown"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &command.Context{
				GroupID: 1,
				UserID:  1,
				Args:    tt.args,
			}
			err := handler.Handle(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_showCommands(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	registry := NewMockRegistry()
	handler := NewHandler(groupRepo, registry)

	tests := []struct {
		name    string
		setup   func()
		groupID int64
		wantErr bool
	}{
		{
			name: "show commands with some disabled",
			setup: func() {
				registry.Register(&MockCommand{name: "ping", description: "Ping", permission: user.PermissionUser})
				registry.Register(&MockCommand{name: "help", description: "Help", permission: user.PermissionUser})
				registry.Register(&MockCommand{name: "ban", description: "Ban", permission: user.PermissionAdmin})

				g := group.NewGroup(1, "Test Group", "supergroup")
				g.EnableCommand("ping", 1)
				g.EnableCommand("help", 1)
				g.DisableCommand("ban", 1)
				groupRepo.Save(g)
			},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "group not found",
			setup: func() {
				registry.commands = make(map[string]command.Handler)
				registry.Register(&MockCommand{name: "test", description: "Test", permission: user.PermissionUser})
			},
			groupID: 999,
			wantErr: false,
		},
		{
			name: "no commands registered",
			setup: func() {
				registry.commands = make(map[string]command.Handler)
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			groupID: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			registry.commands = make(map[string]command.Handler)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
			}

			err := handler.showCommands(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("showCommands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_enableCommand(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	registry := NewMockRegistry()
	handler := NewHandler(groupRepo, registry)

	tests := []struct {
		name    string
		setup   func()
		args    []string
		groupID int64
		wantErr bool
	}{
		{
			name: "enable existing command",
			setup: func() {
				registry.Register(&MockCommand{name: "test", description: "Test", permission: user.PermissionUser})
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.DisableCommand("test", 1)
				groupRepo.Save(g)
			},
			args:    []string{"enable", "test"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "enable command with slash prefix",
			setup: func() {
				registry.Register(&MockCommand{name: "test", description: "Test", permission: user.PermissionUser})
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"enable", "/test"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "enable non-existing command",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"enable", "nonexistent"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "enable without command name",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"enable"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "enable for non-existing group",
			setup: func() {
				registry.Register(&MockCommand{name: "test", description: "Test", permission: user.PermissionUser})
			},
			args:    []string{"enable", "test"},
			groupID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			registry.commands = make(map[string]command.Handler)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
				Args:    tt.args,
			}

			err := handler.enableCommand(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("enableCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_disableCommand(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	registry := NewMockRegistry()
	handler := NewHandler(groupRepo, registry)

	tests := []struct {
		name    string
		setup   func()
		args    []string
		groupID int64
		wantErr bool
	}{
		{
			name: "disable existing command",
			setup: func() {
				registry.Register(&MockCommand{name: "test", description: "Test", permission: user.PermissionUser})
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.EnableCommand("test", 1)
				groupRepo.Save(g)
			},
			args:    []string{"disable", "test"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "disable command with slash prefix",
			setup: func() {
				registry.Register(&MockCommand{name: "test", description: "Test", permission: user.PermissionUser})
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"disable", "/test"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "disable manage command (should fail)",
			setup: func() {
				registry.Register(&MockCommand{name: "manage", description: "Manage", permission: user.PermissionAdmin})
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"disable", "manage"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "disable non-existing command",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"disable", "nonexistent"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "disable without command name",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"disable"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "disable for non-existing group",
			setup: func() {
				registry.Register(&MockCommand{name: "test", description: "Test", permission: user.PermissionUser})
			},
			args:    []string{"disable", "test"},
			groupID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			registry.commands = make(map[string]command.Handler)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
				Args:    tt.args,
			}

			err := handler.disableCommand(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("disableCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_showCommandStatus(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	registry := NewMockRegistry()
	handler := NewHandler(groupRepo, registry)

	tests := []struct {
		name    string
		setup   func()
		args    []string
		groupID int64
		wantErr bool
	}{
		{
			name: "show status of enabled command",
			setup: func() {
				registry.Register(&MockCommand{name: "test", description: "Test command", permission: user.PermissionUser})
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.EnableCommand("test", 123)
				groupRepo.Save(g)
			},
			args:    []string{"status", "test"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "show status of disabled command",
			setup: func() {
				registry.Register(&MockCommand{name: "test", description: "Test command", permission: user.PermissionAdmin})
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.DisableCommand("test", 456)
				groupRepo.Save(g)
			},
			args:    []string{"status", "test"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "show status with slash prefix",
			setup: func() {
				registry.Register(&MockCommand{name: "test", description: "Test command", permission: user.PermissionUser})
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"status", "/test"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "show status of non-existing command",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"status", "nonexistent"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "show status without command name",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"status"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "show status for non-existing group",
			setup: func() {
				registry.Register(&MockCommand{name: "test", description: "Test", permission: user.PermissionUser})
			},
			args:    []string{"status", "test"},
			groupID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			registry.commands = make(map[string]command.Handler)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
				Args:    tt.args,
			}

			err := handler.showCommandStatus(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("showCommandStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_showHelp(t *testing.T) {
	handler := &Handler{}

	ctx := &command.Context{
		GroupID: 1,
		UserID:  1,
	}

	err := handler.showHelp(ctx)
	if err != nil {
		t.Errorf("showHelp() error = %v", err)
	}
}

func TestGetPermissionLabel(t *testing.T) {
	tests := []struct {
		name       string
		permission user.Permission
		want       string
	}{
		{
			name:       "no permission",
			permission: user.PermissionNone,
			want:       "🚫 无权限",
		},
		{
			name:       "user permission",
			permission: user.PermissionUser,
			want:       "👤 普通用户",
		},
		{
			name:       "admin permission",
			permission: user.PermissionAdmin,
			want:       "👮 管理员",
		},
		{
			name:       "super admin permission",
			permission: user.PermissionSuperAdmin,
			want:       "⭐ 超级管理员",
		},
		{
			name:       "owner permission",
			permission: user.PermissionOwner,
			want:       "👑 群主",
		},
		{
			name:       "unknown permission",
			permission: user.Permission(999),
			want:       "❓ 未知",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPermissionLabel(tt.permission); got != tt.want {
				t.Errorf("getPermissionLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_Integration(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	registry := NewMockRegistry()
	handler := NewHandler(groupRepo, registry)

	// Register commands
	registry.Register(&MockCommand{name: "ping", description: "Ping command", permission: user.PermissionUser})
	registry.Register(&MockCommand{name: "help", description: "Help command", permission: user.PermissionUser})
	registry.Register(&MockCommand{name: "ban", description: "Ban command", permission: user.PermissionAdmin})

	// Create group
	g := group.NewGroup(1, "Test Group", "supergroup")
	groupRepo.Save(g)

	t.Run("full workflow", func(t *testing.T) {
		// 1. List commands
		ctx := &command.Context{GroupID: 1, UserID: 1, Args: []string{}}
		if err := handler.Handle(ctx); err != nil {
			t.Errorf("List commands failed: %v", err)
		}

		// 2. Disable a command
		ctx.Args = []string{"disable", "ping"}
		if err := handler.Handle(ctx); err != nil {
			t.Errorf("Disable command failed: %v", err)
		}

		// Verify command was disabled
		g, _ := groupRepo.FindByID(1)
		if g.IsCommandEnabled("ping") {
			t.Error("Command should be disabled")
		}

		// 3. Show status
		ctx.Args = []string{"status", "ping"}
		if err := handler.Handle(ctx); err != nil {
			t.Errorf("Show status failed: %v", err)
		}

		// 4. Enable the command again
		ctx.Args = []string{"enable", "ping"}
		if err := handler.Handle(ctx); err != nil {
			t.Errorf("Enable command failed: %v", err)
		}

		// Verify command was enabled
		g, _ = groupRepo.FindByID(1)
		if !g.IsCommandEnabled("ping") {
			t.Error("Command should be enabled")
		}
	})
}

// Verify Handler implements command.Handler interface
var _ command.Handler = (*Handler)(nil)
