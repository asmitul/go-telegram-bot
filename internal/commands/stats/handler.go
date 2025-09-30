package stats

import (
	"fmt"
	"runtime"
	"time"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
)

// Stats 统计数据
type Stats struct {
	BotStartTime    time.Time
	TotalMessages   int64
	CommandsHandled int64
	ActiveGroups    int
	ActiveUsers     int
}

// Handler Stats 命令处理器
type Handler struct {
	groupRepo group.Repository
	userRepo  user.Repository
	stats     *Stats
}

// NewHandler 创建 Stats 命令处理器
func NewHandler(groupRepo group.Repository, userRepo user.Repository, stats *Stats) *Handler {
	if stats == nil {
		stats = &Stats{
			BotStartTime: time.Now(),
		}
	}
	return &Handler{
		groupRepo: groupRepo,
		userRepo:  userRepo,
		stats:     stats,
	}
}

// Name 命令名称
func (h *Handler) Name() string {
	return "stats"
}

// Description 命令描述
func (h *Handler) Description() string {
	return "显示群组和机器人统计信息"
}

// RequiredPermission 所需权限
func (h *Handler) RequiredPermission() user.Permission {
	return user.PermissionUser // 所有用户都可以使用
}

// IsEnabled 检查命令是否在群组中启用
func (h *Handler) IsEnabled(groupID int64) bool {
	g, err := h.groupRepo.FindByID(groupID)
	if err != nil {
		return true // 默认启用
	}
	return g.IsCommandEnabled(h.Name())
}

// Handle 处理命令
func (h *Handler) Handle(ctx *command.Context) error {
	// 检查是否指定了子命令
	if len(ctx.Args) > 0 {
		switch ctx.Args[0] {
		case "bot":
			return h.showBotStats(ctx)
		case "group":
			return h.showGroupStats(ctx)
		case "user":
			return h.showUserStats(ctx)
		default:
			return h.showAllStats(ctx)
		}
	}

	// 默认显示所有统计
	return h.showAllStats(ctx)
}

// showAllStats 显示所有统计信息
func (h *Handler) showAllStats(ctx *command.Context) error {
	var response string

	// Bot 统计
	botStats := h.getBotStats()
	response += "🤖 *机器人统计*\n"
	response += fmt.Sprintf("运行时间: %s\n", formatDuration(time.Since(h.stats.BotStartTime)))
	response += fmt.Sprintf("处理消息数: %d\n", h.stats.TotalMessages)
	response += fmt.Sprintf("处理命令数: %d\n", h.stats.CommandsHandled)
	response += fmt.Sprintf("内存使用: %.2f MB\n", botStats.MemoryUsageMB)
	response += fmt.Sprintf("协程数: %d\n\n", botStats.Goroutines)

	// 群组统计
	groupStats := h.getGroupStats(ctx.GroupID)
	response += "👥 *群组统计*\n"
	response += fmt.Sprintf("活跃群组数: %d\n", h.stats.ActiveGroups)
	if groupStats != nil {
		response += fmt.Sprintf("当前群组成员: %d (估算)\n", groupStats.MemberCount)
		response += fmt.Sprintf("管理员数: %d\n", groupStats.AdminCount)
	}
	response += "\n"

	// 用户统计
	response += "👤 *用户统计*\n"
	response += fmt.Sprintf("活跃用户数: %d\n", h.stats.ActiveUsers)

	response += "\n💡 提示: 使用 `/stats bot`、`/stats group` 或 `/stats user` 查看详细统计"

	return sendMessage(ctx, response)
}

// showBotStats 显示机器人统计
func (h *Handler) showBotStats(ctx *command.Context) error {
	botStats := h.getBotStats()

	response := "🤖 *机器人详细统计*\n\n"
	response += fmt.Sprintf("*运行时间*: %s\n", formatDuration(time.Since(h.stats.BotStartTime)))
	response += fmt.Sprintf("*启动时间*: %s\n", h.stats.BotStartTime.Format("2006-01-02 15:04:05"))
	response += fmt.Sprintf("*处理消息*: %d\n", h.stats.TotalMessages)
	response += fmt.Sprintf("*处理命令*: %d\n", h.stats.CommandsHandled)
	response += fmt.Sprintf("*平均响应*: %.2f msg/min\n\n", botStats.AvgMessagesPerMin)

	response += "*系统信息*\n"
	response += fmt.Sprintf("内存使用: %.2f MB\n", botStats.MemoryUsageMB)
	response += fmt.Sprintf("协程数: %d\n", botStats.Goroutines)
	response += fmt.Sprintf("Go 版本: %s\n", runtime.Version())
	response += fmt.Sprintf("CPU 核心: %d\n", runtime.NumCPU())

	return sendMessage(ctx, response)
}

