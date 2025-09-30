package group

import (
	"context"
	"fmt"

	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/errors"
	"telegram-bot/pkg/validator"
)

// GetConfigUseCase 获取配置用例
type GetConfigUseCase struct {
	groupRepo group.Repository
	userRepo  user.Repository
}

// NewGetConfigUseCase 创建获取配置用例
func NewGetConfigUseCase(groupRepo group.Repository, userRepo user.Repository) *GetConfigUseCase {
	return &GetConfigUseCase{
		groupRepo: groupRepo,
		userRepo:  userRepo,
	}
}

// GroupConfigOutput 群组配置输出
type GroupConfigOutput struct {
	GroupID   int64
	Title     string
	Type      string
	Commands  map[string]*CommandConfig
	Settings  map[string]interface{}
	CreatedAt string
	UpdatedAt string
}

// CommandConfig 命令配置
type CommandConfig struct {
	CommandName string
	Enabled     bool
	UpdatedAt   string
	UpdatedBy   int64
}

// GetGroupConfig 获取群组配置
func (uc *GetConfigUseCase) GetGroupConfig(ctx context.Context, operatorID, groupID int64) (*GroupConfigOutput, error) {
	// 验证输入
	if err := validator.UserID(operatorID); err != nil {
		return nil, err
	}
	if err := validator.GroupID(groupID); err != nil {
		return nil, err
	}

	// 获取群组
	grp, err := uc.groupRepo.FindByID(groupID)
	if err != nil {
		if err == group.ErrGroupNotFound {
			return nil, errors.NotFound("GROUP_NOT_FOUND", "群组不存在")
		}
		return nil, errors.Wrap(err, "获取群组信息失败").
			WithContext("group_id", fmt.Sprintf("%d", groupID))
	}

	// 构建输出
	output := &GroupConfigOutput{
		GroupID:   grp.ID,
		Title:     grp.Title,
		Type:      grp.Type,
		Commands:  make(map[string]*CommandConfig),
		Settings:  grp.Settings,
		CreatedAt: grp.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: grp.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	// 转换命令配置
	for cmdName, config := range grp.Commands {
		output.Commands[cmdName] = &CommandConfig{
			CommandName: config.CommandName,
			Enabled:     config.Enabled,
			UpdatedAt:   config.UpdatedAt.Format("2006-01-02 15:04:05"),
			UpdatedBy:   config.UpdatedBy,
		}
	}

	return output, nil
}

// GetAllGroupConfigsOutput 所有群组配置输出
type GetAllGroupConfigsOutput struct {
	Groups []*GroupConfigOutput
	Total  int
}

// GetAllGroupConfigs 获取所有群组配置
func (uc *GetConfigUseCase) GetAllGroupConfigs(ctx context.Context, operatorID int64) (*GetAllGroupConfigsOutput, error) {
	// 验证输入
	if err := validator.UserID(operatorID); err != nil {
		return nil, err
	}

	// 检查操作者权限（需要超级管理员权限才能查看所有群组）
	operator, err := uc.userRepo.FindByID(operatorID)
	if err != nil {
		return nil, errors.Wrap(err, "获取操作者信息失败").
			WithContext("operator_id", fmt.Sprintf("%d", operatorID))
	}

	// 检查是否至少在一个群组中是超级管理员
	isSuperAdmin := false
	for _, perm := range operator.Permissions {
		if perm >= user.PermissionSuperAdmin {
			isSuperAdmin = true
			break
		}
	}

	if !isSuperAdmin {
		return nil, errors.Permission("INSUFFICIENT_PERMISSION", "需要超级管理员权限")
	}

	// 获取所有群组
	groups, err := uc.groupRepo.FindAll()
	if err != nil {
		return nil, errors.Wrap(err, "获取群组列表失败")
	}

	// 构建输出
	output := &GetAllGroupConfigsOutput{
		Groups: make([]*GroupConfigOutput, 0, len(groups)),
		Total:  len(groups),
	}

	for _, grp := range groups {
		groupConfig := &GroupConfigOutput{
			GroupID:   grp.ID,
			Title:     grp.Title,
			Type:      grp.Type,
			Commands:  make(map[string]*CommandConfig),
			Settings:  grp.Settings,
			CreatedAt: grp.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: grp.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		// 转换命令配置
		for cmdName, config := range grp.Commands {
			groupConfig.Commands[cmdName] = &CommandConfig{
				CommandName: config.CommandName,
				Enabled:     config.Enabled,
				UpdatedAt:   config.UpdatedAt.Format("2006-01-02 15:04:05"),
				UpdatedBy:   config.UpdatedBy,
			}
		}

		output.Groups = append(output.Groups, groupConfig)
	}

	return output, nil
}

// UpdateGroupSettingsInput 更新群组设置输入
type UpdateGroupSettingsInput struct {
	OperatorID int64                  // 操作者ID
	GroupID    int64                  // 群组ID
	Settings   map[string]interface{} // 要更新的设置
}

// UpdateGroupSettings 更新群组设置
func (uc *GetConfigUseCase) UpdateGroupSettings(ctx context.Context, input UpdateGroupSettingsInput) error {
	// 验证输入
	if err := validator.UserID(input.OperatorID); err != nil {
		return err
	}
	if err := validator.GroupID(input.GroupID); err != nil {
		return err
	}
	if len(input.Settings) == 0 {
		return errors.Validation("EMPTY_SETTINGS", "设置不能为空")
	}

	// 检查操作者权限
	operator, err := uc.userRepo.FindByID(input.OperatorID)
	if err != nil {
		return errors.Wrap(err, "获取操作者信息失败").
			WithContext("operator_id", fmt.Sprintf("%d", input.OperatorID))
	}

	if !operator.IsAdmin(input.GroupID) {
		return errors.Permission("INSUFFICIENT_PERMISSION", "需要管理员权限")
	}

	// 获取群组
	grp, err := uc.groupRepo.FindByID(input.GroupID)
	if err != nil {
		if err == group.ErrGroupNotFound {
			// 如果群组不存在，创建新群组
			grp = group.NewGroup(input.GroupID, "", "group")
		} else {
			return errors.Wrap(err, "获取群组信息失败").
				WithContext("group_id", fmt.Sprintf("%d", input.GroupID))
		}
	}

	// 更新设置
	for key, value := range input.Settings {
		grp.SetSetting(key, value)
	}

	// 保存更新
	if err := uc.groupRepo.Update(grp); err != nil {
		// 如果更新失败，尝试保存新群组
		if err := uc.groupRepo.Save(grp); err != nil {
			return errors.Wrap(err, "保存群组配置失败")
		}
	}

	return nil
}

// GetGroupSetting 获取单个群组设置
func (uc *GetConfigUseCase) GetGroupSetting(ctx context.Context, operatorID, groupID int64, key string) (interface{}, error) {
	// 验证输入
	if err := validator.UserID(operatorID); err != nil {
		return nil, err
	}
	if err := validator.GroupID(groupID); err != nil {
		return nil, err
	}
	if key == "" {
		return nil, errors.Validation("EMPTY_KEY", "设置键不能为空")
	}

	// 获取群组
	grp, err := uc.groupRepo.FindByID(groupID)
	if err != nil {
		if err == group.ErrGroupNotFound {
			return nil, errors.NotFound("GROUP_NOT_FOUND", "群组不存在")
		}
		return nil, errors.Wrap(err, "获取群组信息失败").
			WithContext("group_id", fmt.Sprintf("%d", groupID))
	}

	// 获取设置
	value, ok := grp.GetSetting(key)
	if !ok {
		return nil, errors.NotFound("SETTING_NOT_FOUND", "设置不存在")
	}

	return value, nil
}

// SetGroupSetting 设置单个群组配置项
func (uc *GetConfigUseCase) SetGroupSetting(ctx context.Context, operatorID, groupID int64, key string, value interface{}) error {
	// 验证输入
	if err := validator.UserID(operatorID); err != nil {
		return err
	}
	if err := validator.GroupID(groupID); err != nil {
		return err
	}
	if key == "" {
		return errors.Validation("EMPTY_KEY", "设置键不能为空")
	}

	// 检查操作者权限
	operator, err := uc.userRepo.FindByID(operatorID)
	if err != nil {
		return errors.Wrap(err, "获取操作者信息失败").
			WithContext("operator_id", fmt.Sprintf("%d", operatorID))
	}

	if !operator.IsAdmin(groupID) {
		return errors.Permission("INSUFFICIENT_PERMISSION", "需要管理员权限")
	}

	// 获取群组
	grp, err := uc.groupRepo.FindByID(groupID)
	if err != nil {
		if err == group.ErrGroupNotFound {
			// 如果群组不存在，创建新群组
			grp = group.NewGroup(groupID, "", "group")
		} else {
			return errors.Wrap(err, "获取群组信息失败").
				WithContext("group_id", fmt.Sprintf("%d", groupID))
		}
	}

	// 设置配置项
	grp.SetSetting(key, value)

	// 保存更新
	if err := uc.groupRepo.Update(grp); err != nil {
		// 如果更新失败，尝试保存新群组
		if err := uc.groupRepo.Save(grp); err != nil {
			return errors.Wrap(err, "保存群组配置失败")
		}
	}

	return nil
}
