# 待处理功能：用户权限管理命令

## 📋 功能概述

**当前状态**: 权限系统已实现，但缺少管理命令

**目标**: 添加完整的用户权限管理命令，让 SuperAdmin/Owner 可以通过命令管理群组成员权限

---

## ✅ 已实现的基础设施

### 权限系统核心
- **权限级别** (`internal/domain/user/user.go:12-25`):
  ```go
  const (
      PermissionNone       // 0
      PermissionUser       // 1 - 默认用户
      PermissionAdmin      // 2 - 管理员
      PermissionSuperAdmin // 3 - 超级管理员
      PermissionOwner      // 4 - 群主
  )
  ```

- **用户实体** (`internal/domain/user/user.go:32-83`):
  - `Permissions map[int64]Permission` - 按群组存储权限
  - `GetPermission(groupID)` - 获取特定群组权限
  - `SetPermission(groupID, perm)` - 设置群组权限
  - `HasPermission(groupID, required)` - 检查权限
  - `IsSuperAdmin(groupID)` / `IsAdmin(groupID)` - 权限判断

- **User Repository** (`internal/domain/user/user.go:86-93`):
  ```go
  type Repository interface {
      FindByID(id int64) (*User, error)
      FindByUsername(username string) (*User, error)
      Save(user *User) error
      Update(user *User) error          // ✅ 用于权限更新
      Delete(id int64) error
      FindAdminsByGroup(groupID int64) ([]*User, error)  // ✅ 用于管理员列表
  }
  ```

- **权限中间件** (`internal/middleware/permission.go`):
  - 自动加载用户并注入到 `ctx.User`
  - 首次用户自动创建（默认 PermissionUser）

- **Context 权限辅助** (`internal/handler/context.go:160-191`):
  - `HasPermission(required)` - 检查权限
  - `RequirePermission(required)` - 要求权限，不足返回错误

---

## ❌ 缺失的功能

目前**没有任何命令**可以管理用户权限，所有权限修改只能通过直接操作数据库。

需要实现以下 5 个命令：

### 1. `/promote` - 提升用户权限
### 2. `/demote` - 降低用户权限
### 3. `/setperm` - 直接设置权限
### 4. `/listadmins` - 查看管理员列表
### 5. `/myperm` - 查看自己的权限

---

## 📋 详细实现方案

### 1. `/promote` - 提升用户权限

**文件**: `internal/handlers/command/promote.go`

**功能**: 将用户权限提升一级
- User → Admin
- Admin → SuperAdmin
- SuperAdmin → Owner（仅 Owner 可执行）

**权限要求**: SuperAdmin

**用法**:
```
/promote @username       # 通过 @ 指定用户
/promote                 # 回复某人消息时执行
```

**实现要点**:
```go
type PromoteHandler struct {
    *BaseCommand
    userRepo UserRepository
}

func (h *PromoteHandler) Handle(ctx *handler.Context) error {
    // 1. 检查权限
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }

    // 2. 获取目标用户（从参数或回复消息）
    targetUser, err := h.getTargetUser(ctx)
    if err != nil {
        return ctx.Reply("❌ 无法识别目标用户")
    }

    // 3. 获取当前权限
    currentPerm := targetUser.GetPermission(ctx.ChatID)

    // 4. 计算新权限
    newPerm := currentPerm + 1
    if newPerm > user.PermissionOwner {
        return ctx.Reply("❌ 该用户已是最高权限")
    }

    // 5. 权限保护：不能提升到比自己高的等级
    if !ctx.User.HasPermission(ctx.ChatID, newPerm) {
        return ctx.Reply("❌ 您无权提升用户到该等级")
    }

    // 6. 设置新权限
    targetUser.SetPermission(ctx.ChatID, newPerm)

    // 7. 保存到数据库
    if err := h.userRepo.Update(targetUser); err != nil {
        return ctx.Reply("❌ 权限更新失败")
    }

    // 8. 成功反馈
    return ctx.ReplyMarkdown(fmt.Sprintf(
        "✅ 用户 %s 权限已提升: %s → %s",
        targetUser.Username,
        currentPerm.String(),
        newPerm.String(),
    ))
}
```

---

### 2. `/demote` - 降低用户权限

**文件**: `internal/handlers/command/demote.go`

**功能**: 将用户权限降低一级
- Owner → SuperAdmin
- SuperAdmin → Admin
- Admin → User