// showGroupStats 显示群组统计
func (h *Handler) showGroupStats(ctx *command.Context) error {
	groupStats := h.getGroupStats(ctx.GroupID)
	if groupStats == nil {
		return sendMessage(ctx, "⚠️ 无法获取群组统计信息")
	}

	response := "👥 *群组详细统计*\n\n"
	response += fmt.Sprintf("*群组 ID*: `%d`\n", ctx.GroupID)
	response += fmt.Sprintf("*群组名称*: %s\n", groupStats.GroupTitle)
	response += fmt.Sprintf("*成员数*: %d (估算)\n", groupStats.MemberCount)
	response += fmt.Sprintf("*管理员数*: %d\n", groupStats.AdminCount)
	response += fmt.Sprintf("*启用命令数*: %d\n", groupStats.EnabledCommands)
	response += fmt.Sprintf("*禁用命令数*: %d\n", groupStats.DisabledCommands)
	response += fmt.Sprintf("*创建时间*: %s\n", groupStats.CreatedAt)

	return sendMessage(ctx, response)
}

// showUserStats 显示用户统计
func (h *Handler) showUserStats(ctx *command.Context) error {
	userStats := h.getUserStats(ctx.UserID, ctx.GroupID)
	if userStats == nil {
		return sendMessage(ctx, "⚠️ 无法获取用户统计信息")
	}

	response := "👤 *用户详细统计*\n\n"
	response += fmt.Sprintf("*用户 ID*: `%d`\n", ctx.UserID)
	response += fmt.Sprintf("*用户名*: @%s\n", userStats.Username)
	response += fmt.Sprintf("*姓名*: %s\n", userStats.FullName)
	response += fmt.Sprintf("*当前权限*: %s\n", getPermissionLabel(userStats.Permission))
	response += fmt.Sprintf("*管理群组数*: %d\n", userStats.AdminGroupCount)

	return sendMessage(ctx, response)
}

// BotStats 机器人统计
type BotStats struct {
	MemoryUsageMB     float64
	Goroutines        int
	AvgMessagesPerMin float64
}

// getBotStats 获取机器人统计
func (h *Handler) getBotStats() *BotStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(h.stats.BotStartTime)
	avgMsgPerMin := 0.0
	if uptime.Minutes() > 0 {
		avgMsgPerMin = float64(h.stats.TotalMessages) / uptime.Minutes()
	}

	return &BotStats{
		MemoryUsageMB:     float64(m.Alloc) / 1024 / 1024,
		Goroutines:        runtime.NumGoroutine(),
		AvgMessagesPerMin: avgMsgPerMin,
	}
}

// GroupStats 群组统计
type GroupStats struct {
	GroupTitle       string
	MemberCount      int
	AdminCount       int
	EnabledCommands  int
	DisabledCommands int
	CreatedAt        string
}

// getGroupStats 获取群组统计
func (h *Handler) getGroupStats(groupID int64) *GroupStats {
	g, err := h.groupRepo.FindByID(groupID)
	if err != nil {
		return nil
	}

	// 统计管理员数量
	admins, err := h.userRepo.FindAdminsByGroup(groupID)
	adminCount := 0
	if err == nil {
		adminCount = len(admins)
	}

	// 统计启用/禁用的命令
	enabledCount := 0
	disabledCount := 0
	for _, cmd := range g.Commands {
		if cmd.Enabled {
			enabledCount++
		} else {
			disabledCount++
		}
	}

	return &GroupStats{
		GroupTitle:       g.Title,
		MemberCount:      1000, // 模拟数据，实际需要通过 Telegram API 获取
		AdminCount:       adminCount,
		EnabledCommands:  enabledCount,
		DisabledCommands: disabledCount,
		CreatedAt:        g.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// UserStats 用户统计
type UserStats struct {
	Username        string
	FullName        string
	Permission      user.Permission
	AdminGroupCount int
}

// getUserStats 获取用户统计
func (h *Handler) getUserStats(userID, groupID int64) *UserStats {
	u, err := h.userRepo.FindByID(userID)
	if err != nil {
		return nil
	}

	// 统计管理的群组数
	adminGroupCount := 0
	for _, perm := range u.Permissions {
		if perm >= user.PermissionAdmin {
			adminGroupCount++
		}
	}

	fullName := u.FirstName
	if u.LastName != "" {
		fullName += " " + u.LastName
	}

	return &UserStats{
		Username:        u.Username,
		FullName:        fullName,
		Permission:      u.GetPermission(groupID),
		AdminGroupCount: adminGroupCount,
	}
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d 天 %d 小时 %d 分钟", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d 小时 %d 分钟", hours, minutes)
	}
	return fmt.Sprintf("%d 分钟", minutes)
}

// getPermissionLabel 获取权限标签
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

// IncrementMessage 增加消息计数
func (h *Handler) IncrementMessage() {
	h.stats.TotalMessages++
}

// IncrementCommand 增加命令计数
func (h *Handler) IncrementCommand() {
	h.stats.CommandsHandled++
}

// UpdateActiveGroups 更新活跃群组数
func (h *Handler) UpdateActiveGroups(count int) {
	h.stats.ActiveGroups = count
}

// UpdateActiveUsers 更新活跃用户数
func (h *Handler) UpdateActiveUsers(count int) {
	h.stats.ActiveUsers = count
}

// sendMessage 发送消息的辅助函数
func sendMessage(ctx *command.Context, text string) error {
	// 这里实际会调用 Telegram API
	fmt.Printf("Send to group %d: %s\n", ctx.GroupID, text)
	return nil
}
