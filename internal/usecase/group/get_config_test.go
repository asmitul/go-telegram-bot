package group

import (
	"context"
	"testing"

	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/errors"
)

func setupTestDataForConfig() (*mockGroupRepository, *mockUserRepository) {
	groupRepo := newMockGroupRepository()
	userRepo := newMockUserRepository()

	// 创建测试群组
	testGroup := group.NewGroup(-100, "Test Group", "supergroup")
	testGroup.EnableCommand("/start", 1)
	testGroup.SetSetting("welcome_message", "Welcome!")
	testGroup.SetSetting("max_members", 1000)
	groupRepo.Save(testGroup)

	anotherGroup := group.NewGroup(-200, "Another Group", "group")
	groupRepo.Save(anotherGroup)

	// 创建测试用户
	superAdmin := user.NewUser(1, "superadmin", "Super", "Admin")
	superAdmin.SetPermission(-100, user.PermissionSuperAdmin)
	superAdmin.SetPermission(-200, user.PermissionSuperAdmin)
	userRepo.Save(superAdmin)

	admin := user.NewUser(2, "admin", "Admin", "User")
	admin.SetPermission(-100, user.PermissionAdmin)
	userRepo.Save(admin)

	normalUser := user.NewUser(3, "user", "Normal", "User")
	normalUser.SetPermission(-100, user.PermissionUser)
	userRepo.Save(normalUser)

	return groupRepo, userRepo
}

func TestGetGroupConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		operatorID int64
		groupID    int64
		wantErr    bool
		errCode    string
	}{
		{
			name:       "get existing group config",
			operatorID: 1,
			groupID:    -100,
			wantErr:    false,
		},
		{
			name:       "get non-existent group",
			operatorID: 1,
			groupID:    -999,
			wantErr:    true,
			errCode:    "GROUP_NOT_FOUND",
		},
		{
			name:       "invalid group id",
			operatorID: 1,
			groupID:    100,
			wantErr:    true,
			errCode:    "INVALID_GROUP_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo, userRepo := setupTestDataForConfig()
			uc := NewGetConfigUseCase(groupRepo, userRepo)

			output, err := uc.GetGroupConfig(ctx, tt.operatorID, tt.groupID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGroupConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			if !tt.wantErr {
				if output.GroupID != tt.groupID {
					t.Errorf("expected group id %d, got %d", tt.groupID, output.GroupID)
				}
				if output.Commands == nil {
					t.Error("commands should not be nil")
				}
				if output.Settings == nil {
					t.Error("settings should not be nil")
				}
			}
		})
	}
}

func TestGetAllGroupConfigs(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		operatorID int64
		wantCount  int
		wantErr    bool
		errCode    string
	}{
		{
			name:       "superadmin gets all groups",
			operatorID: 1,
			wantCount:  2,
			wantErr:    false,
		},
		{
			name:       "admin cannot get all groups",
			operatorID: 2,
			wantErr:    true,
			errCode:    "INSUFFICIENT_PERMISSION",
		},
		{
			name:       "normal user cannot get all groups",
			operatorID: 3,
			wantErr:    true,
			errCode:    "INSUFFICIENT_PERMISSION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo, userRepo := setupTestDataForConfig()
			uc := NewGetConfigUseCase(groupRepo, userRepo)

			output, err := uc.GetAllGroupConfigs(ctx, tt.operatorID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllGroupConfigs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			if !tt.wantErr {
				if output.Total != tt.wantCount {
					t.Errorf("expected %d groups, got %d", tt.wantCount, output.Total)
				}
				if len(output.Groups) != tt.wantCount {
					t.Errorf("expected %d groups in list, got %d", tt.wantCount, len(output.Groups))
				}
			}
		})
	}
}

