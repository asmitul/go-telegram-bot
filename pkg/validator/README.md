# Validator Package

提供通用验证器，支持字段验证、格式验证、链式验证等功能。

## 功能特性

- **字段验证**: 必填、长度、范围等基础验证
- **格式验证**: 正则表达式、邮箱、URL 等格式验证
- **业务验证**: 用户 ID、群组 ID、命令名称等业务规则验证
- **链式验证**: 支持链式调用，一次验证多个规则
- **错误集合**: 收集所有验证错误，便于批量处理

## 使用示例

### 基础验证

```go
import "github.com/yourusername/go-telegram-bot/pkg/validator"

// 验证必填字段
if err := validator.Required(username, "用户名"); err != nil {
    return err
}

// 验证长度
if err := validator.MinLength(password, 8, "密码"); err != nil {
    return err
}

if err := validator.MaxLength(bio, 200, "个人简介"); err != nil {
    return err
}

// 验证长度范围
if err := validator.LengthRange(title, 5, 100, "标题"); err != nil {
    return err
}
```

### Telegram 业务验证

```go
// 验证用户 ID
if err := validator.UserID(userID); err != nil {
    return err
}

// 验证群组 ID
if err := validator.GroupID(groupID); err != nil {
    return err
}

// 验证用户名（5-32 字符，字母数字下划线）
if err := validator.Username("test_user"); err != nil {
    return err
}

// 验证命令名称
if err := validator.CommandName("/start"); err != nil {
    return err
}

// 验证文本消息
if err := validator.TextMessage(text, 1, 4096); err != nil {
    return err
}
```

### 格式验证

```go
// 正则表达式验证
if err := validator.Pattern(code, `^\d{6}$`, "验证码"); err != nil {
    return err
}

// 邮箱验证
if err := validator.Email("user@example.com"); err != nil {
    return err
}

// URL 验证
if err := validator.URL("https://example.com"); err != nil {
    return err
}
```

### 切片验证

```go
// 验证值是否在允许列表中
allowedRoles := []string{"admin", "user", "guest"}
if err := validator.InSlice(role, allowedRoles, "角色"); err != nil {
    return err
}

// 验证值是否不在禁止列表中
forbiddenWords := []string{"spam", "ad", "abuse"}
if err := validator.NotInSlice(word, forbiddenWords, "关键词"); err != nil {
    return err
}
```

### 链式验证

```go
// 创建链式验证器
chain := validator.NewChain().
    Add(validator.UserID(userID)).
    Add(validator.Username(username)).
    Add(validator.Required(bio, "个人简介")).
    Add(validator.MaxLength(bio, 200, "个人简介"))

// 检查是否全部通过
if !chain.IsValid() {
    // 获取第一个错误
    return chain.Error()
}

// 或获取所有错误
result := chain.Result()
for _, err := range result.AllErrors() {
    fmt.Println(err)
}
```

### 完整示例

```go
package main

import (
    "fmt"
    "telegram-bot/pkg/validator"
)

type CreateUserRequest struct {
    UserID   int64
    Username string
    Bio      string
    Email    string
}

func (r *CreateUserRequest) Validate() error {
    chain := validator.NewChain().
        Add(validator.UserID(r.UserID)).
        Add(validator.Username(r.Username)).
        Add(validator.MaxLength(r.Bio, 200, "个人简介"))

    // 如果提供了邮箱，验证格式
    if r.Email != "" {
        chain.Add(validator.Email(r.Email))
    }

    return chain.Error()
}

func main() {
    req := &CreateUserRequest{
        UserID:   12345,
        Username: "test_user",
        Bio:      "Hello world",
        Email:    "test@example.com",
    }

    if err := req.Validate(); err != nil {
        fmt.Println("验证失败:", err)
        return
    }

    fmt.Println("验证通过")
}
```

## API 文档

### 基础验证函数

- `Required(value, fieldName string) error` - 验证必填字段
- `MinLength(value string, min int, fieldName string) error` - 验证最小长度
- `MaxLength(value string, max int, fieldName string) error` - 验证最大长度
- `LengthRange(value string, min, max int, fieldName string) error` - 验证长度范围
- `Pattern(value, pattern, fieldName string) error` - 正则表达式验证

### Telegram 业务验证

- `UserID(id int64) error` - 验证用户 ID（必须 > 0）
- `GroupID(id int64) error` - 验证群组 ID（必须 < 0）
- `Username(username string) error` - 验证用户名格式
- `CommandName(command string) error` - 验证命令名称
- `TextMessage(text string, minLen, maxLen int) error` - 验证文本消息

### 格式验证

- `Email(email string) error` - 验证邮箱格式
- `URL(url string) error` - 验证 URL 格式

### 切片验证

- `InSlice(value string, slice []string, fieldName string) error` - 验证值是否在切片中
- `NotInSlice(value string, slice []string, fieldName string) error` - 验证值是否不在切片中

### 链式验证

```go
type Chain struct { ... }

// 创建新的链式验证器
func NewChain() *Chain

// 添加验证规则
func (c *Chain) Add(err error) *Chain

// 获取验证结果
func (c *Chain) Result() *Result

// 获取第一个错误
func (c *Chain) Error() error

// 是否验证通过
func (c *Chain) IsValid() bool
```

### 验证结果

```go
type Result struct {
    Valid  bool
    Errors []error
}

// 添加错误
func (r *Result) AddError(err error)

// 获取第一个错误
func (r *Result) Error() error

// 获取所有错误
func (r *Result) AllErrors() []error
```

## 验证规则

### 用户名规则

- 长度：5-32 字符
- 字符：仅字母、数字、下划线
- 示例：`test_user`, `user123`

### 命令名称规则

- 必须以 `/` 开头
- 命令名：仅字母、数字、下划线
- 示例：`/start`, `/my_command`

### 群组 ID 规则

- Telegram 群组 ID 通常为负数
- 示例：`-100123456789`

### 用户 ID 规则

- 必须大于 0
- 示例：`12345`

## 自定义验证器

实现 `Validator` 接口：

```go
type MyValidator struct {
    value string
}

func (v *MyValidator) Validate() error {
    if v.value == "invalid" {
        return errors.Validation("INVALID_VALUE", "值不合法")
    }
    return nil
}

// 使用
v := &MyValidator{value: "test"}
if err := v.Validate(); err != nil {
    return err
}
```

## 最佳实践

1. **使用链式验证**: 对于多个验证规则，使用 `Chain` 提高代码可读性
2. **集中验证**: 在 domain 层或 request 对象中集中处理验证逻辑
3. **错误码统一**: 验证错误统一使用 `errors.Validation` 类型
4. **字段名本地化**: 根据需要提供中英文字段名
5. **可选字段**: 对于可选字段，先检查是否为空再验证格式