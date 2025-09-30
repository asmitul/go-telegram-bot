package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 应用配置
type Config struct {
	// Telegram 配置
	TelegramToken string
	Debug         bool

	// MongoDB 配置
	MongoURI     string
	DatabaseName string
	MongoTimeout time.Duration

	// 应用配置
	Environment string
	LogLevel    string
	LogFormat   string // "text" 或 "json"
	Port        int

	// 限流配置
	RateLimitEnabled bool
	RateLimitPerMin  int

	// 监控配置
	MetricsEnabled bool
	MetricsPort    int
}

// Load 加载配置
func Load() (*Config, error) {
	cfg := &Config{
		TelegramToken:    getEnv("TELEGRAM_TOKEN", ""),
		Debug:            getEnvBool("DEBUG", false),
		MongoURI:         getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DatabaseName:     getEnv("DATABASE_NAME", "telegram_bot"),
		MongoTimeout:     getEnvDuration("MONGO_TIMEOUT", 10*time.Second),
		Environment:      getEnv("ENVIRONMENT", "development"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		LogFormat:        getEnv("LOG_FORMAT", "text"),
		Port:             getEnvInt("PORT", 8080),
		RateLimitEnabled: getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitPerMin:  getEnvInt("RATE_LIMIT_PER_MIN", 20),
		MetricsEnabled:   getEnvBool("METRICS_ENABLED", true),
		MetricsPort:      getEnvInt("METRICS_PORT", 9091),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.TelegramToken == "" {
		return fmt.Errorf("TELEGRAM_TOKEN is required")
	}

	if c.MongoURI == "" {
		return fmt.Errorf("MONGO_URI is required")
	}

	if c.DatabaseName == "" {
		return fmt.Errorf("DATABASE_NAME is required")
	}

	return nil
}

// IsProduction 是否为生产环境
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment 是否为开发环境
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔类型环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

// getEnvInt 获取整数类型环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// getEnvDuration 获取时间间隔类型环境变量
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