func TestUpdateGroupSettings(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   UpdateGroupSettingsInput
		wantErr bool
		errCode string
	}{
		{
			name: "admin updates settings",
			input: UpdateGroupSettingsInput{
				OperatorID: 2,
				GroupID:    -100,
				Settings: map[string]interface{}{
					"welcome_message": "New Welcome!",
					"max_members":     2000,
				},
			},
			wantErr: false,
		},
		{
			name: "normal user cannot update settings",
			input: UpdateGroupSettingsInput{
				OperatorID: 3,
				GroupID:    -100,
				Settings: map[string]interface{}{
					"test": "value",
				},
			},
			wantErr: true,
			errCode: "INSUFFICIENT_PERMISSION",
		},
		{
			name: "empty settings",
			input: UpdateGroupSettingsInput{
				OperatorID: 2,
				GroupID:    -100,
				Settings:   map[string]interface{}{},
			},
			wantErr: true,
			errCode: "EMPTY_SETTINGS",
		},
		{
			name: "invalid group id",
			input: UpdateGroupSettingsInput{
				OperatorID: 2,
				GroupID:    100,
				Settings: map[string]interface{}{
					"test": "value",
				},
			},
			wantErr: true,
			errCode: "INVALID_GROUP_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo, userRepo := setupTestDataForConfig()
			uc := NewGetConfigUseCase(groupRepo, userRepo)

			err := uc.UpdateGroupSettings(ctx, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateGroupSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			// 验证设置是否已更新
			if !tt.wantErr {
				grp, _ := groupRepo.FindByID(tt.input.GroupID)
				for key, expectedValue := range tt.input.Settings {
					actualValue, ok := grp.GetSetting(key)
					if !ok {
						t.Errorf("setting %s not found", key)
					}
					if actualValue != expectedValue {
						t.Errorf("setting %s: expected %v, got %v", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

func TestGetGroupSetting(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		operatorID int64
		groupID    int64
		key        string
		wantValue  interface{}
		wantErr    bool
		errCode    string
	}{
		{
			name:       "get existing setting",
			operatorID: 1,
			groupID:    -100,
			key:        "welcome_message",
			wantValue:  "Welcome!",
			wantErr:    false,
		},
		{
			name:       "get non-existent setting",
			operatorID: 1,
			groupID:    -100,
			key:        "non_existent",
			wantErr:    true,
			errCode:    "SETTING_NOT_FOUND",
		},
		{
			name:       "get from non-existent group",
			operatorID: 1,
			groupID:    -999,
			key:        "test",
			wantErr:    true,
			errCode:    "GROUP_NOT_FOUND",
		},
		{
			name:       "empty key",
			operatorID: 1,
			groupID:    -100,
			key:        "",
			wantErr:    true,
			errCode:    "EMPTY_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo, userRepo := setupTestDataForConfig()
			uc := NewGetConfigUseCase(groupRepo, userRepo)

			value, err := uc.GetGroupSetting(ctx, tt.operatorID, tt.groupID, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGroupSetting() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			if !tt.wantErr {
				if value != tt.wantValue {
					t.Errorf("expected value %v, got %v", tt.wantValue, value)
				}
			}
		})
	}
}

func TestSetGroupSetting(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		operatorID int64
		groupID    int64
		key        string
		value      interface{}
		wantErr    bool
		errCode    string
	}{
		{
			name:       "admin sets setting",
			operatorID: 2,
			groupID:    -100,
			key:        "new_setting",
			value:      "new_value",
			wantErr:    false,
		},
		{
			name:       "normal user cannot set setting",
			operatorID: 3,
			groupID:    -100,
			key:        "test",
			value:      "value",
			wantErr:    true,
			errCode:    "INSUFFICIENT_PERMISSION",
		},
		{
			name:       "empty key",
			operatorID: 2,
			groupID:    -100,
			key:        "",
			value:      "value",
			wantErr:    true,
			errCode:    "EMPTY_KEY",
		},
		{
			name:       "cannot set in group without permission",
			operatorID: 1,
			groupID:    -300,
			key:        "test",
			value:      "value",
			wantErr:    true,
			errCode:    "INSUFFICIENT_PERMISSION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo, userRepo := setupTestDataForConfig()
			uc := NewGetConfigUseCase(groupRepo, userRepo)

			err := uc.SetGroupSetting(ctx, tt.operatorID, tt.groupID, tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetGroupSetting() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errCode != "" {
				if !errors.HasCode(err, tt.errCode) {
					t.Errorf("expected error code %s, got %s", tt.errCode, errors.GetCode(err))
				}
			}

			// 验证设置是否已更新
			if !tt.wantErr {
				grp, _ := groupRepo.FindByID(tt.groupID)
				actualValue, ok := grp.GetSetting(tt.key)
				if !ok {
					t.Errorf("setting %s not found", tt.key)
				}
				if actualValue != tt.value {
					t.Errorf("expected value %v, got %v", tt.value, actualValue)
				}
			}
		})
	}
}