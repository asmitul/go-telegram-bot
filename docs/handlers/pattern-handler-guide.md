# 正则匹配处理器开发指南

## 📚 目录

- [概述](#概述)
- [核心概念](#核心概念)
- [快速开始](#快速开始)
- [完整代码示例](#完整代码示例)
- [正则表达式最佳实践](#正则表达式最佳实践)
- [注册流程](#注册流程)
- [测试方法](#测试方法)
- [实际场景示例](#实际场景示例)
- [常见问题](#常见问题)

---

## 概述

**正则匹配处理器** (Pattern Handler) 是本机器人框架的四大处理器类型之一，用于处理复杂的文本模式匹配场景。

### 适用场景

- ✅ 需要提取消息中的特定信息（如城市名、订单号、金额等）
- ✅ 支持多种表达方式的同一意图（如"天气 北京"、"北京天气"、"查天气 北京"）
- ✅ 需要验证输入格式（如电话号码、邮箱、身份证等）
- ✅ 复杂的关键词组合匹配

### 不适用场景

- ❌ 简单的精确匹配 → 使用 **命令处理器** (`/command`)
- ❌ 简单的关键词包含 → 使用 **关键词处理器** (Keyword Handler)
- ❌ 需要处理所有消息 → 使用 **监听器** (Listener)

---

## 核心概念

### 处理器接口

所有正则匹配处理器必须实现 `handler.Handler` 接口：

```go
type Handler interface {
    Match(ctx *Context) bool      // 判断是否匹配
    Handle(ctx *Context) error    // 处理消息
    Priority() int                // 优先级（300-399）
    ContinueChain() bool          // 是否继续执行后续处理器
}
```

### 优先级规则

- **优先级范围**：`300-399`
- **数值越小，优先级越高**（越早执行）
- **标准优先级**：`300`（推荐）
- **特殊情况**：
  - `301-310`：高优先级正则（如安全过滤、敏感词检测）
  - `390-399`：低优先级正则（如兜底匹配）

### 执行链控制

- `ContinueChain() = true`：继续执行后续处理器（用于监控、日志）
- `ContinueChain() = false`：停止执行后续处理器（推荐，避免误触发）

---

## 快速开始

### 步骤 1：创建处理器文件

在 `internal/handlers/pattern/` 目录下创建新文件，例如 `balance.go`：

```bash
touch internal/handlers/pattern/balance.go
```

### 步骤 2：编写处理器代码

参考以下模板：

```go
package pattern

import (
    "regexp"
    "telegram-bot/internal/handler"
)

type BalanceHandler struct {
    pattern   *regexp.Regexp
    chatTypes []string
}

func NewBalanceHandler() *BalanceHandler {
    return &BalanceHandler{
        pattern:   regexp.MustCompile(`(?i)余额`),
        chatTypes: []string{"private", "group", "supergroup"},
    }
}

func (h *BalanceHandler) Match(ctx *handler.Context) bool {
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }
    return h.pattern.MatchString(ctx.Text)
}

func (h *BalanceHandler) Handle(ctx *handler.Context) error {
    return ctx.Reply("查询余额功能")
}

func (h *BalanceHandler) Priority() int {
    return 300
}

func (h *BalanceHandler) ContinueChain() bool {
    return false
}

func (h *BalanceHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

### 步骤 3：注册处理器

在 `cmd/bot/main.go` 的 `registerHandlers()` 函数中添加：

```go
// 3. 正则处理器（优先级 300）
router.Register(pattern.NewWeatherHandler())
router.Register(pattern.NewBalanceHandler())  // 新增
```

### 步骤 4：测试

向机器人发送消息 `余额` 或 `查询余额`，验证功能。

---

## 完整代码示例

### 示例 1：余额查询（带信息提取）

```go
package pattern

import (
    "fmt"
    "regexp"
    "telegram-bot/internal/handler"
)

type BalanceHandler struct {
    pattern   *regexp.Regexp
    chatTypes []string
}

func NewBalanceHandler() *BalanceHandler {
    return &BalanceHandler{
        // 支持：查询余额、余额查询、我的余额、balance
        pattern:   regexp.MustCompile(`(?i)(查询|查|我的)?余额|balance`),
        chatTypes: []string{"private"}, // 仅私聊
    }
}

func (h *BalanceHandler) Match(ctx *handler.Context) bool {
    // 1. 检查聊天类型
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }

    // 2. 检查正则匹配
    return h.pattern.MatchString(ctx.Text)
}

func (h *BalanceHandler) Handle(ctx *handler.Context) error {
    // TODO: 从数据库或外部服务查询真实余额
    // userID := ctx.UserID
    // balance, err := balanceService.GetBalance(userID)

    response := fmt.Sprintf(
        "💰 *余额查询*\n\n"+
            "👤 用户: %s\n"+
            "🆔 ID: `%d`\n"+
            "💵 可用余额: ¥1,234.56\n"+
            "🔒 冻结余额: ¥0.00\n"+
            "📅 更新时间: 2025-10-02 14:30:00\n\n"+
            "💡 _提示：这是示例数据_",
        ctx.FirstName,
        ctx.UserID,
    )

    return ctx.ReplyMarkdown(response)
}

func (h *BalanceHandler) Priority() int {
    return 300
}

func (h *BalanceHandler) ContinueChain() bool {
    return false // 匹配后停止，避免触发其他处理器
}

func (h *BalanceHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

### 示例 2：订单查询（捕获组提取）

```go
package pattern

import (
    "fmt"
    "regexp"
    "telegram-bot/internal/handler"
)

type OrderHandler struct {
    pattern   *regexp.Regexp
    chatTypes []string
}

func NewOrderHandler() *OrderHandler {
    return &OrderHandler{
        // 捕获订单号：订单 20250101123456 或 查询订单 20250101123456
        pattern:   regexp.MustCompile(`(?i)(查询|查)?订单\s*([A-Z0-9]{10,20})`),
        chatTypes: []string{"private", "group", "supergroup"},
    }
}

func (h *OrderHandler) Match(ctx *handler.Context) bool {
    if !h.isSupportedChatType(ctx.ChatType) {
        return false
    }
    return h.pattern.MatchString(ctx.Text)
}

func (h *OrderHandler) Handle(ctx *handler.Context) error {
    // 提取订单号（捕获组索引 2）
    matches := h.pattern.FindStringSubmatch(ctx.Text)
    if len(matches) < 3 {
        return ctx.Reply("❌ 订单号格式错误")
    }

    orderID := matches[2]

    // TODO: 从数据库查询订单
    // order, err := orderService.GetOrder(orderID)

    response := fmt.Sprintf(
        "📦 *订单详情*\n\n"+
            "🆔 订单号: `%s`\n"+
            "📊 状态: 已发货\n"+
            "📅 下单时间: 2025-09-30 10:20:30\n"+
            "🚚 物流单号: SF1234567890\n\n"+
            "💡 _点击订单号可复制_",
        orderID,
    )

    return ctx.ReplyMarkdown(response)
}

func (h *OrderHandler) Priority() int {
    return 305 // 稍高优先级
}

func (h *OrderHandler) ContinueChain() bool {
    return false
}

func (h *OrderHandler) isSupportedChatType(chatType string) bool {
    for _, t := range h.chatTypes {
        if t == chatType {
            return true
        }
    }
    return false
}
```

---

## 正则表达式最佳实践

### 常用修饰符

| 修饰符 | 说明 | 示例 |
|--------|------|------|
| `(?i)` | 不区分大小写 | `(?i)hello` 匹配 "Hello", "HELLO" |
| `(?m)` | 多行模式 | `^` 和 `$` 匹配每行的开始/结束 |
| `(?s)` | 单行模式 | `.` 匹配包括换行符在内的所有字符 |

### 捕获组

```go
// 基本捕获组
pattern := regexp.MustCompile(`订单\s+(\d+)`)
matches := pattern.FindStringSubmatch("订单 12345")
// matches[0] = "订单 12345"
// matches[1] = "12345"

// 命名捕获组（Go 1.15+）
pattern := regexp.MustCompile(`订单\s+(?P<orderid>\d+)`)

// 非捕获组（只匹配不捕获）
pattern := regexp.MustCompile(`(?:查询|查)?订单\s+(\d+)`)
```

### 常见模式

```go
// 1. 城市/地点提取
regexp.MustCompile(`(?i)天气\s+(.+)`)

// 2. 金额提取
regexp.MustCompile(`(?i)充值\s+(\d+(?:\.\d{1,2})?)`)

// 3. 手机号验证
regexp.MustCompile(`^1[3-9]\d{9}$`)

// 4. 邮箱验证
regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// 5. 多关键词组合
regexp.MustCompile(`(?i)(查询|查|我的)?(余额|balance|钱包)`)

// 6. 可选前缀 + 必选主体
regexp.MustCompile(`(?i)(?:请|帮我)?查询订单\s+([A-Z0-9]+)`)
```

### 性能优化

```go
// ✅ 推荐：预编译正则表达式（在构造函数中）
func NewHandler() *Handler {
    return &Handler{
        pattern: regexp.MustCompile(`pattern`), // 编译一次
    }
}

// ❌ 避免：每次匹配都编译
func (h *Handler) Match(ctx *handler.Context) bool {
    pattern := regexp.MustCompile(`pattern`) // 重复编译，性能差
    return pattern.MatchString(ctx.Text)
}
```

### 安全建议

```go
// ⚠️ 避免过于宽泛的模式（可能匹配意外内容）
regexp.MustCompile(`.*余额.*`) // 太宽泛

// ✅ 推荐：明确的边界
regexp.MustCompile(`(?i)^(查询|查)?余额$`)

// ✅ 限制捕获长度（防止恶意输入）
regexp.MustCompile(`订单\s+([A-Z0-9]{10,20})`) // 限制 10-20 个字符
```

---

## 注册流程

### 1. 在 `cmd/bot/main.go` 中注册

找到 `registerHandlers()` 函数（约 242 行）：

```go
func registerHandlers(
    router *handler.Router,
    groupRepo *mongodb.GroupRepository,
    userRepo *mongodb.UserRepository,
    appLogger logger.Logger,
) {
    // 1. 命令处理器（优先级 100）
    router.Register(command.NewPingHandler(groupRepo))
    router.Register(command.NewHelpHandler(groupRepo, router))
    router.Register(command.NewStatsHandler(groupRepo, userRepo))

    // 2. 关键词处理器（优先级 200）
    router.Register(keyword.NewGreetingHandler())

    // 3. 正则处理器（优先级 300）
    router.Register(pattern.NewWeatherHandler())
    router.Register(pattern.NewBalanceHandler())    // 新增
    router.Register(pattern.NewOrderHandler())      // 新增

    // 4. 监听器（优先级 900+）
    router.Register(listener.NewMessageLoggerHandler(appLogger))
    router.Register(listener.NewAnalyticsHandler())

    // 更新日志统计
    appLogger.Info("Registered handlers breakdown",
        "commands", 3,
        "keywords", 1,
        "patterns", 3,  // 更新数量
        "listeners", 2,
    )
}
```

### 2. 如果需要依赖注入

```go
// 示例：传入数据库仓储
func NewBalanceHandler(userRepo UserRepository) *BalanceHandler {
    return &BalanceHandler{
        pattern:  regexp.MustCompile(`(?i)余额`),
        userRepo: userRepo,
    }
}

// 注册时传入依赖
router.Register(pattern.NewBalanceHandler(userRepo))
```

---

## 测试方法

### 1. 单元测试

创建 `internal/handlers/pattern/balance_test.go`：

```go
package pattern

import (
    "testing"
    "telegram-bot/internal/handler"
    "github.com/stretchr/testify/assert"
)

func TestBalanceHandler_Match(t *testing.T) {
    h := NewBalanceHandler()

    tests := []struct {
        name     string
        text     string
        chatType string
        want     bool
    }{
        {"匹配-查询余额", "查询余额", "private", true},
        {"匹配-余额", "余额", "private", true},
        {"匹配-Balance", "Balance", "private", true},
        {"不匹配-其他文本", "你好", "private", false},
        {"不匹配-群组", "余额", "group", false}, // 仅私聊
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := &handler.Context{
                Text:     tt.text,
                ChatType: tt.chatType,
            }
            got := h.Match(ctx)
            assert.Equal(t, tt.want, got)
        })
    }
}

func TestBalanceHandler_Priority(t *testing.T) {
    h := NewBalanceHandler()
    assert.Equal(t, 300, h.Priority())
}
```

运行测试：

```bash
go test ./internal/handlers/pattern/... -v
```

### 2. 集成测试

在实际 Telegram 环境中测试：

1. 启动机器人：`make run`
2. 向机器人发送测试消息：
   - `余额`
   - `查询余额`
   - `我的余额`
   - `balance`
3. 检查日志输出和返回结果

### 3. 正则表达式在线测试

推荐工具：
- [Regex101](https://regex101.com/) - 支持 Go 语法，实时测试
- [RegExr](https://regexr.com/) - 可视化匹配结果

---

## 实际场景示例

### 场景 1：电话号码提取

```go
type PhoneHandler struct {
    pattern   *regexp.Regexp
    chatTypes []string
}

func NewPhoneHandler() *PhoneHandler {
    return &PhoneHandler{
        // 中国手机号：1开头，第二位3-9，共11位
        pattern:   regexp.MustCompile(`1[3-9]\d{9}`),
        chatTypes: []string{"private"},
    }
}

func (h *PhoneHandler) Handle(ctx *handler.Context) error {
    phone := h.pattern.FindString(ctx.Text)
    return ctx.Reply(fmt.Sprintf("检测到手机号：%s", phone))
}
```

### 场景 2：金额充值

```go
type RechargeHandler struct {
    pattern   *regexp.Regexp
    chatTypes []string
}

func NewRechargeHandler() *RechargeHandler {
    return &RechargeHandler{
        // 匹配：充值 100、充值100元、recharge 50.5
        pattern: regexp.MustCompile(`(?i)(充值|recharge)\s*(\d+(?:\.\d{1,2})?)`),
        chatTypes: []string{"private"},
    }
}

func (h *RechargeHandler) Handle(ctx *handler.Context) error {
    matches := h.pattern.FindStringSubmatch(ctx.Text)
    if len(matches) < 3 {
        return ctx.Reply("❌ 金额格式错误")
    }

    amount := matches[2]

    response := fmt.Sprintf(
        "💳 *充值确认*\n\n"+
            "💰 金额: ¥%s\n"+
            "👤 用户: %s\n\n"+
            "请确认后点击支付按钮",
        amount,
        ctx.FirstName,
    )

    return ctx.ReplyMarkdown(response)
}
```

### 场景 3：多语言支持

```go
type HelpPatternHandler struct {
    pattern   *regexp.Regexp
    chatTypes []string
}

func NewHelpPatternHandler() *HelpPatternHandler {
    return &HelpPatternHandler{
        // 支持中英文：帮助、help、?、？
        pattern: regexp.MustCompile(`^(?i)(帮助|help|\?|？)$`),
        chatTypes: []string{"private", "group", "supergroup"},
    }
}

func (h *HelpPatternHandler) Handle(ctx *handler.Context) error {
    // 根据消息内容判断语言（匹配中文则用中文回复）
    if strings.Contains(ctx.Text, "帮助") || strings.Contains(ctx.Text, "？") {
        return ctx.Reply("请输入 /help 查看帮助")
    }
    return ctx.Reply("Type /help to see available commands")
}
```

### 场景 4：URL 提取与验证

```go
type LinkHandler struct {
    pattern   *regexp.Regexp
    chatTypes []string
}

func NewLinkHandler() *LinkHandler {
    return &LinkHandler{
        pattern: regexp.MustCompile(`https?://[^\s]+`),
        chatTypes: []string{"group", "supergroup"},
    }
}

func (h *LinkHandler) Handle(ctx *handler.Context) error {
    // 提取所有链接
    links := h.pattern.FindAllString(ctx.Text, -1)

    // TODO: 验证链接安全性
    // for _, link := range links {
    //     if isMalicious(link) {
    //         return ctx.Reply("⚠️ 检测到恶意链接！")
    //     }
    // }

    return nil // 继续处理
}

func (h *LinkHandler) ContinueChain() bool {
    return true // 允许其他处理器继续处理
}
```

---

## 常见问题

### Q1：正则处理器和命令处理器有什么区别？

| 特性 | 命令处理器 | 正则处理器 |
|------|-----------|-----------|
| **触发方式** | `/command` 开头 | 正则表达式匹配 |
| **权限系统** | 内置 `BaseCommand` | 需手动实现 |
| **优先级** | 100-199 | 300-399 |
| **信息提取** | 参数解析 | 捕获组提取 |
| **适用场景** | 明确的功能指令 | 自然语言输入 |

### Q2：如何避免多个正则处理器互相冲突？

1. **设置合理的优先级**：重要的处理器使用更低的数字
2. **精确的正则表达式**：避免过于宽泛的模式
3. **使用 `ContinueChain() = false`**：匹配后停止执行链

### Q3：正则性能会影响机器人吗？

- ✅ **预编译正则表达式**（在构造函数中）性能很好
- ✅ Go 的 `regexp` 包采用 RE2 引擎，性能稳定
- ⚠️ 避免过于复杂的正则（如深层嵌套、大量回溯）
- ⚠️ 限制捕获组数量和输入长度

### Q4：如何处理中英文混合输入？

```go
// 使用 Unicode 字符类
regexp.MustCompile(`(?i)(查询|query)\s+([\p{Han}a-zA-Z]+)`)

// \p{Han} 匹配中文字符
// [\p{Han}a-zA-Z]+ 同时匹配中文和英文
```

### Q5：如何调试正则不匹配的问题？

```go
func (h *Handler) Match(ctx *handler.Context) bool {
    matched := h.pattern.MatchString(ctx.Text)

    // 临时添加调试日志
    if !matched {
        log.Printf("Pattern '%s' did not match text: '%s'",
            h.pattern.String(), ctx.Text)
    }

    return matched
}
```

### Q6：可以在一个处理器中使用多个正则表达式吗？

可以，示例：

```go
type MultiPatternHandler struct {
    balancePattern *regexp.Regexp
    orderPattern   *regexp.Regexp
    chatTypes      []string
}

func (h *MultiPatternHandler) Match(ctx *handler.Context) bool {
    return h.balancePattern.MatchString(ctx.Text) ||
           h.orderPattern.MatchString(ctx.Text)
}

func (h *MultiPatternHandler) Handle(ctx *handler.Context) error {
    if h.balancePattern.MatchString(ctx.Text) {
        return h.handleBalance(ctx)
    }
    if h.orderPattern.MatchString(ctx.Text) {
        return h.handleOrder(ctx)
    }
    return nil
}
```

---

## 附录

### 参考资源

- [Go 正则表达式官方文档](https://pkg.go.dev/regexp)
- [RE2 语法参考](https://github.com/google/re2/wiki/Syntax)
- [Regex101 在线测试](https://regex101.com/)
- 项目中的示例：`internal/handlers/pattern/weather.go`

### 相关文档

- [命令处理器开发指南](./command-handler-guide.md)（待创建）
- [关键词处理器开发指南](./keyword-handler-guide.md)（待创建）
- [架构总览文档](../../CLAUDE.md)

---

**最后更新**: 2025-10-03
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team
