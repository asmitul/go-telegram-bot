# 待处理功能：群组配置管理命令

## 📋 功能概述

**当前状态**: 群组配置系统已完整实现，但缺少管理命令

**目标**: 添加完整的群组配置管理命令，让管理员可以通过命令管理群组设置

---

## ✅ 已实现的基础设施

### 群组实体核心 (`internal/domain/group/group.go`)

**Group 结构** (group.go:21-29):
```go
type Group struct {
    ID        int64
    Title     string
    Type      string                    // "group", "supergroup", "channel"
    Commands  map[string]*CommandConfig // 命令启用/禁用配置
    Settings  map[string]interface{}    // 通用配置存储（key-value）
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 已实现的命令配置功能

**CommandConfig 结构** (group.go:12-18):
```go
type CommandConfig struct {
    CommandName string
    Enabled     bool      // 命令是否启用
    UpdatedAt   time.Time
    UpdatedBy   int64     // 谁修改的
}
```

**命令管理方法** (group.go:45-98):
- ✅ `IsCommandEnabled(commandName)` - 检查命令是否启用（默认 true）
- ✅ `EnableCommand(commandName, userID)` - 启用命令
- ✅ `DisableCommand(commandName, userID)` - 禁用命令
- ✅ `GetCommandConfig(commandName)` - 获取命令配置

### 已实现的通用配置功能

**通用配置方法** (group.go:100-110):
- ✅ `SetSetting(key, value)` - 设置任意配置项
- ✅ `GetSetting(key)` - 获取配置项

### Group Repository

**Repository 接口** (group.go:112-119):
```go
type Repository interface {
    FindByID(id int64) (*Group, error)
    Save(group *Group) error
    Update(group *Group) error          // ✅ 用于保存配置
    Delete(id int64) error
    FindAll() ([]*Group, error)
}
```

**MongoDB 实现** (`internal/adapter/repository/mongodb/group_repository.go`):
- ✅ 完整的 CRUD 操作
- ✅ 索引优化（title, type, updated_at）
- ✅ 领域对象 ↔ 文档转换

### BaseCommand 自动检查

**命令自动启用检查** (internal/handlers/command/base.go:76-82):
```go
// BaseCommand.Match() 方法中会自动检查命令是否被禁用
if ctx.IsGroup() && c.groupRepo != nil {
    g, err := c.groupRepo.FindByID(ctx.ChatID)
    if err == nil && !g.IsCommandEnabled(c.name) {
        return false  // 命令被禁用，不执行
    }
}
```

---

## ❌ 缺失的功能

目前**没有任何命令**可以管理群组配置，所有配置修改只能通过直接操作数据库。

**文档中有示例但未实现**：
- 在 `docs/handlers/command-handler-guide.md:945-989` 有 `/enable` 命令的完整示例代码
- 但实际的 `internal/handlers/command/` 目录中**没有实现**

需要实现以下命令：

### 命令配置管理
1. `/enable <命令名>` - 启用命令
2. `/disable <命令名>` - 禁用命令
3. `/cmdlist` - 查看所有命令的启用状态

### 通用配置管理
4. `/config set <key> <value>` - 设置配置项
5. `/config get <key>` - 获取配置项
6. `/config list` - 列出所有配置
7. `/config del <key>` - 删除配置项

---

## 📋 详细实现方案

### 第一部分：命令启用/禁用管理

---

### 1. `/enable` - 启用命令

**文件**: `internal/handlers/command/enable.go`

**功能**: 启用某个被禁用的命令

**权限要求**: SuperAdmin

**用法**:
```
/enable ping          # 启用 ping 命令
/enable stats         # 启用 stats 命令
```

**实现代码**:
```go
package command

