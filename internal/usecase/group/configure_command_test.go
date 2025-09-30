package group

import (
	"context"
	"testing"

	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/errors"
)

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

func setupTestData() (*mockGroupRepository, *mockUserRepository) {
	groupRepo := newMockGroupRepository()
	userRepo := newMockUserRepository()

	// 创建测试群组
	testGroup := group.NewGroup(-100, "Test Group", "supergroup")
	groupRepo.Save(testGroup)

	// 创建测试用户
	admin := user.NewUser(1, "admin", "Admin", "User")
	admin.SetPermission(-100, user.PermissionAdmin)
	userRepo.Save(admin)

	normalUser := user.NewUser(2, "user", "Normal", "User")
	normalUser.SetPermission(-100, user.PermissionUser)
	userRepo.Save(normalUser)

	return groupRepo, userRepo
}

func TestEnableCommand(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   EnableCommandInput
		wantErr bool
		errCode string
	}{
		{
			name: "admin enables command",
			input: EnableCommandInput{
				OperatorID:  1,
				GroupID:     -100,
				CommandName: "/test",
			},
			wantErr: false,
		},
		{
			name: "normal user cannot enable command",
			input: EnableCommandInput{
				OperatorID:  2,
				GroupID:     -100,
				CommandName: "/test",
			},
			wantErr: true,
			errCode: "INSUFFICIENT_PERMISSION",
		},
		{
			name: "invalid group id",
			input: EnableCommandInput{
				OperatorID:  1,
				GroupID:     100,
				CommandName: "/test",
			},
			wantErr: true,
			errCode: "INVALID_GROUP_ID",
		},
		{
			name: "invalid command name",
			input: EnableCommandInput{
				OperatorID:  1,
				GroupID:     -100,
				CommandName: "test",
			},
			wantErr: true,
			errCode: "INVALID_COMMAND_FORMAT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo, userRepo := setupTestData()
			uc := NewConfigureCommandUseCase(groupRepo, userRepo)

			err := uc.EnableCommand(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnableCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			// 验证命令是否已启用
			if !tt.wantErr {
				grp, _ := groupRepo.FindByID(tt.input.GroupID)
				if !grp.IsCommandEnabled(tt.input.CommandName) {
					t.Error("command should be enabled")
				}
			}
		})
	}
}

func TestDisableCommand(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   DisableCommandInput
		wantErr bool
		errCode string
	}{
		{
			name: "admin disables command",
			input: DisableCommandInput{
				OperatorID:  1,
				GroupID:     -100,
				CommandName: "/test",
			},
			wantErr: false,
		},
		{
			name: "normal user cannot disable command",
			input: DisableCommandInput{
				OperatorID:  2,
				GroupID:     -100,
				CommandName: "/test",
			},
			wantErr: true,
			errCode: "INSUFFICIENT_PERMISSION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo, userRepo := setupTestData()
			uc := NewConfigureCommandUseCase(groupRepo, userRepo)

			err := uc.DisableCommand(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("DisableCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			// 验证命令是否已禁用
			if !tt.wantErr {
				grp, _ := groupRepo.FindByID(tt.input.GroupID)
				if grp.IsCommandEnabled(tt.input.CommandName) {
					t.Error("command should be disabled")
				}
			}
		})
	}
}

func TestGetCommandStatus(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		input         GetCommandStatusInput
		wantEnabled   bool
		wantErr       bool
		errCode       string
	}{
		{
			name: "get status of enabled command",
			input: GetCommandStatusInput{
				OperatorID:  1,
				GroupID:     -100,
				CommandName: "/start",
			},
			wantEnabled: true,
			wantErr:     false,
		},
		{
			name: "get status from non-existent group",
			input: GetCommandStatusInput{
				OperatorID:  1,
				GroupID:     -200,
				CommandName: "/test",
			},
			wantEnabled: true, // 默认启用
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo, userRepo := setupTestData()
			uc := NewConfigureCommandUseCase(groupRepo, userRepo)

			output, err := uc.GetCommandStatus(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommandStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			if !tt.wantErr {
				if output.Enabled != tt.wantEnabled {
					t.Errorf("expected enabled=%v, got %v", tt.wantEnabled, output.Enabled)
				}
				if output.CommandName != tt.input.CommandName {
					t.Errorf("expected command name %s, got %s", tt.input.CommandName, output.CommandName)
				}
			}
		})
	}
}

func TestBatchConfigure(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   BatchConfigureInput
		wantErr bool
		errCode string
	}{
		{
			name: "admin batch configures commands",
			input: BatchConfigureInput{
				OperatorID: 1,
				GroupID:    -100,
				Commands: map[string]bool{
					"/start": true,
					"/help":  true,
					"/ban":   false,
				},
			},
			wantErr: false,
		},
		{
			name: "normal user cannot batch configure",
			input: BatchConfigureInput{
				OperatorID: 2,
				GroupID:    -100,
				Commands: map[string]bool{
					"/start": true,
				},
			},
			wantErr: true,
			errCode: "INSUFFICIENT_PERMISSION",
		},
		{
			name: "empty commands list",
			input: BatchConfigureInput{
				OperatorID: 1,
				GroupID:    -100,
				Commands:   map[string]bool{},
			},
			wantErr: true,
			errCode: "EMPTY_COMMANDS",
		},
		{
			name: "invalid command name in batch",
			input: BatchConfigureInput{
				OperatorID: 1,
				GroupID:    -100,
				Commands: map[string]bool{
					"invalid": true,
				},
			},
			wantErr: true,
			errCode: "INVALID_COMMAND_FORMAT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo, userRepo := setupTestData()
			uc := NewConfigureCommandUseCase(groupRepo, userRepo)

			err := uc.BatchConfigure(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("BatchConfigure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			// 验证批量配置结果
			if !tt.wantErr {
				grp, _ := groupRepo.FindByID(tt.input.GroupID)
				for cmdName, enabled := range tt.input.Commands {
					if grp.IsCommandEnabled(cmdName) != enabled {
						t.Errorf("command %s enabled state mismatch", cmdName)
					}
				}
			}
		})
	}
}

func TestListCommands(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		operatorID int64
		groupID    int64
		setup      func(*mockGroupRepository)
		wantCount  int
		wantErr    bool
	}{
		{
			name:       "list commands from group with configs",
			operatorID: 1,
			groupID:    -100,
			setup: func(repo *mockGroupRepository) {
				grp, _ := repo.FindByID(-100)
				grp.EnableCommand("/start", 1)
				grp.EnableCommand("/help", 1)
				grp.DisableCommand("/ban", 1)
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:       "list commands from non-existent group",
			operatorID: 1,
			groupID:    -200,
			setup:      func(repo *mockGroupRepository) {},
			wantCount:  0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo, userRepo := setupTestData()
			uc := NewConfigureCommandUseCase(groupRepo, userRepo)

			// 运行设置
			tt.setup(groupRepo)

			output, err := uc.ListCommands(ctx, tt.operatorID, tt.groupID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if output.Total != tt.wantCount {
					t.Errorf("expected %d commands, got %d", tt.wantCount, output.Total)
				}
				if len(output.Commands) != tt.wantCount {
					t.Errorf("expected %d commands in list, got %d", tt.wantCount, len(output.Commands))
				}
			}
		})
	}
}