**权限要求**: SuperAdmin

**用法**:
```
/demote @username
/demote                  # 回复消息时
```

**实现要点**:
- 逻辑与 `/promote` 类似，但 `newPerm = currentPerm - 1`
- 不能降低到 PermissionNone（最低为 PermissionUser）
- 权限保护：不能降低比自己高的用户

---

### 3. `/setperm` - 直接设置权限

**文件**: `internal/handlers/command/setperm.go`

**功能**: 直接设置用户到指定权限等级

**权限要求**: Owner

**用法**:
```
/setperm @username admin
/setperm @username superadmin
/setperm @username owner
```

**实现要点**:
```go
func (h *SetPermHandler) Handle(ctx *handler.Context) error {
    // 1. 检查权限（必须是 Owner）
    if err := ctx.RequirePermission(user.PermissionOwner); err != nil {
        return err
    }

    // 2. 解析参数
    args := command.ParseArgs(ctx.Text)
    if len(args) < 2 {
        return ctx.Reply("用法: /setperm @username <admin|superadmin|owner>")
    }

    username := strings.TrimPrefix(args[0], "@")
    permStr := strings.ToLower(args[1])

    // 3. 解析权限等级
    var newPerm user.Permission
    switch permStr {
    case "user":
        newPerm = user.PermissionUser
    case "admin":
        newPerm = user.PermissionAdmin
    case "superadmin":
        newPerm = user.PermissionSuperAdmin
    case "owner":
        newPerm = user.PermissionOwner
    default:
        return ctx.Reply("❌ 无效的权限等级")
    }

    // 4. 查找用户并更新
    targetUser, err := h.userRepo.FindByUsername(username)
    if err != nil {
        return ctx.Reply("❌ 用户不存在")
    }

    targetUser.SetPermission(ctx.ChatID, newPerm)
    if err := h.userRepo.Update(targetUser); err != nil {
        return ctx.Reply("❌ 权限更新失败")
    }

    return ctx.Reply(fmt.Sprintf("✅ 用户 @%s 权限已设置为: %s", username, newPerm.String()))
}
```

---

### 4. `/listadmins` - 查看管理员列表

**文件**: `internal/handlers/command/listadmins.go`

**功能**: 显示当前群组的所有管理员（Admin 及以上）

**权限要求**: User（所有人可查看）

**用法**:
```
/listadmins
```

**输出示例**:
```
👥 当前群组管理员列表:

👑 Owner (1人):
  • @alice

⭐ SuperAdmin (2人):
  • @bob
  • @charlie

🛡 Admin (3人):
  • @david
  • @emma
  • @frank

总计: 6 位管理员
```

**实现要点**:
```go
func (h *ListAdminsHandler) Handle(ctx *handler.Context) error {
    // 1. 查询所有管理员
    admins, err := h.userRepo.FindAdminsByGroup(ctx.ChatID)
    if err != nil {
        return ctx.Reply("❌ 查询失败")
    }

    // 2. 按权限等级分组
    owners := []string{}
    superAdmins := []string{}
    regularAdmins := []string{}

    for _, admin := range admins {
        perm := admin.GetPermission(ctx.ChatID)
        username := "@" + admin.Username
        if admin.Username == "" {
            username = admin.FirstName
        }

        switch perm {
        case user.PermissionOwner:
            owners = append(owners, username)
        case user.PermissionSuperAdmin:
            superAdmins = append(superAdmins, username)
        case user.PermissionAdmin:
            regularAdmins = append(regularAdmins, username)
        }
    }

    // 3. 构建输出
    var sb strings.Builder
    sb.WriteString("👥 当前群组管理员列表:\n\n")

    if len(owners) > 0 {
        sb.WriteString(fmt.Sprintf("👑 Owner (%d人):\n", len(owners)))
        for _, u := range owners {
            sb.WriteString(fmt.Sprintf("  • %s\n", u))
        }
        sb.WriteString("\n")
    }

    if len(superAdmins) > 0 {
        sb.WriteString(fmt.Sprintf("⭐ SuperAdmin (%d人):\n", len(superAdmins)))
        for _, u := range superAdmins {
            sb.WriteString(fmt.Sprintf("  • %s\n", u))
        }
        sb.WriteString("\n")
    }

    if len(regularAdmins) > 0 {
        sb.WriteString(fmt.Sprintf("🛡 Admin (%d人):\n", len(regularAdmins)))
        for _, u := range regularAdmins {
            sb.WriteString(fmt.Sprintf("  • %s\n", u))
        }
        sb.WriteString("\n")
    }

    total := len(owners) + len(superAdmins) + len(regularAdmins)
    sb.WriteString(fmt.Sprintf("总计: %d 位管理员", total))

    return ctx.Reply(sb.String())
}
```

