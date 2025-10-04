package command

import (
	"context"
	"strings"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

// GroupRepository 群组仓储接口（简化版）
type GroupRepository interface {
	FindByID(ctx context.Context, id int64) (*group.Group, error)
}

// UserRepository 用户仓储接口（简化版）
type UserRepository interface {
	FindByID(ctx context.Context, id int64) (*user.User, error)
	FindByUsername(ctx context.Context, username string) (*user.User, error)
	Save(ctx context.Context, user *user.User) error
	Update(ctx context.Context, user *user.User) error
	UpdatePermission(ctx context.Context, userID int64, groupID int64, perm user.Permission) error
	FindAdminsByGroup(ctx context.Context, groupID int64) ([]*user.User, error)
}

// BaseCommand 命令处理器基类
// 提供命令匹配和权限检查的通用逻辑
type BaseCommand struct {
	name        string
	description string
	permission  user.Permission
	chatTypes   []string // 支持的聊天类型：private, group, supergroup, channel
	groupRepo   GroupRepository
}

// NewBaseCommand 创建命令基类
func NewBaseCommand(
	name string,
	description string,
	permission user.Permission,
	chatTypes []string,
	groupRepo GroupRepository,
) *BaseCommand {
	if len(chatTypes) == 0 {
		// 默认支持所有类型
		chatTypes = []string{"private", "group", "supergroup", "channel"}
	}

	return &BaseCommand{
		name:        name,
		description: description,
		permission:  permission,
		chatTypes:   chatTypes,
		groupRepo:   groupRepo,
	}
}

// Match 判断是否匹配此命令
func (c *BaseCommand) Match(ctx *handler.Context) bool {
	// 1. 必须是文本消息
	if ctx.Text == "" {
		return false
	}

	// 2. 必须以 / 开头
	if ctx.Text[0] != '/' {
		return false
	}

	// 3. 解析命令名
	cmdName := parseCommandName(ctx.Text)
	if cmdName != c.name {
		return false
	}

	// 4. 检查聊天类型
	if !c.isSupportedChatType(ctx.ChatType) {
		return false
	}

	// 5. 检查群组是否启用（如果是群组且有 groupRepo）
	if ctx.IsGroup() && c.groupRepo != nil {
		reqCtx := context.TODO() // TODO: 从 handler.Context 传递
		g, err := c.groupRepo.FindByID(reqCtx, ctx.ChatID)
		if err != nil {
			// 区分群组不存在和数据库错误
			if err == group.ErrGroupNotFound {
				// 群组不存在，中间件会创建，此时应该允许命令执行（默认启用）
				return true
			}
			// 数据库错误，为了安全起见，禁止命令执行
			// 注意：这里无法访问 logger，错误会在中间件层记录
			return false
		}

		// 群组存在，检查命令是否启用
		if !g.IsCommandEnabled(c.name) {
			return false
		}
	}

	return true
}

// Priority 命令优先级
func (c *BaseCommand) Priority() int {
	return 100 // 命令优先级为 100
}

// ContinueChain 命令处理后停止链
func (c *BaseCommand) ContinueChain() bool {
	return false
}

// GetName 获取命令名
func (c *BaseCommand) GetName() string {
	return c.name
}

// GetDescription 获取命令描述
func (c *BaseCommand) GetDescription() string {
	return c.description
}

// GetPermission 获取所需权限
func (c *BaseCommand) GetPermission() user.Permission {
	return c.permission
}

// CheckPermission 检查权限
func (c *BaseCommand) CheckPermission(ctx *handler.Context) error {
	return ctx.RequirePermission(c.permission)
}

// isSupportedChatType 检查是否支持该聊天类型
func (c *BaseCommand) isSupportedChatType(chatType string) bool {
	for _, t := range c.chatTypes {
		if t == chatType {
			return true
		}
	}
	return false
}

// parseCommandName 解析命令名
// "/ping" -> "ping"
// "/ping@botname" -> "ping"
// "/ping arg1 arg2" -> "ping"
func parseCommandName(text string) string {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return ""
	}

	cmd := strings.TrimPrefix(parts[0], "/")

	// 移除 @botname
	if idx := strings.Index(cmd, "@"); idx != -1 {
		cmd = cmd[:idx]
	}

	return cmd
}

// ParseArgs 解析命令参数
// "/command arg1 arg2 arg3" -> ["arg1", "arg2", "arg3"]
func ParseArgs(text string) []string {
	parts := strings.Fields(text)
	if len(parts) <= 1 {
		return []string{}
	}
	return parts[1:]
}
