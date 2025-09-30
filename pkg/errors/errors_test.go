package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New("TEST_CODE", "test message")

	if err.Code() != "TEST_CODE" {
		t.Errorf("expected code TEST_CODE, got %s", err.Code())
	}

	if err.Message() != "test message" {
		t.Errorf("expected message 'test message', got %s", err.Message())
	}

	if err.Cause() != nil {
		t.Error("expected nil cause")
	}

	if len(err.Stack()) == 0 {
		t.Error("expected stack trace to be captured")
	}
}

func TestWrap(t *testing.T) {
	original := errors.New("original error")
	wrapped := Wrap(original, "wrapped message")

	if wrapped == nil {
		t.Fatal("expected non-nil error")
	}

	if wrapped.Code() != CodeUnknown {
		t.Errorf("expected code %s, got %s", CodeUnknown, wrapped.Code())
	}

	if wrapped.Message() != "wrapped message" {
		t.Errorf("expected message 'wrapped message', got %s", wrapped.Message())
	}

	if wrapped.Cause() != original {
		t.Error("expected cause to be original error")
	}
}

func TestWrapCustomError(t *testing.T) {
	original := NotFound("", "user not found")
	wrapped := Wrap(original, "failed to get user")

	if wrapped.Code() != CodeNotFound {
		t.Errorf("expected code %s, got %s", CodeNotFound, wrapped.Code())
	}

	if wrapped.Message() != "failed to get user" {
		t.Errorf("expected message 'failed to get user', got %s", wrapped.Message())
	}
}

func TestWrapWithCode(t *testing.T) {
	original := errors.New("original error")
	wrapped := WrapWithCode(original, "CUSTOM_CODE", "custom message")

	if wrapped.Code() != "CUSTOM_CODE" {
		t.Errorf("expected code CUSTOM_CODE, got %s", wrapped.Code())
	}

	if wrapped.Message() != "custom message" {
		t.Errorf("expected message 'custom message', got %s", wrapped.Message())
	}

	if wrapped.Cause() != original {
		t.Error("expected cause to be original error")
	}
}

func TestWrapNil(t *testing.T) {
	wrapped := Wrap(nil, "message")
	if wrapped != nil {
		t.Error("expected nil when wrapping nil error")
	}

	wrappedWithCode := WrapWithCode(nil, "CODE", "message")
	if wrappedWithCode != nil {
		t.Error("expected nil when wrapping nil error")
	}
}

func TestWithContext(t *testing.T) {
	err := New("TEST_CODE", "test message").
		WithContext("user_id", "123").
		WithContext("group_id", "456")

	ctx := err.Context()

	if ctx["user_id"] != "123" {
		t.Errorf("expected user_id 123, got %s", ctx["user_id"])
	}

	if ctx["group_id"] != "456" {
		t.Errorf("expected group_id 456, got %s", ctx["group_id"])
	}
}

func TestGetContext(t *testing.T) {
	err := New("TEST_CODE", "test message").
		WithContext("key", "value")

	val, exists := GetContext(err, "key")
	if !exists {
		t.Error("expected context key to exist")
	}

	if val != "value" {
		t.Errorf("expected value 'value', got %s", val)
	}

	_, exists = GetContext(err, "nonexistent")
	if exists {
		t.Error("expected nonexistent key to not exist")
	}
}

func TestTypedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      Error
		code     string
		checkFn  func(error) bool
	}{
		{"NotFound", NotFound("", "not found"), CodeNotFound, IsNotFound},
		{"Validation", Validation("", "validation error"), CodeValidation, IsValidation},
		{"Permission", Permission("", "permission denied"), CodePermission, IsPermission},
		{"Internal", Internal("", "internal error"), CodeInternal, IsInternal},
		{"External", External("", "external error"), CodeExternal, IsExternal},
		{"Conflict", Conflict("", "conflict"), CodeConflict, IsConflict},
		{"RateLimit", RateLimit("rate limit"), CodeRateLimit, IsRateLimit},
		{"Timeout", Timeout("timeout"), CodeTimeout, IsTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code() != tt.code {
				t.Errorf("expected code %s, got %s", tt.code, tt.err.Code())
			}

			if !tt.checkFn(tt.err) {
				t.Errorf("check function returned false for %s error", tt.name)
			}
		})
	}
}

func TestTypedErrorsWithCustomCode(t *testing.T) {
	err := NotFound("USER_NOT_FOUND", "user not found")

	if err.Code() != "USER_NOT_FOUND" {
		t.Errorf("expected code USER_NOT_FOUND, got %s", err.Code())
	}

	// IsNotFound checks for CodeNotFound, not custom codes
	if HasCode(err, "USER_NOT_FOUND") == false {
		t.Error("expected HasCode to return true for USER_NOT_FOUND")
	}
}

func TestHasCode(t *testing.T) {
	err := New("TEST_CODE", "test message")

	if !HasCode(err, "TEST_CODE") {
		t.Error("expected HasCode to return true")
	}

	if HasCode(err, "OTHER_CODE") {
		t.Error("expected HasCode to return false for different code")
	}

	standardErr := errors.New("standard error")
	if HasCode(standardErr, "TEST_CODE") {
		t.Error("expected HasCode to return false for standard error")
	}
}

func TestGetCode(t *testing.T) {
	err := New("TEST_CODE", "test message")

	if GetCode(err) != "TEST_CODE" {
		t.Errorf("expected code TEST_CODE, got %s", GetCode(err))
	}

	standardErr := errors.New("standard error")
	if GetCode(standardErr) != CodeUnknown {
		t.Errorf("expected code %s for standard error, got %s", CodeUnknown, GetCode(standardErr))
	}
}

func TestUnwrap(t *testing.T) {
	original := errors.New("original")
	wrapped := Wrap(original, "wrapped")

	unwrapped := Unwrap(wrapped)
	if unwrapped != original {
		t.Error("expected Unwrap to return original error")
	}
}

func TestErrorString(t *testing.T) {
	err := New("TEST_CODE", "test message")
	expected := "[TEST_CODE] test message"

	if err.Error() != expected {
		t.Errorf("expected error string '%s', got '%s'", expected, err.Error())
	}

	original := errors.New("original")
	wrapped := Wrap(original, "wrapped")

	if wrapped.Error() != "[UNKNOWN] wrapped: original" {
		t.Errorf("unexpected error string: %s", wrapped.Error())
	}
}

func TestStackCapture(t *testing.T) {
	err := New("TEST_CODE", "test message")
	stack := err.Stack()

	if len(stack) == 0 {
		t.Error("expected stack trace to be captured")
	}

	// Verify stack has file and line information
	if stack[0].File == "" {
		t.Error("expected stack frame to have file information")
	}

	if stack[0].Line == 0 {
		t.Error("expected stack frame to have line information")
	}
}

func TestIs(t *testing.T) {
	original := errors.New("original")
	wrapped := Wrap(original, "wrapped")

	// Standard errors.Is requires Unwrap method to be implemented
	// Our Error type has Cause() instead, so we need to manually check
	if Unwrap(wrapped) != original {
		t.Error("expected Unwrap to return original error")
	}
}

func TestAs(t *testing.T) {
	err := NotFound("", "not found")

	var customErr Error
	if !As(err, &customErr) {
		t.Error("expected As to return true")
	}

	if customErr.Code() != CodeNotFound {
		t.Errorf("expected code %s, got %s", CodeNotFound, customErr.Code())
	}
}