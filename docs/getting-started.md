# 项目快速入门指南

## 📚 目录

- [项目简介](#项目简介)
- [环境准备](#环境准备)
- [快速开始](#快速开始)
- [项目结构](#项目结构)
- [开发工作流](#开发工作流)
- [核心概念](#核心概念)
- [第一个功能](#第一个功能)
- [常用命令](#常用命令)
- [调试技巧](#调试技巧)
- [下一步学习](#下一步学习)

---

## 项目简介

这是一个**生产级 Telegram 机器人框架**，采用 Go 语言开发，核心特性包括：

### ✨ 核心亮点

- **🎯 统一消息处理架构**：4 种处理器（命令、关键词、正则、监听器）
- **🔐 完整的权限系统**：4 级权限，按群组隔离
- **🛡️ 健全的中间件**：错误恢复、日志、权限、限流
- **📊 生产级监控**：Prometheus + Grafana
- **⚡ 优雅的设计**：清晰的架构，易于扩展

### 🎓 适合谁？

- ✅ 想要快速开发 Telegram 机器人的 Go 开发者
- ✅ 需要生产级架构的机器人项目
- ✅ 学习消息处理框架设计的开发者

---

## 环境准备

### 必需软件

| 软件 | 版本要求 | 用途 | 安装验证 |
|------|---------|------|---------|
| **Go** | 1.21+ | 编译运行 | `go version` |
| **MongoDB** | 4.4+ | 数据存储 | `mongod --version` |
| **Git** | 任意 | 版本控制 | `git --version` |

### 推荐软件

| 软件 | 用途 | 安装验证 |
|------|------|---------|
| **Docker** | 容器化部署 | `docker --version` |
| **Docker Compose** | 多容器编排 | `docker-compose --version` |
| **Make** | 构建工具 | `make --version` |
| **Air** | 热重载（可选） | `air -v` |

### 安装 Go（如果未安装）

**macOS/Linux:**
```bash
# macOS (使用 Homebrew)
brew install go

# Ubuntu/Debian
sudo apt update
sudo apt install golang-go

# 验证安装
go version  # 应该显示 go1.21 或更高版本
```

**Windows:**
1. 下载安装包：https://go.dev/dl/
2. 安装后验证：`go version`

### 安装 MongoDB（如果未安装）

**macOS:**
```bash
brew tap mongodb/brew
brew install mongodb-community@7.0
brew services start mongodb-community@7.0
```

**Ubuntu/Debian:**
```bash
sudo apt install mongodb
sudo systemctl start mongodb
sudo systemctl enable mongodb
```

**Windows:**
1. 下载安装包：https://www.mongodb.com/try/download/community
2. 安装并启动服务

**使用 Docker（推荐）:**
```bash
docker run -d -p 27017:27017 --name mongo mongo:7.0
```

---

## 快速开始

### 1. 克隆项目

```bash
git clone <your-repo-url>
cd go-telegram-bot
```

### 2. 配置环境变量

```bash
# 复制配置模板
cp .env.example .env

# 编辑 .env 文件
nano .env  # 或使用你喜欢的编辑器
```

**必填配置**：
```bash
# .env 文件
TELEGRAM_TOKEN=<你的_bot_token>  # 从 @BotFather 获取
MONGO_URI=mongodb://localhost:27017
DATABASE_NAME=telegram_bot
LOG_LEVEL=info
LOG_FORMAT=json
```

**如何获取 Bot Token**：
1. 在 Telegram 中搜索 [@BotFather](https://t.me/BotFather)
2. 发送 `/newbot`
3. 按提示设置机器人名称和用户名
4. 获得类似 `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz` 的 Token
5. 复制到 `.env` 文件的 `TELEGRAM_TOKEN` 中

### 3. 安装依赖

```bash
# 下载 Go 模块依赖
go mod download

# 安装开发工具（可选）
make install-tools
```

### 4. 运行项目

**方式 1：直接运行（推荐新手）**
```bash
go run ./cmd/bot
```

**方式 2：使用 Make**
```bash
make run
```

**方式 3：编译后运行**
```bash
make build
./bin/bot
```

**方式 4：使用 Docker（推荐生产环境）**
```bash
# 启动所有服务（Bot + MongoDB + Prometheus + Grafana）
make docker-up

# 查看日志
make docker-logs

# 停止服务
make docker-down
```

### 5. 测试机器人

启动成功后，在 Telegram 中：

1. 搜索你的机器人（用户名）
2. 点击 "Start" 或发送 `/start`
3. 尝试以下命令：
   - `/ping` - 测试机器人是否在线
   - `/help` - 查看帮助信息
   - `你好` - 触发问候语（仅私聊）
   - `天气 北京` - 触发正则匹配（示例）

**日志输出示例**：
```
INFO  🚀 Bot starting... version=2.0.0
INFO  ✅ MongoDB connected successfully
INFO  ✅ Database indexes created
INFO  ✅ Middlewares registered
INFO  ✅ Handlers registered count=7
INFO  ✅ Scheduler initialized jobs=2
INFO  ✅ Bot is running
```

---

## 项目结构

### 核心目录

```
go-telegram-bot/
├── cmd/                     # 应用入口
│   └── bot/
│       └── main.go          # 主程序入口
│
├── internal/                # 内部代码（不对外暴露）
│   ├── handler/             # 🌟 核心框架
│   │   ├── handler.go       # Handler 接口定义
│   │   ├── context.go       # 消息上下文
│   │   └── router.go        # 路由器
│   │
│   ├── handlers/            # 🌟 处理器实现
│   │   ├── command/         # 命令处理器 (/ping, /help)
│   │   ├── keyword/         # 关键词处理器 (你好, 谢谢)
│   │   ├── pattern/         # 正则处理器 (天气查询)
│   │   └── listener/        # 监听器 (日志, 统计)
│   │
│   ├── middleware/          # 🌟 中间件
│   │   ├── recovery.go      # 错误恢复
│   │   ├── logging.go       # 日志记录
│   │   ├── permission.go    # 权限检查
│   │   └── ratelimit.go     # 限流控制
│   │
│   ├── domain/              # 领域模型
│   │   ├── user/            # 用户实体
│   │   └── group/           # 群组实体
│   │
│   ├── adapter/             # 外部适配器
│   │   ├── repository/      # 数据库访问
│   │   └── telegram/        # Telegram API 适配
│   │
│   ├── scheduler/           # 🌟 定时任务
│   │   ├── scheduler.go     # 调度器
│   │   └── jobs.go          # 任务定义
│   │
│   └── config/              # 配置加载
│       └── config.go
│
├── pkg/                     # 可复用的包（可对外暴露）
│   └── logger/              # 日志工具
│
├── test/                    # 测试文件
│   ├── mocks/               # Mock 对象
│   └── integration/         # 集成测试
│
├── deployments/             # 部署配置
│   └── docker/              # Docker 相关
│       ├── Dockerfile
│       └── docker-compose.yml
│
├── monitoring/              # 监控配置
│   ├── prometheus/          # Prometheus 配置
│   └── grafana/             # Grafana 仪表板
│
├── docs/                    # 📖 文档
│   ├── getting-started.md   # 本文档
│   ├── handlers/            # 处理器开发指南
│   └── scheduler-guide.md   # 定时任务指南
│
├── .env.example             # 环境变量模板
├── Makefile                 # Make 命令定义
├── go.mod                   # Go 模块定义
├── CLAUDE.md                # Claude Code 项目说明
└── README.md                # 项目说明
```

### 文件说明

| 路径 | 作用 | 重要性 |
|------|------|--------|
| `cmd/bot/main.go` | 程序入口，初始化所有组件 | ⭐⭐⭐⭐⭐ |
| `internal/handler/` | 核心框架，理解消息处理机制 | ⭐⭐⭐⭐⭐ |
| `internal/handlers/` | 实际的功能实现 | ⭐⭐⭐⭐⭐ |
| `internal/middleware/` | 全局中间件 | ⭐⭐⭐⭐ |
| `internal/domain/` | 业务实体 | ⭐⭐⭐⭐ |
| `internal/scheduler/` | 定时任务 | ⭐⭐⭐ |

---

## 开发工作流

### 典型的开发流程

```
1. 创建新分支
   ↓
2. 修改代码
   ↓
3. 运行测试
   ↓
4. 本地验证
   ↓
5. 提交代码
   ↓
6. 推送到远程
   ↓
7. 创建 Pull Request
```

### 详细步骤

#### 1. 创建功能分支

```bash
git checkout -b feature/my-new-feature
```

#### 2. 开发新功能

参考 [第一个功能](#第一个功能) 章节

#### 3. 格式化代码

```bash
make fmt
```

#### 4. 运行代码检查

```bash
make lint
```

#### 5. 运行测试

```bash
# 运行所有测试
make test

# 只运行单元测试
make test-unit

# 生成覆盖率报告
make test-coverage
```

#### 6. 本地验证

```bash
# 启动机器人
make run

# 在 Telegram 中测试功能
```

#### 7. 提交代码

```bash
git add .
git commit -m "feat: add my new feature"
```

**提交信息规范**（推荐）：
- `feat:` - 新功能
- `fix:` - 修复 Bug
- `docs:` - 文档更新
- `refactor:` - 重构代码
- `test:` - 添加测试
- `chore:` - 构建工具或辅助工具变动

#### 8. 推送到远程

```bash
git push origin feature/my-new-feature
```

#### 9. 创建 Pull Request

在 GitHub/GitLab 上创建 PR，等待 Code Review

---

## 核心概念

### 1. Handler（处理器）

所有消息处理逻辑都是 Handler，实现 4 个方法：

```go
type Handler interface {
    Match(ctx *Context) bool      // 是否匹配这条消息
    Handle(ctx *Context) error    // 如何处理这条消息
    Priority() int                // 优先级（数字越小越高）
    ContinueChain() bool          // 是否继续执行后续处理器
}
```

### 2. Context（上下文）

每条消息都有一个 Context，包含：

```go
type Context struct {
    // 消息信息
    Text      string
    MessageID int

    // 用户信息
    UserID    int64
    Username  string
    FirstName string

    // 聊天信息
    ChatID    int64
    ChatType  string  // "private", "group", "supergroup"

    // 辅助方法
    Reply(text string) error
    IsPrivate() bool
    HasPermission(perm Permission) bool
}
```

### 3. Router（路由器）

Router 负责：
1. 收集所有 Handler
2. 按优先级排序
3. 逐个执行匹配的 Handler
4. 应用中间件

### 4. Middleware（中间件）

中间件包装 Handler，在执行前后添加逻辑：

```
Request → Recovery → Logging → Permission → Handler → Response
```

### 5. 处理器优先级

| 优先级范围 | 处理器类型 | 示例 |
|-----------|-----------|------|
| 100-199 | 命令 | `/ping`, `/help` |
| 200-299 | 关键词 | "你好", "谢谢" |
| 300-399 | 正则 | "天气 北京" |
| 900-999 | 监听器 | 日志记录 |

---

## 第一个功能

让我们创建一个 `/version` 命令来显示机器人版本。

### 步骤 1：创建命令处理器

创建文件 `internal/handlers/command/version.go`：

```go
package command

import (
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

type VersionHandler struct {
	*BaseCommand
}

func NewVersionHandler(groupRepo GroupRepository) *VersionHandler {
	return &VersionHandler{
		BaseCommand: NewBaseCommand(
			"version",                                  // 命令名
			"查看机器人版本",                              // 描述
			user.PermissionUser,                        // 所需权限（所有人可用）
			[]string{"private", "group", "supergroup"}, // 支持的聊天类型
			groupRepo,
		),
	}
}

func (h *VersionHandler) Handle(ctx *handler.Context) error {
	// 检查权限
	if err := h.CheckPermission(ctx); err != nil {
		return err
	}

	// 返回版本信息
	return ctx.Reply("🤖 Bot Version: v2.0.0\n✅ Status: Running")
}
```

### 步骤 2：注册处理器

编辑 `cmd/bot/main.go`，在 `registerHandlers()` 函数中添加：

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
	router.Register(command.NewVersionHandler(groupRepo))  // 新增这一行

	// ... 其他处理器
}
```

### 步骤 3：测试

```bash
# 重启机器人
make run

# 在 Telegram 中发送
/version
```

**预期输出**：
```
🤖 Bot Version: v2.0.0
✅ Status: Running
```

### 🎉 恭喜！

你已经成功创建了第一个功能！

---

## 常用命令

### 开发命令

```bash
make help           # 查看所有可用命令
make run            # 运行机器人
make build          # 编译为二进制文件
make test           # 运行测试
make fmt            # 格式化代码
make lint           # 代码检查
```

### Docker 命令

```bash
make docker-up      # 启动所有服务
make docker-down    # 停止所有服务
make docker-logs    # 查看日志
make docker-restart # 重启机器人
make docker-clean   # 清理所有容器和数据
```

### 测试命令

```bash
make test                # 所有测试
make test-unit           # 单元测试
make test-integration    # 集成测试
make test-coverage       # 生成覆盖率报告
```

### 其他命令

```bash
make deps               # 下载依赖
make clean              # 清理构建产物
make install-tools      # 安装开发工具
```

---

## 调试技巧

### 1. 查看日志

**本地运行**：
```bash
# 日志直接输出到终端
make run
```

**Docker 运行**：
```bash
# 实时查看日志
make docker-logs

# 查看最近 100 行
docker-compose -f deployments/docker/docker-compose.yml logs --tail=100 bot
```

### 2. 调整日志级别

编辑 `.env` 文件：

```bash
# 开发环境：显示详细日志
LOG_LEVEL=debug
LOG_FORMAT=text

# 生产环境：精简日志
LOG_LEVEL=info
LOG_FORMAT=json
```

### 3. 使用 Delve 调试器

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试
dlv debug ./cmd/bot

# 在代码中设置断点
(dlv) break main.main
(dlv) continue
```

### 4. 添加调试日志

在代码中添加日志：

```go
func (h *MyHandler) Handle(ctx *handler.Context) error {
    appLogger.Debug("Debug info", "user_id", ctx.UserID, "text", ctx.Text)
    appLogger.Info("Processing message", "text", ctx.Text)
    appLogger.Warn("Warning message")
    appLogger.Error("Error occurred", "error", err)

    return nil
}
```

### 5. 测试单个处理器

```bash
# 只运行特定测试
go test -v ./internal/handlers/command/ -run TestVersionHandler
```

### 6. MongoDB 数据检查

```bash
# 进入 MongoDB 容器
docker exec -it <container_id> mongosh

# 查看数据库
use telegram_bot

# 查看用户
db.users.find().pretty()

# 查看群组
db.groups.find().pretty()
```

---

## 下一步学习

### 📖 推荐学习路径

1. **理解核心概念**（你在这里）✅
   - ✅ 项目结构
   - ✅ 开发工作流
   - ✅ 第一个功能

2. **深入处理器开发**
   - 📄 [命令处理器开发指南](./handlers/command-handler-guide.md)
   - 📄 [关键词处理器开发指南](./handlers/keyword-handler-guide.md)
   - 📄 [正则匹配处理器开发指南](./handlers/pattern-handler-guide.md)
   - 📄 [监听器开发指南](./handlers/listener-handler-guide.md)

3. **高级功能**
   - 📄 [定时任务开发指南](./scheduler-guide.md)
   - 📄 中间件开发指南（即将推出）
   - 📄 Repository 开发指南（即将推出）

4. **部署上线**
   - 📄 部署运维指南（即将推出）
   - 📄 监控告警指南（即将推出）

### 🎯 学习建议

**第 1 周**：
- ✅ 搭建开发环境
- ✅ 运行并理解现有功能
- ✅ 创建 1-2 个简单命令

**第 2 周**：
- 深入学习 4 种处理器类型
- 创建关键词和正则处理器
- 理解权限系统

**第 3 周**：
- 学习中间件开发
- 添加定时任务
- 编写单元测试

**第 4 周**：
- 部署到测试环境
- 配置监控告警
- 性能优化

### 📚 相关文档

- [CLAUDE.md](../CLAUDE.md) - 项目架构总览
- [README.md](../README.md) - 项目说明
- [处理器开发指南](./handlers/) - 详细的开发文档

### 🤝 获取帮助

- **文档问题**：查看 `docs/` 目录下的详细文档
- **代码问题**：参考 `internal/` 目录下的现有实现
- **Bug 反馈**：提交 GitHub Issue
- **功能建议**：提交 GitHub Issue 或 Pull Request

---

## 常见问题

### Q1：机器人无法启动？

**检查清单**：
1. ✅ MongoDB 是否运行？`docker ps` 或 `systemctl status mongodb`
2. ✅ `.env` 文件中的 `TELEGRAM_TOKEN` 是否正确？
3. ✅ 网络是否可以访问 Telegram API？

### Q2：命令无响应？

**排查步骤**：
1. 检查日志：`make docker-logs` 或终端输出
2. 确认命令已注册：查看启动日志中的 "Handlers registered"
3. 确认权限：检查用户是否有足够权限

### Q3：如何修改日志格式？

编辑 `.env`：
```bash
# 方式 1：JSON 格式（生产环境）
LOG_FORMAT=json

# 方式 2：文本格式（开发环境）
LOG_FORMAT=text
```

### Q4：如何禁用某个功能？

注释掉 `cmd/bot/main.go` 中的注册代码：

```go
// router.Register(command.NewStatsHandler(groupRepo, userRepo))
```

### Q5：测试失败怎么办？

```bash
# 查看详细错误
make test

# 只运行失败的测试
go test -v ./path/to/package -run TestFailedFunc

# 清理并重试
make clean
go mod tidy
make test
```

---

## 附录

### 环境变量完整列表

| 变量名 | 必需 | 默认值 | 说明 |
|--------|-----|--------|------|
| `TELEGRAM_TOKEN` | ✅ | 无 | Bot API Token |
| `MONGO_URI` | ✅ | `mongodb://localhost:27017` | MongoDB 连接串 |
| `DATABASE_NAME` | ❌ | `telegram_bot` | 数据库名称 |
| `LOG_LEVEL` | ❌ | `info` | 日志级别 (debug/info/warn/error) |
| `LOG_FORMAT` | ❌ | `json` | 日志格式 (json/text) |

### Make 命令速查表

| 命令 | 作用 | 使用场景 |
|------|------|---------|
| `make run` | 运行机器人 | 开发时快速启动 |
| `make build` | 编译二进制 | 生成可执行文件 |
| `make test` | 运行测试 | 验证代码正确性 |
| `make fmt` | 格式化代码 | 提交前规范代码 |
| `make lint` | 代码检查 | 发现潜在问题 |
| `make docker-up` | 启动所有服务 | Docker 环境开发 |
| `make clean` | 清理产物 | 重新构建 |

---

**编写日期**: 2025-10-02
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team

🎉 **欢迎加入开发！** 如有问题，请查阅详细文档或提交 Issue。

