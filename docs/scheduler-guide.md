# 定时任务开发指南

## 📚 目录

- [概述](#概述)
- [系统架构](#系统架构)
- [快速开始](#快速开始)
- [完整代码示例](#完整代码示例)
- [时间表达式](#时间表达式)
- [任务生命周期](#任务生命周期)
- [实际场景示例](#实际场景示例)
- [最佳实践](#最佳实践)
- [测试方法](#测试方法)
- [常见问题](#常见问题)

---

## 概述

本机器人框架内置了一个**轻量级的定时任务调度器**（Scheduler），用于执行周期性任务，如数据清理、统计报表、自动解禁等。

### 适用场景

- ✅ 定期清理过期数据（日志、缓存、临时文件）
- ✅ 生成统计报表（每日/每小时）
- ✅ 定时发送通知/提醒
- ✅ 自动化运维（健康检查、备份、监控）
- ✅ 缓存预热和刷新
- ✅ 批量处理任务

### 核心特性

- 🚀 **轻量级**：无需外部依赖（如 cron、Redis）
- 🔄 **自动重试**：任务失败不影响下次执行
- ⏱️ **超时控制**：单个任务最多执行 5 分钟
- 🛡️ **优雅关闭**：程序退出时等待任务完成（最多 30 秒）
- 📊 **日志记录**：自动记录任务执行情况
- 🔧 **简单易用**：实现 `Job` 接口即可

---

## 系统架构

### 核心组件

```
┌─────────────────────────────────────────┐
│           Scheduler (调度器)             │
│                                         │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐│
│  │ Job 1   │  │ Job 2   │  │ Job N   ││
│  │ (每1小时)│  │ (每5分钟)│  │ (每1天) ││
│  └────┬────┘  └────┬────┘  └────┬────┘│
│       │            │             │     │
│       └────────────┴─────────────┘     │
│              goroutine pool            │
└─────────────────────────────────────────┘
```

### Job 接口

所有定时任务必须实现 `Job` 接口（位于 `internal/scheduler/scheduler.go`）：

```go
type Job interface {
    Name() string           // 任务名称
    Run(ctx context.Context) error  // 执行任务
    Schedule() string       // 调度时间表达式
}
```

### Scheduler 调度器

调度器负责：
1. **管理任务**：添加、启动、停止任务
2. **执行调度**：按时间间隔触发任务
3. **并发控制**：每个任务在独立的 goroutine 中运行
4. **超时控制**：每个任务执行最多 5 分钟
5. **日志记录**：记录任务启动、完成、失败
6. **优雅关闭**：等待所有任务完成后退出

---

## 快速开始

### 步骤 1：创建任务文件

在 `internal/scheduler/` 目录下创建任务，或直接在 `jobs.go` 中添加。

### 步骤 2：实现 Job 接口

```go
package scheduler

import (
    "context"
    "telegram-bot/pkg/logger"
)

// MyCustomJob 自定义任务
type MyCustomJob struct {
    logger logger.Logger
}

func NewMyCustomJob(log logger.Logger) *MyCustomJob {
    return &MyCustomJob{
        logger: log,
    }
}

// Name 返回任务名称
func (j *MyCustomJob) Name() string {
    return "MyCustomJob"
}

// Schedule 返回调度时间（每 30 分钟执行一次）
func (j *MyCustomJob) Schedule() string {
    return "30m"
}

// Run 执行任务
func (j *MyCustomJob) Run(ctx context.Context) error {
    j.logger.Info("执行自定义任务...")

    // 业务逻辑
    // ...

    j.logger.Info("自定义任务执行完成")
    return nil
}
```

### 步骤 3：注册任务

在 `cmd/bot/main.go` 中注册任务：

```go
// 10. 初始化定时任务调度器
taskScheduler := scheduler.NewScheduler(appLogger)

// 添加定时任务
taskScheduler.AddJob(scheduler.NewCleanupExpiredDataJob(db, appLogger))
taskScheduler.AddJob(scheduler.NewStatisticsReportJob(userRepo, groupRepo, appLogger))
taskScheduler.AddJob(scheduler.NewMyCustomJob(appLogger)) // 新增

appLogger.Info("✅ Scheduler initialized", "jobs", len(taskScheduler.GetJobs()))
```

### 步骤 4：启动调度器

调度器在 `main.go` 中自动启动（第 149 行）：

```go
// 13. 启动定时任务调度器
taskScheduler.Start()
appLogger.Info("✅ Scheduler started")
```

---

## 完整代码示例

### 示例 1：数据清理任务（项目内置）

```go
package scheduler

import (
    "context"
    "fmt"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "telegram-bot/pkg/logger"
)

// CleanupExpiredDataJob 清理过期数据任务
type CleanupExpiredDataJob struct {
    db     *mongo.Database
    logger logger.Logger
}

func NewCleanupExpiredDataJob(db *mongo.Database, log logger.Logger) *CleanupExpiredDataJob {
    return &CleanupExpiredDataJob{
        db:     db,
        logger: log,
    }
}

func (j *CleanupExpiredDataJob) Name() string {
    return "CleanupExpiredData"
}

func (j *CleanupExpiredDataJob) Schedule() string {
    return "1d" // 每天执行一次
}

func (j *CleanupExpiredDataJob) Run(ctx context.Context) error {
    j.logger.Info("Starting cleanup expired data job")

    // 清理过期的警告记录（超过90天）
    warningsDeleted, err := j.cleanupExpiredWarnings(ctx)
    if err != nil {
        j.logger.Error("Failed to cleanup expired warnings", "error", err)
        return fmt.Errorf("cleanup warnings failed: %w", err)
    }

    // 清理不活跃的用户数据（超过180天未活跃）
    usersDeleted, err := j.cleanupInactiveUsers(ctx)
    if err != nil {
        j.logger.Error("Failed to cleanup inactive users", "error", err)
        // 不返回错误，继续执行
    }

    j.logger.Info("Cleanup expired data completed",
        "warnings_deleted", warningsDeleted,
        "users_deleted", usersDeleted,
    )

    return nil
}

func (j *CleanupExpiredDataJob) cleanupExpiredWarnings(ctx context.Context) (int64, error) {
    collection := j.db.Collection("warnings")

    // 删除90天前的警告
    cutoffTime := time.Now().Add(-90 * 24 * time.Hour)
    filter := bson.M{
        "created_at": bson.M{"$lt": cutoffTime},
    }

    result, err := collection.DeleteMany(ctx, filter)
    if err != nil {
        return 0, err
    }

    return result.DeletedCount, nil
}

func (j *CleanupExpiredDataJob) cleanupInactiveUsers(ctx context.Context) (int64, error) {
    collection := j.db.Collection("users")

    // 删除180天未活跃的普通用户（非管理员）
    cutoffTime := time.Now().Add(-180 * 24 * time.Hour)
    filter := bson.M{
        "updated_at": bson.M{"$lt": cutoffTime},
        "permissions": bson.M{
            "$not": bson.M{
                "$elemMatch": bson.M{
                    "level": bson.M{"$gte": 2}, // Admin 及以上不删除
                },
            },
        },
    }

    result, err := collection.DeleteMany(ctx, filter)
    if err != nil {
        return 0, err
    }

    return result.DeletedCount, nil
}
```

### 示例 2：统计报告任务（项目内置）

```go
package scheduler

import (
    "context"
    "time"

    "telegram-bot/internal/domain/group"
    "telegram-bot/internal/domain/user"
    "telegram-bot/pkg/logger"
)

// StatisticsReportJob 统计报告任务
type StatisticsReportJob struct {
    userRepo  user.Repository
    groupRepo group.Repository
    logger    logger.Logger
}

func NewStatisticsReportJob(userRepo user.Repository, groupRepo group.Repository, log logger.Logger) *StatisticsReportJob {
    return &StatisticsReportJob{
        userRepo:  userRepo,
        groupRepo: groupRepo,
        logger:    log,
    }
}

func (j *StatisticsReportJob) Name() string {
    return "StatisticsReport"
}

func (j *StatisticsReportJob) Schedule() string {
    return "1h" // 每小时执行一次
}

func (j *StatisticsReportJob) Run(ctx context.Context) error {
    j.logger.Info("Starting statistics report job")

    stats := map[string]interface{}{
        "timestamp": time.Now(),
    }

    // TODO: 实际统计逻辑
    // totalUsers, _ := j.userRepo.Count(ctx)
    // totalGroups, _ := j.groupRepo.Count(ctx)
    // stats["total_users"] = totalUsers
    // stats["total_groups"] = totalGroups

    j.logger.Info("Statistics report generated", "stats", stats)

    return nil
}
```

### 示例 3：自动解禁任务（项目内置）

```go
package scheduler

import (
    "context"
    "fmt"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "telegram-bot/pkg/logger"
)

// AutoUnbanJob 自动解除临时封禁任务
type AutoUnbanJob struct {
    db     *mongo.Database
    logger logger.Logger
}

func NewAutoUnbanJob(db *mongo.Database, log logger.Logger) *AutoUnbanJob {
    return &AutoUnbanJob{
        db:     db,
        logger: log,
    }
}

func (j *AutoUnbanJob) Name() string {
    return "AutoUnban"
}

func (j *AutoUnbanJob) Schedule() string {
    return "5m" // 每5分钟执行一次
}

func (j *AutoUnbanJob) Run(ctx context.Context) error {
    j.logger.Info("Starting auto unban job")

    collection := j.db.Collection("bans")

    // 查找已过期且未解除的封禁记录
    filter := bson.M{
        "banned_until": bson.M{"$lte": time.Now()},
        "unbanned":     false,
    }

    cursor, err := collection.Find(ctx, filter)
    if err != nil {
        return fmt.Errorf("failed to query expired bans: %w", err)
    }
    defer cursor.Close(ctx)

    var unbanCount int64
    for cursor.Next(ctx) {
        var ban struct {
            ID          interface{} `bson:"_id"`
            UserID      int64       `bson:"user_id"`
            GroupID     int64       `bson:"group_id"`
            BannedUntil time.Time   `bson:"banned_until"`
        }

        if err := cursor.Decode(&ban); err != nil {
            j.logger.Warn("Failed to decode ban record", "error", err)
            continue
        }

        // 标记为已解除
        update := bson.M{
            "$set": bson.M{
                "unbanned":    true,
                "unbanned_at": time.Now(),
            },
        }

        _, err := collection.UpdateOne(ctx, bson.M{"_id": ban.ID}, update)
        if err != nil {
            j.logger.Error("Failed to mark ban as unbanned",
                "user_id", ban.UserID,
                "group_id", ban.GroupID,
                "error", err,
            )
            continue
        }

        j.logger.Info("User auto-unbanned",
            "user_id", ban.UserID,
            "group_id", ban.GroupID,
        )

        unbanCount++
    }

    j.logger.Info("Auto unban job completed", "unban_count", unbanCount)

    return nil
}
```

### 示例 4：使用 SimpleJob（快捷方式）

```go
// 在 main.go 中直接创建简单任务
taskScheduler.AddJob(scheduler.NewSimpleJob(
    "QuickTask",        // 任务名称
    "10m",              // 每10分钟
    func(ctx context.Context) error {
        appLogger.Info("快速任务执行中...")

        // 业务逻辑

        return nil
    },
))
```

---

## 时间表达式

### 支持的格式

定时任务的 `Schedule()` 方法返回时间表达式，支持以下格式：

| 格式 | 说明 | 示例 |
|------|------|------|
| `30s` | 秒 | 每 30 秒执行 |
| `5m` | 分钟 | 每 5 分钟执行 |
| `1h` | 小时 | 每 1 小时执行 |
| `2h30m` | 混合 | 每 2 小时 30 分钟执行 |
| `1d` | 天（特殊支持） | 每 1 天执行 |

### 示例

```go
func (j *MyJob) Schedule() string {
    return "30s"    // 每30秒
    return "1m"     // 每1分钟
    return "5m"     // 每5分钟
    return "15m"    // 每15分钟
    return "30m"    // 每30分钟
    return "1h"     // 每1小时
    return "2h"     // 每2小时
    return "12h"    // 每12小时
    return "1d"     // 每1天（24小时）
    return "7d"     // 每7天
}
```

### 注意事项

- ⚠️ **不支持 cron 表达式**（如 `0 0 * * *`）
- ⚠️ **不支持指定具体时间**（如每天凌晨 3 点）
- ✅ **间隔时间**：从任务**完成**时开始计时下一次执行
- ✅ **立即执行**：任务启动后会立即执行一次，然后按间隔调度

---

## 任务生命周期

### 启动流程

```
1. main.go: 创建 Scheduler
   ↓
2. main.go: 添加 Job (AddJob)
   ↓
3. main.go: 启动调度器 (Start)
   ↓
4. Scheduler: 为每个 Job 启动 goroutine
   ↓
5. Scheduler: 立即执行一次 Job.Run()
   ↓
6. Scheduler: 创建 Ticker，按间隔调度
```

### 执行流程

```
Ticker 触发
   ↓
创建带超时的 Context (5分钟)
   ↓
执行 Job.Run(ctx)
   ↓
记录执行时间和结果
   ↓
如果成功: INFO 日志
如果失败: ERROR 日志
   ↓
等待下一次 Ticker 触发
```

### 关闭流程

```
1. 接收到关闭信号 (SIGINT/SIGTERM)
   ↓
2. taskScheduler.Stop()
   ↓
3. 取消所有任务的 Context
   ↓
4. 等待所有任务完成 (最多30秒)
   ↓
5. 记录日志并退出
```

---

## 实际场景示例

### 场景 1：定时备份数据库

```go
package scheduler

import (
    "context"
    "fmt"
    "os/exec"
    "time"

    "telegram-bot/pkg/logger"
)

type DatabaseBackupJob struct {
    logger     logger.Logger
    mongoURI   string
    backupPath string
}

func NewDatabaseBackupJob(log logger.Logger, mongoURI, backupPath string) *DatabaseBackupJob {
    return &DatabaseBackupJob{
        logger:     log,
        mongoURI:   mongoURI,
        backupPath: backupPath,
    }
}

func (j *DatabaseBackupJob) Name() string {
    return "DatabaseBackup"
}

func (j *DatabaseBackupJob) Schedule() string {
    return "1d" // 每天备份一次
}

func (j *DatabaseBackupJob) Run(ctx context.Context) error {
    j.logger.Info("Starting database backup")

    // 生成备份文件名
    timestamp := time.Now().Format("20060102_150405")
    backupFile := fmt.Sprintf("%s/backup_%s.gz", j.backupPath, timestamp)

    // 执行 mongodump
    cmd := exec.CommandContext(ctx,
        "mongodump",
        "--uri", j.mongoURI,
        "--archive="+backupFile,
        "--gzip",
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        j.logger.Error("Backup failed", "error", err, "output", string(output))
        return fmt.Errorf("mongodump failed: %w", err)
    }

    j.logger.Info("Database backup completed", "file", backupFile)

    // 可选：上传到云存储
    // uploadToS3(backupFile)

    // 可选：清理旧备份（保留最近7天）
    // cleanOldBackups(j.backupPath, 7)

    return nil
}
```

### 场景 2：定时发送通知

```go
package scheduler

import (
    "context"
    "fmt"
    "time"

    "github.com/go-telegram/bot"
    "telegram-bot/pkg/logger"
)

type DailyReminderJob struct {
    logger    logger.Logger
    bot       *bot.Bot
    channelID int64
}

func NewDailyReminderJob(log logger.Logger, b *bot.Bot, channelID int64) *DailyReminderJob {
    return &DailyReminderJob{
        logger:    log,
        bot:       b,
        channelID: channelID,
    }
}

func (j *DailyReminderJob) Name() string {
    return "DailyReminder"
}

func (j *DailyReminderJob) Schedule() string {
    return "1d" // 每天执行
}

func (j *DailyReminderJob) Run(ctx context.Context) error {
    j.logger.Info("Sending daily reminder")

    // 构建消息
    message := fmt.Sprintf(
        "📅 *每日提醒*\n\n"+
            "日期: %s\n"+
            "提醒内容：\n"+
            "• 记得查看待办事项\n"+
            "• 检查系统运行状态\n"+
            "• 回顾今日目标",
        time.Now().Format("2006-01-02"),
    )

    // 发送消息
    _, err := j.bot.SendMessage(ctx, &bot.SendMessageParams{
        ChatID:    j.channelID,
        Text:      message,
        ParseMode: "Markdown",
    })

    if err != nil {
        j.logger.Error("Failed to send reminder", "error", err)
        return fmt.Errorf("send message failed: %w", err)
    }

    j.logger.Info("Daily reminder sent successfully")
    return nil
}
```

### 场景 3：健康检查任务

```go
package scheduler

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "telegram-bot/pkg/logger"
)

type HealthCheckJob struct {
    logger    logger.Logger
    endpoints []string
}

func NewHealthCheckJob(log logger.Logger, endpoints []string) *HealthCheckJob {
    return &HealthCheckJob{
        logger:    log,
        endpoints: endpoints,
    }
}

func (j *HealthCheckJob) Name() string {
    return "HealthCheck"
}

func (j *HealthCheckJob) Schedule() string {
    return "5m" // 每5分钟检查一次
}

func (j *HealthCheckJob) Run(ctx context.Context) error {
    j.logger.Info("Starting health check")

    var failedEndpoints []string

    for _, endpoint := range j.endpoints {
        if err := j.checkEndpoint(ctx, endpoint); err != nil {
            j.logger.Warn("Endpoint unhealthy",
                "endpoint", endpoint,
                "error", err,
            )
            failedEndpoints = append(failedEndpoints, endpoint)
        }
    }

    if len(failedEndpoints) > 0 {
        // 发送告警
        j.logger.Error("Health check failed",
            "failed_count", len(failedEndpoints),
            "failed_endpoints", failedEndpoints,
        )

        // 可选：发送 Telegram 通知给管理员
        // notifyAdmin("健康检查失败: " + strings.Join(failedEndpoints, ", "))

        return fmt.Errorf("%d endpoints unhealthy", len(failedEndpoints))
    }

    j.logger.Info("Health check completed", "all_healthy", true)
    return nil
}

func (j *HealthCheckJob) checkEndpoint(ctx context.Context, endpoint string) error {
    client := &http.Client{Timeout: 10 * time.Second}

    req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
    if err != nil {
        return err
    }

    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("status code: %d", resp.StatusCode)
    }

    return nil
}
```

### 场景 4：缓存预热任务

```go
package scheduler

import (
    "context"

    "telegram-bot/pkg/logger"
)

type CacheWarmupJob struct {
    logger     logger.Logger
    warmupFunc func(ctx context.Context) error
}

func NewCacheWarmupJob(log logger.Logger, warmupFunc func(ctx context.Context) error) *CacheWarmupJob {
    return &CacheWarmupJob{
        logger:     log,
        warmupFunc: warmupFunc,
    }
}

func (j *CacheWarmupJob) Name() string {
    return "CacheWarmup"
}

func (j *CacheWarmupJob) Schedule() string {
    return "30m" // 每30分钟预热一次
}

func (j *CacheWarmupJob) Run(ctx context.Context) error {
    j.logger.Info("Starting cache warmup")

    if j.warmupFunc != nil {
        if err := j.warmupFunc(ctx); err != nil {
            return fmt.Errorf("cache warmup failed: %w", err)
        }
    }

    j.logger.Info("Cache warmup completed")
    return nil
}
```

---

## 最佳实践

### 1. 超时控制

每个任务最多执行 **5 分钟**（在 `scheduler.go:150` 定义）。

```go
func (j *MyJob) Run(ctx context.Context) error {
    // 检查 context 是否已取消
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // 长时间操作应该检查 context
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            process(item)
        }
    }

    return nil
}
```

### 2. 错误处理

```go
func (j *MyJob) Run(ctx context.Context) error {
    // ✅ 推荐：捕获 panic
    defer func() {
        if r := recover(); r != nil {
            j.logger.Error("Job panic", "panic", r)
        }
    }()

    // ✅ 推荐：记录详细错误
    if err := j.doSomething(); err != nil {
        j.logger.Error("Operation failed", "error", err)
        return fmt.Errorf("doSomething failed: %w", err)
    }

    // ✅ 推荐：部分失败不中断
    errors := 0
    for _, item := range items {
        if err := process(item); err != nil {
            j.logger.Warn("Item processing failed", "item", item, "error", err)
            errors++
        }
    }

    if errors > 0 {
        j.logger.Warn("Job completed with errors", "error_count", errors)
        // 不返回 error，避免影响下次执行
    }

    return nil
}
```

### 3. 并发安全

```go
type MyJob struct {
    logger logger.Logger
    cache  map[string]int
    mu     sync.RWMutex  // 保护共享数据
}

func (j *MyJob) Run(ctx context.Context) error {
    j.mu.Lock()
    defer j.mu.Unlock()

    // 修改共享数据
    j.cache["key"]++

    return nil
}
```

### 4. 数据库操作

```go
func (j *MyJob) Run(ctx context.Context) error {
    // ✅ 使用带超时的 context
    collection := j.db.Collection("users")

    // ✅ 批量操作
    bulkOps := []mongo.WriteModel{}
    for _, item := range items {
        bulkOps = append(bulkOps, mongo.NewDeleteOneModel().SetFilter(
            bson.M{"_id": item.ID},
        ))
    }

    if len(bulkOps) > 0 {
        result, err := collection.BulkWrite(ctx, bulkOps)
        if err != nil {
            return err
        }
        j.logger.Info("Bulk operation completed", "deleted", result.DeletedCount)
    }

    return nil
}
```

### 5. 资源清理

```go
func (j *MyJob) Run(ctx context.Context) error {
    // ✅ 使用 defer 确保资源释放
    file, err := os.Open("data.txt")
    if err != nil {
        return err
    }
    defer file.Close()

    // ✅ 数据库游标
    cursor, err := collection.Find(ctx, filter)
    if err != nil {
        return err
    }
    defer cursor.Close(ctx)

    return nil
}
```

### 6. 幂等性

```go
func (j *MyJob) Run(ctx context.Context) error {
    // ✅ 确保任务可以重复执行
    // 例如：使用唯一标识避免重复处理

    // 检查是否已处理
    processed, err := j.isProcessed(ctx, recordID)
    if err != nil {
        return err
    }
    if processed {
        j.logger.Info("Record already processed", "id", recordID)
        return nil
    }

    // 处理记录
    if err := j.processRecord(ctx, recordID); err != nil {
        return err
    }

    // 标记为已处理
    return j.markAsProcessed(ctx, recordID)
}
```

---

## 测试方法

### 1. 单元测试

创建 `internal/scheduler/jobs_test.go`：

```go
package scheduler

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock Logger
type MockLogger struct {
    mock.Mock
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
    m.Called(msg, fields)
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
    m.Called(msg, fields)
}

func TestMyJob_Run(t *testing.T) {
    mockLogger := new(MockLogger)
    mockLogger.On("Info", mock.Anything, mock.Anything).Return()

    job := NewMyCustomJob(mockLogger)

    ctx := context.Background()
    err := job.Run(ctx)

    assert.NoError(t, err)
    mockLogger.AssertExpectations(t)
}

func TestMyJob_Schedule(t *testing.T) {
    job := NewMyCustomJob(nil)
    assert.Equal(t, "30m", job.Schedule())
}

func TestMyJob_Name(t *testing.T) {
    job := NewMyCustomJob(nil)
    assert.Equal(t, "MyCustomJob", job.Name())
}
```

运行测试：

```bash
go test ./internal/scheduler/... -v
```

### 2. 集成测试

```go
func TestScheduler_Integration(t *testing.T) {
    logger := logger.New(logger.Config{Level: logger.DebugLevel})
    scheduler := NewScheduler(logger)

    // 创建测试任务
    executed := false
    job := NewSimpleJob("TestJob", "1s", func(ctx context.Context) error {
        executed = true
        return nil
    })

    scheduler.AddJob(job)
    scheduler.Start()

    // 等待任务执行
    time.Sleep(2 * time.Second)

    scheduler.Stop()

    assert.True(t, executed, "Job should have been executed")
}
```

### 3. 手动触发测试

在开发时，可以临时修改 `Schedule()` 返回值为短间隔（如 `10s`）进行快速测试：

```go
func (j *MyJob) Schedule() string {
    // 开发时使用
    return "10s"

    // 生产环境使用
    // return "1d"
}
```

或者直接调用 `Run()` 方法：

```go
func main() {
    logger := logger.New(logger.Config{Level: logger.DebugLevel})
    job := NewMyCustomJob(logger)

    ctx := context.Background()
    if err := job.Run(ctx); err != nil {
        log.Fatalf("Job failed: %v", err)
    }
}
```

---

## 常见问题

### Q1：定时任务失败会影响下次执行吗？

**不会**。任务失败只会记录错误日志，不影响下次调度。

### Q2：如何实现每天凌晨 3 点执行？

目前框架**不支持指定具体时间**。推荐方案：

1. **使用系统 cron**（推荐）
2. **修改调度器**：添加对 cron 表达式的支持（可使用 `github.com/robfig/cron` 库）

### Q3：任务执行时间超过间隔怎么办？

任务从**完成时**开始计时下一次执行，不会重叠。

例如：任务每 5 分钟执行，但某次执行了 8 分钟，下次执行时间是 **8 分钟后 + 5 分钟 = 13 分钟后**。

### Q4：如何动态添加/删除任务?

目前**不支持运行时动态修改**。任务在调度器启动时注册，运行期间不可变。

如需动态任务，可以：
1. 创建一个"任务调度器任务"，动态从数据库读取任务列表并执行
2. 修改 `Scheduler` 添加 `RemoveJob()` 和 `AddJob()` 的并发安全版本

### Q5：任务可以访问 Bot 实例吗？

可以，通过构造函数传入：

```go
type MyJob struct {
    bot    *bot.Bot
    logger logger.Logger
}

func NewMyJob(b *bot.Bot, log logger.Logger) *MyJob {
    return &MyJob{
        bot:    b,
        logger: log,
    }
}

// 在 main.go 中注册
taskScheduler.AddJob(scheduler.NewMyJob(telegramBot, appLogger))
```

### Q6：如何查看任务执行历史？

当前只有日志记录。如需持久化历史，可以：

1. **修改 `executeJob()`**：在任务完成后写入数据库
2. **创建任务执行记录表**：
   ```go
   type JobExecution struct {
       JobName   string
       StartTime time.Time
       EndTime   time.Time
       Duration  time.Duration
       Success   bool
       Error     string
   }
   ```

### Q7：如何临时禁用某个任务？

**方式 1**：注释掉注册代码

```go
// taskScheduler.AddJob(scheduler.NewMyJob(logger))
```

**方式 2**：添加开关控制

```go
if os.Getenv("ENABLE_MY_JOB") == "true" {
    taskScheduler.AddJob(scheduler.NewMyJob(logger))
}
```

---

## 附录

### 相关资源

- [Scheduler 源码](../internal/scheduler/scheduler.go)
- [内置任务示例](../internal/scheduler/jobs.go)
- [主程序入口](../cmd/bot/main.go) (第 125-150 行)

### 相关文档

- [架构总览](./CLAUDE.md)
- [命令处理器开发指南](./handlers/command-handler-guide.md)
- [监听器开发指南](./handlers/listener-handler-guide.md)

### 扩展阅读

- [robfig/cron](https://github.com/robfig/cron) - 如需支持 cron 表达式
- [Go Context 包文档](https://pkg.go.dev/context)
- [Go Time 包文档](https://pkg.go.dev/time)

---

**编写日期**: 2025-10-02
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team
