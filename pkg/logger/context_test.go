package logger

import (
	"context"
	"testing"
)

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "test-trace-123"

	ctx = WithTraceID(ctx, traceID)
	result := GetTraceID(ctx)

	if result != traceID {
		t.Errorf("expected traceID %s, got %s", traceID, result)
	}
}

func TestGetTraceID_Empty(t *testing.T) {
	ctx := context.Background()
	result := GetTraceID(ctx)

	if result != "" {
		t.Errorf("expected empty traceID, got %s", result)
	}
}

func TestWithUserID(t *testing.T) {
	ctx := context.Background()
	userID := int64(12345)

	ctx = WithUserID(ctx, userID)
	result := GetUserID(ctx)

	if result != userID {
		t.Errorf("expected userID %d, got %d", userID, result)
	}
}

func TestGetUserID_Zero(t *testing.T) {
	ctx := context.Background()
	result := GetUserID(ctx)

	if result != 0 {
		t.Errorf("expected userID 0, got %d", result)
	}
}

func TestWithGroupID(t *testing.T) {
	ctx := context.Background()
	groupID := int64(-12345)

	ctx = WithGroupID(ctx, groupID)
	result := GetGroupID(ctx)

	if result != groupID {
		t.Errorf("expected groupID %d, got %d", groupID, result)
	}
}

func TestGetGroupID_Zero(t *testing.T) {
	ctx := context.Background()
	result := GetGroupID(ctx)

	if result != 0 {
		t.Errorf("expected groupID 0, got %d", result)
	}
}

func TestGenerateTraceID(t *testing.T) {
	traceID1 := GenerateTraceID()
	traceID2 := GenerateTraceID()

	if len(traceID1) != 16 {
		t.Errorf("expected traceID length 16, got %d", len(traceID1))
	}

	if traceID1 == traceID2 {
		t.Error("expected different trace IDs")
	}
}

func TestWithContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")
	ctx = WithUserID(ctx, 456)
	ctx = WithGroupID(ctx, -789)

	logger := Default().(*StandardLogger)
	contextLogger := logger.WithContext(ctx).(*StandardLogger)

	if contextLogger.fields["trace_id"] != "trace-123" {
		t.Error("expected trace_id in fields")
	}
	if contextLogger.fields["user_id"] != int64(456) {
		t.Error("expected user_id in fields")
	}
	if contextLogger.fields["group_id"] != int64(-789) {
		t.Error("expected group_id in fields")
	}
}

func TestWithContext_EmptyContext(t *testing.T) {
	ctx := context.Background()

	logger := Default().(*StandardLogger)
	contextLogger := logger.WithContext(ctx).(*StandardLogger)

	// 应该返回相同的logger（没有新字段）
	if len(contextLogger.fields) != 0 {
		t.Errorf("expected no fields, got %d", len(contextLogger.fields))
	}
}
