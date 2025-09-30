package validator

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"telegram-bot/pkg/errors"
)

// Validator 验证器接口
type Validator interface {
	// Validate 执行验证，返回验证错误
	Validate() error
}

// Result 验证结果
type Result struct {
	Valid  bool
	Errors []error
}

// AddError 添加错误
func (r *Result) AddError(err error) {
	r.Valid = false
	r.Errors = append(r.Errors, err)
}

// Error 返回第一个错误
func (r *Result) Error() error {
	if len(r.Errors) > 0 {
		return r.Errors[0]
	}
	return nil
}

// AllErrors 返回所有错误
func (r *Result) AllErrors() []error {
	return r.Errors
}

// Required 验证必填字段
func Required(value string, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return errors.Validation("FIELD_REQUIRED", fieldName+"不能为空")
	}
	return nil
}

// MinLength 验证最小长度
func MinLength(value string, min int, fieldName string) error {
	length := utf8.RuneCountInString(value)
	if length < min {
		return errors.Validation("MIN_LENGTH", fieldName+"长度不能少于指定字符")
	}
	return nil
}

// MaxLength 验证最大长度
func MaxLength(value string, max int, fieldName string) error {
	length := utf8.RuneCountInString(value)
	if length > max {
		return errors.Validation("MAX_LENGTH", fieldName+"长度不能超过指定字符")
	}
	return nil
}

// LengthRange 验证长度范围
func LengthRange(value string, min, max int, fieldName string) error {
	length := utf8.RuneCountInString(value)
	if length < min || length > max {
		return errors.Validation("LENGTH_RANGE", fieldName+"长度必须在指定范围内")
	}
	return nil
}

// Pattern 验证正则表达式
func Pattern(value string, pattern string, fieldName string) error {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return errors.Internal("REGEX_ERROR", "正则表达式错误")
	}
	if !matched {
		return errors.Validation("PATTERN_MISMATCH", fieldName+"格式不正确")
	}
	return nil
}

// UserID 验证用户 ID
func UserID(id int64) error {
	if id <= 0 {
		return errors.Validation("INVALID_USER_ID", "用户ID必须大于0")
	}
	return nil
}

// GroupID 验证群组 ID
func GroupID(id int64) error {
	// Telegram 群组 ID 通常是负数
	if id >= 0 {
		return errors.Validation("INVALID_GROUP_ID", "群组ID格式不正确")
	}
	return nil
}

// Username 验证用户名格式
func Username(username string) error {
	if username == "" {
		return errors.Validation("EMPTY_USERNAME", "用户名不能为空")
	}

	// Telegram 用户名规则：5-32 字符，只能包含字母、数字和下划线
	if len(username) < 5 || len(username) > 32 {
		return errors.Validation("INVALID_USERNAME_LENGTH", "用户名长度必须在5-32个字符之间")
	}

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, username)
	if !matched {
		return errors.Validation("INVALID_USERNAME_FORMAT", "用户名只能包含字母、数字和下划线")
	}

	return nil
}

// CommandName 验证命令名称
func CommandName(command string) error {
	if command == "" {
		return errors.Validation("EMPTY_COMMAND", "命令名称不能为空")
	}

	// 命令必须以 / 开头
	if !strings.HasPrefix(command, "/") {
		return errors.Validation("INVALID_COMMAND_FORMAT", "命令必须以/开头")
	}

	// 移除 / 后验证命令名
	cmdName := strings.TrimPrefix(command, "/")
	if cmdName == "" {
		return errors.Validation("EMPTY_COMMAND_NAME", "命令名称不能为空")
	}

	// 命令名只能包含字母、数字和下划线
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, cmdName)
	if !matched {
		return errors.Validation("INVALID_COMMAND_NAME", "命令名称只能包含字母、数字和下划线")
	}

	return nil
}

// TextMessage 验证文本消息
func TextMessage(text string, minLen, maxLen int) error {
	if err := Required(text, "消息内容"); err != nil {
		return err
	}

	length := utf8.RuneCountInString(text)
	if length < minLen {
		return errors.Validation("MESSAGE_TOO_SHORT", "消息内容太短")
	}

	if length > maxLen {
		return errors.Validation("MESSAGE_TOO_LONG", "消息内容太长")
	}

	return nil
}

// InSlice 验证值是否在切片中
func InSlice(value string, slice []string, fieldName string) error {
	for _, item := range slice {
		if item == value {
			return nil
		}
	}
	return errors.Validation("INVALID_VALUE", fieldName+"的值不在允许的范围内")
}

// NotInSlice 验证值是否不在切片中
func NotInSlice(value string, slice []string, fieldName string) error {
	for _, item := range slice {
		if item == value {
			return errors.Validation("FORBIDDEN_VALUE", fieldName+"的值不允许使用")
		}
	}
	return nil
}

// Email 验证邮箱格式
func Email(email string) error {
	if email == "" {
		return errors.Validation("EMPTY_EMAIL", "邮箱不能为空")
	}

	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	if !matched {
		return errors.Validation("INVALID_EMAIL", "邮箱格式不正确")
	}

	return nil
}

// URL 验证 URL 格式
func URL(url string) error {
	if url == "" {
		return errors.Validation("EMPTY_URL", "URL不能为空")
	}

	pattern := `^https?://[^\s]+$`
	matched, _ := regexp.MatchString(pattern, url)
	if !matched {
		return errors.Validation("INVALID_URL", "URL格式不正确")
	}

	return nil
}

// Chain 链式验证器
type Chain struct {
	result *Result
}

// NewChain 创建新的链式验证器
func NewChain() *Chain {
	return &Chain{
		result: &Result{Valid: true, Errors: []error{}},
	}
}

// Add 添加验证规则
func (c *Chain) Add(err error) *Chain {
	if err != nil {
		c.result.AddError(err)
	}
	return c
}

// Result 获取验证结果
func (c *Chain) Result() *Result {
	return c.result
}

// Error 获取第一个错误
func (c *Chain) Error() error {
	return c.result.Error()
}

// IsValid 是否验证通过
func (c *Chain) IsValid() bool {
	return c.result.Valid
}
