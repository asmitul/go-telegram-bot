package logger

import (
	"regexp"
	"strings"
)

// Sanitizer 敏感信息脱敏器
type Sanitizer struct {
	patterns map[string]*regexp.Regexp
}

// NewSanitizer 创建新的脱敏器
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		patterns: map[string]*regexp.Regexp{
			// Token/API Key 模式 (Telegram bot token)
			"token": regexp.MustCompile(`\d{8,10}:[A-Za-z0-9_-]{30,}`),
			// 密码模式
			"password": regexp.MustCompile(`(?i)(password|passwd|pwd|secret)\s*[:=]\s*["']?([^\s"',}]+)["']?`),
			// Email 模式
			"email": regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
			// 信用卡模式
			"credit_card": regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),
			// 手机号模式（中国）
			"phone": regexp.MustCompile(`\b1[3-9]\d{9}\b`),
		},
	}
}

// Sanitize 脱敏字符串
func (s *Sanitizer) Sanitize(text string) string {
	result := text

	// Token 脱敏
	if s.patterns["token"].MatchString(result) {
		result = s.patterns["token"].ReplaceAllStringFunc(result, func(match string) string {
			if len(match) > 15 {
				return match[:8] + "..." + match[len(match)-4:]
			}
			return "***TOKEN***"
		})
	}

	// 密码脱敏
	if s.patterns["password"].MatchString(result) {
		result = s.patterns["password"].ReplaceAllString(result, "$1=***")
	}

	// Email 脱敏
	if s.patterns["email"].MatchString(result) {
		result = s.patterns["email"].ReplaceAllStringFunc(result, func(match string) string {
			parts := strings.Split(match, "@")
			if len(parts) == 2 && len(parts[0]) > 2 {
				return parts[0][:2] + "***@" + parts[1]
			}
			return "***@" + parts[1]
		})
	}

	// 信用卡脱敏
	if s.patterns["credit_card"].MatchString(result) {
		result = s.patterns["credit_card"].ReplaceAllString(result, "****-****-****-$1")
	}

	// 手机号脱敏
	if s.patterns["phone"].MatchString(result) {
		result = s.patterns["phone"].ReplaceAllStringFunc(result, func(match string) string {
			if len(match) == 11 {
				return match[:3] + "****" + match[7:]
			}
			return match
		})
	}

	return result
}

// SanitizeFields 脱敏字段值
func (s *Sanitizer) SanitizeFields(fields map[string]interface{}) map[string]interface{} {
	sensitiveKeys := map[string]bool{
		"password":     true,
		"passwd":       true,
		"pwd":          true,
		"secret":       true,
		"token":        true,
		"api_key":      true,
		"access_token": true,
		"private_key":  true,
	}

	result := make(map[string]interface{})
	for k, v := range fields {
		key := strings.ToLower(k)
		if sensitiveKeys[key] {
			result[k] = "***"
		} else if str, ok := v.(string); ok {
			result[k] = s.Sanitize(str)
		} else {
			result[k] = v
		}
	}
	return result
}

// Global sanitizer instance
var globalSanitizer = NewSanitizer()

// SanitizeString 使用全局脱敏器脱敏字符串
func SanitizeString(text string) string {
	return globalSanitizer.Sanitize(text)
}

// SanitizeFields 使用全局脱敏器脱敏字段
func SanitizeFields(fields map[string]interface{}) map[string]interface{} {
	return globalSanitizer.SanitizeFields(fields)
}
