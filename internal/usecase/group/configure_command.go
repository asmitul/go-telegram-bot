package group

import (
	"context"
	"fmt"

	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/errors"
	"telegram-bot/pkg/validator"
)

// ConfigureCommandUseCase 命令配置用例
type ConfigureCommandUseCase struct {
	groupRepo group.Repository
	userRepo  user.Repository
}

// NewConfigureCommandUseCase 创建命令配置用例
func NewConfigureCommandUseCase(groupRepo group.Repository, userRepo user.Repository) *ConfigureCommandUseCase {
	return &ConfigureCommandUseCase{
		groupRepo: groupRepo,
		userRepo:  userRepo,
	}
}

// EnableCommandInput 启用命令输入
type EnableCommandInput struct {
	OperatorID  int64  // 操作者ID
	GroupID     int64  // 群组ID
	CommandName string // 命令名称
}

// EnableCommand 启用命令
func (uc *ConfigureCommandUseCase) EnableCommand(ctx context.Context, input EnableCommandInput) error {
	// 验证输入
	if err := validator.UserID(input.OperatorID); err != nil {
		return err
	}
	if err := validator.GroupID(input.GroupID); err != nil {
		return err
	}
	if err := validator.CommandName(input.CommandName); err != nil {
		return err
	}

	// 检查操作者权限
	if err := uc.checkPermission(ctx, input.OperatorID, input.GroupID); err != nil {
		return err
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

	// 启用命令
	grp.EnableCommand(input.CommandName, input.OperatorID)

	// 保存更新
	if err := uc.groupRepo.Update(grp); err != nil {
		// 如果更新失败，尝试保存新群组
		if err := uc.groupRepo.Save(grp); err != nil {
			return errors.Wrap(err, "保存群组配置失败")
		}
	}

	return nil
}

// DisableCommandInput 禁用命令输入
type DisableCommandInput struct {
	OperatorID  int64  // 操作者ID
	GroupID     int64  // 群组ID
	CommandName string // 命令名称
}

// DisableCommand 禁用命令
func (uc *ConfigureCommandUseCase) DisableCommand(ctx context.Context, input DisableCommandInput) error {
	// 验证输入
	if err := validator.UserID(input.OperatorID); err != nil {
		return err
	}
	if err := validator.GroupID(input.GroupID); err != nil {
		return err
	}
	if err := validator.CommandName(input.CommandName); err != nil {
		return err
	}

	// 检查操作者权限
	if err := uc.checkPermission(ctx, input.OperatorID, input.GroupID); err != nil {
		return err
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

	// 禁用命令
	grp.DisableCommand(input.CommandName, input.OperatorID)

	// 保存更新
	if err := uc.groupRepo.Update(grp); err != nil {
		// 如果更新失败，尝试保存新群组
		if err := uc.groupRepo.Save(grp); err != nil {
			return errors.Wrap(err, "保存群组配置失败")
		}
	}

	return nil
}

// GetCommandStatusInput 获取命令状态输入
type GetCommandStatusInput struct {
	OperatorID  int64  // 操作者ID
	GroupID     int64  // 群组ID
	CommandName string // 命令名称
}

// CommandStatusOutput 命令状态输出
type CommandStatusOutput struct {
	CommandName string
	Enabled     bool
	UpdatedBy   int64
	UpdatedAt   string
}

// GetCommandStatus 获取命令状态
func (uc *ConfigureCommandUseCase) GetCommandStatus(ctx context.Context, input GetCommandStatusInput) (*CommandStatusOutput, error) {
	// 验证输入
	if err := validator.UserID(input.OperatorID); err != nil {
		return nil, err
	}
	if err := validator.GroupID(input.GroupID); err != nil {
		return nil, err
	}
	if err := validator.CommandName(input.CommandName); err != nil {
		return nil, err
	}

	// 获取群组
	grp, err := uc.groupRepo.FindByID(input.GroupID)
	if err != nil {
		if err == group.ErrGroupNotFound {
			// 群组不存在，返回默认状态（启用）
			return &CommandStatusOutput{
				CommandName: input.CommandName,
				Enabled:     true,
			}, nil
		}
		return nil, errors.Wrap(err, "获取群组信息失败").
			WithContext("group_id", fmt.Sprintf("%d", input.GroupID))
	}

	// 获取命令配置
	config := grp.GetCommandConfig(input.CommandName)

	return &CommandStatusOutput{
		CommandName: config.CommandName,
		Enabled:     config.Enabled,
		UpdatedBy:   config.UpdatedBy,
		UpdatedAt:   config.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// BatchConfigureInput 批量配置输入
type BatchConfigureInput struct {
	OperatorID   int64             // 操作者ID
	GroupID      int64             // 群组ID
	Commands     map[string]bool   // 命令名称 -> 启用状态
}

// BatchConfigure 批量配置命令
func (uc *ConfigureCommandUseCase) BatchConfigure(ctx context.Context, input BatchConfigureInput) error {
	// 验证输入
	if err := validator.UserID(input.OperatorID); err != nil {
		return err
	}
	if err := validator.GroupID(input.GroupID); err != nil {
		return err
	}
	if len(input.Commands) == 0 {
		return errors.Validation("EMPTY_COMMANDS", "命令列表不能为空")
	}

	// 验证所有命令名称
	for cmdName := range input.Commands {
		if err := validator.CommandName(cmdName); err != nil {
			return err
		}
	}

	// 检查操作者权限
	if err := uc.checkPermission(ctx, input.OperatorID, input.GroupID); err != nil {
		return err
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

	// 批量配置命令
	for cmdName, enabled := range input.Commands {
		if enabled {
			grp.EnableCommand(cmdName, input.OperatorID)
		} else {
			grp.DisableCommand(cmdName, input.OperatorID)
		}
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

// ListCommandsOutput 命令列表输出
type ListCommandsOutput struct {
	Commands []*CommandStatusOutput
	Total    int
}

// ListCommands 列出所有命令配置
func (uc *ConfigureCommandUseCase) ListCommands(ctx context.Context, operatorID, groupID int64) (*ListCommandsOutput, error) {
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
			// 群组不存在，返回空列表
			return &ListCommandsOutput{
				Commands: []*CommandStatusOutput{},
				Total:    0,
			}, nil
		}
		return nil, errors.Wrap(err, "获取群组信息失败").
			WithContext("group_id", fmt.Sprintf("%d", groupID))
	}

	// 构建输出
	output := &ListCommandsOutput{
		Commands: make([]*CommandStatusOutput, 0, len(grp.Commands)),
		Total:    len(grp.Commands),
	}

	for _, config := range grp.Commands {
		output.Commands = append(output.Commands, &CommandStatusOutput{
			CommandName: config.CommandName,
			Enabled:     config.Enabled,
			UpdatedBy:   config.UpdatedBy,
			UpdatedAt:   config.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return output, nil
}

// checkPermission 检查操作者权限（必须是管理员或更高）
func (uc *ConfigureCommandUseCase) checkPermission(ctx context.Context, operatorID, groupID int64) error {
	operator, err := uc.userRepo.FindByID(operatorID)
	if err != nil {
		return errors.Wrap(err, "获取操作者信息失败").
			WithContext("operator_id", fmt.Sprintf("%d", operatorID))
	}

	if !operator.IsAdmin(groupID) {
		return errors.Permission("INSUFFICIENT_PERMISSION", "需要管理员权限")
	}

	return nil
}
