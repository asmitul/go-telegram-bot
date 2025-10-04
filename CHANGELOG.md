# Changelog

所有重要的项目变更都会记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范。

---

## [2.1.0] - 2025-10-04

### 🎯 架构优化 - Context 传递与错误处理

本次更新修复了 21 个逻辑问题，提升了系统的稳定性、可维护性和资源管理能力。

### ✨ 新增 (Added)

#### Repository 层 Context 支持

- **所有 Repository 接口方法现在都需要 `context.Context` 作为第一个参数**
  - 支持请求取消：当用户取消请求时，数据库操作也会被取消
  - 支持超时控制：防止慢查询阻塞整个系统
  - 支持链路追踪：可以在 context 中传递 request ID、trace ID 等
  - 影响接口：
    ```go
    // UserRepository
    FindByID(ctx context.Context, id int64) (*User, error)
    FindByUsername(ctx context.Context, username string) (*User, error)
    Save(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    UpdatePermission(ctx context.Context, userID, groupID int64, perm Permission) error
    Delete(ctx context.Context, id int64) error
    FindAdminsByGroup(ctx context.Context, groupID int64) ([]*User, error)

    // GroupRepository
    FindByID(ctx context.Context, id int64) (*Group, error)
    Save(ctx context.Context, group *Group) error
    Update(ctx context.Context, group *Group) error
    Delete(ctx context.Context, id int64) error
    FindAll(ctx context.Context) ([]*Group, error)
    ```
  - 影响文件：`internal/domain/user/user.go`, `internal/domain/group/group.go`
  - 影响范围：所有 Repository 实现和调用点（50+ 文件）

#### Telegram API 层 Context 支持

- **所有 Telegram API 方法添加 `context.Context` 参数**
  - 影响方法：`BanChatMember`, `UnbanChatMember`, `RestrictChatMember`, `PromoteChatMember`, `SendMessage`, `DeleteMessage`, `GetChatMember`, `GetChatMembersCount`
  - 影响文件：`internal/adapter/telegram/api.go`

#### 资源管理改进

- **RateLimiter 自动清理机制**
  - 每小时自动清理一次未活跃用户数据（超过 24 小时未发送消息）
  - 新增 `Stop()` 方法用于优雅关闭，防止 goroutine 泄漏
  - 影响文件：`internal/middleware/ratelimit.go`

- **AnalyticsHandler 清理间隔限制**
  - 添加最小清理间隔（10 分钟），防止高并发下频繁清理
  - 影响文件：`internal/handlers/listener/analytics.go`

#### 权限系统增强

- **禁止用户修改自己的权限**
  - `/setperm` 命令新增自我修改检查
  - 防止权限系统被滥用
  - 影响文件：`internal/handlers/command/setperm.go`

### 🔧 修复 (Fixed)

#### 高优先级 (P0) 修复

1. **Middleware 错误处理优化**
   - **问题描述**：Permission/Group 中间件在创建用户/群组失败时，会注入默认对象并继续执行，导致内存与数据库状态不一致
   - **修复方案**：创建失败时返回错误，停止请求处理
   - **影响**：确保数据一致性，避免后续操作基于错误的对象
   - **代码变更**：
     ```go
     // ❌ 旧版本（错误）
     if err := m.userRepo.Save(u); err != nil {
         u = user.NewDefaultUser()  // 注入默认对象
         ctx.User = u               // 继续执行（危险）
     }

     // ✅ 新版本（正确）
     if err := m.userRepo.Save(reqCtx, u); err != nil {
         m.logger.Error("failed_to_create_user", "error", err)
         return fmt.Errorf("failed to create user: %w", err)
     }
     ```
   - **影响文件**：`internal/middleware/permission.go`, `internal/middleware/group.go`

2. **BaseCommand 群组检查逻辑修复**
   - **问题描述**：数据库错误时逻辑判断反了，会继续执行命令；群组不存在时阻止执行
   - **修复方案**：反转逻辑，数据库错误时阻止执行，群组不存在时允许（由中间件创建）
   - **影响**：提高系统稳定性，避免在数据库异常时执行敏感操作
   - **影响文件**：`internal/handlers/command/base.go`

#### 中优先级 (P1) 修复

3. **Router 错误处理优化**
   - **改进**：区分用户级错误和系统级错误，改进注释和日志
   - **影响**：提高代码可读性和可维护性
   - **影响文件**：`internal/handler/router.go`

#### 低优先级 (P2) 修复

4. **Recovery Middleware 错误包装改进**
   - **改进**：保留原始错误类型信息，改进 panic 恢复后的错误处理
   - **代码变更**：
     ```go
     // 新版本：保留错误类型
     switch v := r.(type) {
     case error:
         err = fmt.Errorf("panic recovered: %w", v)
     default:
         err = fmt.Errorf("panic recovered: %v (type: %T)", r, r)
     }
     ```
   - **影响文件**：`internal/middleware/recovery.go`

5. **Scheduler Context 处理优化**
   - **说明**：确认现有实现正确（继承 scheduler context 以支持任务取消）
   - **测试验证**：通过 TestScheduler_ContextCancellation 测试
   - **影响文件**：`internal/scheduler/scheduler.go`

