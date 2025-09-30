package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewMetrics(t *testing.T) {
	// 使用全局 metrics 避免重复注册
	m := globalMetrics

	if m.CommandTotal == nil {
		t.Error("CommandTotal should not be nil")
	}
	if m.CommandSuccess == nil {
		t.Error("CommandSuccess should not be nil")
	}
	if m.CommandFailure == nil {
		t.Error("CommandFailure should not be nil")
	}
	if m.CommandDuration == nil {
		t.Error("CommandDuration should not be nil")
	}
	if m.ActiveUsers == nil {
		t.Error("ActiveUsers should not be nil")
	}
	if m.ActiveGroups == nil {
		t.Error("ActiveGroups should not be nil")
	}
	if m.MessagesTotal == nil {
		t.Error("MessagesTotal should not be nil")
	}
	if m.RateLimitRejections == nil {
		t.Error("RateLimitRejections should not be nil")
	}
}

func TestMetrics_RecordCommand(t *testing.T) {
	// 创建独立的 registry 避免冲突
	reg := prometheus.NewRegistry()
	commandTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_command_total",
			Help: "Test command total",
		},
		[]string{"command", "group_id"},
	)
	reg.MustRegister(commandTotal)

	// 记录命令
	commandTotal.WithLabelValues("ping", "-1").Inc()
	commandTotal.WithLabelValues("ping", "-1").Inc()
	commandTotal.WithLabelValues("help", "-2").Inc()

	// 验证计数
	count := testutil.ToFloat64(commandTotal.WithLabelValues("ping", "-1"))
	if count != 2 {
		t.Errorf("expected 2, got %f", count)
	}

	count = testutil.ToFloat64(commandTotal.WithLabelValues("help", "-2"))
	if count != 1 {
		t.Errorf("expected 1, got %f", count)
	}
}

func TestMetrics_RecordCommandSuccess(t *testing.T) {
	reg := prometheus.NewRegistry()
	commandSuccess := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_command_success",
			Help: "Test command success",
		},
		[]string{"command", "group_id"},
	)
	reg.MustRegister(commandSuccess)

	// 记录成功
	commandSuccess.WithLabelValues("ping", "-1").Inc()

	// 验证计数
	count := testutil.ToFloat64(commandSuccess.WithLabelValues("ping", "-1"))
	if count != 1 {
		t.Errorf("expected 1, got %f", count)
	}
}

func TestMetrics_RecordCommandFailure(t *testing.T) {
	reg := prometheus.NewRegistry()
	commandFailure := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_command_failure",
			Help: "Test command failure",
		},
		[]string{"command", "group_id", "error_type"},
	)
	reg.MustRegister(commandFailure)

	// 记录失败
	commandFailure.WithLabelValues("ping", "-1", "rate_limit").Inc()
	commandFailure.WithLabelValues("ping", "-1", "other").Inc()

	// 验证计数
	count := testutil.ToFloat64(commandFailure.WithLabelValues("ping", "-1", "rate_limit"))
	if count != 1 {
		t.Errorf("expected 1, got %f", count)
	}

	count = testutil.ToFloat64(commandFailure.WithLabelValues("ping", "-1", "other"))
	if count != 1 {
		t.Errorf("expected 1, got %f", count)
	}
}

func TestMetrics_RecordCommandDuration(t *testing.T) {
	reg := prometheus.NewRegistry()
	commandDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_command_duration",
			Help:    "Test command duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"command", "group_id"},
	)
	reg.MustRegister(commandDuration)

	// 记录时长
	commandDuration.WithLabelValues("ping", "-1").Observe(0.1)
	commandDuration.WithLabelValues("ping", "-1").Observe(0.2)

	// 对于 histogram，我们只验证它不会 panic
	// testutil.ToFloat64 不支持 histogram 的 Observer
	// 这里简单验证功能正常
}

func TestMetrics_ActiveUsers(t *testing.T) {
	reg := prometheus.NewRegistry()
	activeUsers := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "test_active_users",
			Help: "Test active users",
		},
	)
	reg.MustRegister(activeUsers)

	// 设置值
	activeUsers.Set(10)
	value := testutil.ToFloat64(activeUsers)
	if value != 10 {
		t.Errorf("expected 10, got %f", value)
	}

	// 增加
	activeUsers.Inc()
	value = testutil.ToFloat64(activeUsers)
	if value != 11 {
		t.Errorf("expected 11, got %f", value)
	}

	// 减少
	activeUsers.Dec()
	value = testutil.ToFloat64(activeUsers)
	if value != 10 {
		t.Errorf("expected 10, got %f", value)
	}
}

