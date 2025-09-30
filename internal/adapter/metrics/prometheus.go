package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics 指标收集器
type Metrics struct {
	// Counter - 命令执行次数
	CommandTotal *prometheus.CounterVec
	// Counter - 命令执行成功次数
	CommandSuccess *prometheus.CounterVec
	// Counter - 命令执行失败次数
	CommandFailure *prometheus.CounterVec
	// Histogram - 命令执行时长
	CommandDuration *prometheus.HistogramVec
	// Gauge - 活跃用户数
	ActiveUsers prometheus.Gauge
	// Gauge - 活跃群组数
	ActiveGroups prometheus.Gauge
	// Counter - 处理的消息总数
	MessagesTotal prometheus.Counter
	// Counter - 限流拒绝次数
	RateLimitRejections *prometheus.CounterVec
}

// NewMetrics 创建新的指标收集器
func NewMetrics() *Metrics {
	return &Metrics{
		CommandTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "telegram_bot_command_total",
				Help: "Total number of commands executed",
			},
			[]string{"command", "group_id"},
		),
		CommandSuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "telegram_bot_command_success_total",
				Help: "Total number of successful commands",
			},
			[]string{"command", "group_id"},
		),
		CommandFailure: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "telegram_bot_command_failure_total",
				Help: "Total number of failed commands",
			},
			[]string{"command", "group_id", "error_type"},
		),
		CommandDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "telegram_bot_command_duration_seconds",
				Help:    "Command execution duration in seconds",
				Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
			},
			[]string{"command", "group_id"},
		),
		ActiveUsers: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "telegram_bot_active_users",
				Help: "Number of active users",
			},
		),
		ActiveGroups: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "telegram_bot_active_groups",
				Help: "Number of active groups",
			},
		),
		MessagesTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "telegram_bot_messages_total",
				Help: "Total number of messages processed",
			},
		),
		RateLimitRejections: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "telegram_bot_ratelimit_rejections_total",
				Help: "Total number of requests rejected by rate limiter",
			},
			[]string{"user_id", "command"},
		),
	}
}

// RecordCommand 记录命令执行
func (m *Metrics) RecordCommand(command string, groupID int64) {
	m.CommandTotal.WithLabelValues(command, int64ToString(groupID)).Inc()
}

// RecordCommandSuccess 记录命令执行成功
func (m *Metrics) RecordCommandSuccess(command string, groupID int64) {
	m.CommandSuccess.WithLabelValues(command, int64ToString(groupID)).Inc()
}

// RecordCommandFailure 记录命令执行失败
func (m *Metrics) RecordCommandFailure(command string, groupID int64, errorType string) {
	m.CommandFailure.WithLabelValues(command, int64ToString(groupID), errorType).Inc()
}

// RecordCommandDuration 记录命令执行时长
func (m *Metrics) RecordCommandDuration(command string, groupID int64, duration float64) {
	m.CommandDuration.WithLabelValues(command, int64ToString(groupID)).Observe(duration)
}

// SetActiveUsers 设置活跃用户数
func (m *Metrics) SetActiveUsers(count float64) {
	m.ActiveUsers.Set(count)
}

// SetActiveGroups 设置活跃群组数
func (m *Metrics) SetActiveGroups(count float64) {
	m.ActiveGroups.Set(count)
}

// IncActiveUsers 增加活跃用户数
func (m *Metrics) IncActiveUsers() {
	m.ActiveUsers.Inc()
}

// DecActiveUsers 减少活跃用户数
func (m *Metrics) DecActiveUsers() {
	m.ActiveUsers.Dec()
}

// IncActiveGroups 增加活跃群组数
func (m *Metrics) IncActiveGroups() {
	m.ActiveGroups.Inc()
}

// DecActiveGroups 减少活跃群组数
func (m *Metrics) DecActiveGroups() {
	m.ActiveGroups.Dec()
}

// RecordMessage 记录消息处理
func (m *Metrics) RecordMessage() {
	m.MessagesTotal.Inc()
}

// RecordRateLimitRejection 记录限流拒绝
func (m *Metrics) RecordRateLimitRejection(userID int64, command string) {
	m.RateLimitRejections.WithLabelValues(int64ToString(userID), command).Inc()
}

// int64ToString 将 int64 转换为字符串
func int64ToString(n int64) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}
