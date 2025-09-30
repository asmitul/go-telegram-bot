package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestStandardLogger(t *testing.T) {
	var buf bytes.Buffer

	log := NewStandardLogger(Config{
		Level:      LevelDebug,
		Output:     &buf,
		TimeFormat: "2006-01-02 15:04:05",
	})

	// 测试 Debug
	log.Debug("debug message", "key", "value")
	if !strings.Contains(buf.String(), "[DEBUG]") {
		t.Error("Debug log not found")
	}
	if !strings.Contains(buf.String(), "debug message") {
		t.Error("Debug message not found")
	}
	if !strings.Contains(buf.String(), "key=value") {
		t.Error("Debug field not found")
	}
	buf.Reset()

	// 测试 Info
	log.Info("info message")
	if !strings.Contains(buf.String(), "[INFO]") {
		t.Error("Info log not found")
	}
	buf.Reset()

	// 测试 Warn
	log.Warn("warn message")
	if !strings.Contains(buf.String(), "[WARN]") {
		t.Error("Warn log not found")
	}
	buf.Reset()

	// 测试 Error
	log.Error("error message")
	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Error("Error log not found")
	}
	buf.Reset()
}

func TestLogLevel(t *testing.T) {
	var buf bytes.Buffer

	log := NewStandardLogger(Config{
		Level:  LevelWarn, // 只输出 Warn 和 Error
		Output: &buf,
	})

	log.Debug("debug")
	log.Info("info")
	if buf.Len() > 0 {
		t.Error("Debug and Info should not be logged")
	}

	log.Warn("warn")
	if !strings.Contains(buf.String(), "warn") {
		t.Error("Warn should be logged")
	}
	buf.Reset()

	log.Error("error")
	if !strings.Contains(buf.String(), "error") {
		t.Error("Error should be logged")
	}
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer

	log := NewStandardLogger(Config{
		Level:  LevelInfo,
		Output: &buf,
	})

	// 添加持久字段
	logWithUser := log.WithField("user_id", 12345)
	logWithUser.Info("user action")

	if !strings.Contains(buf.String(), "user_id=12345") {
		t.Error("user_id field not found")
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer

	log := NewStandardLogger(Config{
		Level:  LevelInfo,
		Output: &buf,
	})

	logWithFields := log.WithFields(map[string]interface{}{
		"user_id":  12345,
		"group_id": 67890,
	})
	logWithFields.Info("action")

	output := buf.String()
	if !strings.Contains(output, "user_id=12345") {
		t.Error("user_id field not found")
	}
	if !strings.Contains(output, "group_id=67890") {
		t.Error("group_id field not found")
	}
}

func TestJSONLogger(t *testing.T) {
	var buf bytes.Buffer

	log := NewJSONLogger(Config{
		Level:  LevelInfo,
		Output: &buf,
	})

	log.Info("test message", "key", "value")
	output := buf.String()

	if !strings.Contains(output, `"level":"INFO"`) {
		t.Error("JSON level not found")
	}
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Error("JSON message not found")
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Error("JSON field not found")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"unknown", LevelInfo}, // 默认
	}

	for _, tt := range tests {
		got := ParseLevel(tt.input)
		if got != tt.want {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
	}

	for _, tt := range tests {
		got := tt.level.String()
		if got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}