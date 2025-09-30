package welcome

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

// Test cases

func TestHandler_Name(t *testing.T) {
	handler := &Handler{}
	if got := handler.Name(); got != "welcome" {
		t.Errorf("Name() = %v, want %v", got, "welcome")
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
	handler := NewHandler(groupRepo)

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
				g.EnableCommand("welcome", 1)
				groupRepo.Save(g)
			},
			groupID: 1,
			want:    true,
		},
		{
			name: "disabled in group",
			setup: func() {
				g := group.NewGroup(2, "Test Group 2", "supergroup")
				g.DisableCommand("welcome", 1)
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
	handler := NewHandler(groupRepo)

	// Setup test data
	g := group.NewGroup(1, "Test Group", "supergroup")
	g.SetSetting(SettingKeyEnabled, true)
	g.SetSetting(SettingKeyMessage, "Welcome {user}!")
	groupRepo.Save(g)

	u := user.NewUser(1, "testuser", "Test", "User")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "show config",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "enable welcome",
			args:    []string{"on"},
			wantErr: false,
		},
		{
			name:    "disable welcome",
			args:    []string{"off"},
			wantErr: false,
		},
		{
			name:    "set message",
			args:    []string{"set", "Hello", "{user}!"},
			wantErr: false,
		},
		{
			name:    "reset message",
			args:    []string{"reset"},
			wantErr: false,
		},
		{
			name:    "test message",
			args:    []string{"test"},
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
				User:    u,
			}
			err := handler.Handle(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_showWelcomeConfig(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	handler := NewHandler(groupRepo)

	tests := []struct {
		name    string
		setup   func()
		groupID int64
		wantErr bool
	}{
		{
			name: "group exists with config",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.SetSetting(SettingKeyEnabled, true)
				g.SetSetting(SettingKeyMessage, "Welcome {user}!")
				groupRepo.Save(g)
			},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "group not found",
			setup: func() {
				// no group
			},
			groupID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			tt.setup()

			u := user.NewUser(1, "testuser", "Test", "User")
			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
				User:    u,
			}

			err := handler.showWelcomeConfig(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("showWelcomeConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_enableWelcome(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	handler := NewHandler(groupRepo)

	tests := []struct {
		name    string
		setup   func()
		groupID int64
		wantErr bool
	}{
		{
			name: "enable for existing group",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "enable for non-existing group",
			setup: func() {
				// no group
			},
			groupID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
			}

			err := handler.enableWelcome(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("enableWelcome() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify it was enabled
			g, _ := groupRepo.FindByID(tt.groupID)
			if g != nil {
				if enabled, ok := g.GetSetting(SettingKeyEnabled); ok {
					if !enabled.(bool) {
						t.Error("Welcome should be enabled")
					}
				}
			}
		})
	}
}

func TestHandler_disableWelcome(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	handler := NewHandler(groupRepo)

	tests := []struct {
		name    string
		setup   func()
		groupID int64
		wantErr bool
	}{
		{
			name: "disable for existing group",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.SetSetting(SettingKeyEnabled, true)
				groupRepo.Save(g)
			},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "disable for non-existing group",
			setup: func() {
				// no group
			},
			groupID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
			}

			err := handler.disableWelcome(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("disableWelcome() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.groupID != 999 {
				// Verify it was disabled
				g, _ := groupRepo.FindByID(tt.groupID)
				if g != nil {
					if enabled, ok := g.GetSetting(SettingKeyEnabled); ok {
						if enabled.(bool) {
							t.Error("Welcome should be disabled")
						}
					}
				}
			}
		})
	}
}

func TestHandler_setWelcomeMessage(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	handler := NewHandler(groupRepo)

	tests := []struct {
		name    string
		setup   func()
		args    []string
		groupID int64
		wantErr bool
	}{
		{
			name: "set valid message",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"set", "Welcome", "{user}", "to", "{group}!"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "set message without text",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				groupRepo.Save(g)
			},
			args:    []string{"set"},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "set message for non-existing group",
			setup: func() {
				// no group
			},
			args:    []string{"set", "Hello!"},
			groupID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
				Args:    tt.args,
			}

			err := handler.setWelcomeMessage(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("setWelcomeMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_setWelcomeMessage_TooLong(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	handler := NewHandler(groupRepo)

	g := group.NewGroup(1, "Test Group", "supergroup")
	groupRepo.Save(g)

	// Create a message longer than 500 characters
	longMessage := make([]string, 502)
	for i := range longMessage {
		longMessage[i] = "a"
	}

	ctx := &command.Context{
		GroupID: 1,
		UserID:  1,
		Args:    append([]string{"set"}, longMessage...),
	}

	err := handler.setWelcomeMessage(ctx)
	if err != nil {
		t.Errorf("setWelcomeMessage() should not return error for too long message")
	}
}

func TestHandler_resetWelcomeMessage(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	handler := NewHandler(groupRepo)

	tests := []struct {
		name    string
		setup   func()
		groupID int64
		wantErr bool
	}{
		{
			name: "reset existing message",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.SetSetting(SettingKeyMessage, "Custom message")
				groupRepo.Save(g)
			},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "reset for non-existing group",
			setup: func() {
				// no group
			},
			groupID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
			}

			err := handler.resetWelcomeMessage(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("resetWelcomeMessage() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.groupID != 999 {
				// Verify message was reset
				g, _ := groupRepo.FindByID(tt.groupID)
				if g != nil {
					if msg, ok := g.GetSetting(SettingKeyMessage); ok {
						if msg.(string) != DefaultWelcomeMessage {
							t.Errorf("Message should be reset to default")
						}
					}
				}
			}
		})
	}
}

func TestHandler_testWelcomeMessage(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	handler := NewHandler(groupRepo)

	tests := []struct {
		name    string
		setup   func()
		groupID int64
		wantErr bool
	}{
		{
			name: "test with existing group",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.SetSetting(SettingKeyMessage, "Welcome {user} to {group}!")
				groupRepo.Save(g)
			},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "test with non-existing group",
			setup: func() {
				// no group
			},
			groupID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			tt.setup()

			u := user.NewUser(1, "testuser", "Test", "User")
			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
				User:    u,
			}

			err := handler.testWelcomeMessage(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("testWelcomeMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_OnNewMember(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	handler := NewHandler(groupRepo)

	tests := []struct {
		name        string
		setup       func()
		groupID     int64
		newUser     *user.User
		wantMessage bool
	}{
		{
			name: "welcome enabled with custom message",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.SetSetting(SettingKeyEnabled, true)
				g.SetSetting(SettingKeyMessage, "欢迎 {user} 加入 {group}!")
				groupRepo.Save(g)
			},
			groupID: 1,
			newUser: user.NewUser(123, "newuser", "New", "User"),
			wantMessage: true,
		},
		{
			name: "welcome disabled",
			setup: func() {
				g := group.NewGroup(2, "Test Group 2", "supergroup")
				g.SetSetting(SettingKeyEnabled, false)
				groupRepo.Save(g)
			},
			groupID: 2,
			newUser: user.NewUser(124, "newuser2", "New", "User2"),
			wantMessage: false,
		},
		{
			name: "group not found - default behavior",
			setup: func() {
				// no group
			},
			groupID: 999,
			newUser: user.NewUser(125, "newuser3", "New", "User3"),
			wantMessage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo.groups = make(map[int64]*group.Group)
			tt.setup()

			message, shouldSend := handler.OnNewMember(tt.groupID, tt.newUser)

			if shouldSend != tt.wantMessage {
				t.Errorf("OnNewMember() shouldSend = %v, want %v", shouldSend, tt.wantMessage)
			}

			if tt.wantMessage && message == "" {
				t.Error("OnNewMember() should return non-empty message when enabled")
			}

			if !tt.wantMessage && message != "" {
				t.Error("OnNewMember() should return empty message when disabled")
			}
		})
	}
}

func TestHandler_formatWelcomeMessage(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name     string
		template string
		user     *user.User
		group    *group.Group
		contains []string
	}{
		{
			name:     "replace user name",
			template: "Welcome {user}!",
			user:     user.NewUser(1, "testuser", "Test", "User"),
			group:    nil,
			contains: []string{"Test User"},
		},
		{
			name:     "replace username",
			template: "Hello {username}!",
			user:     user.NewUser(1, "testuser", "Test", "User"),
			group:    nil,
			contains: []string{"@testuser"},
		},
		{
			name:     "replace user id",
			template: "User ID: {userid}",
			user:     user.NewUser(123, "testuser", "Test", "User"),
			group:    nil,
			contains: []string{"123"},
		},
		{
			name:     "replace group name",
			template: "Welcome to {group}!",
			user:     user.NewUser(1, "testuser", "Test", "User"),
			group:    group.NewGroup(1, "Test Group", "supergroup"),
			contains: []string{"Test Group"},
		},
		{
			name:     "replace all variables",
			template: "{user} ({username}) joined {group}. ID: {userid}",
			user:     user.NewUser(123, "testuser", "Test", "User"),
			group:    group.NewGroup(1, "Test Group", "supergroup"),
			contains: []string{"Test User", "@testuser", "Test Group", "123"},
		},
		{
			name:     "user without username",
			template: "Hello {username}!",
			user:     user.NewUser(1, "", "Test", "User"),
			group:    nil,
			contains: []string{"Test User"},
		},
		{
			name:     "user without last name",
			template: "Welcome {user}!",
			user:     user.NewUser(1, "testuser", "Test", ""),
			group:    nil,
			contains: []string{"Test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.formatWelcomeMessage(tt.template, tt.user, tt.group)

			for _, substr := range tt.contains {
				if !contains(result, substr) {
					t.Errorf("formatWelcomeMessage() result should contain %q, got %q", substr, result)
				}
			}
		})
	}
}

func TestHandler_isWelcomeEnabled(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name  string
		group *group.Group
		want  bool
	}{
		{
			name: "explicitly enabled",
			group: func() *group.Group {
				g := group.NewGroup(1, "Test", "supergroup")
				g.SetSetting(SettingKeyEnabled, true)
				return g
			}(),
			want: true,
		},
		{
			name: "explicitly disabled",
			group: func() *group.Group {
				g := group.NewGroup(1, "Test", "supergroup")
				g.SetSetting(SettingKeyEnabled, false)
				return g
			}(),
			want: false,
		},
		{
			name:  "default behavior",
			group: group.NewGroup(1, "Test", "supergroup"),
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.isWelcomeEnabled(tt.group); got != tt.want {
				t.Errorf("isWelcomeEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_getWelcomeMessage(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name  string
		group *group.Group
		want  string
	}{
		{
			name: "custom message",
			group: func() *group.Group {
				g := group.NewGroup(1, "Test", "supergroup")
				g.SetSetting(SettingKeyMessage, "Custom welcome!")
				return g
			}(),
			want: "Custom welcome!",
		},
		{
			name:  "default message",
			group: group.NewGroup(1, "Test", "supergroup"),
			want:  DefaultWelcomeMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.getWelcomeMessage(tt.group); got != tt.want {
				t.Errorf("getWelcomeMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Verify Handler implements command.Handler interface
var _ command.Handler = (*Handler)(nil)