import (
    "strings"
    "telegram-bot/internal/domain/group"
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
    // 1. 检查权限
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    // 2. 解析参数
    args := ParseArgs(ctx.Text)
    if len(args) == 0 {
        return ctx.Reply("❌ 用法: /enable <命令名>")
    }

    commandName := strings.TrimPrefix(args[0], "/")

    // 3. 加载群组
    g, err := h.groupRepo.FindByID(ctx.ChatID)
    if err != nil {
        // 群组不存在，创建新群组
        g = group.NewGroup(ctx.ChatID, ctx.ChatTitle, ctx.ChatType)
        if err := h.groupRepo.Save(g); err != nil {
            return ctx.Reply("❌ 保存失败")
        }
    }

    // 4. 启用命令
    g.EnableCommand(commandName, ctx.UserID)

    // 5. 保存
    if err := h.groupRepo.Update(g); err != nil {
        return ctx.Reply("❌ 保存失败")
    }

    return ctx.Reply("✅ 命令 /" + commandName + " 已启用")
}
```

---

### 2. `/disable` - 禁用命令

**文件**: `internal/handlers/command/disable.go`

**功能**: 禁用某个命令（防止普通用户使用）

**权限要求**: SuperAdmin

**用法**:
```
/disable ping         # 禁用 ping 命令
```

**实现要点**:
```go
type DisableCommandHandler struct {
    *BaseCommand
    groupRepo GroupRepository
}

func (h *DisableCommandHandler) Handle(ctx *handler.Context) error {
    // 解析参数
    args := ParseArgs(ctx.Text)
    if len(args) == 0 {
        return ctx.Reply("❌ 用法: /disable <命令名>")
    }

    commandName := strings.TrimPrefix(args[0], "/")

    // 保护措施：不能禁用核心命令
    protectedCommands := []string{"enable", "disable", "help"}
    for _, protected := range protectedCommands {
        if commandName == protected {
            return ctx.Reply("❌ 不能禁用该命令")
        }
    }

    // 加载群组并禁用命令
    g, err := h.loadOrCreateGroup(ctx)
    if err != nil {
        return ctx.Reply("❌ 加载群组失败")
    }

    g.DisableCommand(commandName, ctx.UserID)

    if err := h.groupRepo.Update(g); err != nil {
        return ctx.Reply("❌ 保存失败")
    }

    return ctx.Reply("✅ 命令 /" + commandName + " 已禁用")
}
```

**重要保护**:
- 不能禁用 `/enable` 和 `/disable` 命令本身（避免死锁）
- 不能禁用 `/help` 命令（保证用户能查看帮助）

---

### 3. `/cmdlist` - 查看命令状态

**文件**: `internal/handlers/command/cmdlist.go`

**功能**: 显示所有命令的启用/禁用状态

**权限要求**: Admin

**用法**:
```
/cmdlist
```

**输出示例**:
```
📋 群组命令配置状态:

✅ 已启用 (3):
  • /ping - 测试机器人响应
  • /help - 显示帮助信息
  • /stats - 查看统计信息

❌ 已禁用 (1):
  • /admin - 管理员命令

💡 使用 /enable <命令名> 启用命令
💡 使用 /disable <命令名> 禁用命令
```

**实现要点**:
```go
type CmdListHandler struct {
    *BaseCommand
    groupRepo GroupRepository
    router    *handler.Router  // 用于获取所有注册的命令
}

func (h *CmdListHandler) Handle(ctx *handler.Context) error {
    // 1. 加载群组配置
    g, err := h.groupRepo.FindByID(ctx.ChatID)
    if err != nil {
        // 群组不存在，说明所有命令都是默认启用
        g = group.NewGroup(ctx.ChatID, ctx.ChatTitle, ctx.ChatType)
    }

    // 2. 获取所有注册的命令
    allCommands := h.getAllCommands()

    // 3. 分类
    enabled := []string{}
    disabled := []string{}

    for cmdName, cmdDesc := range allCommands {
        if g.IsCommandEnabled(cmdName) {
            enabled = append(enabled, fmt.Sprintf("  • /%s - %s", cmdName, cmdDesc))
        } else {
            disabled = append(disabled, fmt.Sprintf("  • /%s - %s", cmdName, cmdDesc))
        }
    }

    // 4. 构建输出
    var sb strings.Builder
    sb.WriteString("📋 群组命令配置状态:\n\n")

    if len(enabled) > 0 {
        sb.WriteString(fmt.Sprintf("✅ 已启用 (%d):\n", len(enabled)))
        sb.WriteString(strings.Join(enabled, "\n"))
        sb.WriteString("\n\n")
    }

    if len(disabled) > 0 {
        sb.WriteString(fmt.Sprintf("❌ 已禁用 (%d):\n", len(disabled)))
        sb.WriteString(strings.Join(disabled, "\n"))
        sb.WriteString("\n\n")
    }

    sb.WriteString("💡 使用 /enable <命令名> 启用命令\n")
    sb.WriteString("💡 使用 /disable <命令名> 禁用命令")

    return ctx.Reply(sb.String())
}

