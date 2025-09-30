package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRetryConfig tests retry configuration
func TestRetryConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		logger := &MockLogger{}
		config := DefaultRetryConfig(logger)

		assert.NotNil(t, config)
		assert.Equal(t, 3, config.MaxRetries)
		assert.Equal(t, 100*time.Millisecond, config.InitialDelay)
		assert.Equal(t, 10*time.Second, config.MaxDelay)
		assert.Equal(t, 2.0, config.Multiplier)
		assert.Equal(t, logger, config.Logger)
	})

	t.Run("custom config", func(t *testing.T) {
		logger := &MockLogger{}
		config := &RetryConfig{
			MaxRetries:   5,
			InitialDelay: 200 * time.Millisecond,
			MaxDelay:     30 * time.Second,
			Multiplier:   3.0,
			Logger:       logger,
		}

		assert.Equal(t, 5, config.MaxRetries)
		assert.Equal(t, 200*time.Millisecond, config.InitialDelay)
		assert.Equal(t, 30*time.Second, config.MaxDelay)
		assert.Equal(t, 3.0, config.Multiplier)
	})
}

// TestExponentialBackoffRetrier tests the exponential backoff retrier
func TestExponentialBackoffRetrier(t *testing.T) {
	t.Run("successful on first attempt", func(t *testing.T) {
		logger := &MockLogger{}
		config := &RetryConfig{
			MaxRetries:   3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Logger:       logger,
		}
		retrier := NewRetrier(config)

		callCount := 0
		err := retrier.Do(context.Background(), func() error {
			callCount++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, callCount)
		assert.Equal(t, 0, logger.WarnCallCount)
		assert.Equal(t, 0, logger.ErrorCallCount)
	})

	t.Run("successful after retries", func(t *testing.T) {
		logger := &MockLogger{}
		config := &RetryConfig{
			MaxRetries:   3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Logger:       logger,
		}
		retrier := NewRetrier(config)

		callCount := 0
		err := retrier.Do(context.Background(), func() error {
			callCount++
			if callCount < 3 {
				return errors.New("temporary error")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, callCount)
		assert.Equal(t, 2, logger.WarnCallCount) // 2 failed attempts
		assert.Equal(t, 0, logger.ErrorCallCount)
	})

	t.Run("all attempts fail", func(t *testing.T) {
		logger := &MockLogger{}
		config := &RetryConfig{
			MaxRetries:   2,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Logger:       logger,
		}
		retrier := NewRetrier(config)

		expectedErr := errors.New("persistent error")
		callCount := 0
		err := retrier.Do(context.Background(), func() error {
			callCount++
			return expectedErr
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed after 3 attempts")
		assert.Equal(t, 3, callCount) // MaxRetries + 1
		assert.Equal(t, 2, logger.WarnCallCount)  // 2 retry warnings
		assert.Equal(t, 1, logger.ErrorCallCount) // 1 final error
	})

	t.Run("context cancelled during execution", func(t *testing.T) {
		logger := &MockLogger{}
		config := &RetryConfig{
			MaxRetries:   3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Logger:       logger,
		}
		retrier := NewRetrier(config)

		ctx, cancel := context.WithCancel(context.Background())

		callCount := 0
		err := retrier.Do(ctx, func() error {
			callCount++
			if callCount == 2 {
				cancel()
			}
			return errors.New("error")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retry cancelled")
		assert.Equal(t, 2, callCount)
	})

	t.Run("context cancelled during backoff", func(t *testing.T) {
		logger := &MockLogger{}
		config := &RetryConfig{
			MaxRetries:   5,
			InitialDelay: 500 * time.Millisecond, // Long delay
			MaxDelay:     10 * time.Second,
			Multiplier:   2.0,
			Logger:       logger,
		}
		retrier := NewRetrier(config)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		callCount := 0
		err := retrier.Do(ctx, func() error {
			callCount++
			return errors.New("error")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
		// Should fail quickly due to context timeout
		assert.True(t, callCount < 3)
	})

	t.Run("with description", func(t *testing.T) {
		logger := &MockLogger{}
		config := &RetryConfig{
			MaxRetries:   2,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Logger:       logger,
		}
		retrier := NewRetrier(config)

		callCount := 0
		err := retrier.DoWithDescription(context.Background(), "test_operation", func() error {
			callCount++
			if callCount < 2 {
				return errors.New("temporary error")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, callCount)
		assert.Equal(t, 1, logger.WarnCallCount)
		// Check that description is logged (in real logger)
	})
}

// TestCalculateDelay tests exponential backoff delay calculation
func TestCalculateDelay(t *testing.T) {
	logger := &MockLogger{}
	config := &RetryConfig{
		MaxRetries:   5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Logger:       logger,
	}
	retrier := NewRetrier(config)

	tests := []struct {
		attempt      int
		expectedMin  time.Duration
		expectedMax  time.Duration
	}{
		{0, 100 * time.Millisecond, 100 * time.Millisecond},   // 100 * 2^0
		{1, 200 * time.Millisecond, 200 * time.Millisecond},   // 100 * 2^1
		{2, 400 * time.Millisecond, 400 * time.Millisecond},   // 100 * 2^2
		{3, 800 * time.Millisecond, 800 * time.Millisecond},   // 100 * 2^3
		{4, 1600 * time.Millisecond, 1600 * time.Millisecond}, // 100 * 2^4
		{5, 3200 * time.Millisecond, 3200 * time.Millisecond}, // 100 * 2^5
		{10, 5 * time.Second, 5 * time.Second},                // Capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run(string(rune('0'+tt.attempt)), func(t *testing.T) {
			delay := retrier.calculateDelay(tt.attempt)
			assert.GreaterOrEqual(t, delay, tt.expectedMin)
			assert.LessOrEqual(t, delay, tt.expectedMax)
		})
	}
}

// TestIsRetryableError tests retryable error detection
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		retryable  bool
	}{
		{"nil error", nil, false},
		{"connection refused", errors.New("connection refused"), true},
		{"connection reset", errors.New("connection reset by peer"), true},
		{"timeout", errors.New("request timeout"), true},
		{"temporary failure", errors.New("temporary failure in name resolution"), true},
		{"too many requests", errors.New("429 Too Many Requests"), true},
		{"rate limit", errors.New("rate limit exceeded"), true},
		{"503 error", errors.New("503 Service Unavailable"), true},
		{"502 error", errors.New("502 Bad Gateway"), true},
		{"504 error", errors.New("504 Gateway Timeout"), true},
		{"permission denied", errors.New("permission denied"), false},
		{"invalid parameter", errors.New("invalid parameter"), false},
		{"not found", errors.New("404 not found"), false},
		{"bad request", errors.New("400 bad request"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

// TestStringHelpers tests helper functions
func TestStringHelpers(t *testing.T) {
	t.Run("toLower", func(t *testing.T) {
		assert.Equal(t, "hello", toLower("HELLO"))
		assert.Equal(t, "hello world", toLower("Hello World"))
		assert.Equal(t, "123abc", toLower("123ABC"))
	})

	t.Run("indexOf", func(t *testing.T) {
		assert.Equal(t, 0, indexOf("hello", "hello"))
		assert.Equal(t, 6, indexOf("hello world", "world"))
		assert.Equal(t, -1, indexOf("hello", "xyz"))
		assert.Equal(t, 2, indexOf("aabaa", "ba"))
	})

	t.Run("contains", func(t *testing.T) {
		assert.True(t, contains("hello world", "world"))
		assert.True(t, contains("HELLO WORLD", "world"))
		assert.True(t, contains("hello world", "WORLD"))
		assert.False(t, contains("hello", "xyz"))
	})
}

// TestRetryableAPI tests the retryable API wrapper
func TestRetryableAPI(t *testing.T) {
	t.Run("BanChatMember with retry", func(t *testing.T) {
		logger := &MockLogger{}
		config := &RetryConfig{
			MaxRetries:   2,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Logger:       logger,
		}
		retrier := NewRetrier(config)

		// Test retrier with a function that fails once then succeeds
		callCount := 0
		err := retrier.DoWithDescription(context.Background(), "ban_test", func() error {
			callCount++
			if callCount == 1 {
				return errors.New("temporary error")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, callCount) // Failed once, succeeded on retry
		assert.Equal(t, 1, logger.WarnCallCount) // One warning for the retry
	})

	t.Run("SendMessage with all retries failed", func(t *testing.T) {
		logger := &MockLogger{}
		config := &RetryConfig{
			MaxRetries:   2,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
			Logger:       logger,
		}
		retrier := NewRetrier(config)

		// Test retrier with a function that always fails
		callCount := 0
		err := retrier.DoWithDescription(context.Background(), "send_test", func() error {
			callCount++
			return errors.New("persistent error")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed after")
		assert.Equal(t, 3, callCount) // 3 attempts total
		assert.Equal(t, 2, logger.WarnCallCount) // Two warnings for the retries
		assert.Equal(t, 1, logger.ErrorCallCount) // One final error
	})
}

// TestNewRetrier_Panic tests that nil config panics
func TestNewRetrier_Panic(t *testing.T) {
	assert.Panics(t, func() {
		NewRetrier(nil)
	})
}

// Benchmark tests
func BenchmarkRetrier_Success(b *testing.B) {
	logger := &MockLogger{}
	config := DefaultRetryConfig(logger)
	retrier := NewRetrier(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		retrier.Do(context.Background(), func() error {
			return nil
		})
	}
}

func BenchmarkRetrier_WithRetries(b *testing.B) {
	logger := &MockLogger{}
	config := &RetryConfig{
		MaxRetries:   2,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		Logger:       logger,
	}
	retrier := NewRetrier(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempt := 0
		retrier.Do(context.Background(), func() error {
			attempt++
			if attempt < 2 {
				return errors.New("error")
			}
			return nil
		})
	}
}

