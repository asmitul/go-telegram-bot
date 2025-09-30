# Error Package

提供结构化错误处理，支持错误码、错误包装、上下文信息和堆栈跟踪。

## 功能特性

- **错误码**: 预定义的错误码常量，便于错误分类和处理
- **错误包装**: 支持包装标准错误和自定义错误，保留原始错误信息
- **上下文信息**: 为错误添加键值对上下文，便于调试
- **堆栈跟踪**: 自动捕获错误发生时的调用栈
- **类型检查**: 提供便捷的错误类型检查函数
- **标准兼容**: 兼容标准库 `errors.Is` 和 `errors.As`

## 使用示例

### 创建错误

```go
import "github.com/yourusername/go-telegram-bot/pkg/errors"

// 使用预定义类型创建错误
err := errors.NotFound("", "用户不存在")
err := errors.Validation("INVALID_EMAIL", "邮箱格式不正确")
err := errors.Permission("", "权限不足")

// 使用自定义错误码
err := errors.New("CUSTOM_CODE", "自定义错误信息")
```

### 包装错误

```go
// 包装标准错误
if err := doSomething(); err != nil {
    return errors.Wrap(err, "执行操作失败")
}

// 使用自定义错误码包装
if err := doSomething(); err != nil {
    return errors.WrapWithCode(err, "OPERATION_FAILED", "执行操作失败")
}
```

### 添加上下文

```go
err := errors.NotFound("", "用户不存在").
    WithContext("user_id", "12345").
    WithContext("operation", "get_user")

// 获取上下文
if val, exists := errors.GetContext(err, "user_id"); exists {
    fmt.Println("User ID:", val)
}
```

### 错误类型检查

```go
if errors.IsNotFound(err) {
    // 处理资源不存在错误
    return http.StatusNotFound
}

if errors.IsValidation(err) {
    // 处理验证错误
    return http.StatusBadRequest
}

if errors.IsPermission(err) {
    // 处理权限错误
    return http.StatusForbidden
}
```

### 获取错误码

```go
code := errors.GetCode(err)
switch code {
case errors.CodeNotFound:
    // 处理资源不存在
case errors.CodeValidation:
    // 处理验证错误
}
```

### 堆栈跟踪

```go
err := errors.Internal("", "数据库连接失败")

// 获取堆栈信息
stack := err.Stack()
for i, frame := range stack {
    fmt.Printf("%d. %s:%d %s\n", i+1, frame.File, frame.Line, frame.Function)
}

// 格式化输出
fmt.Println(errors.FormatStack(stack))
```

### 完整示例

```go
package main

import (
    "fmt"
    "github.com/yourusername/go-telegram-bot/pkg/errors"
)

func getUser(id string) error {
    // 模拟数据库查询失败
    if id == "" {
        return errors.Validation("INVALID_USER_ID", "用户ID不能为空")
    }

    // 模拟用户不存在
    return errors.NotFound("USER_NOT_FOUND", "用户不存在").
        WithContext("user_id", id)
}

func main() {
    err := getUser("")

    if err != nil {
        // 检查错误类型
        if errors.IsValidation(err) {
            fmt.Println("验证错误:", err.Error())
        }

        // 获取错误码
        fmt.Println("错误码:", errors.GetCode(err))

        // 获取上下文
        if val, exists := errors.GetContext(err, "user_id"); exists {
            fmt.Println("用户ID:", val)
        }

        // 打印堆栈
        if e, ok := err.(errors.Error); ok {
            fmt.Println(errors.FormatStack(e.Stack()))
        }
    }
}
```

## 错误码列表

| 错误码 | 常量 | 说明 |
|--------|------|------|
| `NOT_FOUND` | `CodeNotFound` | 资源不存在 |
| `VALIDATION_ERROR` | `CodeValidation` | 验证错误 |
| `PERMISSION_DENIED` | `CodePermission` | 权限不足 |
| `INTERNAL_ERROR` | `CodeInternal` | 内部错误 |
| `EXTERNAL_ERROR` | `CodeExternal` | 外部服务错误 |
| `CONFLICT` | `CodeConflict` | 冲突错误（如重复创建） |
| `RATE_LIMIT_EXCEEDED` | `CodeRateLimit` | 限流错误 |
| `TIMEOUT` | `CodeTimeout` | 超时错误 |
| `UNKNOWN` | `CodeUnknown` | 未知错误 |

## API 文档

### 错误创建函数

- `New(code, message string) Error` - 创建新错误
- `NotFound(code, message string) Error` - 创建资源不存在错误
- `Validation(code, message string) Error` - 创建验证错误
- `Permission(code, message string) Error` - 创建权限错误
- `Internal(code, message string) Error` - 创建内部错误
- `External(code, message string) Error` - 创建外部服务错误
- `Conflict(code, message string) Error` - 创建冲突错误
- `RateLimit(message string) Error` - 创建限流错误
- `Timeout(message string) Error` - 创建超时错误

### 错误包装函数

- `Wrap(err error, message string) Error` - 包装错误
- `WrapWithCode(err error, code, message string) Error` - 使用自定义错误码包装错误

### 错误检查函数

- `IsNotFound(err error) bool` - 检查是否为资源不存在错误
- `IsValidation(err error) bool` - 检查是否为验证错误
- `IsPermission(err error) bool` - 检查是否为权限错误
- `IsInternal(err error) bool` - 检查是否为内部错误
- `IsExternal(err error) bool` - 检查是否为外部服务错误
- `IsConflict(err error) bool` - 检查是否为冲突错误
- `IsRateLimit(err error) bool` - 检查是否为限流错误
- `IsTimeout(err error) bool` - 检查是否为超时错误

### 工具函数

- `GetCode(err error) string` - 获取错误码
- `HasCode(err error, code string) bool` - 检查错误是否包含指定错误码
- `GetContext(err error, key string) (string, bool)` - 获取上下文信息
- `Unwrap(err error) error` - 解包错误
- `Is(err, target error) bool` - 兼容标准库 errors.Is
- `As(err error, target interface{}) bool` - 兼容标准库 errors.As

## 最佳实践

1. **使用预定义错误类型**: 优先使用 `NotFound`、`Validation` 等预定义函数，而不是直接使用 `New`
2. **添加上下文**: 为错误添加相关上下文信息，便于调试
3. **保留原始错误**: 使用 `Wrap` 包装底层错误，保留完整错误链
4. **统一错误码**: 在团队内部定义统一的错误码规范
5. **避免过度包装**: 不要在每一层都包装错误，只在需要添加额外信息时包装