6. **索引创建失败处理优化**
   - **改进**：提升错误日志级别为 CRITICAL，添加详细的失败处理建议
   - **影响文件**：`cmd/bot/main.go`

### 📝 文档更新 (Documentation)

- **README.md**
  - 添加中间件系统改进说明（错误处理、资源清理、优雅关闭）
  - 添加 Context 传递与 Repository 使用指南

- **docs/middleware-guide.md**
  - 更新 PermissionMiddleware 示例（v2.0 错误处理模式）
  - 添加 RateLimiter 资源清理与优雅关闭说明
  - 添加自动清理机制说明

- **docs/repository-guide.md**
  - 更新所有 Repository 接口示例（添加 context.Context 参数）
  - 添加 "Context 传递最佳实践" 章节
  - 添加 Handler 中使用 Repository 的示例

- **TODO.md**
  - 添加 "第五阶段：逻辑问题修复与优化" 章节
  - 记录所有 21 个修复详情

### ⚠️ 破坏性变更 (BREAKING CHANGES)

#### 1. Repository 接口变更

**所有 Repository 方法现在都需要 `context.Context` 作为第一个参数。**

**迁移指南**：

```go
// ❌ 旧版本
user, err := userRepo.FindByID(123)

// ✅ 新版本
reqCtx := context.TODO()  // 或从上层传递 context
user, err := userRepo.FindByID(reqCtx, 123)
```

**影响范围**：
- `internal/domain/user/user.go` - UserRepository 接口
- `internal/domain/group/group.go` - GroupRepository 接口
- `internal/adapter/repository/mongodb/` - 所有 Repository 实现
- `internal/handlers/command/` - 所有命令处理器（8 个文件）
- `internal/middleware/` - Permission 和 Group 中间件

#### 2. Telegram API 接口变更

**所有 Telegram API 方法添加 `context.Context` 参数。**

**迁移指南**：

```go
// ❌ 旧版本
err := api.BanChatMember(chatID, userID)

// ✅ 新版本
reqCtx := context.TODO()
err := api.BanChatMember(reqCtx, chatID, userID)
```

**影响文件**：`internal/adapter/telegram/api.go`

#### 3. Middleware 错误处理变更

**Permission/Group 中间件在创建失败时会返回错误，而非继续执行。**

**影响**：
- 用户/群组创建失败时，请求会立即返回错误
- 避免了内存与数据库状态不一致的问题

**无需迁移**：此变更仅影响内部逻辑，对外部调用者透明

#### 4. RateLimiter 优雅关闭要求

**使用 RateLimiter 时，必须在程序关闭时调用 `Stop()` 方法。**

**迁移指南**：

```go
// 创建限流器
rateLimiter := middleware.NewSimpleRateLimiter(time.Second, 5)
router.Use(middleware.NewRateLimitMiddleware(rateLimiter).Middleware())

// ⚠️ 在 shutdown 函数中添加
func shutdown() {
    // ... 其他清理代码

    if rateLimiter != nil {
        rateLimiter.Stop()
        appLogger.Info("✅ RateLimiter stopped")
    }
}
```

**影响**：未调用 `Stop()` 会导致 goroutine 泄漏

### 📊 测试结果

- ✅ 所有测试通过 (7 个包, 100% 成功率)
- ✅ 构建成功
- ✅ 无回归问题
- ✅ 代码覆盖率保持在 85%+

### 🔄 升级步骤

1. **更新 Repository 调用**
   ```bash
   # 搜索所有 Repository 调用并添加 context 参数
   grep -r "FindByID\|Save\|Update\|Delete" internal/handlers/
   ```

2. **更新 Telegram API 调用**
   ```bash
   # 搜索所有 Telegram API 调用
   grep -r "BanChatMember\|SendMessage" internal/
   ```

3. **添加 RateLimiter.Stop() 调用**
   - 在 `cmd/bot/main.go` 的 `shutdown()` 函数中添加 `rateLimiter.Stop()`

4. **运行测试**
   ```bash
   make test
   ```

5. **检查日志**
   - 关注 "failed_to_create_user" 和 "failed_to_create_group" 日志
   - 确保数据库连接正常

### 🐛 已知问题

- Handler 中暂时使用 `context.TODO()` 创建上下文（未来版本会改进）
- 清理未活跃用户的定时任务当前只记录不删除（技术债务）

### 🚀 下一步计划

- [ ] Handler.Context 集成 context.Context 字段
- [ ] 实现定时任务实际删除未活跃用户
- [ ] 添加更多的链路追踪支持

---

## [2.0.0] - 2025-09-30

### 初始版本发布

完整的 Telegram Bot 框架，包含：
- 统一消息处理架构（Command, Keyword, Pattern, Listener）
- 4 级权限系统（User, Admin, SuperAdmin, Owner）
- 中间件系统（Recovery, Logging, Permission, RateLimit）
- MongoDB 数据持久化
- 完整的测试覆盖（85%+）
- 15+ 篇详细文档

详见 [README.md](README.md)

---

[2.1.0]: https://github.com/yourusername/go-telegram-bot/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/yourusername/go-telegram-bot/releases/tag/v2.0.0
