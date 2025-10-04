# 开发者 API 参考文档

## 📚 目录

- [概述](#概述)
- [Context API](#context-api)
- [Handler Interface](#handler-interface)
- [Router API](#router-api)
- [BaseCommand API](#basecommand-api)
- [Domain Models](#domain-models)
- [Repository Interfaces](#repository-interfaces)
- [Utility Functions](#utility-functions)
- [Type Definitions](#type-definitions)

---

## 概述

本文档提供 Telegram Bot 框架的完整 API 参考，包括核心接口、方法签名、参数说明和返回值。这是面向开发者的技术文档。

### 导入路径

```go
import (
    "telegram-bot/internal/handler"
    "telegram-bot/internal/handlers/command"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/domain/group"
)
```

### 版本信息

- **Go 版本**: 1.25+
- **框架版本**: v1.0
- **最后更新**: 2025-10-03

---

## Context API

### 类型定义

```go
type Context struct {
    // 原始对象
    Ctx     context.Context
    Bot     *bot.Bot
    Update  *models.Update
    Message *models.Message

    // 聊天信息
    ChatType  string // "private", "group", "supergroup", "channel"
    ChatID    int64
    ChatTitle string

    // 用户信息
    UserID    int64
    Username  string
    FirstName string
    LastName  string
    User      *user.User // 数据库用户对象（由中间件注入）

    // 消息内容
    Text      string
    MessageID int

    // 回复消息
    ReplyTo *ReplyInfo

    // 上下文存储（用于处理器之间传递数据）
    values map[string]interface{}
}
```

---

### 聊天类型判断

#### IsPrivate

检查是否为私聊。

```go
func (c *Context) IsPrivate() bool
```

**返回值**:
- `true`: 私聊
- `false`: 非私聊

**示例**:
```go
func (h *MyHandler) Handle(ctx *handler.Context) error {
    if ctx.IsPrivate() {
        return ctx.Reply("这是私聊消息")
    }
    return ctx.Reply("这是群组消息")
}
```

---

#### IsGroup

检查是否为群组（包括普通群组和超级群组）。

```go
func (c *Context) IsGroup() bool
```

**返回值**:
- `true`: 群组或超级群组
- `false`: 私聊或频道

**实现细节**:
```go
// 匹配 "group" 或 "supergroup"
return c.ChatType == "group" || c.ChatType == "supergroup"
```

---

#### IsChannel

检查是否为频道。

```go
func (c *Context) IsChannel() bool
```

**返回值**:
- `true`: 频道
- `false`: 非频道

---

### 消息发送

#### Reply

回复当前消息（纯文本）。

```go
func (c *Context) Reply(text string) error
```

**参数**:
- `text`: 消息内容（纯文本）

**返回值**:
- `error`: 发送失败时返回错误

**特点**:
- 自动引用原消息
- 使用 `ReplyParameters` 参数

**示例**:
```go
return ctx.Reply("收到消息: " + ctx.Text)
```

---

#### ReplyMarkdown

回复消息（Markdown 格式）。

```go
func (c *Context) ReplyMarkdown(text string) error
```

**参数**:
- `text`: Markdown 格式的消息内容

**Markdown 语法**:
```markdown
*粗体*
_斜体_
`代码`
[链接](https://example.com)
```

**示例**:
```go
return ctx.ReplyMarkdown("*加粗文本* 和 `代码块`")
```

---

#### ReplyHTML

回复消息（HTML 格式）。

```go
func (c *Context) ReplyHTML(text string) error
```

**参数**:
- `text`: HTML 格式的消息内容

**HTML 标签**:
```html
<b>粗体</b>
<i>斜体</i>
<code>代码</code>
<a href="url">链接</a>
<pre>预格式化文本</pre>
```

**示例**:
```go
return ctx.ReplyHTML("<b>重要</b>: 操作成功！")
```

---

#### Send

发送消息（不引用原消息）。

```go
func (c *Context) Send(text string) error
```

**参数**:
- `text`: 消息内容

**区别**: 不使用 `ReplyParameters`，不引用原消息。

**示例**:
```go
return ctx.Send("系统通知")
```

---

#### SendMarkdown

发送消息（Markdown 格式，不引用）。

```go
func (c *Context) SendMarkdown(text string) error
```

**参数**:
- `text`: Markdown 格式的消息内容

---

#### SendHTML

发送消息（HTML 格式，不引用）。

```go
func (c *Context) SendHTML(text string) error
```

**参数**:
- `text`: HTML 格式的消息内容

---

### 消息管理

#### DeleteMessage

删除当前消息。

```go
func (c *Context) DeleteMessage() error
```

**返回值**:
- `error`: 删除失败时返回错误

**权限要求**:
- Bot 需要有删除消息权限
- 只能删除 48 小时内的消息

**示例**:
```go
// 删除违规消息
if containsBadWord(ctx.Text) {
    return ctx.DeleteMessage()
}
```

---

### 权限检查

#### HasPermission

检查用户是否有指定权限。

```go
func (c *Context) HasPermission(required user.Permission) bool
```

**参数**:
- `required`: 所需权限等级

**返回值**:
- `true`: 权限足够
- `false`: 权限不足

**权限等级**:
```go
user.PermissionUser        // 普通用户
user.PermissionAdmin       // 管理员
user.PermissionSuperAdmin  // 超级管理员
user.PermissionOwner       // 所有者
```

**示例**:
```go
if !ctx.HasPermission(user.PermissionAdmin) {
    return ctx.Reply("❌ 需要管理员权限")
}
```

---

#### RequirePermission

要求特定权限，不满足时返回错误。

```go
func (c *Context) RequirePermission(required user.Permission) error
```

**参数**:
- `required`: 所需权限等级

**返回值**:
- `nil`: 权限足够
- `error`: 权限不足，包含详细错误信息

**错误格式**:
```
❌ 权限不足！需要权限: Admin，当前权限: User
```

**示例**:
```go
func (h *StatsHandler) Handle(ctx *handler.Context) error {
    if err := ctx.RequirePermission(user.PermissionUser); err != nil {
        return err
    }
    // 执行统计逻辑
}
```

---

### 上下文存储

#### Set

在上下文中存储键值对。

```go
func (c *Context) Set(key string, value interface{})
```

**参数**:
- `key`: 键名
- `value`: 任意类型的值

**用途**:
- 中间件向处理器传递数据
- 处理器之间共享数据

**示例**:
```go
// 中间件中设置
ctx.Set("start_time", time.Now())

// 处理器中使用
startTime, _ := ctx.Get("start_time")
```

---

#### Get

从上下文中获取值。

```go
func (c *Context) Get(key string) (interface{}, bool)
```

**参数**:
- `key`: 键名

**返回值**:
- `interface{}`: 存储的值
- `bool`: 是否存在该键

**示例**:
```go
if val, ok := ctx.Get("user_data"); ok {
    userData := val.(map[string]interface{})
    // 使用 userData
}
```

**类型断言**:
```go
// 字符串
str, ok := ctx.Get("key")
if ok {
    s := str.(string)
}

// 整数
num, ok := ctx.Get("count")
if ok {
    n := num.(int)
}
```

---

## Handler Interface

处理器接口，所有处理器都必须实现。

```go
type Handler interface {
    Match(ctx *Context) bool     // 是否匹配此消息
    Handle(ctx *Context) error   // 处理消息
    Priority() int               // 优先级（数字越小越高）
    ContinueChain() bool         // 是否继续执行后续处理器
}
```

---

### Match

判断是否应该处理此消息。

```go
Match(ctx *Context) bool
```

**参数**:
- `ctx`: 消息上下文

**返回值**:
- `true`: 匹配，将执行 Handle 方法
- `false`: 不匹配，跳过此处理器

**实现示例**:
```go
func (h *KeywordHandler) Match(ctx *handler.Context) bool {
    return strings.Contains(ctx.Text, h.keyword)
}
```

---

### Handle

处理消息的核心逻辑。

```go
Handle(ctx *Context) error
```

**参数**:
- `ctx`: 消息上下文

**返回值**:
- `error`: 处理失败时返回错误

**注意**:
- 只有 `Match()` 返回 `true` 时才会调用
- 错误会被中间件捕获并记录

**实现示例**:
```go
func (h *PingHandler) Handle(ctx *handler.Context) error {
    return ctx.Reply("Pong!")
}
```

---

### Priority

返回处理器的优先级。

```go
Priority() int
```

**返回值**:
- `int`: 优先级数值，越小越优先

**优先级范围**:
```
0-99:    系统级
100-199: 命令处理器
200-299: 关键词处理器
300-399: 正则模式处理器
400-499: 交互式处理器
900-999: 监听器（日志、分析）
```

**示例**:
```go
func (h *CommandHandler) Priority() int { return 100 }
func (h *KeywordHandler) Priority() int { return 200 }
func (h *ListenerHandler) Priority() int { return 900 }
```

---

### ContinueChain

决定是否继续执行后续处理器。

```go
ContinueChain() bool
```

**返回值**:
- `true`: 继续执行后续匹配的处理器
- `false`: 停止处理链

**使用场景**:

| 处理器类型 | 返回值 | 原因 |
|-----------|-------|------|
| 命令 | `false` | 命令已处理，无需继续 |
| 关键词 | `true` | 可能需要记录日志 |
| 监听器 | `true` | 只是观察，不影响其他处理器 |

**示例**:
```go
func (h *CommandHandler) ContinueChain() bool {
    return false  // 命令处理完就结束
}

func (h *LoggerHandler) ContinueChain() bool {
    return true   // 日志记录后继续
}
```

---

## Router API

消息路由器，负责分发消息到匹配的处理器。

### NewRouter

创建新的路由器实例。

```go
func NewRouter() *Router
```

**返回值**:
- `*Router`: 路由器实例

**示例**:
```go
router := handler.NewRouter()
```

---

### Register

注册处理器。

```go
func (r *Router) Register(h Handler)
```

**参数**:
- `h`: 实现 Handler 接口的处理器

**特点**:
- 自动按优先级排序
- 线程安全
- 可多次调用

**示例**:
```go
router.Register(command.NewPingHandler(groupRepo))
router.Register(keyword.NewGreetingHandler())
router.Register(listener.NewLoggerHandler(logger))
```

---

### Use

注册全局中间件。

```go
func (r *Router) Use(mw Middleware)
```

**参数**:
- `mw`: 中间件函数

**中间件类型**:
```go
type Middleware func(HandlerFunc) HandlerFunc
type HandlerFunc func(*Context) error
```

**执行顺序**:
- 按注册顺序执行
- 洋葱模型（后进先出）

**示例**:
```go
router.Use(middleware.Recovery())
router.Use(middleware.Logging(logger))
router.Use(middleware.Permission(userRepo))
```

---

### Route

路由消息到匹配的处理器。

```go
func (r *Router) Route(ctx *Context) error
```

**参数**:
- `ctx`: 消息上下文

**返回值**:
- `error`: 处理过程中的错误

**执行流程**:
1. 遍历所有处理器（按优先级）
2. 调用 `Match()` 检查是否匹配
3. 匹配时构建中间件链并执行 `Handle()`
4. 检查 `ContinueChain()`，决定是否继续

**示例**:
```go
// 在 main 函数中
updates := bot.GetUpdatesChan(params)
for update := range updates {
    ctx := telegram.ConvertUpdate(update, bot)
    if err := router.Route(ctx); err != nil {
        logger.Error("route error", "error", err)
    }
}
```

---

### Count

返回已注册的处理器数量。

```go
func (r *Router) Count() int
```

**返回值**:
- `int`: 处理器数量

**用途**: 调试、监控

**示例**:
```go
logger.Info("handlers registered", "count", router.Count())
```

---

### GetHandlers

获取所有已注册的处理器（用于调试）。

```go
func (r *Router) GetHandlers() []Handler
```

**返回值**:
- `[]Handler`: 处理器列表（副本）

**注意**: 返回的是副本，修改不会影响路由器。

**示例**:
```go
handlers := router.GetHandlers()
for _, h := range handlers {
    fmt.Printf("Handler priority: %d\n", h.Priority())
}
```

---

## BaseCommand API

命令处理器基类，提供命令匹配和权限检查的通用逻辑。

### NewBaseCommand

创建命令基类实例。

```go
func NewBaseCommand(
    name string,
    description string,
    permission user.Permission,
    chatTypes []string,
    groupRepo GroupRepository,
) *BaseCommand
```

**参数**:
- `name`: 命令名（不含 `/`）
- `description`: 命令描述
- `permission`: 所需权限等级
- `chatTypes`: 支持的聊天类型（`nil` 表示全部支持）
- `groupRepo`: 群组仓储（用于检查命令启用状态）

**chatTypes 可选值**:
```go
[]string{"private"}              // 仅私聊
[]string{"group", "supergroup"}  // 仅群组
[]string{"private", "group"}     // 私聊和群组
nil                              // 所有类型
```

**示例**:
```go
type PingHandler struct {
    *command.BaseCommand
}

func NewPingHandler(groupRepo command.GroupRepository) *PingHandler {
    return &PingHandler{
        BaseCommand: command.NewBaseCommand(
            "ping",                          // 命令名
            "测试 Bot 响应速度",                // 描述
            user.PermissionUser,             // 普通用户可用
            []string{"private", "group"},    // 私聊和群组
            groupRepo,
        ),
    }
}
```

---

### Match

判断是否匹配此命令（已由 BaseCommand 实现）。

```go
func (c *BaseCommand) Match(ctx *handler.Context) bool
```

**匹配逻辑**:
1. 检查是否为文本消息
2. 检查是否以 `/` 开头
3. 解析命令名（支持 `@botname` 后缀）
4. 检查聊天类型是否支持
5. 检查群组是否启用该命令

**无需重写**: 子类通常不需要重写此方法。

---

### Priority

返回命令优先级（固定为 100）。

```go
func (c *BaseCommand) Priority() int
```

**返回值**: `100`

---

### ContinueChain

命令处理后停止链（固定返回 false）。

```go
func (c *BaseCommand) ContinueChain() bool
```

**返回值**: `false`

---

### CheckPermission

检查权限（便捷方法）。

```go
func (c *BaseCommand) CheckPermission(ctx *handler.Context) error
```

**参数**:
- `ctx`: 消息上下文

**返回值**:
- `error`: 权限不足时返回错误

**等价于**:
```go
return ctx.RequirePermission(c.permission)
```

**示例**:
```go
func (h *StatsHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }
    // 业务逻辑
}
```

---

### GetName

获取命令名。

```go
func (c *BaseCommand) GetName() string
```

---

### GetDescription

获取命令描述。

```go
func (c *BaseCommand) GetDescription() string
```

---

### GetPermission

获取所需权限。

```go
func (c *BaseCommand) GetPermission() user.Permission
```

---

## Domain Models

### User

用户聚合根。

#### 结构体

```go
type User struct {
    ID          int64
    Username    string
    FirstName   string
    LastName    string
    Permissions map[int64]Permission // groupID -> Permission
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### NewUser

创建新用户。

```go
func NewUser(id int64, username, firstName, lastName string) *User
```

**参数**:
- `id`: Telegram 用户 ID
- `username`: 用户名
- `firstName`: 名字
- `lastName`: 姓氏

**示例**:
```go
user := user.NewUser(123456, "john_doe", "John", "Doe")
```

---

#### GetPermission

获取用户在特定群组的权限。

```go
func (u *User) GetPermission(groupID int64) Permission
```

**参数**:
- `groupID`: 群组 ID（私聊时使用用户 ID）

**返回值**:
- `Permission`: 权限等级（未设置时默认 `PermissionUser`）

**示例**:
```go
perm := user.GetPermission(ctx.ChatID)
if perm >= user.PermissionAdmin {
    // 管理员操作
}
```

---

#### SetPermission

设置用户在特定群组的权限。

```go
func (u *User) SetPermission(groupID int64, perm Permission)
```

**参数**:
- `groupID`: 群组 ID
- `perm`: 权限等级

**副作用**: 更新 `UpdatedAt` 字段

**示例**:
```go
user.SetPermission(groupID, user.PermissionAdmin)
userRepo.Update(user)
```

---

#### HasPermission

检查用户是否有足够权限。

```go
func (u *User) HasPermission(groupID int64, required Permission) bool
```

**参数**:
- `groupID`: 群组 ID
- `required`: 所需权限等级

**返回值**:
- `bool`: `>=` 比较结果

---

#### IsAdmin

检查是否为管理员（Admin 及以上）。

```go
func (u *User) IsAdmin(groupID int64) bool
```

---

#### IsSuperAdmin

检查是否为超级管理员（SuperAdmin 及以上）。

```go
func (u *User) IsSuperAdmin(groupID int64) bool
```

---

### Group

群组聚合根。

#### 结构体

```go
type Group struct {
    ID        int64
    Title     string
    Type      string                    // "group", "supergroup", "channel"
    Commands  map[string]*CommandConfig // commandName -> config
    Settings  map[string]interface{}    // 自定义配置
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

#### NewGroup

创建新群组。

```go
func NewGroup(id int64, title, groupType string) *Group
```

**参数**:
- `id`: 群组 ID
- `title`: 群组标题
- `groupType`: 群组类型（`"group"`, `"supergroup"`, `"channel"`）

---

#### IsCommandEnabled

检查命令是否启用。

```go
func (g *Group) IsCommandEnabled(commandName string) bool
```

**参数**:
- `commandName`: 命令名（不含 `/`）

**返回值**:
- `bool`: 默认为 `true`

---

#### EnableCommand

启用命令。

```go
func (g *Group) EnableCommand(commandName string, userID int64)
```

**参数**:
- `commandName`: 命令名
- `userID`: 操作者用户 ID

---

#### DisableCommand

禁用命令。

```go
func (g *Group) DisableCommand(commandName string, userID int64)
```

**参数**:
- `commandName`: 命令名
- `userID`: 操作者用户 ID

---

#### GetCommandConfig

获取命令配置。

```go
func (g *Group) GetCommandConfig(commandName string) *CommandConfig
```

**返回值**: 未配置时返回默认启用的配置

---

#### SetSetting / GetSetting

设置/获取自定义配置项。

```go
func (g *Group) SetSetting(key string, value interface{})
func (g *Group) GetSetting(key string) (interface{}, bool)
```

**用途**: 存储群组自定义设置（如语言、欢迎消息等）

**示例**:
```go
group.SetSetting("language", "zh-CN")
group.SetSetting("welcome_message", "欢迎加入！")

if lang, ok := group.GetSetting("language"); ok {
    locale := lang.(string)
}
```

---

## Repository Interfaces

### User Repository

```go
type Repository interface {
    FindByID(id int64) (*User, error)
    FindByUsername(username string) (*User, error)
    Save(user *User) error
    Update(user *User) error
    Delete(id int64) error
    FindAdminsByGroup(groupID int64) ([]*User, error)
}
```

**FindByID**: 根据用户 ID 查询

**FindByUsername**: 根据用户名查询

**Save**: 保存新用户

**Update**: 更新现有用户

**Delete**: 删除用户

**FindAdminsByGroup**: 查询群组的所有管理员

---

### Group Repository

```go
type Repository interface {
    FindByID(id int64) (*Group, error)
    Save(group *Group) error
    Update(group *Group) error
    Delete(id int64) error
    FindAll() ([]*Group, error)
}
```

**FindByID**: 根据群组 ID 查询

**Save**: 保存新群组

**Update**: 更新现有群组

**Delete**: 删除群组

**FindAll**: 查询所有群组

---

## Utility Functions

### ParseArgs

解析命令参数。

```go
func ParseArgs(text string) []string
```

**参数**:
- `text`: 完整消息文本

**返回值**:
- `[]string`: 参数列表（不含命令名）

**示例**:
```go
// 输入: "/stats user @alice"
args := command.ParseArgs(ctx.Text)
// 输出: ["user", "@alice"]

if len(args) < 1 {
    return ctx.Reply("用法: /stats [user @username]")
}
action := args[0]
```

---

### parseCommandName

解析命令名（内部函数）。

```go
func parseCommandName(text string) string
```

**功能**:
- 提取命令名
- 移除 `/` 前缀
- 移除 `@botname` 后缀

**示例**:
```
"/ping"            -> "ping"
"/ping@mybot"      -> "ping"
"/stats arg1 arg2" -> "stats"
```

---

## Type Definitions

### Permission

权限等级枚举。

```go
type Permission int

const (
    PermissionNone       Permission = 0  // 无权限
    PermissionUser       Permission = 1  // 普通用户
    PermissionAdmin      Permission = 2  // 管理员
    PermissionSuperAdmin Permission = 3  // 超级管理员
    PermissionOwner      Permission = 4  // 所有者
)
```

**方法**:

#### String

```go
func (p Permission) String() string
```

**返回值**: `"None"`, `"User"`, `"Admin"`, `"SuperAdmin"`, `"Owner"`

#### CanManage

```go
func (p Permission) CanManage(target Permission) bool
```

**功能**: 检查是否可以管理目标权限（必须高于目标）

**示例**:
```go
admin := user.PermissionAdmin
superAdmin := user.PermissionSuperAdmin

superAdmin.CanManage(admin)  // true
admin.CanManage(superAdmin)  // false
```

---

### ReplyInfo

回复消息信息。

```go
type ReplyInfo struct {
    MessageID int
    UserID    int64
    Username  string
    Text      string
}
```

**用途**: 存储被回复消息的信息

**访问方式**:
```go
if ctx.ReplyTo != nil {
    repliedUserID := ctx.ReplyTo.UserID
    repliedText := ctx.ReplyTo.Text
}
```

---

### Middleware

中间件类型定义。

```go
type Middleware func(HandlerFunc) HandlerFunc
type HandlerFunc func(*Context) error
```

**示例实现**:
```go
func LoggingMiddleware(logger Logger) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx *handler.Context) error {
            logger.Info("message_received", "user_id", ctx.UserID)
            err := next(ctx)  // 调用下一个中间件/处理器
            if err != nil {
                logger.Error("handler_error", "error", err)
            }
            return err
        }
    }
}
```

---

## 快速查找

### 按功能分类

#### 消息发送
- `ctx.Reply()` - 回复纯文本
- `ctx.ReplyMarkdown()` - 回复 Markdown
- `ctx.ReplyHTML()` - 回复 HTML
- `ctx.Send()` - 发送纯文本（不引用）
- `ctx.SendMarkdown()` - 发送 Markdown（不引用）
- `ctx.SendHTML()` - 发送 HTML（不引用）

#### 权限管理
- `ctx.HasPermission()` - 检查权限
- `ctx.RequirePermission()` - 要求权限
- `user.GetPermission()` - 获取用户权限
- `user.SetPermission()` - 设置用户权限
- `user.IsAdmin()` - 是否为管理员
- `user.IsSuperAdmin()` - 是否为超级管理员

#### 聊天类型
- `ctx.IsPrivate()` - 是否私聊
- `ctx.IsGroup()` - 是否群组
- `ctx.IsChannel()` - 是否频道

#### 路由管理
- `router.Register()` - 注册处理器
- `router.Use()` - 注册中间件
- `router.Route()` - 路由消息

#### 数据存储
- `ctx.Set()` - 存储上下文数据
- `ctx.Get()` - 获取上下文数据
- `group.SetSetting()` - 设置群组配置
- `group.GetSetting()` - 获取群组配置

---

## 常见模式

### 1. 创建命令处理器

```go
type MyCommandHandler struct {
    *command.BaseCommand
}

func NewMyCommandHandler(groupRepo command.GroupRepository) *MyCommandHandler {
    return &MyCommandHandler{
        BaseCommand: command.NewBaseCommand(
            "mycommand",
            "命令描述",
            user.PermissionUser,
            nil,
            groupRepo,
        ),
    }
}

func (h *MyCommandHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }
    return ctx.Reply("执行成功")
}
```

---

### 2. 创建关键词处理器

```go
type KeywordHandler struct {
    keywords []string
}

func (h *KeywordHandler) Match(ctx *handler.Context) bool {
    for _, kw := range h.keywords {
        if strings.Contains(strings.ToLower(ctx.Text), kw) {
            return true
        }
    }
    return false
}

func (h *KeywordHandler) Handle(ctx *handler.Context) error {
    return ctx.Reply("检测到关键词")
}

func (h *KeywordHandler) Priority() int { return 200 }
func (h *KeywordHandler) ContinueChain() bool { return true }
```

---

### 3. 创建中间件

```go
func MyMiddleware() handler.Middleware {
    return func(next handler.HandlerFunc) handler.HandlerFunc {
        return func(ctx *handler.Context) error {
            // 前置处理
            startTime := time.Now()

            // 调用下一个中间件/处理器
            err := next(ctx)

            // 后置处理
            duration := time.Since(startTime)
            logger.Info("request_duration", "duration", duration)

            return err
        }
    }
}
```

---

### 4. 使用仓储

```go
// 查询用户
user, err := userRepo.FindByID(ctx.UserID)
if err != nil {
    if errors.Is(err, user.ErrUserNotFound) {
        // 用户不存在，创建新用户
        user = user.NewUser(ctx.UserID, ctx.Username, ctx.FirstName, ctx.LastName)
        if err := userRepo.Save(user); err != nil {
            return err
        }
    } else {
        return err
    }
}

// 修改权限
user.SetPermission(ctx.ChatID, user.PermissionAdmin)
return userRepo.Update(user)
```

---

## 附录

### 错误处理最佳实践

```go
func (h *MyHandler) Handle(ctx *handler.Context) error {
    // 1. 参数验证
    args := command.ParseArgs(ctx.Text)
    if len(args) < 1 {
        return ctx.Reply("❌ 参数不足")
    }

    // 2. 权限检查
    if err := ctx.RequirePermission(user.PermissionAdmin); err != nil {
        return err  // 返回权限错误消息
    }

    // 3. 业务逻辑
    result, err := doSomething(args[0])
    if err != nil {
        logger.Error("business error", "error", err)
        return ctx.Reply("❌ 操作失败")
    }

    // 4. 返回结果
    return ctx.Reply("✅ " + result)
}
```

---

### 测试辅助函数

```go
// 创建测试上下文
func createTestContext(text string, userID int64) *handler.Context {
    return &handler.Context{
        Text:      text,
        UserID:    userID,
        ChatID:    123,
        ChatType:  "private",
        MessageID: 1,
        User:      user.NewUser(userID, "testuser", "Test", "User"),
    }
}

// 测试处理器
func TestMyHandler(t *testing.T) {
    h := NewMyHandler()
    ctx := createTestContext("/mycommand arg1", 123)

    if !h.Match(ctx) {
        t.Fatal("should match")
    }

    err := h.Handle(ctx)
    assert.NoError(t, err)
}
```

---

## 相关文档

- [项目快速入门](./getting-started.md)
- [命令处理器开发指南](./handlers/command-handler-guide.md)
- [中间件开发指南](./middleware-guide.md)
- [Repository 开发指南](./repository-guide.md)
- [部署运维指南](./deployment.md)
- [架构总览](../CLAUDE.md)

---

**最后更新**: 2025-10-03
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team
