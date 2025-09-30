package admin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	userUseCase "telegram-bot/internal/usecase/user"
)

// Handler Admin management command handler
type Handler struct {
	groupRepo      group.Repository
	userRepo       user.Repository
	manageAdminUC  *userUseCase.ManageAdminUseCase
}

// NewHandler creates a new admin management command handler
func NewHandler(groupRepo group.Repository, userRepo user.Repository, manageAdminUC *userUseCase.ManageAdminUseCase) *Handler {
	return &Handler{
		groupRepo:     groupRepo,
		userRepo:      userRepo,
		manageAdminUC: manageAdminUC,
	}
}

// Name returns the command name
func (h *Handler) Name() string {
	return "admin"
}

// Description returns the command description
func (h *Handler) Description() string {
	return "管理群组管理员"
}

// RequiredPermission returns the required permission
func (h *Handler) RequiredPermission() user.Permission {
	return user.PermissionAdmin
}

// IsEnabled checks if the command is enabled in the group
func (h *Handler) IsEnabled(groupID int64) bool {
	g, err := h.groupRepo.FindByID(groupID)
	if err != nil {
		return true
	}
	return g.IsCommandEnabled(h.Name())
}

// Handle processes the admin command
func (h *Handler) Handle(ctx *command.Context) error {
	if len(ctx.Args) == 0 {
		return h.listAdmins(ctx)
	}

	subcommand := ctx.Args[0]
	switch subcommand {
	case "list":
		return h.listAdmins(ctx)
	case "promote":
		return h.promoteAdmin(ctx)
	case "demote":
		return h.demoteAdmin(ctx)
	case "info":
		return h.showAdminInfo(ctx)
	default:
		return h.showHelp(ctx)
	}
}

// listAdmins lists all admins in the group
func (h *Handler) listAdmins(ctx *command.Context) error {
	admins, err := h.userRepo.FindAdminsByGroup(ctx.GroupID)
	if err != nil {
		return sendMessage(ctx, "⚠️ 获取管理员列表失败")
	}

	if len(admins) == 0 {
		return sendMessage(ctx, "ℹ️ 当前群组没有管理员")
	}

	response := "👥 *管理员列表*\n\n"

	// Group by permission level
	owners := []*user.User{}
	superAdmins := []*user.User{}
	regularAdmins := []*user.User{}

	for _, admin := range admins {
		perm := admin.GetPermission(ctx.GroupID)
		switch perm {
		case user.PermissionOwner:
			owners = append(owners, admin)
		case user.PermissionSuperAdmin:
			superAdmins = append(superAdmins, admin)
		case user.PermissionAdmin:
			regularAdmins = append(regularAdmins, admin)
		}
	}

	// Display owners
	if len(owners) > 0 {
		response += "👑 *群主*\n"
		for _, admin := range owners {
			response += formatAdminInfo(admin)
		}
		response += "\n"
	}

	// Display super admins
	if len(superAdmins) > 0 {
		response += "⭐ *超级管理员*\n"
		for _, admin := range superAdmins {
			response += formatAdminInfo(admin)
		}
		response += "\n"
	}

	// Display regular admins
	if len(regularAdmins) > 0 {
		response += "👮 *管理员*\n"
		for _, admin := range regularAdmins {
			response += formatAdminInfo(admin)
		}
		response += "\n"
	}

	response += fmt.Sprintf("*总计*: %d 位管理员\n\n", len(admins))
	response += "*可用命令*:\n"
	response += "• `/admin promote <用户ID> <级别>` - 提升权限\n"
	response += "• `/admin demote <用户ID>` - 降低权限\n"
	response += "• `/admin info <用户ID>` - 查看用户信息"

	return sendMessage(ctx, response)
}

// promoteAdmin promotes a user to admin
func (h *Handler) promoteAdmin(ctx *command.Context) error {
	if len(ctx.Args) < 3 {
		return sendMessage(ctx, "❌ 请指定用户ID和权限级别\n\n"+
			"用法: `/admin promote <用户ID> <级别>`\n"+
			"级别: user, admin, superadmin, owner")
	}

	// Parse target user ID
	targetID, err := strconv.ParseInt(ctx.Args[1], 10, 64)
	if err != nil {
		return sendMessage(ctx, "❌ 无效的用户ID")
	}

	// Parse permission level
	permLevel := strings.ToLower(ctx.Args[2])
	var permission user.Permission
	switch permLevel {
	case "user":
		permission = user.PermissionUser
	case "admin":
		permission = user.PermissionAdmin
	case "superadmin":
		permission = user.PermissionSuperAdmin
	case "owner":
		permission = user.PermissionOwner
	default:
		return sendMessage(ctx, "❌ 无效的权限级别\n\n可用级别: user, admin, superadmin, owner")
	}

	// Call use case
	input := userUseCase.PromoteAdminInput{
		OperatorID: ctx.UserID,
		TargetID:   targetID,
		GroupID:    ctx.GroupID,
		Permission: permission,
	}

	if err := h.manageAdminUC.PromoteAdmin(context.Background(), input); err != nil {
		return sendMessage(ctx, fmt.Sprintf("❌ 提升权限失败: %s", err.Error()))
	}

	return sendMessage(ctx, fmt.Sprintf("✅ 已将用户 `%d` 提升为 %s",
		targetID, getPermissionLabel(permission)))
}

