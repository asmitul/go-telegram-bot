package user

import (
	"context"
	"errors"
	"telegram-bot/internal/domain/user"
)

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrInsufficientPermission = errors.New("insufficient permission")
)

// CheckPermissionUseCase 权限检查用例
type CheckPermissionUseCase struct {
	userRepo user.Repository
}

// NewCheckPermissionUseCase 创建权限检查用例
func NewCheckPermissionUseCase(userRepo user.Repository) *CheckPermissionUseCase {
	return &CheckPermissionUseCase{
		userRepo: userRepo,
	}
}

// Execute 执行权限检查
func (uc *CheckPermissionUseCase) Execute(ctx context.Context, userID, groupID int64, required user.Permission) error {
	u, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !u.HasPermission(groupID, required) {
		return ErrInsufficientPermission
	}

	return nil
}

// GetUserPermission 获取用户权限
func (uc *CheckPermissionUseCase) GetUserPermission(ctx context.Context, userID, groupID int64) (user.Permission, error) {
	u, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return user.PermissionNone, ErrUserNotFound
	}

	return u.GetPermission(groupID), nil
}

// IsAdmin 检查是否为管理员
func (uc *CheckPermissionUseCase) IsAdmin(ctx context.Context, userID, groupID int64) (bool, error) {
	u, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return false, ErrUserNotFound
	}

	return u.IsAdmin(groupID), nil
}

// IsSuperAdmin 检查是否为超级管理员
func (uc *CheckPermissionUseCase) IsSuperAdmin(ctx context.Context, userID, groupID int64) (bool, error) {
	u, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return false, ErrUserNotFound
	}

	return u.IsSuperAdmin(groupID), nil
}