---

### 5. `/myperm` - 查看自己的权限

**文件**: `internal/handlers/command/myperm.go`

**功能**: 显示当前用户在当前群组的权限级别

**权限要求**: User（所有人可查看）

**用法**:
```
/myperm
```

**输出示例**:
```
👤 您的权限信息:

群组: 开发讨论组
用户: @alice
权限等级: SuperAdmin ⭐

您可以:
✅ 使用所有用户命令
✅ 使用管理员命令
✅ 提升/降低用户权限
✅ 管理群组配置
```

**实现要点**:
```go
func (h *MyPermHandler) Handle(ctx *handler.Context) error {
    perm := ctx.User.GetPermission(ctx.ChatID)

    var sb strings.Builder
    sb.WriteString("👤 您的权限信息:\n\n")
    sb.WriteString(fmt.Sprintf("群组: %s\n", ctx.ChatTitle))
    sb.WriteString(fmt.Sprintf("用户: @%s\n", ctx.Username))
    sb.WriteString(fmt.Sprintf("权限等级: %s %s\n\n", perm.String(), getPermIcon(perm)))
    sb.WriteString("您可以:\n")

    switch perm {
    case user.PermissionOwner:
        sb.WriteString("✅ 所有权限（群主）\n")
    case user.PermissionSuperAdmin:
        sb.WriteString("✅ 使用所有用户命令\n")
        sb.WriteString("✅ 使用管理员命令\n")
        sb.WriteString("✅ 提升/降低用户权限\n")
        sb.WriteString("✅ 管理群组配置\n")
    case user.PermissionAdmin:
        sb.WriteString("✅ 使用所有用户命令\n")
        sb.WriteString("✅ 使用管理员命令\n")
    case user.PermissionUser:
        sb.WriteString("✅ 使用基础用户命令\n")
    }

    return ctx.Reply(sb.String())
}

func getPermIcon(perm user.Permission) string {
    switch perm {
    case user.PermissionOwner:
        return "👑"
    case user.PermissionSuperAdmin:
        return "⭐"
    case user.PermissionAdmin:
        return "🛡"
    default:
        return "👤"
    }
}
```

---

## 🔧 辅助函数

**文件**: `internal/handlers/command/permission_helpers.go`

```go
package command

import (
    "fmt"
    "strings"
    "telegram-bot/internal/domain/user"
    "telegram-bot/internal/handler"
)

// GetTargetUser 从参数或回复消息中获取目标用户
func GetTargetUser(ctx *handler.Context, userRepo UserRepository) (*user.User, error) {
    // 方式 1: 从参数获取 @username
    args := ParseArgs(ctx.Text)
    if len(args) > 0 {
        username := strings.TrimPrefix(args[0], "@")
        return userRepo.FindByUsername(username)
    }

    // 方式 2: 从回复消息获取
    if ctx.ReplyTo != nil {
        return userRepo.FindByID(ctx.ReplyTo.UserID)
    }

    return nil, fmt.Errorf("no target user specified")
}
```

---

## 📦 文件清单

### 新建文件（6 个）

1. `internal/handlers/command/promote.go` - 提升权限命令
2. `internal/handlers/command/demote.go` - 降低权限命令
3. `internal/handlers/command/setperm.go` - 设置权限命令
4. `internal/handlers/command/listadmins.go` - 管理员列表命令
5. `internal/handlers/command/myperm.go` - 查看自己权限命令
6. `internal/handlers/command/permission_helpers.go` - 辅助函数

### 修改文件（1 个）

7. `cmd/bot/main.go` - 注册新命令到 `registerHandlers()`