// 通过 router 获取所有命令
func (h *CmdListHandler) getAllCommands() map[string]string {
    commands := make(map[string]string)
    handlers := h.router.GetHandlers()

    for _, hdlr := range handlers {
        // 尝试获取命令信息（需要 BaseCommand 提供接口）
        if cmd, ok := hdlr.(interface {
            GetName() string
            GetDescription() string
        }); ok {
            commands[cmd.GetName()] = cmd.GetDescription()
        }
    }

    return commands
}
```

---

### 第二部分：通用配置管理

---

### 4. `/config` - 通用配置管理命令

**文件**: `internal/handlers/command/config.go`

**功能**: 管理群组的通用配置（key-value 存储）

**权限要求**: SuperAdmin

**子命令**:
```
/config set <key> <value>    # 设置配置
/config get <key>            # 获取配置
/config list                 # 列出所有配置
/config del <key>            # 删除配置
```

**实现代码框架**:
```go
package command

import (
    "fmt"
    "strings"
    "time"
    "telegram-bot/internal/domain/group"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
)

type ConfigHandler struct {
    *BaseCommand
    groupRepo GroupRepository
}

func NewConfigHandler(groupRepo GroupRepository) *ConfigHandler {
    return &ConfigHandler{
        BaseCommand: NewBaseCommand(
            "config",
            "管理群组配置",
            user.PermissionSuperAdmin,
            []string{"group", "supergroup"},
            groupRepo,
        ),
        groupRepo: groupRepo,
    }
}

func (h *ConfigHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    args := ParseArgs(ctx.Text)
    if len(args) == 0 {
        return h.showUsage(ctx)
    }

    subCmd := strings.ToLower(args[0])

    switch subCmd {
    case "set":
        return h.handleSet(ctx, args[1:])
    case "get":
        return h.handleGet(ctx, args[1:])
    case "list":
        return h.handleList(ctx)
    case "del", "delete":
        return h.handleDelete(ctx, args[1:])
    default:
        return h.showUsage(ctx)
    }
}

func (h *ConfigHandler) showUsage(ctx *handler.Context) error {
    usage := `📖 /config 命令用法:

/config set <key> <value>  - 设置配置
/config get <key>          - 获取配置
/config list               - 列出所有配置
/config del <key>          - 删除配置

示例:
/config set welcome_msg 欢迎加入本群！
/config get welcome_msg
/config del welcome_msg`

    return ctx.Reply(usage)
}

func (h *ConfigHandler) handleSet(ctx *handler.Context, args []string) error {
    if len(args) < 2 {
        return ctx.Reply("❌ 用法: /config set <key> <value>")
    }

    key := args[0]
    value := strings.Join(args[1:], " ")

    g, err := h.loadOrCreateGroup(ctx)
    if err != nil {
        return ctx.Reply("❌ 加载群组失败")
    }

    g.SetSetting(key, value)

    if err := h.groupRepo.Update(g); err != nil {
        return ctx.Reply("❌ 保存失败")
    }

    return ctx.Reply(fmt.Sprintf("✅ 配置已设置:\n%s = %s", key, value))
}

func (h *ConfigHandler) handleGet(ctx *handler.Context, args []string) error {
    if len(args) == 0 {
        return ctx.Reply("❌ 用法: /config get <key>")
    }

    key := args[0]

    g, err := h.groupRepo.FindByID(ctx.ChatID)
    if err != nil {
        return ctx.Reply("❌ 群组配置不存在")
    }

    value, ok := g.GetSetting(key)
    if !ok {
        return ctx.Reply(fmt.Sprintf("❌ 配置 '%s' 不存在", key))
    }

    return ctx.Reply(fmt.Sprintf("📋 %s = %v", key, value))
}

