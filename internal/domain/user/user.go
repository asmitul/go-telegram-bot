package user

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

// Permission 权限等级
type Permission int

const (
	PermissionUser       Permission = 1
	PermissionAdmin      Permission = 2
	PermissionSuperAdmin Permission = 3
	PermissionOwner      Permission = 4
)

func (p Permission) String() string {
	switch p {
	case PermissionUser:
		return "User"
	case PermissionAdmin:
		return "Admin"
	case PermissionSuperAdmin:
		return "SuperAdmin"
	case PermissionOwner:
		return "Owner"
	default:
		return "Unknown"
	}
}

func (p Permission) CanManage(target Permission) bool {
	return p > target
}

// User 用户聚合根
type User struct {
	ID          int64
	Username    string
	FirstName   string
	LastName    string
	Permissions map[int64]Permission // groupID -> Permission
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewUser 创建新用户
func NewUser(id int64, username, firstName, lastName string) *User {
	now := time.Now()
	return &User{
		ID:          id,
		Username:    username,
		FirstName:   firstName,
		LastName:    lastName,
		Permissions: make(map[int64]Permission),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// GetPermission 获取用户在特定群组的权限
// 返回全局权限和群组权限中的较高值
func (u *User) GetPermission(groupID int64) Permission {
	globalPerm := PermissionUser
	groupPerm := PermissionUser

	// 检查全局权限（groupID = 0）
	if perm, ok := u.Permissions[0]; ok {
		globalPerm = perm
	}

	// 检查群组特定权限
	if perm, ok := u.Permissions[groupID]; ok {
		groupPerm = perm
	}

	// 返回两者中的较高权限
	if globalPerm > groupPerm {
		return globalPerm
	}
	return groupPerm
}

// SetPermission 设置用户在特定群组的权限
func (u *User) SetPermission(groupID int64, perm Permission) {
	u.Permissions[groupID] = perm
	u.UpdatedAt = time.Now()
}

// HasPermission 检查用户是否有足够权限
func (u *User) HasPermission(groupID int64, required Permission) bool {
	return u.GetPermission(groupID) >= required
}

// IsSuperAdmin 检查是否为超级管理员
func (u *User) IsSuperAdmin(groupID int64) bool {
	return u.GetPermission(groupID) >= PermissionSuperAdmin
}

// IsAdmin 检查是否为管理员（包括普通、高级、超级）
func (u *User) IsAdmin(groupID int64) bool {
	return u.GetPermission(groupID) >= PermissionAdmin
}

// Repository 用户仓储接口
type Repository interface {
	FindByID(id int64) (*User, error)
	FindByUsername(username string) (*User, error)
	Save(user *User) error
	Update(user *User) error
	UpdatePermission(userID int64, groupID int64, perm Permission) error // 细粒度权限更新，避免并发冲突
	Delete(id int64) error
	FindAdminsByGroup(groupID int64) ([]*User, error)
}
