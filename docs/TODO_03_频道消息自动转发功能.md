# 待处理功能清单

## 🔥 高优先级

### 1. 频道消息自动转发功能

**需求描述**:
某个已认证的频道，机器人需要把频道的消息转发到别的群组。

**业务场景**:
- 官方频道发布公告 → 自动转发到多个讨论群
- 新闻频道更新内容 → 同步到相关主题群组
- 内部频道通知 → 分发到不同部门群组

**技术方案**: 使用 Listener 处理器 + MongoDB 配置存储

---

#### 📋 实现步骤

**Phase 1: 基础支持（必须先完成）**

1. **修复 Channel 消息处理的基础问题**
   - [ ] 修复 `internal/adapter/telegram/converter.go`
     - 问题: `msg.From` 在频道消息中可能为 `nil`，导致空指针异常
     - 解决: 频道消息使用频道信息代替用户信息
   ```go
   // 需要修改的部分
   var userID int64
   var username, firstName, lastName string

   if msg.From != nil {
       userID = msg.From.ID
       username = msg.From.Username
       firstName = msg.From.FirstName
       lastName = msg.From.LastName
   } else {
       // 频道匿名消息
       userID = msg.Chat.ID
       username = msg.Chat.Username
       firstName = msg.Chat.Title
   }
   ```

   - [ ] 修改 `internal/handler/context.go`
     - 添加 `ForwardMessage()` 方法
     - 修改 `Reply()` 方法，频道消息不使用 `ReplyParameters`
   ```go
   // 新增方法
   func (c *Context) ForwardMessage(toChatID int64) error {
       _, err := c.Bot.ForwardMessage(c.Ctx, &bot.ForwardMessageParams{
           ChatID:     toChatID,
           FromChatID: c.ChatID,
           MessageID:  c.MessageID,
       })
       return err
   }
   ```

---

**Phase 2: 核心功能**

2. **创建转发配置领域模型**
   - [ ] 新建 `internal/domain/forward/forward_config.go`
   ```go
   type ForwardConfig struct {
       ID                 string    // MongoDB ObjectID
       SourceChannelID    int64     // 源频道 ID
       SourceChannelTitle string    // 源频道标题
       TargetGroupIDs     []int64   // 目标群组 ID 列表
       Enabled            bool      // 是否启用
       CreatedBy          int64     // 创建者用户 ID
       CreatedAt          time.Time
       UpdatedAt          time.Time
       Stats              ForwardStats
   }

   type ForwardStats struct {
       TotalForwarded  int64     // 总转发次数
       LastForwardAt   time.Time // 最后转发时间
   }
   ```

   - [ ] 新建 `internal/domain/forward/repository.go`
   ```go
   type Repository interface {
       FindBySourceChannel(channelID int64) (*ForwardConfig, error)
       FindAllEnabled() ([]*ForwardConfig, error)
       Save(config *ForwardConfig) error
       Update(config *ForwardConfig) error
       Delete(channelID int64) error
   }
   ```

3. **实现 MongoDB Repository**
   - [ ] 新建 `internal/adapter/repository/mongodb/forward_repository.go`
   - [ ] 实现所有 Repository 接口方法
   - [ ] 添加索引创建逻辑到 `index_manager.go`
   ```go
   // 需要的索引
   db.forward_configs.createIndex({ source_channel_id: 1 }, { unique: true })
   db.forward_configs.createIndex({ enabled: 1, source_channel_id: 1 })
   ```

4. **创建转发监听器**
   - [ ] 新建 `internal/handlers/listener/channel_forwarder.go`
   ```go
   type ChannelForwarderHandler struct {
       forwardRepo forward.Repository
       logger      logger.Logger
   }

   // Priority: 920 (在 MessageLogger 和 Analytics 之后)
   // ContinueChain: true (不中断其他处理器)

   // Match: 只匹配频道消息 + 在配置的源频道列表中
   // Handle: 转发到所有配置的目标群组
   ```

5. **创建管理命令**
   - [ ] 新建 `internal/handlers/command/forward.go`
   - [ ] 实现子命令:
     - `/forward add <频道ID> <目标群ID1> [目标群ID2...]` - 添加转发规则
     - `/forward remove <频道ID>` - 删除转发规则
     - `/forward list` - 列出所有转发规则
     - `/forward enable <频道ID>` - 启用转发
     - `/forward disable <频道ID>` - 禁用转发
     - `/forward stats [频道ID]` - 查看转发统计
   - [ ] 权限要求: `PermissionSuperAdmin`

6. **注册到主程序**
   - [ ] 修改 `cmd/bot/main.go`
     - 初始化 `ForwardRepository`
     - 注册 `ChannelForwarderHandler`
     - 注册 `/forward` 命令

---

**Phase 3: 增强功能（可选）**

8. **转发增强**
   - [ ] 支持转发延迟（避免频率限制）
   - [ ] 支持批量转发（队列处理）

9. **监控和统计**
   - [ ] 转发失败告警（连续失败 N 次）
   - [ ] 定期统计报告

---

#### 📊 数据库设计

**集合名称**: `forward_configs`

**文档结构**:
```json
{
  "_id": ObjectId("..."),
  "source_channel_id": -1001234567890,
  "source_channel_title": "Official Channel",
  "target_group_ids": [-1001111111111, -1002222222222],
  "enabled": true,
  "created_by": 123456,
  "created_at": ISODate("2025-10-02T10:00:00Z"),
  "updated_at": ISODate("2025-10-02T10:00:00Z"),
  "stats": {
    "total_forwarded": 1234,
    "last_forward_at": ISODate("2025-10-02T12:30:00Z")
  }
}
```