func (h *ConfigHandler) handleList(ctx *handler.Context) error {
    g, err := h.groupRepo.FindByID(ctx.ChatID)
    if err != nil {
        return ctx.Reply("📋 当前没有任何配置")
    }

    if len(g.Settings) == 0 {
        return ctx.Reply("📋 当前没有任何配置")
    }

    var sb strings.Builder
    sb.WriteString("📋 群组配置列表:\n\n")

    for key, value := range g.Settings {
        sb.WriteString(fmt.Sprintf("• %s = %v\n", key, value))
    }

    sb.WriteString(fmt.Sprintf("\n总计: %d 项配置", len(g.Settings)))

    return ctx.Reply(sb.String())
}

func (h *ConfigHandler) handleDelete(ctx *handler.Context, args []string) error {
    if len(args) == 0 {
        return ctx.Reply("❌ 用法: /config del <key>")
    }

    key := args[0]

    g, err := h.groupRepo.FindByID(ctx.ChatID)
    if err != nil {
        return ctx.Reply("❌ 加载群组失败")
    }

    if _, ok := g.GetSetting(key); !ok {
        return ctx.Reply(fmt.Sprintf("❌ 配置 '%s' 不存在", key))
    }

    delete(g.Settings, key)
    g.UpdatedAt = time.Now()

    if err := h.groupRepo.Update(g); err != nil {
        return ctx.Reply("❌ 保存失败")
    }

    return ctx.Reply(fmt.Sprintf("✅ 配置 '%s' 已删除", key))
}

func (h *ConfigHandler) loadOrCreateGroup(ctx *handler.Context) (*group.Group, error) {
    g, err := h.groupRepo.FindByID(ctx.ChatID)
    if err != nil {
        g = group.NewGroup(ctx.ChatID, ctx.ChatTitle, ctx.ChatType)
        if err := h.groupRepo.Save(g); err != nil {
            return nil, err
        }
    }
    return g, nil
}
```

---

## 🎯 常见配置场景示例

### 场景 1: 欢迎消息配置

```bash
# 设置欢迎消息
/config set welcome_msg 欢迎加入本群！请阅读群规。
/config set welcome_enabled true

# 对应的 Listener 处理器读取配置
# value, ok := g.GetSetting("welcome_enabled")
# if ok && value.(bool) { ... }
```

### 场景 2: 自动回复配置

```bash
/config set auto_reply_hello 你好！有什么可以帮助你的？
/config set auto_reply_help 请使用 /help 查看所有命令
```

### 场景 3: 功能开关

```bash
/config set antiflood_enabled true
/config set antiflood_max_messages 5
/config set antiflood_time_window 10
```

---

## 📦 文件清单

### 新建文件（4 个）

1. `internal/handlers/command/enable.go` - 启用命令
2. `internal/handlers/command/disable.go` - 禁用命令
3. `internal/handlers/command/cmdlist.go` - 命令状态列表
4. `internal/handlers/command/config.go` - 通用配置管理

### 修改文件（1 个）

5. `cmd/bot/main.go` - 注册新命令到 `registerHandlers()`

**注册示例**:
```go
func registerHandlers(...) {
    // 现有命令
    router.Register(command.NewPingHandler(groupRepo))
    router.Register(command.NewHelpHandler(groupRepo, router))
    router.Register(command.NewStatsHandler(groupRepo, userRepo))

    // 新增群组配置管理命令
    router.Register(command.NewEnableCommandHandler(groupRepo))
    router.Register(command.NewDisableCommandHandler(groupRepo))
    router.Register(command.NewCmdListHandler(groupRepo, router))
    router.Register(command.NewConfigHandler(groupRepo))
}
```

---

## ⚠️ 重要注意事项

### 1. 命令禁用保护

**不能禁用的核心命令**:
- `/enable` - 否则无法重新启用命令
- `/disable` - 防止逻辑混乱
- `/help` - 保证用户能查看帮助

### 2. 配置值类型处理

`Settings map[string]interface{}` 存储任意类型，使用时需要类型断言：

```go
// 存储
g.SetSetting("max_users", 100)           // int
g.SetSetting("welcome_msg", "欢迎！")     // string
g.SetSetting("enabled", true)            // bool

