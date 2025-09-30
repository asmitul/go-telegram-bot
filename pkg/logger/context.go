package logger

import (
	"context"
	"math/rand"
	"time"
)

type contextKey string

const (
	traceIDKey contextKey = "trace_id"
	userIDKey  contextKey = "user_id"
	groupIDKey contextKey = "group_id"
)

// WithTraceID 在 context 中添加 trace ID
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID 从 context 中获取 trace ID
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// WithUserID 在 context 中添加 user ID
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID 从 context 中获取 user ID
func GetUserID(ctx context.Context) int64 {
	if userID, ok := ctx.Value(userIDKey).(int64); ok {
		return userID
	}
	return 0
}

// WithGroupID 在 context 中添加 group ID
func WithGroupID(ctx context.Context, groupID int64) context.Context {
	return context.WithValue(ctx, groupIDKey, groupID)
}

// GetGroupID 从 context 中获取 group ID
func GetGroupID(ctx context.Context) int64 {
	if groupID, ok := ctx.Value(groupIDKey).(int64); ok {
		return groupID
	}
	return 0
}

// GenerateTraceID 生成随机的 trace ID
func GenerateTraceID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 16

	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// WithContext 从 context 中提取字段并添加到 logger
func (l *StandardLogger) WithContext(ctx context.Context) Logger {
	fields := make(map[string]interface{})

	if traceID := GetTraceID(ctx); traceID != "" {
		fields["trace_id"] = traceID
	}

	if userID := GetUserID(ctx); userID != 0 {
		fields["user_id"] = userID
	}

	if groupID := GetGroupID(ctx); groupID != 0 {
		fields["group_id"] = groupID
	}

	if len(fields) > 0 {
		return l.WithFields(fields)
	}

	return l
}

// WithContext 从 context 中提取字段并添加到 logger
func (l *JSONLogger) WithContext(ctx context.Context) Logger {
	fields := make(map[string]interface{})

	if traceID := GetTraceID(ctx); traceID != "" {
		fields["trace_id"] = traceID
	}

	if userID := GetUserID(ctx); userID != 0 {
		fields["user_id"] = userID
	}

	if groupID := GetGroupID(ctx); groupID != 0 {
		fields["group_id"] = groupID
	}

	if len(fields) > 0 {
		return l.WithFields(fields)
	}

	return l
}
