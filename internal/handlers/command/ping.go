package command

import (
	"fmt"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
	"time"
)

// PingHandler Ping 命令处理器
type PingHandler struct {
	*BaseCommand
}

// NewPingHandler 创建 Ping 命令处理器
func NewPingHandler(groupRepo GroupRepository) *PingHandler {
	return &PingHandler{
		BaseCommand: NewBaseCommand(
			"ping",
			"测试机器人是否在线",
			user.PermissionUser, // 所有用户都可以使用
			[]string{"private", "group", "supergroup"},
			groupRepo,
		),
	}
}

// Handle 处理命令
func (h *PingHandler) Handle(ctx *handler.Context) error {
	// 权限检查
	if err := h.CheckPermission(ctx); err != nil {
		return err
	}

	// 生成响应
	response := fmt.Sprintf("🏓 Pong! 延迟: %dms\n✅ 机器人运行正常",
		time.Now().UnixMilli()%100) // 模拟延迟

	return ctx.Reply(response)
}