// 安全读取
if value, ok := g.GetSetting("enabled"); ok {
    if enabled, ok := value.(bool); ok {
        // 使用 enabled
    }
}
```

### 3. 群组自动创建

当群组首次使用配置命令时，需要自动创建 Group 实体：

```go
g, err := groupRepo.FindByID(ctx.ChatID)
if err != nil {
    g = group.NewGroup(ctx.ChatID, ctx.ChatTitle, ctx.ChatType)
    if err := groupRepo.Save(g); err != nil {
        return err
    }
}
```

### 4. 私聊限制

配置命令应该只在群组中可用，设置 `chatTypes`:

```go
NewBaseCommand(
    "config",
    "管理群组配置",
    user.PermissionSuperAdmin,
    []string{"group", "supergroup"},  // 不包含 "private"
    groupRepo,
)
```

---

## 🧪 测试场景

### 命令启用/禁用测试

- [ ] `/enable ping` 启用命令成功
- [ ] `/disable ping` 禁用命令成功
- [ ] 禁用后的命令无法执行
- [ ] 重新启用后命令恢复正常
- [ ] 尝试禁用 `/enable` 命令 → 被拒绝
- [ ] `/cmdlist` 正确显示所有命令状态

### 通用配置测试

- [ ] `/config set key value` 设置配置成功
- [ ] `/config get key` 正确获取配置
- [ ] `/config list` 显示所有配置
- [ ] `/config del key` 删除配置成功
- [ ] 获取不存在的配置 → 返回错误提示

### 权限测试

- [ ] SuperAdmin 可以使用所有配置命令
- [ ] Admin 无法使用 `/config` → 权限不足
- [ ] User 无法使用 `/enable` → 权限不足

### 群组隔离测试

- [ ] 群组 A 的配置不影响群组 B
- [ ] 同一个 Bot 在多个群组中配置独立

---

## 🚀 使用示例

### 示例 1: 禁用某个命令

```
Alice (SuperAdmin): /disable stats
Bot: ✅ 命令 /stats 已禁用

Bob (User): /stats
[没有响应]

Alice: /enable stats
Bot: ✅ 命令 /stats 已启用

Bob: /stats
Bot: [显示统计信息]
```

### 示例 2: 查看命令状态

```
Alice: /cmdlist
Bot:
📋 群组命令配置状态:

✅ 已启用 (3):
  • /ping - 测试机器人响应
  • /help - 显示帮助信息
  • /enable - 启用指定命令

❌ 已禁用 (1):
  • /stats - 查看统计信息
```

### 示例 3: 配置管理

```
Alice: /config set welcome_msg 欢迎加入我们的群组！
Bot: ✅ 配置已设置:
welcome_msg = 欢迎加入我们的群组！

Alice: /config list
Bot:
📋 群组配置列表:

• welcome_msg = 欢迎加入我们的群组！
• welcome_enabled = true

总计: 2 项配置
```

---

## 📚 相关文档

- 文档示例：`docs/handlers/command-handler-guide.md:945-989` 有 `/enable` 完整代码
- API 参考：`docs/api-reference.md:1040-1060` 记录了 EnableCommand/DisableCommand API
- 仓储指南：`docs/repository-guide.md:297-306` 有配置方法说明

---

## 📊 工作量评估

- **新建文件**: 4 个
- **修改文件**: 1 个
- **预计工作量**: 3-4 小时
- **优先级**: 中
- **复杂度**: 中等
- **依赖**: 无（基础设施已完备）

---

**创建日期**: 2025-10-02
**最后更新**: 2025-10-02
**负责人**: 待分配
**状态**: 待实现
