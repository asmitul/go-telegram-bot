package user

import (
	"context"
	"testing"

	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/errors"
)

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

func setupTestUsers() *mockUserRepository {
	repo := newMockUserRepository()

	// 创建测试用户
	owner := user.NewUser(1, "owner", "Owner", "User")
	owner.SetPermission(-100, user.PermissionOwner)
	repo.Save(owner)

	superAdmin := user.NewUser(2, "superadmin", "Super", "Admin")
	superAdmin.SetPermission(-100, user.PermissionSuperAdmin)
	repo.Save(superAdmin)

	admin := user.NewUser(3, "admin", "Admin", "User")
	admin.SetPermission(-100, user.PermissionAdmin)
	repo.Save(admin)

	normalUser := user.NewUser(4, "user", "Normal", "User")
	normalUser.SetPermission(-100, user.PermissionUser)
	repo.Save(normalUser)

	return repo
}

func TestPromoteAdmin(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   PromoteAdminInput
		wantErr bool
		errCode string
	}{
		{
			name: "owner promotes user to admin",
			input: PromoteAdminInput{
				OperatorID: 1,
				TargetID:   4,
				GroupID:    -100,
				Permission: user.PermissionAdmin,
			},
			wantErr: false,
		},
		{
			name: "admin cannot promote to superadmin",
			input: PromoteAdminInput{
				OperatorID: 3,
				TargetID:   4,
				GroupID:    -100,
				Permission: user.PermissionSuperAdmin,
			},
			wantErr: true,
			errCode: "CANNOT_PROMOTE_HIGHER",
		},
		{
			name: "cannot promote self",
			input: PromoteAdminInput{
				OperatorID: 1,
				TargetID:   1,
				GroupID:    -100,
				Permission: user.PermissionAdmin,
			},
			wantErr: true,
			errCode: "SELF_PROMOTION",
		},
		{
			name: "user cannot promote",
			input: PromoteAdminInput{
				OperatorID: 4,
				TargetID:   3,
				GroupID:    -100,
				Permission: user.PermissionAdmin,
			},
			wantErr: true,
			errCode: "INSUFFICIENT_PERMISSION",
		},
		{
			name: "invalid user id",
			input: PromoteAdminInput{
				OperatorID: 0,
				TargetID:   4,
				GroupID:    -100,
				Permission: user.PermissionAdmin,
			},
			wantErr: true,
			errCode: "INVALID_USER_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 每个测试用例重新初始化数据
			repo := setupTestUsers()
			uc := NewManageAdminUseCase(repo)

			err := uc.PromoteAdmin(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("PromoteAdmin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			// 验证权限是否更新
			if !tt.wantErr {
				target, _ := repo.FindByID(tt.input.TargetID)
				if target.GetPermission(tt.input.GroupID) != tt.input.Permission {
					t.Errorf("permission not updated correctly")
				}
			}
		})
	}
}

