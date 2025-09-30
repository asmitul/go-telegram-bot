package user

import (
	"context"
	"fmt"

	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/errors"
	"telegram-bot/pkg/validator"
)

// ManageAdminUseCase 管理员管理用例
type ManageAdminUseCase struct {
	userRepo user.Repository
}

// NewManageAdminUseCase 创建管理员管理用例
func NewManageAdminUseCase(userRepo user.Repository) *ManageAdminUseCase {
	return &ManageAdminUseCase{
		userRepo: userRepo,
	}
}

// PromoteAdminInput 提升管理员输入
type PromoteAdminInput struct {
	OperatorID int64           // 操作者ID
	TargetID   int64           // 目标用户ID
	GroupID    int64           // 群组ID
	Permission user.Permission // 要设置的权限级别
}

// PromoteAdmin 提升管理员
func (uc *ManageAdminUseCase) PromoteAdmin(ctx context.Context, input PromoteAdminInput) error {
	// 验证输入
	if err := validator.UserID(input.OperatorID); err != nil {
		return err
	}
	if err := validator.UserID(input.TargetID); err != nil {
		return err
	}
	if err := validator.GroupID(input.GroupID); err != nil {
		return err
	}

	// 不能对自己操作
	if input.OperatorID == input.TargetID {
		return errors.Permission("SELF_PROMOTION", "不能对自己进行权限操作")
	}

	// 验证权限级别有效性
	if input.Permission < user.PermissionUser || input.Permission > user.PermissionOwner {
		return errors.Validation("INVALID_PERMISSION", "无效的权限级别")
	}

	// 获取操作者
	operator, err := uc.userRepo.FindByID(input.OperatorID)
	if err != nil {
		return errors.Wrap(err, "获取操作者信息失败").
			WithContext("operator_id", fmt.Sprintf("%d", input.OperatorID))
	}

	// 获取目标用户
	target, err := uc.userRepo.FindByID(input.TargetID)
	if err != nil {
		return errors.Wrap(err, "获取目标用户信息失败").
			WithContext("target_id", fmt.Sprintf("%d", input.TargetID))
	}

	// 检查操作者权限
	operatorPerm := operator.GetPermission(input.GroupID)
	targetPerm := target.GetPermission(input.GroupID)

	// 操作者必须是管理员或更高权限
	if operatorPerm < user.PermissionAdmin {
		return errors.Permission("INSUFFICIENT_PERMISSION", "操作者权限不足")
	}

	// 操作者权限必须高于或等于要设置的权限（先检查，避免提升到比自己高的权限）
	if operatorPerm < input.Permission {
		return errors.Permission("CANNOT_PROMOTE_HIGHER", "无法提升用户到比自己更高的权限等级")
	}

	// 操作者权限必须高于目标用户当前权限
	if !operatorPerm.CanManage(targetPerm) {
		return errors.Permission("CANNOT_MANAGE_TARGET", "无法管理权限等级相同或更高的用户")
	}

	// 设置权限
	target.SetPermission(input.GroupID, input.Permission)

	// 保存更新
	if err := uc.userRepo.Update(target); err != nil {
		return errors.Wrap(err, "更新用户权限失败")
	}

	return nil
}

// DemoteAdminInput 降级管理员输入
type DemoteAdminInput struct {
	OperatorID int64           // 操作者ID
	TargetID   int64           // 目标用户ID
	GroupID    int64           // 群组ID
	Permission user.Permission // 要设置的权限级别（可选，默认为普通用户）
}

// DemoteAdmin 降级管理员
func (uc *ManageAdminUseCase) DemoteAdmin(ctx context.Context, input DemoteAdminInput) error {
	// 验证输入
	if err := validator.UserID(input.OperatorID); err != nil {
		return err
	}
	if err := validator.UserID(input.TargetID); err != nil {
		return err
	}
	if err := validator.GroupID(input.GroupID); err != nil {
		return err
	}

	// 不能对自己操作
	if input.OperatorID == input.TargetID {
		return errors.Permission("SELF_DEMOTION", "不能对自己进行权限操作")
	}

	// 如果未指定权限，默认降级为普通用户
	if input.Permission == 0 {
		input.Permission = user.PermissionUser
	}

	// 获取操作者
	operator, err := uc.userRepo.FindByID(input.OperatorID)
	if err != nil {
		return errors.Wrap(err, "获取操作者信息失败").
			WithContext("operator_id", fmt.Sprintf("%d", input.OperatorID))
	}

	// 获取目标用户
	target, err := uc.userRepo.FindByID(input.TargetID)
	if err != nil {
		return errors.Wrap(err, "获取目标用户信息失败").
			WithContext("target_id", fmt.Sprintf("%d", input.TargetID))
	}

	// 检查操作者权限
	operatorPerm := operator.GetPermission(input.GroupID)
	targetPerm := target.GetPermission(input.GroupID)

	// 操作者必须是管理员或更高权限
	if operatorPerm < user.PermissionAdmin {
		return errors.Permission("INSUFFICIENT_PERMISSION", "操作者权限不足")
	}

	// 操作者权限必须高于目标用户当前权限
	if !operatorPerm.CanManage(targetPerm) {
		return errors.Permission("CANNOT_MANAGE_TARGET", "无法管理权限等级相同或更高的用户")
	}

	// 只能降级，不能提升
	if input.Permission >= targetPerm {
		return errors.Validation("INVALID_DEMOTION", "降级后的权限必须低于当前权限")
	}

	// 设置权限
	target.SetPermission(input.GroupID, input.Permission)

	// 保存更新
	if err := uc.userRepo.Update(target); err != nil {
		return errors.Wrap(err, "更新用户权限失败")
	}

	return nil
}

