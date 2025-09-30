package logger

import (
	"strings"
	"testing"
)

func TestSanitizer_Token(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "Telegram Bot Token",
			input:    "Bot token: 123456789:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefg",
			contains: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Sanitize(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain '%s', got: %s", tt.contains, result)
			}
			if result == tt.input {
				t.Error("expected sanitized output to be different from input")
			}
		})
	}
}

func TestSanitizer_Password(t *testing.T) {
	s := NewSanitizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password with colon",
			input:    "password: secret123",
			expected: "password=***",
		},
		{
			name:     "password with equals",
			input:    "password=secret123",
			expected: "password=***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Sanitize(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected result to contain '%s', got: %s", tt.expected, result)
			}
		})
	}
}

func TestSanitizer_Email(t *testing.T) {
	s := NewSanitizer()

	input := "Email: user@example.com"
	result := s.Sanitize(input)

	if !strings.Contains(result, "***@example.com") {
		t.Errorf("expected sanitized email, got: %s", result)
	}
}

func TestSanitizer_Phone(t *testing.T) {
	s := NewSanitizer()

	input := "Phone: 13812345678"
	result := s.Sanitize(input)

	if !strings.Contains(result, "138****5678") {
		t.Errorf("expected sanitized phone, got: %s", result)
	}
}

func TestSanitizer_CreditCard(t *testing.T) {
	s := NewSanitizer()

	input := "Card: 1234 5678 9012 3456"
	result := s.Sanitize(input)

	if !strings.Contains(result, "****-****-****") {
		t.Errorf("expected sanitized credit card, got: %s", result)
	}
}

func TestSanitizer_SanitizeFields(t *testing.T) {
	s := NewSanitizer()

	fields := map[string]interface{}{
		"username": "testuser",
		"password": "secret123",
		"token":    "abc123",
		"count":    42,
	}

	result := s.SanitizeFields(fields)

	if result["username"] != "testuser" {
		t.Error("expected username to remain unchanged")
	}

	if result["password"] != "***" {
		t.Errorf("expected password to be sanitized, got: %v", result["password"])
	}

	if result["token"] != "***" {
		t.Errorf("expected token to be sanitized, got: %v", result["token"])
	}

	if result["count"] != 42 {
		t.Error("expected count to remain unchanged")
	}
}

func TestSanitizeString(t *testing.T) {
	input := "password: secret123"
	result := SanitizeString(input)

	if !strings.Contains(result, "***") {
		t.Errorf("expected sanitized output, got: %s", result)
	}
}

func TestSanitizeFields_Global(t *testing.T) {
	fields := map[string]interface{}{
		"api_key": "secret",
		"value":   "normal",
	}

	result := SanitizeFields(fields)

	if result["api_key"] != "***" {
		t.Error("expected api_key to be sanitized")
	}

	if result["value"] != "normal" {
		t.Error("expected value to remain unchanged")
	}
}

func TestSanitizer_MultiplePatterns(t *testing.T) {
	s := NewSanitizer()

	input := "User: user@example.com, password: secret123, phone: 13812345678"
	result := s.Sanitize(input)

	// 检查所有敏感信息都被脱敏
	if !strings.Contains(result, "***@example.com") {
		t.Error("expected email to be sanitized")
	}
	if !strings.Contains(result, "password=***") {
		t.Error("expected password to be sanitized")
	}
	if !strings.Contains(result, "138****5678") {
		t.Error("expected phone to be sanitized")
	}
}
