package stats

import (
	"testing"
	"time"

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

// MockUserRepository User Repository mock
type MockUserRepository struct {
	users map[int64]*user.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[int64]*user.User),
	}
}

func (m *MockUserRepository) FindByID(id int64) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	return u, nil
}

func (m *MockUserRepository) FindByUsername(username string) (*user.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}

func (m *MockUserRepository) Save(u *user.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *MockUserRepository) Update(u *user.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *MockUserRepository) Delete(id int64) error {
	delete(m.users, id)
	return nil
}

func (m *MockUserRepository) FindAdminsByGroup(groupID int64) ([]*user.User, error) {
	admins := make([]*user.User, 0)
	for _, u := range m.users {
		if u.GetPermission(groupID) >= user.PermissionAdmin {
			admins = append(admins, u)
		}
	}
	return admins, nil
}

// Test cases

func TestHandler_Name(t *testing.T) {
	handler := &Handler{}
	if got := handler.Name(); got != "stats" {
		t.Errorf("Name() = %v, want %v", got, "stats")
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
	if got := handler.RequiredPermission(); got != user.PermissionUser {
		t.Errorf("RequiredPermission() = %v, want %v", got, user.PermissionUser)
	}
}

func TestHandler_IsEnabled(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	stats := &Stats{BotStartTime: time.Now()}
	handler := NewHandler(groupRepo, userRepo, stats)

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
				g.EnableCommand("stats", 1)
				groupRepo.Save(g)
			},
			groupID: 1,
			want:    true,
		},
		{
			name: "disabled in group",
			setup: func() {
				g := group.NewGroup(2, "Test Group 2", "supergroup")
				g.DisableCommand("stats", 1)
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
	userRepo := NewMockUserRepository()
	stats := &Stats{
		BotStartTime:    time.Now().Add(-24 * time.Hour),
		TotalMessages:   1000,
		CommandsHandled: 200,
		ActiveGroups:    5,
		ActiveUsers:     50,
	}
	handler := NewHandler(groupRepo, userRepo, stats)

	// Setup test data
	g := group.NewGroup(1, "Test Group", "supergroup")
	g.EnableCommand("stats", 1)
	groupRepo.Save(g)

	u := user.NewUser(1, "testuser", "Test", "User")
	u.SetPermission(1, user.PermissionUser)
	userRepo.Save(u)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "show all stats",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "show bot stats",
			args:    []string{"bot"},
			wantErr: false,
		},
		{
			name:    "show group stats",
			args:    []string{"group"},
			wantErr: false,
		},
		{
			name:    "show user stats",
			args:    []string{"user"},
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

func TestHandler_showAllStats(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	stats := &Stats{
		BotStartTime:    time.Now().Add(-48 * time.Hour),
		TotalMessages:   5000,
		CommandsHandled: 1000,
		ActiveGroups:    10,
		ActiveUsers:     100,
	}
	handler := NewHandler(groupRepo, userRepo, stats)

	g := group.NewGroup(1, "Test Group", "supergroup")
	groupRepo.Save(g)

	ctx := &command.Context{
		GroupID: 1,
		UserID:  1,
	}

	err := handler.showAllStats(ctx)
	if err != nil {
		t.Errorf("showAllStats() error = %v", err)
	}
}

func TestHandler_showBotStats(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	stats := &Stats{
		BotStartTime:    time.Now().Add(-72 * time.Hour),
		TotalMessages:   10000,
		CommandsHandled: 2000,
		ActiveGroups:    15,
		ActiveUsers:     150,
	}
	handler := NewHandler(groupRepo, userRepo, stats)

	ctx := &command.Context{
		GroupID: 1,
		UserID:  1,
	}

	err := handler.showBotStats(ctx)
	if err != nil {
		t.Errorf("showBotStats() error = %v", err)
	}
}

func TestHandler_showGroupStats(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	handler := NewHandler(groupRepo, userRepo, nil)

	tests := []struct {
		name    string
		setup   func()
		groupID int64
		wantErr bool
	}{
		{
			name: "group exists",
			setup: func() {
				g := group.NewGroup(1, "Test Group", "supergroup")
				g.EnableCommand("stats", 1)
				g.EnableCommand("help", 1)
				g.DisableCommand("ban", 1)
				groupRepo.Save(g)

				u := user.NewUser(1, "admin", "Admin", "User")
				u.SetPermission(1, user.PermissionAdmin)
				userRepo.Save(u)
			},
			groupID: 1,
			wantErr: false,
		},
		{
			name: "group not found",
			setup: func() {
				// no group added
			},
			groupID: 999,
			wantErr: false, // returns warning message, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset repositories
			groupRepo.groups = make(map[int64]*group.Group)
			userRepo.users = make(map[int64]*user.User)

			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
			}

			err := handler.showGroupStats(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("showGroupStats() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_showUserStats(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	handler := NewHandler(groupRepo, userRepo, nil)

	tests := []struct {
		name    string
		setup   func()
		userID  int64
		groupID int64
		wantErr bool
	}{
		{
			name: "user exists",
			setup: func() {
				u := user.NewUser(1, "testuser", "Test", "User")
				u.SetPermission(1, user.PermissionUser)
				u.SetPermission(2, user.PermissionAdmin)
				userRepo.Save(u)
			},
			userID:  1,
			groupID: 1,
			wantErr: false,
		},
		{
			name: "user not found",
			setup: func() {
				// no user added
			},
			userID:  999,
			groupID: 1,
			wantErr: false, // returns warning message, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset repositories
			userRepo.users = make(map[int64]*user.User)

			tt.setup()

			ctx := &command.Context{
				UserID:  tt.userID,
				GroupID: tt.groupID,
			}

			err := handler.showUserStats(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("showUserStats() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_getBotStats(t *testing.T) {
	stats := &Stats{
		BotStartTime:    time.Now().Add(-1 * time.Hour),
		TotalMessages:   6000,
		CommandsHandled: 1200,
	}
	handler := NewHandler(nil, nil, stats)

	botStats := handler.getBotStats()

	if botStats.MemoryUsageMB <= 0 {
		t.Error("MemoryUsageMB should be greater than 0")
	}

	if botStats.Goroutines <= 0 {
		t.Error("Goroutines should be greater than 0")
	}

	if botStats.AvgMessagesPerMin <= 0 {
		t.Error("AvgMessagesPerMin should be greater than 0")
	}
}

func TestHandler_getGroupStats(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	handler := NewHandler(groupRepo, userRepo, nil)

	// Setup test data
	g := group.NewGroup(1, "Test Group", "supergroup")
	g.EnableCommand("stats", 1)
	g.EnableCommand("help", 1)
	g.DisableCommand("ban", 1)
	groupRepo.Save(g)

	u1 := user.NewUser(1, "admin1", "Admin", "One")
	u1.SetPermission(1, user.PermissionAdmin)
	userRepo.Save(u1)

	u2 := user.NewUser(2, "admin2", "Admin", "Two")
	u2.SetPermission(1, user.PermissionAdmin)
	userRepo.Save(u2)

	groupStats := handler.getGroupStats(1)

	if groupStats == nil {
		t.Fatal("getGroupStats() should not return nil")
	}

	if groupStats.GroupTitle != "Test Group" {
		t.Errorf("GroupTitle = %v, want %v", groupStats.GroupTitle, "Test Group")
	}

	if groupStats.AdminCount != 2 {
		t.Errorf("AdminCount = %v, want %v", groupStats.AdminCount, 2)
	}

	if groupStats.EnabledCommands != 2 {
		t.Errorf("EnabledCommands = %v, want %v", groupStats.EnabledCommands, 2)
	}

	if groupStats.DisabledCommands != 1 {
		t.Errorf("DisabledCommands = %v, want %v", groupStats.DisabledCommands, 1)
	}
}

func TestHandler_getGroupStats_NotFound(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	handler := NewHandler(groupRepo, userRepo, nil)

	groupStats := handler.getGroupStats(999)

	if groupStats != nil {
		t.Error("getGroupStats() should return nil for non-existent group")
	}
}

func TestHandler_getUserStats(t *testing.T) {
	userRepo := NewMockUserRepository()
	handler := NewHandler(nil, userRepo, nil)

	// Setup test data
	u := user.NewUser(1, "testuser", "Test", "User")
	u.SetPermission(1, user.PermissionUser)
	u.SetPermission(2, user.PermissionAdmin)
	u.SetPermission(3, user.PermissionSuperAdmin)
	userRepo.Save(u)

	userStats := handler.getUserStats(1, 1)

	if userStats == nil {
		t.Fatal("getUserStats() should not return nil")
	}

	if userStats.Username != "testuser" {
		t.Errorf("Username = %v, want %v", userStats.Username, "testuser")
	}

	if userStats.FullName != "Test User" {
		t.Errorf("FullName = %v, want %v", userStats.FullName, "Test User")
	}

	if userStats.Permission != user.PermissionUser {
		t.Errorf("Permission = %v, want %v", userStats.Permission, user.PermissionUser)
	}

	if userStats.AdminGroupCount != 2 {
		t.Errorf("AdminGroupCount = %v, want %v", userStats.AdminGroupCount, 2)
	}
}

func TestHandler_getUserStats_NotFound(t *testing.T) {
	userRepo := NewMockUserRepository()
	handler := NewHandler(nil, userRepo, nil)

	userStats := handler.getUserStats(999, 1)

	if userStats != nil {
		t.Error("getUserStats() should return nil for non-existent user")
	}
}

func TestHandler_IncrementMessage(t *testing.T) {
	stats := &Stats{TotalMessages: 100}
	handler := NewHandler(nil, nil, stats)

	handler.IncrementMessage()

	if handler.stats.TotalMessages != 101 {
		t.Errorf("TotalMessages = %v, want %v", handler.stats.TotalMessages, 101)
	}
}

func TestHandler_IncrementCommand(t *testing.T) {
	stats := &Stats{CommandsHandled: 50}
	handler := NewHandler(nil, nil, stats)

	handler.IncrementCommand()

	if handler.stats.CommandsHandled != 51 {
		t.Errorf("CommandsHandled = %v, want %v", handler.stats.CommandsHandled, 51)
	}
}

func TestHandler_UpdateActiveGroups(t *testing.T) {
	stats := &Stats{ActiveGroups: 10}
	handler := NewHandler(nil, nil, stats)

	handler.UpdateActiveGroups(20)

	if handler.stats.ActiveGroups != 20 {
		t.Errorf("ActiveGroups = %v, want %v", handler.stats.ActiveGroups, 20)
	}
}

func TestHandler_UpdateActiveUsers(t *testing.T) {
	stats := &Stats{ActiveUsers: 100}
	handler := NewHandler(nil, nil, stats)

	handler.UpdateActiveUsers(200)

	if handler.stats.ActiveUsers != 200 {
		t.Errorf("ActiveUsers = %v, want %v", handler.stats.ActiveUsers, 200)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "minutes only",
			duration: 30 * time.Minute,
			want:     "30 分钟",
		},
		{
			name:     "hours and minutes",
			duration: 2*time.Hour + 15*time.Minute,
			want:     "2 小时 15 分钟",
		},
		{
			name:     "days hours minutes",
			duration: 3*24*time.Hour + 5*time.Hour + 45*time.Minute,
			want:     "3 天 5 小时 45 分钟",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.duration); got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
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

func TestNewHandler_WithNilStats(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()

	handler := NewHandler(groupRepo, userRepo, nil)

	if handler.stats == nil {
		t.Error("NewHandler() should initialize stats when nil is passed")
	}

	if handler.stats.BotStartTime.IsZero() {
		t.Error("BotStartTime should be initialized")
	}
}

// Verify Handler implements command.Handler interface
var _ command.Handler = (*Handler)(nil)