**索引**:
```javascript
// 唯一索引：一个频道只能有一个转发配置
db.forward_configs.createIndex({ source_channel_id: 1 }, { unique: true })

// 查询优化：快速找到所有启用的配置
db.forward_configs.createIndex({ enabled: 1, source_channel_id: 1 })
```

---

#### ⚠️ 重要注意事项

**权限要求**:
1. Bot 必须是源频道的管理员（才能接收频道消息）
2. Bot 必须在所有目标群组中（且有发送消息权限）
3. 建议 Bot 在目标群组也是管理员（避免反垃圾限制）

**Telegram 限制**:
1. 消息发送频率限制：约 30 条/秒（单个 Bot）
2. 目标群组过多时需要控制并发（避免超限）
3. 转发失败不应影响其他目标群组

**错误处理**:
1. 某个目标群组转发失败 → 记录日志，继续其他群组
2. 连续失败 3 次 → 记录错误，考虑禁用该配置
3. Bot 被踢出目标群 → 自动从配置中移除该群组 ID

**性能优化**:
1. 使用 goroutine 并发转发到多个群组
2. 控制并发数（如最多 10 个并发）
3. 转发失败时使用指数退避重试

---

#### 🧪 测试场景

**基础功能测试**:
1. [ ] 频道发送文本消息 → 自动转发到目标群组
2. [ ] 频道发送图片/视频 → 正确转发
3. [ ] 频道发送文件/音频 → 正确转发
4. [ ] 多个目标群组 → 都能收到消息

**管理命令测试**:
5. [ ] `/forward add` 添加新规则 → 成功保存到数据库
6. [ ] `/forward list` 查看规则 → 正确显示
7. [ ] `/forward disable` 禁用规则 → 不再转发
8. [ ] `/forward enable` 重新启用 → 恢复转发

**异常处理测试**:
9. [ ] Bot 被踢出目标群 → 记录错误，继续其他群组
10. [ ] 目标群组不存在 → 记录错误
11. [ ] Bot 无权限发送消息 → 记录错误
12. [ ] 网络超时 → 重试机制

---

#### 📚 相关文档

需要更新的文档:
- [ ] `docs/handlers/listener-handler-guide.md` - 添加 ChannelForwarder 示例
- [ ] `docs/getting-started.md` - 添加转发功能使用说明
- [ ] `docs/developer-api.md` - 添加 `ForwardMessage()` API 文档
- [ ] `README.md` - 功能列表中添加"频道消息转发"

---

#### 📦 涉及的文件

**新建文件（约 10 个）**:
1. `internal/domain/forward/forward_config.go` - 领域模型
2. `internal/domain/forward/repository.go` - 仓储接口
3. `internal/adapter/repository/mongodb/forward_repository.go` - MongoDB 实现
4. `internal/handlers/listener/channel_forwarder.go` - 转发监听器
5. `internal/handlers/command/forward.go` - 管理命令
6. `test/integration/forward_test.go` - 集成测试
7. `test/mocks/forward_repository_mock.go` - Mock 对象

**修改文件（约 3 个）**:
8. `internal/adapter/telegram/converter.go` - 修复频道消息处理
9. `internal/handler/context.go` - 添加 ForwardMessage 方法
10. `cmd/bot/main.go` - 注册处理器和仓储
11. `internal/adapter/repository/mongodb/index_manager.go` - 添加索引

---

#### 🚀 使用示例

**配置转发规则**:
```
# 添加转发规则（频道 ID → 目标群组 ID）
/forward add -1001234567890 -1001111111111 -1002222222222

# 查看所有规则
/forward list

# 禁用某个频道的转发
/forward disable -1001234567890

# 查看转发统计
/forward stats -1001234567890
```

**效果**:
```
频道 "Official Announcements" 发布新消息
↓
Bot 自动检测到消息
↓
转发到 "讨论群 A" ✅
转发到 "讨论群 B" ✅
转发到 "讨论群 C" ✅
↓
记录统计：total_forwarded++
```

---

**预计工作量**: 4-6 小时
**优先级**: 高
**复杂度**: 中等
**依赖**: 需要先修复 Channel 支持的基础问题

---

**创建日期**: 2025-10-02
**最后更新**: 2025-10-03
**负责人**: 待分配

---

## 📝 其他待处理功能

### 2. 限流中间件启用

**当前状态**: 已实现但未启用
**文件位置**: `internal/middleware/ratelimit.go`
**优先级**: 中

**启用方式**:
```go
// cmd/bot/main.go
rateLimiter := middleware.NewSimpleRateLimiter(time.Second, 5)
router.Use(middleware.NewRateLimitMiddleware(rateLimiter).Middleware())
```

---

### 3. 定时任务完善

**当前状态**: 2 个启用，2 个未启用

**未启用的任务**:
- `AutoUnbanJob` - 自动解封（每 5 分钟）
- `CacheWarmupJob` - 缓存预热（每 30 分钟）

**优先级**: 低

---

### 4. 更多命令实现

**建议添加的命令**:
- `/setperm` - 设置用户权限（SuperAdmin）
- `/config` - 配置群组设置（SuperAdmin）

**优先级**: 中

---

**文档维护**: 定期更新此文件，标记已完成的任务
