package command

import (
	"context"
	"fmt"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

const (
	FeatureCalculator = "calculator" // 计算器功能名称（与 pattern/calculator.go 保持一致）
)

// ToggleCalcHandler 切换计算器功能命令处理器
type ToggleCalcHandler struct {
	*BaseCommand
	groupRepo GroupRepository
}

// NewToggleCalcHandler 创建切换计算器功能命令处理器
func NewToggleCalcHandler(groupRepo GroupRepository, userRepo UserRepository) *ToggleCalcHandler {
	return &ToggleCalcHandler{
		BaseCommand: NewBaseCommand(
			"togglecalc",
			"开启/关闭群组计算器功能",
			user.PermissionAdmin, // 需要 Admin 及以上权限
			[]string{"group", "supergroup"},
			groupRepo,
		),
		groupRepo: groupRepo,
	}
}

// Handle 处理命令
func (h *ToggleCalcHandler) Handle(ctx *handler.Context) error {
	reqCtx := context.TODO()

	// 1. 检查权限
	if err := h.CheckPermission(ctx); err != nil {
		return err
	}

	// 2. 获取群组
	group, err := h.groupRepo.FindByID(reqCtx, ctx.ChatID)
	if err != nil {
		return ctx.Reply("❌ 获取群组信息失败，请稍后重试")
	}

	// 3. 检查当前状态
	currentStatus := group.IsFeatureEnabled(FeatureCalculator)

	// 4. 切换状态
	var statusText string
	if currentStatus {
		// 当前启用，切换为禁用
		group.DisableFeature(FeatureCalculator)
		statusText = "已关闭"
	} else {
		// 当前禁用，切换为启用
		group.EnableFeature(FeatureCalculator)
		statusText = "已开启"
	}

	// 5. 保存到数据库
	if err := h.groupRepo.Update(reqCtx, group); err != nil {
		return ctx.Reply("❌ 保存设置失败，请稍后重试")
	}

	// 6. 返回结果
	return ctx.ReplyHTML(fmt.Sprintf("✅ 计算器功能%s\n\n"+
		"<i>当前状态：%s</i>\n"+
		"<i>提示：群组成员发送数学表达式（如 1+2）时，机器人将%s自动计算并回复结果。</i>",
		statusText,
		getStatusEmoji(!currentStatus),
		getActionText(!currentStatus)))
}

// getStatusEmoji 获取状态表情
func getStatusEmoji(enabled bool) string {
	if enabled {
		return "✅ 已开启"
	}
	return "🚫 已关闭"
}

// getActionText 获取动作描述
func getActionText(enabled bool) string {
	if enabled {
		return ""
	}
	return "不再"
}
