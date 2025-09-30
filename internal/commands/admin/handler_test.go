package admin

import (
	"testing"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	userUseCase "telegram-bot/internal/usecase/user"
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
	if got := handler.Name(); got != "admin" {
		t.Errorf("Name() = %v, want %v", got, "admin")
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
	userRepo := NewMockUserRepository()
	manageAdminUC := userUseCase.NewManageAdminUseCase(userRepo)
	handler := NewHandler(groupRepo, userRepo, manageAdminUC)

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
			groupID: -1,
			want:    true,
		},
		{
			name: "enabled in group",
			setup: func() {
				g := group.NewGroup(-1, "Test Group", "supergroup")
				g.EnableCommand("admin", 1)
				groupRepo.Save(g)
			},
			groupID: -1,
			want:    true,
		},
		{
			name: "disabled in group",
			setup: func() {
				g := group.NewGroup(-2, "Test Group 2", "supergroup")
				g.DisableCommand("admin", 1)
				groupRepo.Save(g)
			},
			groupID: -2,
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
	manageAdminUC := userUseCase.NewManageAdminUseCase(userRepo)
	handler := NewHandler(groupRepo, userRepo, manageAdminUC)

	// Setup test data
	g := group.NewGroup(-1, "Test Group", "supergroup")
	groupRepo.Save(g)

	operator := user.NewUser(1, "operator", "Operator", "User")
	operator.SetPermission(-1, user.PermissionSuperAdmin)
	userRepo.Save(operator)

	target := user.NewUser(2, "target", "Target", "User")
	target.SetPermission(-1, user.PermissionUser)
	userRepo.Save(target)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "list admins",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "list admins explicit",
			args:    []string{"list"},
			wantErr: false,
		},
		{
			name:    "promote user",
			args:    []string{"promote", "2", "admin"},
			wantErr: false,
		},
		{
			name:    "demote user",
			args:    []string{"demote", "2"},
			wantErr: false,
		},
		{
			name:    "show user info",
			args:    []string{"info", "2"},
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
				GroupID: -1,
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

func TestHandler_listAdmins(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	manageAdminUC := userUseCase.NewManageAdminUseCase(userRepo)
	handler := NewHandler(groupRepo, userRepo, manageAdminUC)

	tests := []struct {
		name    string
		setup   func()
		groupID int64
		wantErr bool
	}{
		{
			name: "list with multiple admins",
			setup: func() {
				u1 := user.NewUser(1, "owner", "Owner", "User")
				u1.SetPermission(-1, user.PermissionOwner)
				userRepo.Save(u1)

				u2 := user.NewUser(2, "superadmin", "Super", "Admin")
				u2.SetPermission(-1, user.PermissionSuperAdmin)
				userRepo.Save(u2)

				u3 := user.NewUser(3, "admin", "Admin", "User")
				u3.SetPermission(-1, user.PermissionAdmin)
				userRepo.Save(u3)
			},
			groupID: -1,
			wantErr: false,
		},
		{
			name: "no admins",
			setup: func() {
				userRepo.users = make(map[int64]*user.User)
			},
			groupID: -1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo.users = make(map[int64]*user.User)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
			}

			err := handler.listAdmins(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("listAdmins() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_promoteAdmin(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	manageAdminUC := userUseCase.NewManageAdminUseCase(userRepo)
	handler := NewHandler(groupRepo, userRepo, manageAdminUC)

	tests := []struct {
		name    string
		setup   func()
		args    []string
		userID  int64
		groupID int64
		wantErr bool
	}{
		{
			name: "promote to admin",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionSuperAdmin)
				userRepo.Save(operator)

				target := user.NewUser(2, "target", "Target", "User")
				target.SetPermission(-1, user.PermissionUser)
				userRepo.Save(target)
			},
			args:    []string{"promote", "2", "admin"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
		{
			name: "promote to superadmin",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionOwner)
				userRepo.Save(operator)

				target := user.NewUser(2, "target", "Target", "User")
				target.SetPermission(-1, user.PermissionAdmin)
				userRepo.Save(target)
			},
			args:    []string{"promote", "2", "superadmin"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
		{
			name: "invalid user id",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionSuperAdmin)
				userRepo.Save(operator)
			},
			args:    []string{"promote", "invalid", "admin"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
		{
			name: "invalid permission level",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionSuperAdmin)
				userRepo.Save(operator)

				target := user.NewUser(2, "target", "Target", "User")
				target.SetPermission(-1, user.PermissionUser)
				userRepo.Save(target)
			},
			args:    []string{"promote", "2", "invalid"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
		{
			name: "missing arguments",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionSuperAdmin)
				userRepo.Save(operator)
			},
			args:    []string{"promote", "2"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
		{
			name: "insufficient permission",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionAdmin)
				userRepo.Save(operator)

				target := user.NewUser(2, "target", "Target", "User")
				target.SetPermission(-1, user.PermissionUser)
				userRepo.Save(target)
			},
			args:    []string{"promote", "2", "superadmin"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo.users = make(map[int64]*user.User)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  tt.userID,
				Args:    tt.args,
			}

			err := handler.promoteAdmin(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("promoteAdmin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_demoteAdmin(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	manageAdminUC := userUseCase.NewManageAdminUseCase(userRepo)
	handler := NewHandler(groupRepo, userRepo, manageAdminUC)

	tests := []struct {
		name    string
		setup   func()
		args    []string
		userID  int64
		groupID int64
		wantErr bool
	}{
		{
			name: "demote admin to user",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionSuperAdmin)
				userRepo.Save(operator)

				target := user.NewUser(2, "target", "Target", "User")
				target.SetPermission(-1, user.PermissionAdmin)
				userRepo.Save(target)
			},
			args:    []string{"demote", "2"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
		{
			name: "demote with specific level",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionOwner)
				userRepo.Save(operator)

				target := user.NewUser(2, "target", "Target", "User")
				target.SetPermission(-1, user.PermissionSuperAdmin)
				userRepo.Save(target)
			},
			args:    []string{"demote", "2", "admin"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
		{
			name: "invalid user id",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionSuperAdmin)
				userRepo.Save(operator)
			},
			args:    []string{"demote", "invalid"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
		{
			name: "missing arguments",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionSuperAdmin)
				userRepo.Save(operator)
			},
			args:    []string{"demote"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
		{
			name: "insufficient permission",
			setup: func() {
				operator := user.NewUser(1, "operator", "Operator", "User")
				operator.SetPermission(-1, user.PermissionAdmin)
				userRepo.Save(operator)

				target := user.NewUser(2, "target", "Target", "User")
				target.SetPermission(-1, user.PermissionSuperAdmin)
				userRepo.Save(target)
			},
			args:    []string{"demote", "2"},
			userID:  1,
			groupID: -1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo.users = make(map[int64]*user.User)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  tt.userID,
				Args:    tt.args,
			}

			err := handler.demoteAdmin(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("demoteAdmin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_showAdminInfo(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	manageAdminUC := userUseCase.NewManageAdminUseCase(userRepo)
	handler := NewHandler(groupRepo, userRepo, manageAdminUC)

	tests := []struct {
		name    string
		setup   func()
		args    []string
		groupID int64
		wantErr bool
	}{
		{
			name: "show info for existing user",
			setup: func() {
				u := user.NewUser(1, "testuser", "Test", "User")
				u.SetPermission(-1, user.PermissionAdmin)
				u.SetPermission(2, user.PermissionSuperAdmin)
				userRepo.Save(u)
			},
			args:    []string{"info", "1"},
			groupID: -1,
			wantErr: false,
		},
		{
			name: "show info for user without username",
			setup: func() {
				u := user.NewUser(2, "", "No", "Username")
				u.SetPermission(-1, user.PermissionUser)
				userRepo.Save(u)
			},
			args:    []string{"info", "2"},
			groupID: -1,
			wantErr: false,
		},
		{
			name: "invalid user id",
			setup: func() {
				// no users
			},
			args:    []string{"info", "invalid"},
			groupID: -1,
			wantErr: false,
		},
		{
			name: "missing arguments",
			setup: func() {
				// no users
			},
			args:    []string{"info"},
			groupID: -1,
			wantErr: false,
		},
		{
			name: "user not found",
			setup: func() {
				// no users
			},
			args:    []string{"info", "999"},
			groupID: -1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo.users = make(map[int64]*user.User)
			tt.setup()

			ctx := &command.Context{
				GroupID: tt.groupID,
				UserID:  1,
				Args:    tt.args,
			}

			err := handler.showAdminInfo(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("showAdminInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_showHelp(t *testing.T) {
	handler := &Handler{}

	ctx := &command.Context{
		GroupID: -1,
		UserID:  1,
	}

	err := handler.showHelp(ctx)
	if err != nil {
		t.Errorf("showHelp() error = %v", err)
	}
}

func TestFormatAdminInfo(t *testing.T) {
	tests := []struct {
		name string
		user *user.User
		want string
	}{
		{
			name: "user with full name and username",
			user: func() *user.User {
				u := user.NewUser(1, "testuser", "Test", "User")
				return u
			}(),
			want: "• Test User (@testuser) - `1`\n",
		},
		{
			name: "user without username",
			user: func() *user.User {
				u := user.NewUser(2, "", "No", "Username")
				return u
			}(),
			want: "• No Username - `2`\n",
		},
		{
			name: "user without last name",
			user: func() *user.User {
				u := user.NewUser(3, "single", "Single", "")
				return u
			}(),
			want: "• Single (@single) - `3`\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatAdminInfo(tt.user); got != tt.want {
				t.Errorf("formatAdminInfo() = %v, want %v", got, tt.want)
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

func TestHandler_Integration(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	manageAdminUC := userUseCase.NewManageAdminUseCase(userRepo)
	handler := NewHandler(groupRepo, userRepo, manageAdminUC)

	// Create group
	g := group.NewGroup(-1, "Test Group", "supergroup")
	groupRepo.Save(g)

	// Create operator (super admin)
	operator := user.NewUser(1, "operator", "Operator", "User")
	operator.SetPermission(-1, user.PermissionSuperAdmin)
	userRepo.Save(operator)

	// Create target user
	target := user.NewUser(2, "target", "Target", "User")
	target.SetPermission(-1, user.PermissionUser)
	userRepo.Save(target)

	t.Run("full workflow", func(t *testing.T) {
		ctx := &command.Context{GroupID: -1, UserID: 1}

		// 1. List admins (should show only operator)
		ctx.Args = []string{"list"}
		if err := handler.Handle(ctx); err != nil {
			t.Errorf("List admins failed: %v", err)
		}

		// 2. Promote target to admin
		ctx.Args = []string{"promote", "2", "admin"}
		if err := handler.Handle(ctx); err != nil {
			t.Errorf("Promote failed: %v", err)
		}

		// Verify promotion
		target, _ = userRepo.FindByID(2)
		if target.GetPermission(-1) != user.PermissionAdmin {
			t.Error("User should be promoted to admin")
		}

		// 3. Show target info
		ctx.Args = []string{"info", "2"}
		if err := handler.Handle(ctx); err != nil {
			t.Errorf("Show info failed: %v", err)
		}

		// 4. List admins (should show both)
		ctx.Args = []string{"list"}
		if err := handler.Handle(ctx); err != nil {
			t.Errorf("List admins failed: %v", err)
		}

		// 5. Demote target
		ctx.Args = []string{"demote", "2"}
		if err := handler.Handle(ctx); err != nil {
			t.Errorf("Demote failed: %v", err)
		}

		// Verify demotion
		target, _ = userRepo.FindByID(2)
		if target.GetPermission(-1) != user.PermissionUser {
			t.Error("User should be demoted to user")
		}
	})
}

// Verify Handler implements command.Handler interface
var _ command.Handler = (*Handler)(nil)