// ListAdminsOutput 管理员列表输出
type ListAdminsOutput struct {
	Admins []*AdminInfo
	Total  int
}

// AdminInfo 管理员信息
type AdminInfo struct {
	UserID     int64
	Username   string
	FirstName  string
	LastName   string
	Permission user.Permission
}

// ListAdmins 列出所有管理员
func (uc *ManageAdminUseCase) ListAdmins(ctx context.Context, operatorID, groupID int64) (*ListAdminsOutput, error) {
	// 验证输入
	if err := validator.UserID(operatorID); err != nil {
		return nil, err
	}
	if err := validator.GroupID(groupID); err != nil {
		return nil, err
	}

	// 获取操作者
	operator, err := uc.userRepo.FindByID(operatorID)
	if err != nil {
		return nil, errors.Wrap(err, "获取操作者信息失败").
			WithContext("operator_id", fmt.Sprintf("%d", operatorID))
	}

	// 检查操作者权限（至少是普通用户即可查看管理员列表）
	operatorPerm := operator.GetPermission(groupID)
	if operatorPerm < user.PermissionUser {
		return nil, errors.Permission("INSUFFICIENT_PERMISSION", "操作者权限不足")
	}

	// 获取管理员列表
	admins, err := uc.userRepo.FindAdminsByGroup(groupID)
	if err != nil {
		return nil, errors.Wrap(err, "获取管理员列表失败").
			WithContext("group_id", fmt.Sprintf("%d", groupID))
	}

	// 构建输出
	output := &ListAdminsOutput{
		Admins: make([]*AdminInfo, 0, len(admins)),
		Total:  len(admins),
	}

	for _, admin := range admins {
		output.Admins = append(output.Admins, &AdminInfo{
			UserID:     admin.ID,
			Username:   admin.Username,
			FirstName:  admin.FirstName,
			LastName:   admin.LastName,
			Permission: admin.GetPermission(groupID),
		})
	}

	return output, nil
}

// RemoveAdmin 移除管理员（降级为普通用户）
func (uc *ManageAdminUseCase) RemoveAdmin(ctx context.Context, operatorID, targetID, groupID int64) error {
	return uc.DemoteAdmin(ctx, DemoteAdminInput{
		OperatorID: operatorID,
		TargetID:   targetID,
		GroupID:    groupID,
		Permission: user.PermissionUser,
	})
}

// SetPermission 直接设置用户权限（超级管理员专用）
func (uc *ManageAdminUseCase) SetPermission(ctx context.Context, operatorID, targetID, groupID int64, permission user.Permission) error {
	// 验证输入
	if err := validator.UserID(operatorID); err != nil {
		return err
	}
	if err := validator.UserID(targetID); err != nil {
		return err
	}
	if err := validator.GroupID(groupID); err != nil {
		return err
	}

	// 不能对自己操作
	if operatorID == targetID {
		return errors.Permission("SELF_OPERATION", "不能对自己进行权限操作")
	}

	// 获取操作者
	operator, err := uc.userRepo.FindByID(operatorID)
	if err != nil {
		return errors.Wrap(err, "获取操作者信息失败").
			WithContext("operator_id", fmt.Sprintf("%d", operatorID))
	}

	// 获取目标用户
	target, err := uc.userRepo.FindByID(targetID)
	if err != nil {
		return errors.Wrap(err, "获取目标用户信息失败").
			WithContext("target_id", fmt.Sprintf("%d", targetID))
	}

	// 检查操作者权限（必须是超级管理员或所有者）
	operatorPerm := operator.GetPermission(groupID)
	if operatorPerm < user.PermissionSuperAdmin {
		return errors.Permission("INSUFFICIENT_PERMISSION", "只有超级管理员才能直接设置权限")
	}

	// 不能设置比自己更高的权限
	if operatorPerm < permission {
		return errors.Permission("CANNOT_SET_HIGHER", "无法设置比自己更高的权限等级")
	}

	// 设置权限
	target.SetPermission(groupID, permission)

	// 保存更新
	if err := uc.userRepo.Update(target); err != nil {
		return errors.Wrap(err, "更新用户权限失败")
	}

	return nil
}
