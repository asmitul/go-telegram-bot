package command

import (
	"context"
	"telegram-bot/internal/domain/user"
)

// Context 命令执行上下文
type Context struct {
	Ctx       context.Context
	UserID    int64
	GroupID   int64
	MessageID int
	Text      string
	Args      []string
	User      *user.User
}

// Handler 命令处理器接口
type Handler interface {
	// Name 命令名称（不含 /）
	Name() string

	// Description 命令描述
	Description() string

	// RequiredPermission 所需权限
	RequiredPermission() user.Permission

	// Handle 处理命令
	Handle(ctx *Context) error

	// IsEnabled 是否在该群组启用（可选，默认启用）
	IsEnabled(groupID int64) bool
}

// Response 命令响应
type Response struct {
	Text      string
	ParseMode string // "Markdown", "HTML", ""
	ReplyTo   int    // 回复的消息 ID
}

// Registry 命令注册表
type Registry interface {
	Register(handler Handler) error
	Get(name string) (Handler, bool)
	GetAll() []Handler
	Unregister(name string)
}
