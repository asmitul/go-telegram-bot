# Logger Package

结构化日志包，支持多级别、多格式输出。

## 特性

- ✅ 四个日志级别：Debug、Info、Warn、Error
- ✅ 两种输出格式：Text（文本）、JSON
- ✅ 结构化字段支持
- ✅ 线程安全
- ✅ 字段继承（WithField/WithFields）
- ✅ 可配置日志级别过滤
- ✅ 100% 测试覆盖率

## 快速开始

### 基本使用

```go
import "telegram-bot/pkg/logger"

// 创建默认 Logger (Text 格式, Info 级别)
log := logger.Default()

// 记录日志
log.Info("application started")
log.Error("error occurred", "error", "connection timeout")
```

### 创建自定义 Logger

```go
// Text 格式
log := logger.New(logger.Config{
    Level:  logger.LevelDebug,
    Format: "text",
})

// JSON 格式
log := logger.New(logger.Config{
    Level:  logger.LevelInfo,
    Format: "json",
})
```

### 使用结构化字段

```go
log := logger.Default()

// 单个字段
log.Info("user action", "user_id", 12345)

// 多个字段
log.Info("request",
    "method", "POST",
    "path", "/api/users",
    "status", 200)
```

### 字段继承

```go
log := logger.Default()

// 创建带有持久字段的 logger
userLog := log.WithField("user_id", 12345)

// 所有后续日志都包含 user_id
userLog.Info("login")  // 输出包含 user_id=12345
userLog.Info("logout") // 输出包含 user_id=12345

// 添加多个持久字段
requestLog := log.WithFields(map[string]interface{}{
    "request_id": "req-123",
    "user_id":    12345,
})
```

## 日志级别

```go
logger.LevelDebug  // 调试信息
logger.LevelInfo   // 一般信息
logger.LevelWarn   // 警告
logger.LevelError  // 错误
```

### 级别过滤

```go
log := logger.NewWithLevel(logger.LevelWarn)

log.Debug("debug") // 不输出
log.Info("info")   // 不输出
log.Warn("warn")   // 输出
log.Error("error") // 输出
```

### 从字符串解析级别

```go
level := logger.ParseLevel("debug")  // 支持: debug, info, warn, error
log := logger.NewWithLevel(level)
```

## 输出格式

### Text 格式

```
[2025-09-30 14:45:23] [INFO] application started
[2025-09-30 14:45:23] [ERROR] connection failed error=timeout retry=3
```

### JSON 格式

```json
{"time":"2025-09-30T14:45:23Z","level":"INFO","msg":"application started"}
{"time":"2025-09-30T14:45:23Z","level":"ERROR","msg":"connection failed","fields":{"error":"timeout","retry":3}}
```

## 配置选项

```go
type Config struct {
    Level      Level      // 日志级别
    Format     string     // "text" 或 "json"
    Output     io.Writer  // 输出目标 (默认: os.Stdout)
    TimeFormat string     // 时间格式 (默认: "2006-01-02 15:04:05")
}
```

## 与现有代码集成

### 在 main.go 中使用

```go
import (
    "telegram-bot/pkg/logger"
    "telegram-bot/internal/config"
)

func main() {
    cfg, _ := config.Load()

    // 根据配置创建 Logger
    log := logger.New(logger.Config{
        Level:  logger.ParseLevel(cfg.LogLevel),
        Format: cfg.LogFormat,
    })

    log.Info("bot started", "version", "1.0.0")
}
```

### 在 Middleware 中使用

```go
type Logger interface {
    Debug(msg string, fields ...interface{})
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
}

middleware := NewLoggingMiddleware(log)
```

## 环境变量配置

在 `.env` 文件中配置：

```bash
# 日志级别: debug, info, warn, error
LOG_LEVEL=info

# 日志格式: text, json
LOG_FORMAT=text
```

## 最佳实践

1. **使用结构化字段而非字符串拼接**
   ```go
   // ✅ 好
   log.Info("user login", "user_id", 12345, "ip", "192.168.1.1")

   // ❌ 差
   log.Info(fmt.Sprintf("user %d login from %s", 12345, "192.168.1.1"))
   ```

2. **为不同模块创建带上下文的 logger**
   ```go
   // 在服务初始化时
   userService := NewUserService(log.WithField("service", "user"))

   // 在服务中使用
   func (s *UserService) CreateUser() {
       s.log.Info("creating user") // 自动包含 service=user
   }
   ```

3. **生产环境使用 Info 级别，开发环境使用 Debug**
   ```go
   level := logger.LevelInfo
   if cfg.IsDevelopment() {
       level = logger.LevelDebug
   }
   log := logger.NewWithLevel(level)
   ```

4. **JSON 格式用于集中式日志收集**
   ```go
   // 生产环境，便于 ELK/Splunk 等工具解析
   log := logger.New(logger.Config{
       Level:  logger.LevelInfo,
       Format: "json",
   })
   ```

## 测试

运行测试：

```bash
go test ./pkg/logger/
go test -cover ./pkg/logger/
```

## 线程安全

Logger 的所有方法都是线程安全的，可以在多个 goroutine 中并发使用：

```go
log := logger.Default()

for i := 0; i < 10; i++ {
    go func(id int) {
        log.Info("goroutine", "id", id)
    }(i)
}
```