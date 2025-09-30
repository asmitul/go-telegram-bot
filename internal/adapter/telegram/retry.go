package telegram

import (
	"context"
	"fmt"
	"math"
	"time"

	"telegram-bot/pkg/logger"
)

// RetryConfig 重试配置
type RetryConfig struct {
	// MaxRetries 最大重试次数
	MaxRetries int
	// InitialDelay 初始延迟时间
	InitialDelay time.Duration
	// MaxDelay 最大延迟时间
	MaxDelay time.Duration
	// Multiplier 延迟倍数
	Multiplier float64
	// Logger 日志记录器
	Logger logger.Logger
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig(log logger.Logger) *RetryConfig {
	return &RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		Logger:       log,
	}
}

// RetryableFunc 可重试的函数类型
type RetryableFunc func() error

// Retrier 重试器接口
type Retrier interface {
	// Do 执行带重试的函数
	Do(ctx context.Context, fn RetryableFunc) error
	// DoWithDescription 执行带重试的函数（带描述）
	DoWithDescription(ctx context.Context, description string, fn RetryableFunc) error
}

// ExponentialBackoffRetrier 指数退避重试器
type ExponentialBackoffRetrier struct {
	config *RetryConfig
}

// NewRetrier 创建重试器
func NewRetrier(config *RetryConfig) *ExponentialBackoffRetrier {
	if config == nil {
		panic("retry config cannot be nil")
	}
	return &ExponentialBackoffRetrier{
		config: config,
	}
}

// Do 执行带重试的函数
func (r *ExponentialBackoffRetrier) Do(ctx context.Context, fn RetryableFunc) error {
	return r.DoWithDescription(ctx, "operation", fn)
}

// DoWithDescription 执行带重试的函数（带描述）
func (r *ExponentialBackoffRetrier) DoWithDescription(ctx context.Context, description string, fn RetryableFunc) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			r.config.Logger.Warn("Retry cancelled by context",
				"description", description,
				"attempt", attempt,
				"error", ctx.Err(),
			)
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// 执行函数
		err := fn()
		if err == nil {
			// 成功
			if attempt > 0 {
				r.config.Logger.Info("Retry succeeded",
					"description", description,
					"attempt", attempt+1,
					"total_attempts", attempt+1,
				)
			}
			return nil
		}

		lastErr = err

		// 如果是最后一次尝试，不需要等待
		if attempt == r.config.MaxRetries {
			r.config.Logger.Error("All retry attempts failed",
				"description", description,
				"total_attempts", attempt+1,
				"error", err,
			)
			break
		}

		// 计算退避延迟
		delay := r.calculateDelay(attempt)

		r.config.Logger.Warn("Operation failed, retrying",
			"description", description,
			"attempt", attempt+1,
			"max_attempts", r.config.MaxRetries+1,
			"error", err,
			"next_retry_in", delay,
		)

		// 等待后重试
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled during backoff: %w", ctx.Err())
		case <-time.After(delay):
			// 继续下一次重试
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", r.config.MaxRetries+1, lastErr)
}

// calculateDelay 计算指数退避延迟
func (r *ExponentialBackoffRetrier) calculateDelay(attempt int) time.Duration {
	// 计算指数退避: initialDelay * multiplier^attempt
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(attempt))

	// 限制最大延迟
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	return time.Duration(delay)
}

// IsRetryableError 判断错误是否可以重试
// 这里可以根据具体的错误类型判断是否需要重试
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// TODO: 可以根据具体的 Telegram API 错误码判断
	// 例如：网络错误、超时错误、429 Too Many Requests 等应该重试
	// 而权限错误、参数错误等不应该重试

	errMsg := err.Error()

	// 常见的可重试错误
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"too many requests",
		"rate limit",
		"503",
		"502",
		"504",
	}

	for _, retryable := range retryableErrors {
		if contains(errMsg, retryable) {
			return true
		}
	}

	return false
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

// toLower 转换为小写
func toLower(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// indexOf 查找子串位置
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// RetryableAPI 支持重试的 Telegram API 包装器
type RetryableAPI struct {
	api     *API
	retrier Retrier
	logger  logger.Logger
}

// NewRetryableAPI 创建支持重试的 API
func NewRetryableAPI(api *API, retrier Retrier, log logger.Logger) *RetryableAPI {
	return &RetryableAPI{
		api:     api,
		retrier: retrier,
		logger:  log,
	}
}

// BanChatMember 封禁群组成员（支持重试）
func (r *RetryableAPI) BanChatMember(chatID, userID int64) error {
	return r.retrier.DoWithDescription(
		context.Background(),
		fmt.Sprintf("ban_user_%d_in_chat_%d", userID, chatID),
		func() error {
			return r.api.BanChatMember(chatID, userID)
		},
	)
}

// BanChatMemberWithDuration 临时封禁群组成员（支持重试）
func (r *RetryableAPI) BanChatMemberWithDuration(chatID, userID int64, until time.Time) error {
	return r.retrier.DoWithDescription(
		context.Background(),
		fmt.Sprintf("temp_ban_user_%d_in_chat_%d", userID, chatID),
		func() error {
			return r.api.BanChatMemberWithDuration(chatID, userID, until)
		},
	)
}

// SendMessage 发送消息（支持重试）
func (r *RetryableAPI) SendMessage(chatID int64, text string) error {
	return r.retrier.DoWithDescription(
		context.Background(),
		fmt.Sprintf("send_message_to_chat_%d", chatID),
		func() error {
			return r.api.SendMessage(chatID, text)
		},
	)
}

// SendMessageWithReply 发送回复消息（支持重试）
func (r *RetryableAPI) SendMessageWithReply(chatID int64, text string, replyToMessageID int) error {
	return r.retrier.DoWithDescription(
		context.Background(),
		fmt.Sprintf("send_reply_to_chat_%d_msg_%d", chatID, replyToMessageID),
		func() error {
			return r.api.SendMessageWithReply(chatID, text, replyToMessageID)
		},
	)
}

// DeleteMessage 删除消息（支持重试）
func (r *RetryableAPI) DeleteMessage(chatID int64, messageID int) error {
	return r.retrier.DoWithDescription(
		context.Background(),
		fmt.Sprintf("delete_message_%d_in_chat_%d", messageID, chatID),
		func() error {
			return r.api.DeleteMessage(chatID, messageID)
		},
	)
}

// UnbanChatMember 解封群组成员（支持重试）
func (r *RetryableAPI) UnbanChatMember(chatID, userID int64) error {
	return r.retrier.DoWithDescription(
		context.Background(),
		fmt.Sprintf("unban_user_%d_in_chat_%d", userID, chatID),
		func() error {
			return r.api.UnbanChatMember(chatID, userID)
		},
	)
}
