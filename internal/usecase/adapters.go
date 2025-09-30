package usecase

import (
	"context"

	"telegram-bot/internal/domain/user"
	userUseCase "telegram-bot/internal/usecase/user"
	groupUseCase "telegram-bot/internal/usecase/group"
)

// userManagementAdapter 用户管理用例适配器
type userManagementAdapter struct {
	checkPermissionUC *userUseCase.CheckPermissionUseCase
	manageAdminUC     *userUseCase.ManageAdminUseCase
}

// NewUserManagementAdapter 创建用户管理适配器
func NewUserManagementAdapter(
	checkPermissionUC *userUseCase.CheckPermissionUseCase,
	manageAdminUC *userUseCase.ManageAdminUseCase,
) UserManagement {
	return &userManagementAdapter{
		checkPermissionUC: checkPermissionUC,
		manageAdminUC:     manageAdminUC,
	}
}

func (a *userManagementAdapter) PromoteAdmin(ctx context.Context, input userUseCase.PromoteAdminInput) error {
	return a.manageAdminUC.PromoteAdmin(ctx, input)
}

func (a *userManagementAdapter) DemoteAdmin(ctx context.Context, input userUseCase.DemoteAdminInput) error {
	return a.manageAdminUC.DemoteAdmin(ctx, input)
}

func (a *userManagementAdapter) ListAdmins(ctx context.Context, operatorID, groupID int64) (*userUseCase.ListAdminsOutput, error) {
	return a.manageAdminUC.ListAdmins(ctx, operatorID, groupID)
}

func (a *userManagementAdapter) RemoveAdmin(ctx context.Context, operatorID, targetID, groupID int64) error {
	return a.manageAdminUC.RemoveAdmin(ctx, operatorID, targetID, groupID)
}

func (a *userManagementAdapter) SetPermission(ctx context.Context, operatorID, targetID, groupID int64, permission user.Permission) error {
	return a.manageAdminUC.SetPermission(ctx, operatorID, targetID, groupID, permission)
}

func (a *userManagementAdapter) CheckPermission(ctx context.Context, userID, groupID int64, required user.Permission) error {
	return a.checkPermissionUC.Execute(ctx, userID, groupID, required)
}

func (a *userManagementAdapter) GetUserPermission(ctx context.Context, userID, groupID int64) (user.Permission, error) {
	return a.checkPermissionUC.GetUserPermission(ctx, userID, groupID)
}

func (a *userManagementAdapter) IsAdmin(ctx context.Context, userID, groupID int64) (bool, error) {
	return a.checkPermissionUC.IsAdmin(ctx, userID, groupID)
}

func (a *userManagementAdapter) IsSuperAdmin(ctx context.Context, userID, groupID int64) (bool, error) {
	return a.checkPermissionUC.IsSuperAdmin(ctx, userID, groupID)
}

// groupCommandConfigAdapter 群组命令配置适配器
type groupCommandConfigAdapter struct {
	configureCommandUC *groupUseCase.ConfigureCommandUseCase
}

// NewGroupCommandConfigAdapter 创建群组命令配置适配器
func NewGroupCommandConfigAdapter(configureCommandUC *groupUseCase.ConfigureCommandUseCase) GroupCommandConfig {
	return &groupCommandConfigAdapter{
		configureCommandUC: configureCommandUC,
	}
}

func (a *groupCommandConfigAdapter) EnableCommand(ctx context.Context, input groupUseCase.EnableCommandInput) error {
	return a.configureCommandUC.EnableCommand(ctx, input)
}

func (a *groupCommandConfigAdapter) DisableCommand(ctx context.Context, input groupUseCase.DisableCommandInput) error {
	return a.configureCommandUC.DisableCommand(ctx, input)
}

func (a *groupCommandConfigAdapter) GetCommandStatus(ctx context.Context, input groupUseCase.GetCommandStatusInput) (*groupUseCase.CommandStatusOutput, error) {
	return a.configureCommandUC.GetCommandStatus(ctx, input)
}

func (a *groupCommandConfigAdapter) BatchConfigure(ctx context.Context, input groupUseCase.BatchConfigureInput) error {
	return a.configureCommandUC.BatchConfigure(ctx, input)
}

func (a *groupCommandConfigAdapter) ListCommands(ctx context.Context, operatorID, groupID int64) (*groupUseCase.ListCommandsOutput, error) {
	return a.configureCommandUC.ListCommands(ctx, operatorID, groupID)
}

// groupConfigAdapter 群组配置适配器
type groupConfigAdapter struct {
	getConfigUC *groupUseCase.GetConfigUseCase
}

// NewGroupConfigAdapter 创建群组配置适配器
func NewGroupConfigAdapter(getConfigUC *groupUseCase.GetConfigUseCase) GroupConfig {
	return &groupConfigAdapter{
		getConfigUC: getConfigUC,
	}
}

func (a *groupConfigAdapter) GetGroupConfig(ctx context.Context, operatorID, groupID int64) (*groupUseCase.GroupConfigOutput, error) {
	return a.getConfigUC.GetGroupConfig(ctx, operatorID, groupID)
}

func (a *groupConfigAdapter) GetAllGroupConfigs(ctx context.Context, operatorID int64) (*groupUseCase.GetAllGroupConfigsOutput, error) {
	return a.getConfigUC.GetAllGroupConfigs(ctx, operatorID)
}

func (a *groupConfigAdapter) UpdateGroupSettings(ctx context.Context, input groupUseCase.UpdateGroupSettingsInput) error {
	return a.getConfigUC.UpdateGroupSettings(ctx, input)
}

func (a *groupConfigAdapter) GetGroupSetting(ctx context.Context, operatorID, groupID int64, key string) (interface{}, error) {
	return a.getConfigUC.GetGroupSetting(ctx, operatorID, groupID, key)
}

func (a *groupConfigAdapter) SetGroupSetting(ctx context.Context, operatorID, groupID int64, key string, value interface{}) error {
	return a.getConfigUC.SetGroupSetting(ctx, operatorID, groupID, key, value)
}