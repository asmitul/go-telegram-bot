package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/user"
)

// MockHandler 模拟命令处理器
type MockHandler struct {
	name     string
	handleFn func(ctx *command.Context) error
}

func (m *MockHandler) Name() string {
	return m.name
}

func (m *MockHandler) Description() string {
	return "mock handler"
}

func (m *MockHandler) RequiredPermission() user.Permission {
	return user.PermissionUser
}

func (m *MockHandler) Handle(ctx *command.Context) error {
	if m.handleFn != nil {
		return m.handleFn(ctx)
	}
	return nil
}

func (m *MockHandler) IsEnabled(groupID int64) bool {
	return true
}

var globalMetrics = NewMetrics()

func TestMiddleware_RecordSuccess(t *testing.T) {
	mw := NewMiddleware(globalMetrics)

	handler := &MockHandler{
		name: "test",
		handleFn: func(ctx *command.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	ctx := &command.Context{
		Ctx:     context.Background(),
		UserID:  123,
		GroupID: -1,
	}

	// 执行处理器
	handlerFunc := mw.Record(handler)
	err := handlerFunc(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 注意：由于使用了 promauto，metrics 会注册到全局 registry
	// 这里我们只验证函数是否正常执行，不验证具体的指标值
}

func TestMiddleware_RecordFailure(t *testing.T) {
	mw := NewMiddleware(globalMetrics)

	testError := errors.New("test error")
	handler := &MockHandler{
		name: "test",
		handleFn: func(ctx *command.Context) error {
			return testError
		},
	}

	ctx := &command.Context{
		Ctx:     context.Background(),
		UserID:  123,
		GroupID: -1,
	}

	// 执行处理器
	handlerFunc := mw.Record(handler)
	err := handlerFunc(ctx)

	if err == nil {
		t.Error("expected error, got nil")
	}

	if !errors.Is(err, testError) {
		t.Errorf("expected test error, got %v", err)
	}
}

func TestMiddleware_RecordRateLimitError(t *testing.T) {
	mw := NewMiddleware(globalMetrics)

	handler := &MockHandler{
		name: "test",
		handleFn: func(ctx *command.Context) error {
			return command.ErrRateLimitExceeded
		},
	}

	ctx := &command.Context{
		Ctx:     context.Background(),
		UserID:  123,
		GroupID: -1,
	}

	// 执行处理器
	handlerFunc := mw.Record(handler)
	err := handlerFunc(ctx)

	if err == nil {
		t.Error("expected error, got nil")
	}

	if !errors.Is(err, command.ErrRateLimitExceeded) {
		t.Errorf("expected rate limit error, got %v", err)
	}
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "no error",
			err:      nil,
			expected: "none",
		},
		{
			name:     "rate limit error",
			err:      command.ErrRateLimitExceeded,
			expected: "rate_limit",
		},
		{
			name:     "other error",
			err:      errors.New("test error"),
			expected: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getErrorType(tt.err)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMiddleware_RecordDuration(t *testing.T) {
	mw := NewMiddleware(globalMetrics)

	sleepDuration := 50 * time.Millisecond
	handler := &MockHandler{
		name: "test",
		handleFn: func(ctx *command.Context) error {
			time.Sleep(sleepDuration)
			return nil
		},
	}

	ctx := &command.Context{
		Ctx:     context.Background(),
		UserID:  123,
		GroupID: -1,
	}

	// 记录开始时间
	start := time.Now()

	// 执行处理器
	handlerFunc := mw.Record(handler)
	err := handlerFunc(ctx)

	// 计算实际执行时间
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 验证执行时间至少是我们设置的 sleep 时间
	if elapsed < sleepDuration {
		t.Errorf("expected duration >= %v, got %v", sleepDuration, elapsed)
	}
}

func TestMiddleware_MultipleCommands(t *testing.T) {
	mw := NewMiddleware(globalMetrics)

	// 执行多个不同的命令
	commands := []string{"ping", "help", "stats"}

	for _, cmdName := range commands {
		handler := &MockHandler{
			name: cmdName,
			handleFn: func(ctx *command.Context) error {
				return nil
			},
		}

		ctx := &command.Context{
			Ctx:     context.Background(),
			UserID:  123,
			GroupID: -1,
		}

		handlerFunc := mw.Record(handler)
		err := handlerFunc(ctx)

		if err != nil {
			t.Errorf("expected no error for command %s, got %v", cmdName, err)
		}
	}
}

func TestMiddleware_MultipleGroups(t *testing.T) {
	mw := NewMiddleware(globalMetrics)

	handler := &MockHandler{
		name: "test",
		handleFn: func(ctx *command.Context) error {
			return nil
		},
	}

	// 在不同群组执行相同命令
	groups := []int64{-1, -2, -3}

	for _, groupID := range groups {
		ctx := &command.Context{
			Ctx:     context.Background(),
			UserID:  123,
			GroupID: groupID,
		}

		handlerFunc := mw.Record(handler)
		err := handlerFunc(ctx)

		if err != nil {
			t.Errorf("expected no error for group %d, got %v", groupID, err)
		}
	}
}
