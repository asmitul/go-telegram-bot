# 命令处理器开发指南

## 📚 目录

- [概述](#概述)
- [核心概念](#核心概念)
- [快速开始](#快速开始)
- [完整代码示例](#完整代码示例)
- [BaseCommand 基类详解](#basecommand-基类详解)
- [权限系统](#权限系统)
- [参数解析](#参数解析)
- [注册流程](#注册流程)
- [测试方法](#测试方法)
- [实际场景示例](#实际场景示例)
- [常见问题](#常见问题)

---

## 概述

**命令处理器** (Command Handler) 是本机器人框架的核心处理器类型，用于处理以 `/` 开头的明确指令。

### 适用场景

- ✅ 明确的功能指令（如 `/ping`、`/help`、`/stats`）
- ✅ 需要权限控制的操作（如管理员命令）
- ✅ 支持参数的命令（如 `/ban @user`、`/set limit 100`）
- ✅ 需要群组级别启用/禁用控制的功能

### 不适用场景

- ❌ 自然语言输入 → 使用 **正则匹配处理器** (Pattern Handler)
- ❌ 简单的关键词响应 → 使用 **关键词处理器** (Keyword Handler)
- ❌ 需要监控所有消息 → 使用 **监听器** (Listener)

---

## 核心概念

### 处理器接口

所有命令处理器必须实现 `handler.Handler` 接口：

```go
type Handler interface {
    Match(ctx *Context) bool      // 判断是否匹配
    Handle(ctx *Context) error    // 处理消息
    Priority() int                // 优先级（100-199）
    ContinueChain() bool          // 是否继续执行后续处理器
}
```

### BaseCommand 基类

框架提供 `BaseCommand` 基类，自动处理：
- ✅ 命令名匹配（`/command`、`/command@botname`）
- ✅ 聊天类型过滤（private、group、supergroup、channel）
- ✅ 群组命令启用/禁用检查
- ✅ 参数解析工具函数

### 优先级规则

- **优先级范围**：`100-199`
- **数值越小，优先级越高**（越早执行）
- **标准优先级**：`100`（BaseCommand 默认）
- **特殊情况**：
  - `90-99`：系统级命令（如紧急停机）
  - `100-149`：普通命令
  - `150-199`：低优先级命令（如帮助、关于）

### 执行链控制

- `ContinueChain() = false`：命令执行后停止（**推荐**，BaseCommand 默认）
- `ContinueChain() = true`：允许后续处理器继续处理（罕见场景）

---

## 快速开始

### 步骤 1：创建处理器文件

在 `internal/handlers/command/` 目录下创建新文件，例如 `version.go`：

```bash
touch internal/handlers/command/version.go
```

### 步骤 2：编写处理器代码

```go
package command

import (
    "fmt"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
)

type VersionHandler struct {
    *BaseCommand
}

func NewVersionHandler(groupRepo GroupRepository) *VersionHandler {
    return &VersionHandler{
        BaseCommand: NewBaseCommand(
            "version",                              // 命令名
            "查看机器人版本",                        // 描述
            user.PermissionUser,                    // 所需权限
            []string{"private", "group", "supergroup"}, // 支持的聊天类型
            groupRepo,                              // 群组仓储
        ),
    }
}

func (h *VersionHandler) Handle(ctx *handler.Context) error {
    // 权限检查
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    return ctx.Reply("🤖 Bot Version: v2.0.0")
}
```

### 步骤 3：注册处理器

在 `cmd/bot/main.go` 的 `registerHandlers()` 函数中添加：

```go
// 1. 命令处理器（优先级 100）
router.Register(command.NewPingHandler(groupRepo))
router.Register(command.NewVersionHandler(groupRepo))  // 新增
```

### 步骤 4：测试

向机器人发送 `/version`，验证功能。

---

## 完整代码示例

### 示例 1：简单命令（无参数）

```go
package command

import (
    "fmt"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
    "time"
)

type UptimeHandler struct {
    *BaseCommand
    startTime time.Time
}

func NewUptimeHandler(groupRepo GroupRepository) *UptimeHandler {
    return &UptimeHandler{
        BaseCommand: NewBaseCommand(
            "uptime",
            "查看机器人运行时长",
            user.PermissionUser,
            []string{"private", "group", "supergroup"},
            groupRepo,
        ),
        startTime: time.Now(),
    }
}

func (h *UptimeHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    uptime := time.Since(h.startTime)
    response := fmt.Sprintf(
        "⏱️ *运行时长*\n\n"+
            "已运行: %s\n"+
            "启动时间: %s",
        uptime.Round(time.Second),
        h.startTime.Format("2006-01-02 15:04:05"),
    )

    return ctx.ReplyMarkdown(response)
}
```

### 示例 2：带参数命令

```go
package command

import (
    "fmt"
    "strconv"
    "strings"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
)

type SetLimitHandler struct {
    *BaseCommand
}

func NewSetLimitHandler(groupRepo GroupRepository) *SetLimitHandler {
    return &SetLimitHandler{
        BaseCommand: NewBaseCommand(
            "setlimit",
            "设置消息频率限制",
            user.PermissionAdmin, // 需要管理员权限
            []string{"group", "supergroup"}, // 仅群组
            groupRepo,
        ),
    }
}

func (h *SetLimitHandler) Handle(ctx *handler.Context) error {
    // 权限检查
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    // 解析参数
    args := ParseArgs(ctx.Text)
    if len(args) < 1 {
        return ctx.Reply("❌ 用法: /setlimit <数量>\n例如: /setlimit 10")
    }

    limit, err := strconv.Atoi(args[0])
    if err != nil || limit <= 0 {
        return ctx.Reply("❌ 请输入有效的数字（大于0）")
    }

    // TODO: 保存到数据库
    // groupConfig.MessageLimit = limit
    // groupRepo.Update(groupConfig)

    return ctx.Reply(fmt.Sprintf("✅ 消息频率限制已设置为: %d条/分钟", limit))
}
```

### 示例 3：高级命令（多参数 + 验证）

```go
package command

import (
    "fmt"
    "regexp"
    "strings"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
)

type BanHandler struct {
    *BaseCommand
    userRepo UserRepository
}

func NewBanHandler(groupRepo GroupRepository, userRepo UserRepository) *BanHandler {
    return &BanHandler{
        BaseCommand: NewBaseCommand(
            "ban",
            "封禁用户",
            user.PermissionAdmin,
            []string{"group", "supergroup"},
            groupRepo,
        ),
        userRepo: userRepo,
    }
}

func (h *BanHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    args := ParseArgs(ctx.Text)
    if len(args) < 1 {
        return ctx.Reply(
            "❌ 用法: /ban <用户ID或@用户名> [原因]\n\n" +
                "示例:\n" +
                "  /ban 123456789\n" +
                "  /ban @username 违规发言\n" +
                "  /ban 123456789 spam",
        )
    }

    // 解析用户标识
    userIdentifier := args[0]
    var targetUserID int64
    var err error

    if strings.HasPrefix(userIdentifier, "@") {
        // 通过用户名查找
        username := strings.TrimPrefix(userIdentifier, "@")
        // TODO: 实现用户名查找逻辑
        return ctx.Reply(fmt.Sprintf("⚠️ 用户名查找功能待实现: %s", username))
    } else {
        // 解析用户 ID
        targetUserID, err = parseUserID(userIdentifier)
        if err != nil {
            return ctx.Reply("❌ 无效的用户ID格式")
        }
    }

    // 获取原因
    reason := "违反群规"
    if len(args) > 1 {
        reason = strings.Join(args[1:], " ")
    }

    // 禁止封禁管理员
    targetUser, err := h.userRepo.FindByID(targetUserID)
    if err == nil && targetUser.HasPermission(ctx.ChatID, user.PermissionAdmin) {
        return ctx.Reply("❌ 无法封禁管理员")
    }

    // TODO: 执行封禁逻辑
    // botAPI.BanChatMember(ctx.ChatID, targetUserID)

    response := fmt.Sprintf(
        "🚫 *用户已封禁*\n\n"+
            "用户ID: `%d`\n"+
            "原因: %s\n"+
            "操作者: %s",
        targetUserID,
        reason,
        ctx.FirstName,
    )

    return ctx.ReplyMarkdown(response)
}

func parseUserID(s string) (int64, error) {
    // 移除可能的 "user:" 前缀
    s = strings.TrimPrefix(s, "user:")

    var id int64
    _, err := fmt.Sscanf(s, "%d", &id)
    return id, err
}
```

---

## BaseCommand 基类详解

### 构造函数参数

```go
func NewBaseCommand(
    name string,              // 命令名（不含 /）
    description string,       // 命令描述（用于帮助信息）
    permission user.Permission, // 所需权限
    chatTypes []string,       // 支持的聊天类型
    groupRepo GroupRepository, // 群组仓储（用于检查命令启用状态）
) *BaseCommand
```

### 支持的聊天类型

| 类型 | 说明 | 示例 |
|------|------|------|
| `private` | 私聊 | 与机器人的一对一聊天 |
| `group` | 普通群组 | 早期的 Telegram 群组 |
| `supergroup` | 超级群组 | 支持更多功能的群组 |
| `channel` | 频道 | 单向广播频道 |

**默认值**：如果传入空数组 `[]`，则支持所有类型。

### 自动功能

1. **命令名匹配**
   - `/ping` ✅
   - `/ping@botname` ✅（多机器人场景）
   - `/ping arg1 arg2` ✅（带参数）
   - `/Ping` ❌（区分大小写）

2. **聊天类型过滤**
   ```go
   chatTypes: []string{"private"} // 仅私聊可用
   ```

3. **群组命令启用检查**
   - 自动检查群组配置中是否启用该命令
   - 如果 `groupRepo` 为 `nil`，跳过检查

4. **优先级和链控制**
   - 默认优先级：`100`
   - 默认不继续执行链：`ContinueChain() = false`

### 可用方法

```go
// Getter 方法
GetName() string                    // 获取命令名
GetDescription() string             // 获取命令描述
GetPermission() user.Permission     // 获取所需权限

// 权限检查
CheckPermission(ctx *handler.Context) error // 检查用户权限
```

---

## 权限系统

### 四级权限

框架内置四级权限系统（定义在 `internal/domain/user/user.go`）：

| 权限级别 | 常量 | 说明 | 典型命令 |
|---------|------|------|---------|
| 普通用户 | `PermissionUser` | 默认权限 | `/ping`, `/help` |
| 管理员 | `PermissionAdmin` | 群组管理员 | `/stats`, `/ban` |
| 超级管理员 | `PermissionSuperAdmin` | 可配置命令 | `/enable`, `/disable` |
| 所有者 | `PermissionOwner` | 最高权限 | `/shutdown`, `/setadmin` |

### 权限检查方式

#### 方式 1：使用 BaseCommand（推荐）

```go
func (h *MyHandler) Handle(ctx *handler.Context) error {
    // 一行代码检查权限
    if err := h.CheckPermission(ctx); err != nil {
        return err // 自动返回权限不足错误
    }

    // 业务逻辑
    return ctx.Reply("执行成功")
}
```

#### 方式 2：使用 Context 方法

```go
func (h *MyHandler) Handle(ctx *handler.Context) error {
    // 方式 2a：RequirePermission（返回错误）
    if err := ctx.RequirePermission(user.PermissionAdmin); err != nil {
        return err
    }

    // 方式 2b：HasPermission（返回布尔值）
    if !ctx.HasPermission(user.PermissionAdmin) {
        return ctx.Reply("❌ 需要管理员权限")
    }

    return ctx.Reply("执行成功")
}
```

#### 方式 3：自定义权限逻辑

```go
func (h *MyHandler) Handle(ctx *handler.Context) error {
    // 检查用户是否为群主
    if ctx.User.Permissions[ctx.ChatID] != user.PermissionOwner {
        return ctx.Reply("❌ 仅群主可使用此命令")
    }

    // 或者检查多个权限级别
    perm := ctx.User.Permissions[ctx.ChatID]
    if perm != user.PermissionAdmin && perm != user.PermissionOwner {
        return ctx.Reply("❌ 需要管理员或更高权限")
    }

    return ctx.Reply("执行成功")
}
```

### 按群组分配权限

**重要概念**：用户权限是**按群组**分配的。

```go
// User 结构
type User struct {
    ID          int64
    Username    string
    Permissions map[int64]Permission // 群组ID -> 权限级别
}

// 示例：用户在不同群组有不同权限
user := &User{
    ID: 123456789,
    Permissions: map[int64]Permission{
        -1001234567890: PermissionAdmin,      // 群组A：管理员
        -1009876543210: PermissionUser,       // 群组B：普通用户
        -1005555555555: PermissionOwner,      // 群组C：所有者
    },
}
```

### 权限不足时的响应

```go
// Context.RequirePermission 自动返回的错误消息
return errors.New("权限不足")

// 自定义错误消息
if !ctx.HasPermission(user.PermissionAdmin) {
    return ctx.Reply("⚠️ 此命令需要管理员权限\n\n请联系群主获取权限")
}
```

---

## 参数解析

### ParseArgs 工具函数

```go
// 定义在 internal/handlers/command/base.go
func ParseArgs(text string) []string
```

**示例**：

```go
text := "/ban @username 违规发言"
args := ParseArgs(text)
// args = ["@username", "违规发言"]

text := "/setlimit 100"
args := ParseArgs(text)
// args = ["100"]

text := "/help"
args := ParseArgs(text)
// args = []
```

### 常见解析模式

#### 1. 固定参数数量

```go
func (h *Handler) Handle(ctx *handler.Context) error {
    args := ParseArgs(ctx.Text)

    if len(args) != 2 {
        return ctx.Reply("❌ 用法: /command <arg1> <arg2>")
    }

    arg1 := args[0]
    arg2 := args[1]

    // 处理逻辑
    return nil
}
```

#### 2. 可选参数

```go
func (h *Handler) Handle(ctx *handler.Context) error {
    args := ParseArgs(ctx.Text)

    // 必需参数
    if len(args) < 1 {
        return ctx.Reply("❌ 用法: /kick <用户ID> [原因]")
    }

    userID := args[0]

    // 可选参数
    reason := "违反群规"
    if len(args) > 1 {
        reason = strings.Join(args[1:], " ")
    }

    return ctx.Reply(fmt.Sprintf("踢出用户 %s，原因：%s", userID, reason))
}
```

#### 3. 参数类型转换

```go
func (h *Handler) Handle(ctx *handler.Context) error {
    args := ParseArgs(ctx.Text)

    if len(args) < 1 {
        return ctx.Reply("❌ 用法: /setlimit <数量>")
    }

    // 字符串 -> 整数
    limit, err := strconv.Atoi(args[0])
    if err != nil {
        return ctx.Reply("❌ 请输入有效的数字")
    }

    // 字符串 -> 浮点数
    amount, err := strconv.ParseFloat(args[0], 64)
    if err != nil {
        return ctx.Reply("❌ 请输入有效的金额")
    }

    // 字符串 -> 布尔值
    enabled, err := strconv.ParseBool(args[0]) // "true", "false", "1", "0"
    if err != nil {
        return ctx.Reply("❌ 请输入 true 或 false")
    }

    return nil
}
```

#### 4. 验证参数格式

```go
func (h *Handler) Handle(ctx *handler.Context) error {
    args := ParseArgs(ctx.Text)

    if len(args) < 1 {
        return ctx.Reply("❌ 用法: /setemail <邮箱>")
    }

    email := args[0]

    // 邮箱格式验证
    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    if !emailRegex.MatchString(email) {
        return ctx.Reply("❌ 邮箱格式无效")
    }

    return ctx.Reply(fmt.Sprintf("✅ 邮箱已设置为: %s", email))
}
```

#### 5. 获取完整文本（不分割）

```go
func (h *Handler) Handle(ctx *handler.Context) error {
    // 移除命令部分，保留原始文本（含空格）
    text := strings.TrimSpace(strings.TrimPrefix(ctx.Text, "/announce"))

    if text == "" {
        return ctx.Reply("❌ 用法: /announce <公告内容>")
    }

    // text 保留了所有空格和换行符
    return ctx.Reply(fmt.Sprintf("📢 公告:\n%s", text))
}
```

---

## 注册流程

### 1. 基本注册

在 `cmd/bot/main.go` 的 `registerHandlers()` 函数中：

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
    router.Register(command.NewVersionHandler(groupRepo)) // 新增

    // 更新日志统计
    appLogger.Info("Registered handlers breakdown",
        "commands", 4, // 更新数量
        "keywords", 1,
        "patterns", 1,
        "listeners", 2,
    )
}
```

### 2. 带依赖注入

```go
// 命令需要额外的服务或配置
type MyHandler struct {
    *BaseCommand
    emailService EmailService
    config       *Config
}

func NewMyHandler(groupRepo GroupRepository, emailService EmailService, config *Config) *MyHandler {
    return &MyHandler{
        BaseCommand:  NewBaseCommand("mycommand", "描述", user.PermissionUser, nil, groupRepo),
        emailService: emailService,
        config:       config,
    }
}

// 注册时传入依赖
router.Register(command.NewMyHandler(groupRepo, emailService, config))
```

### 3. 自定义优先级

```go
type UrgentHandler struct {
    *BaseCommand
}

func (h *UrgentHandler) Priority() int {
    return 90 // 覆盖默认的 100，更高优先级
}
```

---

## 测试方法

### 1. 单元测试

创建 `internal/handlers/command/version_test.go`：

```go
package command

import (
    "testing"
    "telegram-bot/internal/domain/group"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock GroupRepository
type MockGroupRepo struct {
    mock.Mock
}

func (m *MockGroupRepo) FindByID(id int64) (*group.Group, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*group.Group), args.Error(1)
}

func TestVersionHandler_Match(t *testing.T) {
    mockRepo := new(MockGroupRepo)
    h := NewVersionHandler(mockRepo)

    tests := []struct {
        name     string
        text     string
        chatType string
        want     bool
    }{
        {"匹配-version", "/version", "private", true},
        {"匹配-带@", "/version@botname", "group", true},
        {"匹配-带参数", "/version arg", "supergroup", true},
        {"不匹配-其他命令", "/help", "private", false},
        {"不匹配-不是命令", "version", "private", false},
        {"不匹配-频道", "/version", "channel", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := &handler.Context{
                Text:     tt.text,
                ChatType: tt.chatType,
                ChatID:   -1001234567890,
            }

            // Mock 群组查询
            if tt.chatType == "group" || tt.chatType == "supergroup" {
                mockRepo.On("FindByID", ctx.ChatID).Return(&group.Group{
                    ID:       ctx.ChatID,
                    Commands: map[string]*group.CommandConfig{},
                }, nil).Once()
            }

            got := h.Match(ctx)
            assert.Equal(t, tt.want, got)
        })
    }
}

func TestVersionHandler_Priority(t *testing.T) {
    mockRepo := new(MockGroupRepo)
    h := NewVersionHandler(mockRepo)
    assert.Equal(t, 100, h.Priority())
}

func TestVersionHandler_ContinueChain(t *testing.T) {
    mockRepo := new(MockGroupRepo)
    h := NewVersionHandler(mockRepo)
    assert.False(t, h.ContinueChain())
}
```

运行测试：

```bash
go test ./internal/handlers/command/... -v
```

### 2. 集成测试

测试完整的命令执行流程：

```go
func TestVersionHandler_Integration(t *testing.T) {
    // 初始化真实的数据库和依赖
    // ...

    // 创建处理器
    h := NewVersionHandler(groupRepo)

    // 模拟用户上下文
    ctx := &handler.Context{
        Text:     "/version",
        ChatType: "private",
        UserID:   123456789,
        User: &user.User{
            ID: 123456789,
            Permissions: map[int64]user.Permission{
                0: user.PermissionUser, // 私聊的权限
            },
        },
    }

    // 执行命令
    err := h.Handle(ctx)
    assert.NoError(t, err)

    // 验证响应（需要 mock Bot API）
}
```

### 3. 手动测试

1. 启动机器人：
   ```bash
   make run
   ```

2. 在 Telegram 中测试：
   - `/version` - 基本功能
   - `/version@botname` - 多机器人场景
   - `/version arg1 arg2` - 带参数
   - 在不同聊天类型中测试（私聊、群组）
   - 使用不同权限的账号测试

3. 检查日志输出：
   ```
   INFO  message_logged chat_type=private user_id=123456789 text=/version
   INFO  command_executed command=version user_id=123456789 duration=5ms
   ```

---

## 实际场景示例

### 场景 1：用户信息查询

```go
package command

import (
    "fmt"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
)

type WhoamiHandler struct {
    *BaseCommand
}

func NewWhoamiHandler(groupRepo GroupRepository) *WhoamiHandler {
    return &WhoamiHandler{
        BaseCommand: NewBaseCommand(
            "whoami",
            "查看自己的用户信息",
            user.PermissionUser,
            []string{"private", "group", "supergroup"},
            groupRepo,
        ),
    }
}

func (h *WhoamiHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    // 获取用户权限
    permission := ctx.User.Permissions[ctx.ChatID]
    permissionName := getPermissionName(permission)

    response := fmt.Sprintf(
        "👤 *用户信息*\n\n"+
            "🆔 ID: `%d`\n"+
            "👤 用户名: @%s\n"+
            "📝 昵称: %s %s\n"+
            "🔒 权限: %s\n"+
            "🌐 语言: %s",
        ctx.UserID,
        ctx.Username,
        ctx.FirstName,
        ctx.LastName,
        permissionName,
        ctx.LanguageCode,
    )

    return ctx.ReplyMarkdown(response)
}

func getPermissionName(perm user.Permission) string {
    switch perm {
    case user.PermissionOwner:
        return "所有者 👑"
    case user.PermissionSuperAdmin:
        return "超级管理员 ⭐"
    case user.PermissionAdmin:
        return "管理员 🔧"
    default:
        return "普通用户 ✅"
    }
}
```

### 场景 2：群组配置管理

```go
package command

import (
    "fmt"
    "strconv"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
)

type EnableCommandHandler struct {
    *BaseCommand
    groupRepo GroupRepository
}

func NewEnableCommandHandler(groupRepo GroupRepository) *EnableCommandHandler {
    return &EnableCommandHandler{
        BaseCommand: NewBaseCommand(
            "enable",
            "启用指定命令",
            user.PermissionSuperAdmin,
            []string{"group", "supergroup"},
            groupRepo,
        ),
        groupRepo: groupRepo,
    }
}

func (h *EnableCommandHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    args := ParseArgs(ctx.Text)
    if len(args) < 1 {
        return ctx.Reply("❌ 用法: /enable <命令名>\n例如: /enable ping")
    }

    commandName := args[0]

    // 获取群组配置
    g, err := h.groupRepo.FindByID(ctx.ChatID)
    if err != nil {
        return fmt.Errorf("获取群组配置失败: %w", err)
    }

    // 启用命令
    g.EnableCommand(commandName, ctx.UserID)

    // 保存
    if err := h.groupRepo.Update(g); err != nil {
        return fmt.Errorf("保存配置失败: %w", err)
    }

    return ctx.Reply(fmt.Sprintf("✅ 命令 /%s 已启用", commandName))
}
```

### 场景 3：批量操作

```go
package command

import (
    "fmt"
    "strings"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
)

type CleanupHandler struct {
    *BaseCommand
}

func NewCleanupHandler(groupRepo GroupRepository) *CleanupHandler {
    return &CleanupHandler{
        BaseCommand: NewBaseCommand(
            "cleanup",
            "清理群组数据",
            user.PermissionOwner,
            []string{"group", "supergroup"},
            groupRepo,
        ),
    }
}

func (h *CleanupHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    args := ParseArgs(ctx.Text)
    if len(args) < 1 {
        return ctx.Reply(
            "❌ 用法: /cleanup <类型>\n\n" +
                "可用类型:\n" +
                "  • warnings - 清除所有警告记录\n" +
                "  • messages - 清除消息统计\n" +
                "  • all - 清除所有数据",
        )
    }

    cleanupType := strings.ToLower(args[0])

    var deletedCount int

    switch cleanupType {
    case "warnings":
        // TODO: 清除警告记录
        deletedCount = 0
    case "messages":
        // TODO: 清除消息统计
        deletedCount = 0
    case "all":
        // TODO: 清除所有数据
        deletedCount = 0
    default:
        return ctx.Reply("❌ 未知的清理类型，请使用: warnings, messages, all")
    }

    return ctx.Reply(fmt.Sprintf("✅ 已清理 %d 条 %s 数据", deletedCount, cleanupType))
}
```

### 场景 4：异步操作

```go
package command

import (
    "fmt"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
    "time"
)

type BackupHandler struct {
    *BaseCommand
}

func NewBackupHandler(groupRepo GroupRepository) *BackupHandler {
    return &BackupHandler{
        BaseCommand: NewBaseCommand(
            "backup",
            "备份群组数据",
            user.PermissionOwner,
            []string{"group", "supergroup"},
            groupRepo,
        ),
    }
}

func (h *BackupHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    // 立即回复用户
    ctx.Reply("⏳ 正在生成备份，请稍候...")

    // 异步执行备份
    go func() {
        time.Sleep(3 * time.Second) // 模拟耗时操作

        // TODO: 实际备份逻辑
        // backupFile := generateBackup(ctx.ChatID)

        // 完成后发送消息
        message := fmt.Sprintf(
            "✅ 备份完成！\n\n" +
                "📦 文件: backup_%d_%s.json\n" +
                "📊 大小: 1.2 MB\n" +
                "⏰ 时间: %s",
            ctx.ChatID,
            time.Now().Format("20060102"),
            time.Now().Format("2006-01-02 15:04:05"),
        )

        ctx.Send(ctx.ChatID, message)

        // TODO: 发送备份文件
        // ctx.SendDocument(ctx.ChatID, backupFile)
    }()

    return nil
}
```

---

## 常见问题

### Q1：命令处理器和正则处理器的区别？

| 特性 | 命令处理器 | 正则处理器 |
|------|-----------|-----------|
| **触发格式** | `/command` | 正则表达式 |
| **权限系统** | 内置 BaseCommand | 需手动实现 |
| **参数解析** | `ParseArgs()` 工具 | 正则捕获组 |
| **群组启用/禁用** | 内置支持 | 需手动实现 |
| **优先级** | 100-199 | 300-399 |
| **适用场景** | 明确的指令 | 自然语言输入 |

### Q2：如何处理命令冲突？

1. **不同命令名**：自动区分，无冲突
2. **相同命令名**：后注册的会被忽略（Router 自动去重）
3. **建议**：使用唯一的命令名

### Q3：如何让命令仅在私聊或仅在群组可用？

```go
// 仅私聊
chatTypes: []string{"private"}

// 仅群组（不含私聊）
chatTypes: []string{"group", "supergroup"}

// 所有类型
chatTypes: []string{"private", "group", "supergroup", "channel"}
// 或
chatTypes: nil // 传 nil 也支持所有类型
```

### Q4：如何获取命令的原始文本（含参数）？

```go
func (h *Handler) Handle(ctx *handler.Context) error {
    // 方式1：完整文本
    fullText := ctx.Text
    // 例如："/ban @user spam" -> "/ban @user spam"

    // 方式2：移除命令部分
    argsText := strings.TrimPrefix(ctx.Text, "/ban")
    argsText = strings.TrimSpace(argsText)
    // 例如："/ban @user spam" -> "@user spam"

    // 方式3：使用 ParseArgs
    args := ParseArgs(ctx.Text)
    // 例如："/ban @user spam" -> ["@user", "spam"]

    return nil
}
```

### Q5：如何处理多语言命令？

```go
type HelpHandler struct {
    *BaseCommand
}

func (h *HelpHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    var response string

    switch ctx.LanguageCode {
    case "zh", "zh-CN", "zh-TW":
        response = "📖 帮助信息\n\n可用命令：..."
    case "en":
        response = "📖 Help\n\nAvailable commands:..."
    default:
        response = "📖 Help / 帮助\n\n..."
    }

    return ctx.Reply(response)
}
```

### Q6：BaseCommand 的权限检查是自动的吗？

**不是自动的**，需要在 `Handle()` 中显式调用：

```go
// ❌ 错误：忘记检查权限
func (h *Handler) Handle(ctx *handler.Context) error {
    // 直接执行业务逻辑
    return ctx.Reply("执行成功")
}

// ✅ 正确：显式调用 CheckPermission
func (h *Handler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }
    return ctx.Reply("执行成功")
}
```

**原因**：保持灵活性，允许在权限检查前执行其他逻辑（如参数验证）。

### Q7：如何调试命令不匹配的问题？

```go
func (h *MyHandler) Match(ctx *handler.Context) bool {
    matched := h.BaseCommand.Match(ctx)

    // 临时添加调试日志
    log.Printf("Command '%s' match result: %v (text: '%s', chatType: '%s')",
        h.GetName(), matched, ctx.Text, ctx.ChatType)

    return matched
}
```

检查以下几点：
- ✅ 命令名是否正确（区分大小写）
- ✅ 聊天类型是否支持
- ✅ 群组是否启用了该命令（如果是群组）
- ✅ 消息是否以 `/` 开头

---

## 附录

### 相关资源

- [BaseCommand 源码](../../internal/handlers/command/base.go)
- [示例命令](../../internal/handlers/command/ping.go)
- [权限系统文档](../../internal/domain/user/user.go)

### 相关文档

- [正则匹配处理器开发指南](./pattern-handler-guide.md)
- [关键词处理器开发指南](./keyword-handler-guide.md)（待创建）
- [监听器开发指南](./listener-handler-guide.md)（待创建）
- [架构总览](../../CLAUDE.md)

---

**编写日期**: 2025-10-02
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team
