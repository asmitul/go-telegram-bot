package errors

import (
	"errors"
	"fmt"
)

// Error 自定义错误接口
type Error interface {
	error
	Code() string                        // 错误码
	Message() string                     // 错误消息
	Cause() error                        // 原始错误
	WithContext(key, val string) Error   // 添加上下文信息
	Context() map[string]string          // 获取上下文信息
	Stack() []Frame                      // 堆栈信息
}

// baseError 基础错误实现
type baseError struct {
	code    string
	message string
	cause   error
	context map[string]string
	stack   []Frame
}

// Error 实现 error 接口
func (e *baseError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("[%s] %s", e.code, e.message)
}

// Code 返回错误码
func (e *baseError) Code() string {
	return e.code
}

// Message 返回错误消息
func (e *baseError) Message() string {
	return e.message
}

// Cause 返回原始错误
func (e *baseError) Cause() error {
	return e.cause
}

// Context 返回上下文信息
func (e *baseError) Context() map[string]string {
	return e.context
}

// Stack 返回堆栈信息
func (e *baseError) Stack() []Frame {
	return e.stack
}

// WithContext 添加上下文信息
func (e *baseError) WithContext(key, val string) Error {
	// 创建新的上下文映射，避免修改原错误
	newCtx := make(map[string]string, len(e.context)+1)
	for k, v := range e.context {
		newCtx[k] = v
	}
	newCtx[key] = val

	return &baseError{
		code:    e.code,
		message: e.message,
		cause:   e.cause,
		context: newCtx,
		stack:   e.stack,
	}
}

// New 创建新错误
func New(code, message string) Error {
	return &baseError{
		code:    code,
		message: message,
		context: make(map[string]string),
		stack:   captureStack(3), // skip: captureStack, New, caller
	}
}

// Wrap 包装错误
func Wrap(err error, message string) Error {
	if err == nil {
		return nil
	}

	// 如果是自定义 Error 类型，保留其错误码
	if e, ok := err.(Error); ok {
		return &baseError{
			code:    e.Code(),
			message: message,
			cause:   e,
			context: make(map[string]string),
			stack:   captureStack(3),
		}
	}

	return &baseError{
		code:    CodeUnknown,
		message: message,
		cause:   err,
		context: make(map[string]string),
		stack:   captureStack(3),
	}
}

// WrapWithCode 使用指定错误码包装错误
func WrapWithCode(err error, code, message string) Error {
	if err == nil {
		return nil
	}

	return &baseError{
		code:    code,
		message: message,
		cause:   err,
		context: make(map[string]string),
		stack:   captureStack(3),
	}
}

// Is 检查错误是否匹配，兼容标准库 errors.Is
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As 转换错误类型，兼容标准库 errors.As
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Unwrap 解包错误
func Unwrap(err error) error {
	if e, ok := err.(Error); ok {
		return e.Cause()
	}
	return errors.Unwrap(err)
}

// GetCode 获取错误码
func GetCode(err error) string {
	if e, ok := err.(Error); ok {
		return e.Code()
	}
	return CodeUnknown
}

// HasCode 检查错误是否包含指定错误码
func HasCode(err error, code string) bool {
	if e, ok := err.(Error); ok {
		return e.Code() == code
	}
	return false
}

// GetContext 获取上下文信息
func GetContext(err error, key string) (string, bool) {
	if e, ok := err.(Error); ok {
		val, exists := e.Context()[key]
		return val, exists
	}
	return "", false
}