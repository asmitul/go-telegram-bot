package metrics

import (
	"errors"
	"telegram-bot/internal/domain/command"
	"time"
)

// Middleware 指标收集中间件
type Middleware struct {
	metrics *Metrics
}

// NewMiddleware 创建指标收集中间件
func NewMiddleware(metrics *Metrics) *Middleware {
	return &Middleware{
		metrics: metrics,
	}
}

// Record 记录指标的中间件函数
func (m *Middleware) Record(handler command.Handler) command.HandlerFunc {
	return func(ctx *command.Context) error {
		commandName := handler.Name()
		groupID := ctx.GroupID

		// 记录命令执行
		m.metrics.RecordCommand(commandName, groupID)

		// 记录开始时间
		start := time.Now()

		// 执行命令
		err := handler.Handle(ctx)

		// 记录执行时长
		duration := time.Since(start).Seconds()
		m.metrics.RecordCommandDuration(commandName, groupID, duration)

		// 记录成功或失败
		if err != nil {
			errorType := getErrorType(err)
			m.metrics.RecordCommandFailure(commandName, groupID, errorType)
		} else {
			m.metrics.RecordCommandSuccess(commandName, groupID)
		}

		return err
	}
}

// getErrorType 获取错误类型
func getErrorType(err error) string {
	if err == nil {
		return "none"
	}

	// 检查是否是限流错误
	if errors.Is(err, command.ErrRateLimitExceeded) {
		return "rate_limit"
	}

	// 其他错误
	return "other"
}
