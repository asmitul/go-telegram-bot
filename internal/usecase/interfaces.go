package usecase

import (
	"context"

	"telegram-bot/internal/domain/user"
	userUseCase "telegram-bot/internal/usecase/user"
	groupUseCase "telegram-bot/internal/usecase/group"
)

// UserManagement 用户管理用例接口
type UserManagement interface {
	// 管理员管理
	PromoteAdmin(ctx context.Context, input userUseCase.PromoteAdminInput) error
	DemoteAdmin(ctx context.Context, input userUseCase.DemoteAdminInput) error
	ListAdmins(ctx context.Context, operatorID, groupID int64) (*userUseCase.ListAdminsOutput, error)
	RemoveAdmin(ctx context.Context, operatorID, targetID, groupID int64) error
	SetPermission(ctx context.Context, operatorID, targetID, groupID int64, permission user.Permission) error

	// 权限检查
	CheckPermission(ctx context.Context, userID, groupID int64, required user.Permission) error
	GetUserPermission(ctx context.Context, userID, groupID int64) (user.Permission, error)
	IsAdmin(ctx context.Context, userID, groupID int64) (bool, error)
	IsSuperAdmin(ctx context.Context, userID, groupID int64) (bool, error)
}

// GroupCommandConfig 群组命令配置用例接口
type GroupCommandConfig interface {
	// 命令配置
	EnableCommand(ctx context.Context, input groupUseCase.EnableCommandInput) error
	DisableCommand(ctx context.Context, input groupUseCase.DisableCommandInput) error
	GetCommandStatus(ctx context.Context, input groupUseCase.GetCommandStatusInput) (*groupUseCase.CommandStatusOutput, error)
	BatchConfigure(ctx context.Context, input groupUseCase.BatchConfigureInput) error
	ListCommands(ctx context.Context, operatorID, groupID int64) (*groupUseCase.ListCommandsOutput, error)
}

// GroupConfig 群组配置用例接口
type GroupConfig interface {
	// 配置获取
	GetGroupConfig(ctx context.Context, operatorID, groupID int64) (*groupUseCase.GroupConfigOutput, error)
	GetAllGroupConfigs(ctx context.Context, operatorID int64) (*groupUseCase.GetAllGroupConfigsOutput, error)

	// 设置更新
	UpdateGroupSettings(ctx context.Context, input groupUseCase.UpdateGroupSettingsInput) error
	GetGroupSetting(ctx context.Context, operatorID, groupID int64, key string) (interface{}, error)
	SetGroupSetting(ctx context.Context, operatorID, groupID int64, key string, value interface{}) error
}

// UseCases 所有用例的聚合接口
type UseCases struct {
	UserManagement     UserManagement
	GroupCommandConfig GroupCommandConfig
	GroupConfig        GroupConfig
}

// NewUseCases 创建用例聚合
func NewUseCases(
	userManagement UserManagement,
	groupCommandConfig GroupCommandConfig,
	groupConfig GroupConfig,
) *UseCases {
	return &UseCases{
		UserManagement:     userManagement,
		GroupCommandConfig: groupCommandConfig,
		GroupConfig:        groupConfig,
	}
}
