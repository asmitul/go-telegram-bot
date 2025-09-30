package telegram

import (
	"errors"
	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/user"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"telegram-bot/test/mocks"
)

// Test PermissionMiddleware
func TestPermissionMiddleware_Check(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockGroupRepo := mocks.NewMockGroupRepository(ctrl)
	mockHandler := mocks.NewMockHandler(ctrl)

	middleware := NewPermissionMiddleware(mockUserRepo, mockGroupRepo)

	t.Run("command disabled in group", func(t *testing.T) {
		mockHandler.EXPECT().IsEnabled(int64(-100)).Return(false)

		mw := middleware.Check(mockHandler)
		handler := mw(func(ctx *command.Context) error {
			return nil
		})

		ctx := &command.Context{
			UserID:  123,
			GroupID: -100,
		}

		err := handler(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "未启用")
	})

	t.Run("user exists with sufficient permission", func(t *testing.T) {
		u := user.NewUser(123, "test", "Test", "User")
		u.SetPermission(-100, user.PermissionAdmin)

		mockHandler.EXPECT().IsEnabled(int64(-100)).Return(true)
		mockHandler.EXPECT().RequiredPermission().Return(user.PermissionUser)
		mockUserRepo.EXPECT().FindByID(int64(123)).Return(u, nil)

		executed := false
		mw := middleware.Check(mockHandler)
		handler := mw(func(ctx *command.Context) error {
			executed = true
			assert.Equal(t, u, ctx.User)
			return nil
		})

		ctx := &command.Context{
			UserID:  123,
			GroupID: -100,
		}

		err := handler(ctx)
		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("user exists with insufficient permission", func(t *testing.T) {
		u := user.NewUser(123, "test", "Test", "User")
		u.SetPermission(-100, user.PermissionUser)

		mockHandler.EXPECT().IsEnabled(int64(-100)).Return(true)
		mockHandler.EXPECT().RequiredPermission().Return(user.PermissionAdmin)
		mockUserRepo.EXPECT().FindByID(int64(123)).Return(u, nil)

		mw := middleware.Check(mockHandler)
		handler := mw(func(ctx *command.Context) error {
			return nil
		})

		ctx := &command.Context{
			UserID:  123,
			GroupID: -100,
		}

		err := handler(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "权限不足")
	})

	t.Run("user not found - create new user", func(t *testing.T) {
		mockHandler.EXPECT().IsEnabled(int64(-100)).Return(true)
		mockHandler.EXPECT().RequiredPermission().Return(user.PermissionUser)
		mockUserRepo.EXPECT().FindByID(int64(123)).Return(nil, user.ErrUserNotFound)
		mockUserRepo.EXPECT().Save(gomock.Any()).Return(nil)

		executed := false
		mw := middleware.Check(mockHandler)
		handler := mw(func(ctx *command.Context) error {
			executed = true
			assert.NotNil(t, ctx.User)
			return nil
		})

		ctx := &command.Context{
			UserID:  123,
			GroupID: -100,
		}

		err := handler(ctx)
		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("user not found - save fails", func(t *testing.T) {
		mockHandler.EXPECT().IsEnabled(int64(-100)).Return(true)
		mockUserRepo.EXPECT().FindByID(int64(123)).Return(nil, user.ErrUserNotFound)
		mockUserRepo.EXPECT().Save(gomock.Any()).Return(errors.New("database error"))

		mw := middleware.Check(mockHandler)
		handler := mw(func(ctx *command.Context) error {
			return nil
		})

		ctx := &command.Context{
			UserID:  123,
			GroupID: -100,
		}

		err := handler(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "创建用户失败")
	})

	t.Run("handler execution error propagates", func(t *testing.T) {
		u := user.NewUser(123, "test", "Test", "User")
		u.SetPermission(-100, user.PermissionAdmin)

		mockHandler.EXPECT().IsEnabled(int64(-100)).Return(true)
		mockHandler.EXPECT().RequiredPermission().Return(user.PermissionUser)
		mockUserRepo.EXPECT().FindByID(int64(123)).Return(u, nil)

		expectedErr := errors.New("handler error")
		mw := middleware.Check(mockHandler)
		handler := mw(func(ctx *command.Context) error {
			return expectedErr
		})

		ctx := &command.Context{
			UserID:  123,
			GroupID: -100,
		}

		err := handler(ctx)
		assert.Equal(t, expectedErr, err)
	})
}

// Test LoggingMiddleware
func TestLoggingMiddleware_Log(t *testing.T) {
	t.Run("logs successful command execution", func(t *testing.T) {
		logger := &MockLogger{}
		middleware := NewLoggingMiddleware(logger)

		mw := middleware.Log()
		handler := mw(func(ctx *command.Context) error {
			return nil
		})

		ctx := &command.Context{
			Text:    "/test",
			UserID:  123,
			GroupID: -100,
		}

		err := handler(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 2, logger.InfoCallCount)   // command_received + command_success
		assert.Equal(t, 0, logger.ErrorCallCount)
	})

	t.Run("logs failed command execution", func(t *testing.T) {
		logger := &MockLogger{}
		middleware := NewLoggingMiddleware(logger)

		expectedErr := errors.New("test error")
		mw := middleware.Log()
		handler := mw(func(ctx *command.Context) error {
			return expectedErr
		})

		ctx := &command.Context{
			Text:    "/test",
			UserID:  123,
			GroupID: -100,
		}

		err := handler(ctx)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, logger.InfoCallCount)   // command_received only
		assert.Equal(t, 1, logger.ErrorCallCount)  // command_failed
	})

	t.Run("logs context information", func(t *testing.T) {
		logger := &MockLogger{}
		middleware := NewLoggingMiddleware(logger)

		mw := middleware.Log()
		handler := mw(func(ctx *command.Context) error {
			return nil
		})

		ctx := &command.Context{
			Text:    "/mycommand",
			UserID:  456,
			GroupID: -200,
		}

		handler(ctx)

		// Verify that context information was logged
		assert.Contains(t, logger.LastInfoFields, "command")
		assert.Contains(t, logger.LastInfoFields, "user_id")
		assert.Contains(t, logger.LastInfoFields, "group_id")
	})
}

// Test RateLimitMiddleware
func TestRateLimitMiddleware_Limit(t *testing.T) {
	t.Run("allows request when under limit", func(t *testing.T) {
		limiter := &MockRateLimiter{AllowResult: true}
		middleware := NewRateLimitMiddleware(limiter)

		executed := false
		mw := middleware.Limit()
		handler := mw(func(ctx *command.Context) error {
			executed = true
			return nil
		})

		ctx := &command.Context{
			UserID: 123,
		}

		err := handler(ctx)
		assert.NoError(t, err)
		assert.True(t, executed)
		assert.Equal(t, int64(123), limiter.LastUserID)
	})

	t.Run("blocks request when over limit", func(t *testing.T) {
		limiter := &MockRateLimiter{AllowResult: false}
		middleware := NewRateLimitMiddleware(limiter)

		executed := false
		mw := middleware.Limit()
		handler := mw(func(ctx *command.Context) error {
			executed = true
			return nil
		})

		ctx := &command.Context{
			UserID: 123,
		}

		err := handler(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "过于频繁")
		assert.False(t, executed)
	})

	t.Run("checks correct user ID", func(t *testing.T) {
		limiter := &MockRateLimiter{AllowResult: true}
		middleware := NewRateLimitMiddleware(limiter)

		mw := middleware.Limit()
		handler := mw(func(ctx *command.Context) error {
			return nil
		})

		ctx := &command.Context{
			UserID: 789,
		}

		handler(ctx)
		assert.Equal(t, int64(789), limiter.LastUserID)
	})
}

// Test Chain
func TestChain(t *testing.T) {
	t.Run("chains middlewares in correct order", func(t *testing.T) {
		var order []string

		mw1 := func(next HandlerFunc) HandlerFunc {
			return func(ctx *command.Context) error {
				order = append(order, "mw1_before")
				err := next(ctx)
				order = append(order, "mw1_after")
				return err
			}
		}

		mw2 := func(next HandlerFunc) HandlerFunc {
			return func(ctx *command.Context) error {
				order = append(order, "mw2_before")
				err := next(ctx)
				order = append(order, "mw2_after")
				return err
			}
		}

		mw3 := func(next HandlerFunc) HandlerFunc {
			return func(ctx *command.Context) error {
				order = append(order, "mw3_before")
				err := next(ctx)
				order = append(order, "mw3_after")
				return err
			}
		}

		chain := Chain(mw1, mw2, mw3)
		handler := chain(func(ctx *command.Context) error {
			order = append(order, "handler")
			return nil
		})

		ctx := &command.Context{}
		err := handler(ctx)

		assert.NoError(t, err)
		assert.Equal(t, []string{
			"mw1_before",
			"mw2_before",
			"mw3_before",
			"handler",
			"mw3_after",
			"mw2_after",
			"mw1_after",
		}, order)
	})

	t.Run("empty chain", func(t *testing.T) {
		executed := false
		chain := Chain()
		handler := chain(func(ctx *command.Context) error {
			executed = true
			return nil
		})

		ctx := &command.Context{}
		err := handler(ctx)

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("single middleware", func(t *testing.T) {
		called := false
		mw := func(next HandlerFunc) HandlerFunc {
			return func(ctx *command.Context) error {
				called = true
				return next(ctx)
			}
		}

		chain := Chain(mw)
		handler := chain(func(ctx *command.Context) error {
			return nil
		})

		ctx := &command.Context{}
		handler(ctx)

		assert.True(t, called)
	})

	t.Run("error propagates through chain", func(t *testing.T) {
		expectedErr := errors.New("handler error")

		mw1 := func(next HandlerFunc) HandlerFunc {
			return func(ctx *command.Context) error {
				return next(ctx)
			}
		}

		mw2 := func(next HandlerFunc) HandlerFunc {
			return func(ctx *command.Context) error {
				return next(ctx)
			}
		}

		chain := Chain(mw1, mw2)
		handler := chain(func(ctx *command.Context) error {
			return expectedErr
		})

		ctx := &command.Context{}
		err := handler(ctx)

		assert.Equal(t, expectedErr, err)
	})

	t.Run("middleware can modify context", func(t *testing.T) {
		mw := func(next HandlerFunc) HandlerFunc {
			return func(ctx *command.Context) error {
				ctx.UserID = 999
				return next(ctx)
			}
		}

		chain := Chain(mw)
		handler := chain(func(ctx *command.Context) error {
			assert.Equal(t, int64(999), ctx.UserID)
			return nil
		})

		ctx := &command.Context{UserID: 123}
		err := handler(ctx)
		assert.NoError(t, err)
	})

	t.Run("middleware can short-circuit", func(t *testing.T) {
		handlerCalled := false

		mw := func(next HandlerFunc) HandlerFunc {
			return func(ctx *command.Context) error {
				// Short-circuit: don't call next
				return errors.New("blocked")
			}
		}

		chain := Chain(mw)
		handler := chain(func(ctx *command.Context) error {
			handlerCalled = true
			return nil
		})

		ctx := &command.Context{}
		err := handler(ctx)

		assert.Error(t, err)
		assert.False(t, handlerCalled)
	})
}

// Test integration of all middlewares
func TestMiddlewareIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockGroupRepo := mocks.NewMockGroupRepository(ctrl)
	mockHandler := mocks.NewMockHandler(ctrl)

	t.Run("full middleware stack", func(t *testing.T) {
		// Setup
		u := user.NewUser(123, "test", "Test", "User")
		u.SetPermission(-100, user.PermissionAdmin)

		mockHandler.EXPECT().IsEnabled(int64(-100)).Return(true)
		mockHandler.EXPECT().RequiredPermission().Return(user.PermissionUser)
		mockUserRepo.EXPECT().FindByID(int64(123)).Return(u, nil)

		// Create middlewares
		permMw := NewPermissionMiddleware(mockUserRepo, mockGroupRepo)
		logMw := NewLoggingMiddleware(&MockLogger{})
		rateMw := NewRateLimitMiddleware(&MockRateLimiter{AllowResult: true})

		// Chain them
		chain := Chain(
			logMw.Log(),
			rateMw.Limit(),
			permMw.Check(mockHandler),
		)

		executed := false
		handler := chain(func(ctx *command.Context) error {
			executed = true
			assert.NotNil(t, ctx.User)
			return nil
		})

		ctx := &command.Context{
			Text:    "/test",
			UserID:  123,
			GroupID: -100,
		}

		err := handler(ctx)
		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("rate limit blocks before permission check", func(t *testing.T) {
		// Rate limiter should block before we even check permissions
		rateMw := NewRateLimitMiddleware(&MockRateLimiter{AllowResult: false})
		permMw := NewPermissionMiddleware(mockUserRepo, mockGroupRepo)

		chain := Chain(
			rateMw.Limit(),
			permMw.Check(mockHandler),
		)

		handler := chain(func(ctx *command.Context) error {
			return nil
		})

		ctx := &command.Context{
			UserID:  123,
			GroupID: -100,
		}

		err := handler(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "过于频繁")
		// No calls to mockUserRepo should happen
	})
}

// Mock implementations
type MockLogger struct {
	InfoCallCount   int
	ErrorCallCount  int
	LastInfoFields  []interface{}
	LastErrorFields []interface{}
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.InfoCallCount++
	m.LastInfoFields = fields
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.ErrorCallCount++
	m.LastErrorFields = fields
}

type MockRateLimiter struct {
	AllowResult bool
	LastUserID  int64
}

func (m *MockRateLimiter) Allow(userID int64) bool {
	m.LastUserID = userID
	return m.AllowResult
}