func TestDemoteAdmin(t *testing.T) {
	ctx := context.Background()

	tests := []struct{
		name    string
		input   DemoteAdminInput
		wantErr bool
		errCode string
	}{
		{
			name: "owner demotes admin to user",
			input: DemoteAdminInput{
				OperatorID: 1,
				TargetID:   3,
				GroupID:    -100,
				Permission: user.PermissionUser,
			},
			wantErr: false,
		},
		{
			name: "cannot demote self",
			input: DemoteAdminInput{
				OperatorID: 1,
				TargetID:   1,
				GroupID:    -100,
				Permission: user.PermissionUser,
			},
			wantErr: true,
			errCode: "SELF_DEMOTION",
		},
		{
			name: "admin cannot demote superadmin",
			input: DemoteAdminInput{
				OperatorID: 3,
				TargetID:   2,
				GroupID:    -100,
				Permission: user.PermissionUser,
			},
			wantErr: true,
			errCode: "CANNOT_MANAGE_TARGET",
		},
		{
			name: "cannot promote when demoting",
			input: DemoteAdminInput{
				OperatorID: 1,
				TargetID:   3,
				GroupID:    -100,
				Permission: user.PermissionSuperAdmin,
			},
			wantErr: true,
			errCode: "INVALID_DEMOTION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 每个测试用例重新初始化数据
			repo := setupTestUsers()
			uc := NewManageAdminUseCase(repo)

			err := uc.DemoteAdmin(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("DemoteAdmin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			// 验证权限是否更新
			if !tt.wantErr {
				target, _ := repo.FindByID(tt.input.TargetID)
				if target.GetPermission(tt.input.GroupID) != tt.input.Permission {
					t.Errorf("permission not updated correctly")
				}
			}
		})
	}
}

func TestListAdmins(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		operatorID  int64
		groupID     int64
		wantCount   int
		wantErr     bool
		errCode     string
	}{
		{
			name:       "list admins as owner",
			operatorID: 1,
			groupID:    -100,
			wantCount:  3, // owner, superadmin, admin
			wantErr:    false,
		},
		{
			name:       "list admins as normal user",
			operatorID: 4,
			groupID:    -100,
			wantCount:  3,
			wantErr:    false,
		},
		{
			name:       "invalid operator id",
			operatorID: 0,
			groupID:    -100,
			wantErr:    true,
			errCode:    "INVALID_USER_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 每个测试用例重新初始化数据
			repo := setupTestUsers()
			uc := NewManageAdminUseCase(repo)

			output, err := uc.ListAdmins(ctx, tt.operatorID, tt.groupID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListAdmins() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			if !tt.wantErr {
				if output.Total != tt.wantCount {
					t.Errorf("expected %d admins, got %d", tt.wantCount, output.Total)
				}
				if len(output.Admins) != tt.wantCount {
					t.Errorf("expected %d admins in list, got %d", tt.wantCount, len(output.Admins))
				}
			}
		})
	}
}

func TestRemoveAdmin(t *testing.T) {
	repo := setupTestUsers()
	uc := NewManageAdminUseCase(repo)
	ctx := context.Background()

	// Remove admin (降级为普通用户)
	err := uc.RemoveAdmin(ctx, 1, 3, -100)
	if err != nil {
		t.Errorf("RemoveAdmin() error = %v", err)
	}

	// 验证权限已更新
	target, _ := repo.FindByID(3)
	if target.GetPermission(-100) != user.PermissionUser {
		t.Errorf("expected PermissionUser, got %v", target.GetPermission(-100))
	}
}

func TestSetPermission(t *testing.T) {
	repo := setupTestUsers()
	uc := NewManageAdminUseCase(repo)
	ctx := context.Background()

	tests := []struct {
		name       string
		operatorID int64
		targetID   int64
		groupID    int64
		permission user.Permission
		wantErr    bool
		errCode    string
	}{
		{
			name:       "superadmin sets permission",
			operatorID: 2,
			targetID:   4,
			groupID:    -100,
			permission: user.PermissionAdmin,
			wantErr:    false,
		},
		{
			name:       "admin cannot set permission",
			operatorID: 3,
			targetID:   4,
			groupID:    -100,
			permission: user.PermissionAdmin,
			wantErr:    true,
			errCode:    "INSUFFICIENT_PERMISSION",
		},
		{
			name:       "cannot set higher permission than self",
			operatorID: 2,
			targetID:   4,
			groupID:    -100,
			permission: user.PermissionOwner,
			wantErr:    true,
			errCode:    "CANNOT_SET_HIGHER",
		},
		{
			name:       "cannot set permission on self",
			operatorID: 2,
			targetID:   2,
			groupID:    -100,
			permission: user.PermissionAdmin,
			wantErr:    true,
			errCode:    "SELF_OPERATION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.SetPermission(ctx, tt.operatorID, tt.targetID, tt.groupID, tt.permission)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPermission() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			// 验证权限是否更新
			if !tt.wantErr {
				target, _ := repo.FindByID(tt.targetID)
				if target.GetPermission(tt.groupID) != tt.permission {
					t.Errorf("permission not updated correctly, expected %v, got %v",
						tt.permission, target.GetPermission(tt.groupID))
				}
			}
		})
	}
}