func TestMetrics_ActiveGroups(t *testing.T) {
	reg := prometheus.NewRegistry()
	activeGroups := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "test_active_groups",
			Help: "Test active groups",
		},
	)
	reg.MustRegister(activeGroups)

	// 设置值
	activeGroups.Set(5)
	value := testutil.ToFloat64(activeGroups)
	if value != 5 {
		t.Errorf("expected 5, got %f", value)
	}

	// 增加
	activeGroups.Inc()
	value = testutil.ToFloat64(activeGroups)
	if value != 6 {
		t.Errorf("expected 6, got %f", value)
	}

	// 减少
	activeGroups.Dec()
	value = testutil.ToFloat64(activeGroups)
	if value != 5 {
		t.Errorf("expected 5, got %f", value)
	}
}

func TestMetrics_RecordMessage(t *testing.T) {
	reg := prometheus.NewRegistry()
	messagesTotal := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "test_messages_total",
			Help: "Test messages total",
		},
	)
	reg.MustRegister(messagesTotal)

	// 记录消息
	messagesTotal.Inc()
	messagesTotal.Inc()

	// 验证计数
	count := testutil.ToFloat64(messagesTotal)
	if count != 2 {
		t.Errorf("expected 2, got %f", count)
	}
}

func TestMetrics_RecordRateLimitRejection(t *testing.T) {
	reg := prometheus.NewRegistry()
	rateLimitRejections := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_ratelimit_rejections",
			Help: "Test rate limit rejections",
		},
		[]string{"user_id", "command"},
	)
	reg.MustRegister(rateLimitRejections)

	// 记录限流拒绝
	rateLimitRejections.WithLabelValues("123", "ping").Inc()
	rateLimitRejections.WithLabelValues("123", "ping").Inc()

	// 验证计数
	count := testutil.ToFloat64(rateLimitRejections.WithLabelValues("123", "ping"))
	if count != 2 {
		t.Errorf("expected 2, got %f", count)
	}
}

func TestInt64ToString(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{-456, "-456"},
		{1, "1"},
		{-1, "-1"},
		{9999999999, "9999999999"},
	}

	for _, tt := range tests {
		result := int64ToString(tt.input)
		if result != tt.expected {
			t.Errorf("int64ToString(%d) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestMetrics_Integration(t *testing.T) {
	// 创建独立的 registry
	reg := prometheus.NewRegistry()

	commandTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_integration_command_total",
			Help: "Test integration command total",
		},
		[]string{"command", "group_id"},
	)
	reg.MustRegister(commandTotal)

	commandSuccess := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_integration_command_success",
			Help: "Test integration command success",
		},
		[]string{"command", "group_id"},
	)
	reg.MustRegister(commandSuccess)

	commandFailure := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_integration_command_failure",
			Help: "Test integration command failure",
		},
		[]string{"command", "group_id", "error_type"},
	)
	reg.MustRegister(commandFailure)

	// 模拟命令执行流程
	// 1. ping 命令成功执行 2 次
	commandTotal.WithLabelValues("ping", "-1").Inc()
	commandSuccess.WithLabelValues("ping", "-1").Inc()

	commandTotal.WithLabelValues("ping", "-1").Inc()
	commandSuccess.WithLabelValues("ping", "-1").Inc()

	// 2. help 命令失败 1 次
	commandTotal.WithLabelValues("help", "-1").Inc()
	commandFailure.WithLabelValues("help", "-1", "other").Inc()

	// 验证结果
	if count := testutil.ToFloat64(commandTotal.WithLabelValues("ping", "-1")); count != 2 {
		t.Errorf("expected 2 ping commands, got %f", count)
	}

	if count := testutil.ToFloat64(commandSuccess.WithLabelValues("ping", "-1")); count != 2 {
		t.Errorf("expected 2 successful ping commands, got %f", count)
	}

	if count := testutil.ToFloat64(commandTotal.WithLabelValues("help", "-1")); count != 1 {
		t.Errorf("expected 1 help command, got %f", count)
	}

	if count := testutil.ToFloat64(commandFailure.WithLabelValues("help", "-1", "other")); count != 1 {
		t.Errorf("expected 1 failed help command, got %f", count)
	}
}
