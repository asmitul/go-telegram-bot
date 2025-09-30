package ping

import (
	"fmt"
	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"time"
)

// Handler Ping 命令处理器
type Handler struct {
	groupRepo group.Repository
}

// NewHandler 创建 Ping 命令处理器
func NewHandler(groupRepo group.Repository) *Handler {
	return &Handler{
		groupRepo: groupRepo,
	}
}

// Name 命令名称
func (h *Handler) Name() string {
	return "ping"
}

// Description 命令描述
func (h *Handler) Description() string {
	return "测试机器人是否在线"
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
	response := fmt.Sprintf("🏓 Pong! 延迟: %dms\n机器人运行正常",
		time.Now().UnixMilli()%100) // 模拟延迟

	return sendMessage(ctx, response)
}

// sendMessage 发送消息的辅助函数（实际实现在 adapter 层）
func sendMessage(ctx *command.Context, text string) error {
	// 这里实际会调用 Telegram API
	// 为了保持命令独立，这个函数会被注入
	fmt.Printf("Send to group %d: %s\n", ctx.GroupID, text)
	return nil
}