// demoteAdmin demotes an admin
func (h *Handler) demoteAdmin(ctx *command.Context) error {
	if len(ctx.Args) < 2 {
		return sendMessage(ctx, "❌ 请指定用户ID\n\n用法: `/admin demote <用户ID>`")
	}

	// Parse target user ID
	targetID, err := strconv.ParseInt(ctx.Args[1], 10, 64)
	if err != nil {
		return sendMessage(ctx, "❌ 无效的用户ID")
	}

	// Parse optional permission level (default to user)
	permission := user.PermissionUser
	if len(ctx.Args) >= 3 {
		permLevel := strings.ToLower(ctx.Args[2])
		switch permLevel {
		case "user":
			permission = user.PermissionUser
		case "admin":
			permission = user.PermissionAdmin
		case "superadmin":
			permission = user.PermissionSuperAdmin
		default:
			return sendMessage(ctx, "❌ 无效的权限级别\n\n可用级别: user, admin, superadmin")
		}
	}

	// Call use case
	input := userUseCase.DemoteAdminInput{
		OperatorID: ctx.UserID,
		TargetID:   targetID,
		GroupID:    ctx.GroupID,
		Permission: permission,
	}

	if err := h.manageAdminUC.DemoteAdmin(context.Background(), input); err != nil {
		return sendMessage(ctx, fmt.Sprintf("❌ 降低权限失败: %s", err.Error()))
	}

	return sendMessage(ctx, fmt.Sprintf("✅ 已将用户 `%d` 降级为 %s",
		targetID, getPermissionLabel(permission)))
}

// showAdminInfo shows detailed info about an admin
func (h *Handler) showAdminInfo(ctx *command.Context) error {
	if len(ctx.Args) < 2 {
		return sendMessage(ctx, "❌ 请指定用户ID\n\n用法: `/admin info <用户ID>`")
	}

	// Parse user ID
	userID, err := strconv.ParseInt(ctx.Args[1], 10, 64)
	if err != nil {
		return sendMessage(ctx, "❌ 无效的用户ID")
	}

	// Get user
	u, err := h.userRepo.FindByID(userID)
	if err != nil {
		return sendMessage(ctx, fmt.Sprintf("❌ 找不到用户 `%d`", userID))
	}

	// Get permission in current group
	perm := u.GetPermission(ctx.GroupID)

	// Build response
	fullName := u.FirstName
	if u.LastName != "" {
		fullName += " " + u.LastName
	}

	username := u.Username
	if username != "" {
		username = "@" + username
	} else {
		username = "无"
	}

	response := "👤 *用户信息*\n\n"
	response += fmt.Sprintf("*ID*: `%d`\n", u.ID)
	response += fmt.Sprintf("*姓名*: %s\n", fullName)
	response += fmt.Sprintf("*用户名*: %s\n", username)
	response += fmt.Sprintf("*当前权限*: %s\n", getPermissionLabel(perm))
	response += fmt.Sprintf("*创建时间*: %s\n", u.CreatedAt.Format("2006-01-02 15:04:05"))

	// Count admin groups
	adminGroupCount := 0
	for _, p := range u.Permissions {
		if p >= user.PermissionAdmin {
			adminGroupCount++
		}
	}
	response += fmt.Sprintf("*管理群组数*: %d", adminGroupCount)

	return sendMessage(ctx, response)
}

// showHelp shows help for the admin command
func (h *Handler) showHelp(ctx *command.Context) error {
	response := "❌ 未知的子命令\n\n"
	response += "*可用命令*:\n"
	response += "• `/admin` 或 `/admin list` - 列出所有管理员\n"
	response += "• `/admin promote <用户ID> <级别>` - 提升权限\n"
	response += "  级别: user, admin, superadmin, owner\n"
	response += "• `/admin demote <用户ID>` - 降低权限为普通用户\n"
	response += "• `/admin info <用户ID>` - 查看用户详细信息\n\n"
	response += "*权限说明*:\n"
	response += "• user - 普通用户\n"
	response += "• admin - 管理员\n"
	response += "• superadmin - 超级管理员\n"
	response += "• owner - 群主"

	return sendMessage(ctx, response)
}

// formatAdminInfo formats admin info for display
func formatAdminInfo(u *user.User) string {
	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}

	username := ""
	if u.Username != "" {
		username = " (@" + u.Username + ")"
	}

	return fmt.Sprintf("• %s%s - `%d`\n", name, username, u.ID)
}

// getPermissionLabel returns a human-readable permission label
func getPermissionLabel(perm user.Permission) string {
	switch perm {
	case user.PermissionNone:
		return "🚫 无权限"
	case user.PermissionUser:
		return "👤 普通用户"
	case user.PermissionAdmin:
		return "👮 管理员"
	case user.PermissionSuperAdmin:
		return "⭐ 超级管理员"
	case user.PermissionOwner:
		return "👑 群主"
	default:
		return "❓ 未知"
	}
}

// sendMessage helper function to send message
func sendMessage(ctx *command.Context, text string) error {
	// This will be called by Telegram API
	fmt.Printf("Send to group %d: %s\n", ctx.GroupID, text)
	return nil
}