```go
func registerHandlers(...) {
    // 现有命令
    router.Register(command.NewPingHandler(groupRepo))
    router.Register(command.NewHelpHandler(groupRepo, router))
    router.Register(command.NewStatsHandler(groupRepo, userRepo))

    // 新增权限管理命令
    router.Register(command.NewPromoteHandler(groupRepo, userRepo))
    router.Register(command.NewDemoteHandler(groupRepo, userRepo))
    router.Register(command.NewSetPermHandler(groupRepo, userRepo))
    router.Register(command.NewListAdminsHandler(groupRepo, userRepo))
    router.Register(command.NewMyPermHandler(groupRepo))

    // ... 其他处理器
}
```

---

## ⚠️ 重要注意事项

### 权限保护规则

1. **不能越级操作**:
   - Admin 不能提升用户到 SuperAdmin
   - SuperAdmin 不能提升用户到 Owner
   - 只有 Owner 可以提升到 Owner

2. **不能降低比自己高的用户**:
   - Admin 不能降低 SuperAdmin
   - SuperAdmin 不能降低 Owner

3. **Owner 特殊规则**:
   - `/setperm` 只有 Owner 可以使用
   - Owner 可以设置任何用户到任何等级
   - 建议：一个群组只设置 1-2 个 Owner

### 用户识别方式

支持两种方式指定目标用户：

1. **参数方式**: `/promote @username`
2. **回复方式**: 回复某人消息后输入 `/promote`

优先级：参数 > 回复消息

### 群组隔离

- 所有权限修改只在当前群组生效
- 使用 `ctx.ChatID` 作为 `groupID`
- 私聊中的权限操作无意义（可选择性禁止）

### 错误处理

- 用户不存在 → 返回友好错误提示
- 权限不足 → 返回当前权限和所需权限
- 数据库操作失败 → 记录日志并返回通用错误

---

## 🧪 测试场景

### 基础功能测试

- [ ] `/promote` 提升 User → Admin 成功
- [ ] `/promote` 提升 Admin → SuperAdmin 成功（执行者为 SuperAdmin）
- [ ] `/promote` 提升 Admin → SuperAdmin 失败（执行者为 Admin）
- [ ] `/demote` 降低 Admin → User 成功
- [ ] `/setperm` 设置权限成功（执行者为 Owner）
- [ ] `/setperm` 失败（执行者为 SuperAdmin）
- [ ] `/listadmins` 正确显示所有管理员
- [ ] `/myperm` 显示当前用户权限

### 边界测试

- [ ] 提升已是最高权限的用户 → 返回错误
- [ ] 降低已是最低权限的用户 → 返回错误
- [ ] 对不存在的用户操作 → 返回错误
- [ ] 参数格式错误 → 返回用法提示

### 权限保护测试

- [ ] Admin 尝试提升到 Owner → 拒绝
- [ ] SuperAdmin 尝试降低 Owner → 拒绝
- [ ] 非 Owner 使用 `/setperm` → 拒绝

### 用户识别测试

- [ ] `/promote @username` 正常工作
- [ ] 回复消息后 `/promote` 正常工作
- [ ] 无参数且无回复消息 → 返回错误

---

## 📚 相关文档更新

需要更新的文档:

- [ ] `docs/handlers/command-handler-guide.md` - 添加权限管理命令示例
- [ ] `docs/getting-started.md` - 添加权限管理使用说明
- [ ] `README.md` - 功能列表中添加"用户权限管理"
- [ ] `CLAUDE.md` - 更新已实现的命令列表

---

## 📊 工作量评估

- **新建文件**: 6 个
- **修改文件**: 1 个
- **预计工作量**: 2-3 小时
- **优先级**: 中
- **复杂度**: 简单
- **依赖**: 无（基础设施已完备）

---

## 🚀 使用示例

### 场景 1: 提升用户为管理员

```
Alice: /promote @bob
Bot: ✅ 用户 bob 权限已提升: User → Admin
```

### 场景 2: 查看管理员列表

```
Bob: /listadmins
Bot:
👥 当前群组管理员列表:

👑 Owner (1人):
  • @alice

🛡 Admin (2人):
  • @bob
  • @charlie

总计: 3 位管理员
```

### 场景 3: 查看自己的权限

```
Bob: /myperm
Bot:
👤 您的权限信息:

群组: 开发讨论组
用户: @bob
权限等级: Admin 🛡

您可以:
✅ 使用所有用户命令
✅ 使用管理员命令
```

---

**创建日期**: 2025-10-02
**最后更新**: 2025-10-02
**负责人**: 待分配
**状态**: 待实现
