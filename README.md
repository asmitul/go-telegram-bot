<div align="center">

# 🤖 Telegram Bot Framework

**生产级 Telegram 机器人开发框架 · 统一消息处理架构**

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/go-telegram-bot)](https://goreportcard.com/report/github.com/yourusername/go-telegram-bot)
[![Code Coverage](https://img.shields.io/badge/coverage-85%25-brightgreen)](coverage.html)

[English](README.md) | [中文文档](README_CN.md)

</div>

---

## 📖 目录

- [为什么选择这个框架？](#为什么选择这个框架)
- [核心特性](#核心特性)
- [架构设计](#架构设计)
- [快速开始](#快速开始)
- [命令列表](#命令列表)
- [开发指南](#开发指南)
- [部署方案](#部署方案)
- [项目统计](#项目统计)
- [完整文档](#完整文档)
- [常见问题](#常见问题)
- [贡献指南](#贡献指南)

---

## 🌟 为什么选择这个框架？

### 传统方式 vs 本框架

<table>
<tr>
<th>传统 Telegram Bot 开发</th>
<th>使用本框架</th>
</tr>
<tr>
<td>

```go
// ❌ 混乱的 if-else 判断
if strings.HasPrefix(msg, "/start") {
    // 处理 start
} else if strings.HasPrefix(msg, "/help") {
    // 处理 help
} else if strings.Contains(msg, "你好") {
    // 处理问候
}
// ... 无穷无尽的判断
```

</td>
<td>

```go
// ✅ 清晰的 Handler 架构
router.Register(command.NewStartHandler(repo))
router.Register(command.NewHelpHandler(repo))
router.Register(keyword.NewGreetingHandler())
// 自动路由、优先级排序
```

</td>
</tr>
<tr>
<td>

- ❌ 代码混乱，难以维护
- ❌ 没有权限系统
- ❌ 缺少错误处理
- ❌ 无法扩展

</td>
<td>

- ✅ 架构清晰，易于扩展
- ✅ 内置 4 级权限系统
- ✅ 完善的中间件和错误恢复
- ✅ 生产级代码质量

</td>
</tr>
</table>

### 核心优势

🎯 **统一消息处理架构** - 4 种处理器类型（命令、关键词、正则、监听器），自动路由和优先级排序

🔐 **完整的权限系统** - User/Admin/SuperAdmin/Owner 四级权限，按群组隔离

🛡️ **健全的中间件** - 错误恢复、日志记录、权限加载、限流保护

⚡ **生产可用** - 优雅关闭、健康检查、性能优化、完整测试（85%+ 覆盖率）

📚 **文档齐全** - 15+ 篇详细文档，从入门到精通

---

## ✨ 核心特性

### 🎯 统一消息处理

- **四种处理器类型**
  - 📝 **命令处理器** (Priority: 100-199) - 处理 `/command` 格式命令
  - 🔍 **关键词处理器** (Priority: 200-299) - 自然语言关键词检测
  - 🎨 **正则处理器** (Priority: 300-399) - 复杂模式匹配和信息提取
  - 👂 **监听器** (Priority: 900-999) - 日志记录、数据统计

- **灵活匹配机制** - 每个处理器自主决定是否处理消息
- **优先级控制** - 自动按优先级排序，支持链式执行或中断
- **全聊天类型支持** - 私聊、群组、超级群组、频道

### 🔐 权限系统

| 权限级别 | 数值 | 说明 | 能力 |
|---------|------|------|------|
| 👤 **User** | 1 | 普通用户（默认） | 使用基础命令 |
| 🛡️ **Admin** | 2 | 管理员 | 管理群组内容 |
| ⚡ **SuperAdmin** | 3 | 超级管理员 | 提升/降低权限 |
| 👑 **Owner** | 4 | 所有者 | 完全控制 |

**核心特性**：
- ✅ **按群组隔离** - 同一用户在不同群组可拥有不同权限
- ✅ **自动加载** - 中间件自动从数据库加载用户权限
- ✅ **便捷检查** - `ctx.HasPermission()` 和 `ctx.RequirePermission()`
- ✅ **管理命令** - `/promote`, `/demote`, `/setperm`, `/listadmins`

### 🛡️ 中间件系统

```go
// 洋葱模型 - 层层包装，职责清晰
Request → Recovery → Logging → Permission → Handler → Response
```

| 中间件 | 功能 | 优势 |
|--------|------|------|
| **Recovery** | 捕获 panic | 防止程序崩溃 |
| **Logging** | 记录所有消息 | 审计和调试 |
| **Permission** | 自动加载用户 | 无需手动查询 |
| **RateLimit** | 令牌桶限流 | 防止滥用 |

### 🏗️ 架构设计

采用 **Handler 模式** + **中间件链** 的清晰架构：

```
┌─────────────────────────────────────────────────────────┐
│                   Telegram Update                        │
└─────────────────┬───────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────┐
│              Converter (Update → Context)                │
└─────────────────┬───────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────┐
│                    Router.Route()                        │
│  • 获取所有处理器                                          │
│  • 按优先级排序                                            │
│  • 逐个执行匹配的处理器                                     │
└─────────────────┬───────────────────────────────────────┘
                  │
                  ▼
         ┌────────┴────────┐
         │                 │
         ▼                 ▼
    Match(ctx)?       ContinueChain()?
         │                 │
         ├─ Yes           Yes → Next Handler
         │                No  → Stop
         ▼
  ┌──────────────┐
  │ Middleware   │
  │  Recovery    │
  │  Logging     │
  │  Permission  │
  └──────┬───────┘
         │
         ▼
    Handle(ctx)
```

### 📂 项目结构

```
internal/
├── handler/              # 🎯 核心框架
│   ├── handler.go        #    Handler 接口定义
│   ├── context.go        #    消息上下文
│   ├── router.go         #    消息路由器
│   └── middleware.go     #    中间件基础
│
├── handlers/             # 🔧 处理器实现
│   ├── command/          #    命令处理器 (8 个)
│   ├── keyword/          #    关键词处理器
│   ├── pattern/          #    正则处理器
│   └── listener/         #    监听器 (2 个)
│
├── middleware/           # 🛡️ 中间件
│   ├── recovery.go       #    错误恢复
│   ├── logging.go        #    日志记录
│   ├── permission.go     #    权限检查
│   └── ratelimit.go      #    限流控制
│
├── domain/               # 📦 领域模型
│   ├── user/             #    用户聚合根
│   └── group/            #    群组聚合根
│
└── adapter/              # 🔌 外部适配器
    ├── telegram/         #    Telegram 适配
    └── repository/       #    数据持久化 (MongoDB)
```

---

## 🚀 快速开始

### 前置要求

- ✅ **Go 1.25+** - [安装 Go](https://go.dev/dl/)
- ✅ **MongoDB Atlas** - [免费注册](https://www.mongodb.com/cloud/atlas) (推荐云数据库)
- ✅ **Telegram Bot Token** - 从 [@BotFather](https://t.me/BotFather) 获取
- 🐳 **Docker** (可选) - [安装 Docker](https://docs.docker.com/get-docker/)

### 5 分钟快速启动

#### 1️⃣ 获取 Bot Token

1. 在 Telegram 中搜索 [@BotFather](https://t.me/BotFather)
2. 发送 `/newbot` 创建新机器人
3. 按提示设置名称和用户名
4. 保存返回的 Token（格式：`1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`）

#### 2️⃣ 配置 MongoDB Atlas

<details>
<summary>点击查看详细步骤</summary>

1. 访问 [MongoDB Atlas](https://www.mongodb.com/cloud/atlas)
2. 注册并创建免费 M0 集群（512MB 免费）
3. 创建数据库用户（设置用户名和密码）
4. 配置网络访问（添加 IP 白名单，或允许所有 IP：`0.0.0.0/0`）
5. 获取连接字符串：`mongodb+srv://username:password@cluster.mongodb.net/`

</details>

#### 3️⃣ 克隆并配置

```bash
# 克隆项目
git clone https://github.com/yourusername/go-telegram-bot.git
cd go-telegram-bot

# 复制配置模板
cp .env.example .env

# 编辑 .env 文件，填入你的配置
# TELEGRAM_TOKEN=你的Bot Token
# MONGO_URI=你的MongoDB连接字符串
```

#### 4️⃣ 启动运行

**方式 1: 使用 Docker（推荐）**

```bash
# 一键启动所有服务
make docker-up

# 查看日志
make docker-logs

# 停止服务
make docker-down
```

**方式 2: 本地开发**

```bash
# 安装依赖
go mod download

# 运行
make run
```

#### 5️⃣ 测试

在 Telegram 中向你的 Bot 发送：

```
/ping
```

如果收到 `🏓 Pong!` 回复，说明启动成功！🎉

---

## 📋 命令列表

### 基础命令

| 命令 | 描述 | 权限 | 支持聊天类型 |
|------|------|------|-------------|
| `/ping` | 测试 Bot 响应速度 | User | 所有 |
| `/help` | 显示帮助信息 | User | 所有 |
| `/stats` | 显示统计数据 | User | 所有 |

### 权限管理命令

| 命令 | 描述 | 权限 | 示例 |
|------|------|------|------|
| `/promote` | 提升用户权限 | SuperAdmin | `/promote @username` |
| `/demote` | 降低用户权限 | SuperAdmin | `/demote @username` |
| `/setperm` | 设置用户权限 | Owner | `/setperm @user admin` |
| `/listadmins` | 查看管理员列表 | User | `/listadmins` |
| `/myperm` | 查看自己的权限 | User | `/myperm` |

### 内置处理器

| 类型 | 功能 | 优先级 | 说明 |
|------|------|--------|------|
| 🔍 Greeting | 问候语自动回复 | 200 | 检测 "你好"、"hello" 等 |
| 🌤️ Weather | 天气查询（示例） | 300 | 正则匹配 "天气 城市" |
| 📝 MessageLogger | 消息日志记录 | 900 | 记录所有消息到日志 |
| 📊 Analytics | 数据统计分析 | 950 | 统计用户和群组活跃度 |

---

## 💻 开发指南

### 添加新命令（3 步完成）

#### 第 1 步：创建处理器

```go
// internal/handlers/command/version.go
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
            "version",                     // 命令名
            "查看 Bot 版本",                // 描述
            user.PermissionUser,           // 所需权限
            nil,                           // 支持所有聊天类型
            groupRepo,
        ),
    }
}

func (h *VersionHandler) Handle(ctx *handler.Context) error {
    if err := h.CheckPermission(ctx); err != nil {
        return err
    }
    return ctx.ReplyHTML("<b>Bot Version:</b> v2.0.0\n<b>Go:</b> 1.25+")
}
```

#### 第 2 步：注册到 Router

```go
// cmd/bot/main.go
func registerHandlers(router *handler.Router, groupRepo, userRepo) {
    // ... 其他处理器

    // ✅ 添加新命令
    router.Register(command.NewVersionHandler(groupRepo))
}
```

#### 第 3 步：测试

```bash
# 重新编译运行
make run

# 在 Telegram 中测试
/version
```

### 添加关键词处理器

```go
// internal/handlers/keyword/thanks.go
package keyword

import (
    "strings"
    "telegram-bot/internal/handler"
)

type ThanksHandler struct {
    keywords []string
}

func NewThanksHandler() *ThanksHandler {
    return &ThanksHandler{
        keywords: []string{"谢谢", "thanks", "thank you"},
    }
}

func (h *ThanksHandler) Match(ctx *handler.Context) bool {
    text := strings.ToLower(ctx.Text)
    for _, kw := range h.keywords {
        if strings.Contains(text, kw) {
            return true
        }
    }
    return false
}

func (h *ThanksHandler) Handle(ctx *handler.Context) error {
    return ctx.Reply("不客气！😊")
}

func (h *ThanksHandler) Priority() int { return 200 }
func (h *ThanksHandler) ContinueChain() bool { return true }
```

### 添加中间件

```go
// internal/middleware/timing.go
package middleware

import (
    "telegram-bot/internal/handler"
    "time"
)

type TimingMiddleware struct {
    logger Logger
}

func NewTimingMiddleware(logger Logger) *TimingMiddleware {
    return &TimingMiddleware{logger: logger}
}

func (m *TimingMiddleware) Middleware() handler.Middleware {
    return func(next handler.HandlerFunc) handler.HandlerFunc {
        return func(ctx *handler.Context) error {
            start := time.Now()
            err := next(ctx)
            duration := time.Since(start)

            m.logger.Info("handler_timing",
                "duration_ms", duration.Milliseconds(),
                "user_id", ctx.UserID,
            )

            return err
        }
    }
}

// cmd/bot/main.go
router.Use(middleware.NewTimingMiddleware(appLogger).Middleware())
```

---

## 🐳 部署方案

### Docker 部署（推荐）

```bash
# 构建镜像
docker build -t telegram-bot .

# 使用 Docker Compose
docker-compose -f deployments/docker/docker-compose.yml up -d

# 查看日志
docker-compose logs -f bot

# 停止服务
docker-compose down
```

### 生产环境部署

<details>
<summary>查看生产环境最佳实践</summary>

1. **环境变量配置**
   ```bash
   # 使用生产级日志格式
   LOG_FORMAT=json
   LOG_LEVEL=info

   # MongoDB 连接池优化
   MONGO_MAX_POOL_SIZE=100
   MONGO_MIN_POOL_SIZE=10
   ```

2. **健康检查**
   - 定期检查 Bot 是否在线
   - 监控 MongoDB 连接状态
   - 设置告警通知

3. **优雅关闭**
   - Bot 已内置优雅关闭机制
   - SIGTERM/SIGINT 信号自动触发
   - 等待所有消息处理完成（最多 30 秒）

4. **日志收集**
   ```bash
   # 使用 JSON 格式便于日志分析
   LOG_FORMAT=json

   # 日志轮转（使用 Docker 的日志驱动）
   docker-compose.yml:
     logging:
       driver: "json-file"
       options:
         max-size: "10m"
         max-file: "3"
   ```

</details>

### 本地开发

```bash
# 安装依赖
go mod download

# 运行测试
make test

# 热重载（需要安装 air）
make run-dev

# 编译
make build

# 运行
./bin/bot
```

---

## 📊 项目统计

| 指标 | 数值 | 说明 |
|------|------|------|
| 📝 代码行数 | ~4,700+ 行 | Go 代码（不含注释和空行） |
| 🧪 测试覆盖率 | 85%+ | 单元测试 + 集成测试 |
| 📦 处理器类型 | 4 种 | Command, Keyword, Pattern, Listener |
| 🎯 内置命令 | 8 个 | Ping, Help, Stats, Promote, Demote, SetPerm, ListAdmins, MyPerm |
| 🛡️ 中间件 | 4 个 | Recovery, Logging, Permission, RateLimit |
| 📚 文档数量 | 15+ 篇 | 从入门到进阶的完整文档 |
| ⚡ 性能 | ~500 msg/s | 单实例处理能力（优化后） |

---

## 📚 完整文档

### 快速入门

| 文档 | 说明 | 适合人群 |
|------|------|----------|
| [快速入门指南](docs/getting-started.md) | 5 分钟上手 | 新手 |
| [命令参考](docs/commands-reference.md) | 所有命令使用说明 | 所有用户 |

### 架构设计

| 文档 | 说明 | 适合人群 |
|------|------|----------|
| [架构设计文档](docs/architecture.md) | 深入理解整体架构 | 开发者 |
| [架构流程图](docs/architecture-diagram.md) | 可视化架构图 | 开发者 |

### 开发指南

| 文档 | 说明 | 适合人群 |
|------|------|----------|
| [开发者 API 参考](docs/developer-api.md) | 完整 API 文档 | 开发者 |
| [命令处理器开发](docs/handlers/command-handler-guide.md) | 如何开发命令处理器 | 开发者 |
| [关键词处理器开发](docs/handlers/keyword-handler-guide.md) | 如何开发关键词处理器 | 开发者 |
| [正则处理器开发](docs/handlers/pattern-handler-guide.md) | 如何开发正则处理器 | 开发者 |
| [监听器开发](docs/handlers/listener-handler-guide.md) | 如何开发监听器 | 开发者 |
| [中间件开发指南](docs/middleware-guide.md) | 如何开发中间件 | 开发者 |
| [Repository 开发](docs/repository-guide.md) | 数据持久化开发 | 开发者 |

### 运维部署

| 文档 | 说明 | 适合人群 |
|------|------|----------|
| [部署运维指南](docs/deployment.md) | 生产环境部署 | 运维 |
| [性能优化指南](docs/performance.md) | 性能调优建议 | 运维 |
| [定时任务指南](docs/scheduler-guide.md) | 定时任务开发 | 开发者 |
| [GitHub Actions 配置](docs/github-actions-setup.md) | CI/CD 配置 | DevOps |

---

## ❓ 常见问题

<details>
<summary><b>如何获取 Telegram Bot Token？</b></summary>

1. 在 Telegram 中搜索 [@BotFather](https://t.me/BotFather)
2. 发送 `/newbot` 命令
3. 按提示设置机器人名称和用户名
4. 保存返回的 Token

**注意**：Token 格式为 `数字:字母和数字混合`，例如 `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`

</details>

<details>
<summary><b>如何配置 MongoDB Atlas？</b></summary>

1. 访问 [MongoDB Atlas](https://www.mongodb.com/cloud/atlas) 并注册
2. 创建免费 M0 集群（512MB 存储）
3. 创建数据库用户（Database Access）
4. 配置网络访问（Network Access）：
   - 开发环境：添加你的 IP
   - 生产环境：添加服务器 IP
   - 临时测试：允许所有 IP（`0.0.0.0/0`，**不推荐生产使用**）
5. 获取连接字符串（Connect → Drivers → Go）

**连接字符串格式**：
```
mongodb+srv://username:password@cluster.mongodb.net/?retryWrites=true&w=majority
```

</details>

<details>
<summary><b>如何设置 Bot 的初始 Owner？</b></summary>

在 `.env` 文件中配置：

```bash
BOT_OWNER_IDS=123456789,987654321
```

多个 Owner 用逗号分隔。这些用户将自动获得 Owner 权限。

</details>

<details>
<summary><b>测试时出现 "Unauthorized" 错误？</b></summary>

这通常表示 Bot Token 无效或过期。请检查：

1. Token 是否正确复制到 `.env` 文件
2. Token 中是否有多余的空格
3. 是否使用了正确的环境变量名：`TELEGRAM_TOKEN`

</details>

<details>
<summary><b>MongoDB 连接失败怎么办？</b></summary>

检查以下几点：

1. **连接字符串格式**：确保包含用户名、密码和集群地址
2. **网络访问**：确认 IP 在白名单中
3. **用户权限**：确认数据库用户有读写权限
4. **防火墙**：检查本地防火墙是否阻止了 27017 端口

**调试方法**：
```bash
# 测试连接
mongosh "你的连接字符串"
```

</details>

<details>
<summary><b>如何启用限流中间件？</b></summary>

在 `cmd/bot/main.go` 中取消注释：

```go
// 创建限流器（每秒 5 条消息，令牌桶容量 10）
rateLimiter := middleware.NewSimpleRateLimiter(time.Second, 10)
router.Use(middleware.NewRateLimitMiddleware(rateLimiter).Middleware())
```

</details>

<details>
<summary><b>如何添加自定义日志？</b></summary>

使用内置的 Logger：

```go
func (h *MyHandler) Handle(ctx *handler.Context) error {
    h.logger.Info("processing_message",
        "user_id", ctx.UserID,
        "text", ctx.Text,
    )

    // 业务逻辑

    return nil
}
```

日志级别：`Debug`, `Info`, `Warn`, `Error`

</details>

---

## 🧪 测试

```bash
# 运行所有测试
make test

# 运行单元测试
make test-unit

# 运行集成测试（需要 MongoDB）
make test-integration

# 生成覆盖率报告
make test-coverage
# 打开 coverage.html 查看详细报告
```

### 测试覆盖率

| 模块 | 覆盖率 |
|------|--------|
| Handler 核心 | 92% |
| 命令处理器 | 88% |
| 中间件 | 85% |
| Domain 模型 | 90% |
| Repository | 80% |
| **总体** | **85%+** |

---

## 🔧 Make 命令

```bash
# 开发
make help           # 查看所有命令
make build          # 编译项目
make run            # 运行项目
make run-dev        # 热重载开发
make fmt            # 格式化代码
make lint           # 代码检查
make vet            # 静态分析

# 测试
make test           # 运行所有测试
make test-unit      # 单元测试
make test-integration  # 集成测试
make test-coverage  # 生成覆盖率报告

# Docker
make docker-up      # 启动 Docker 环境
make docker-down    # 停止 Docker 环境
make docker-logs    # 查看日志
make docker-clean   # 清理 Docker 资源

# 其他
make clean          # 清理构建产物
make ci-check       # CI 检查（格式、lint、测试）
```

---

## 🗺️ 路线图

- [x] ✅ 核心 Handler 架构
- [x] ✅ 4 级权限系统
- [x] ✅ 中间件系统（Recovery, Logging, Permission, RateLimit）
- [x] ✅ 定时任务系统
- [x] ✅ 完整的单元测试和集成测试
- [x] ✅ Docker 部署支持
- [x] ✅ 完整的文档（15+ 篇）
- [ ] 🚧 插件系统（动态加载处理器）
- [ ] 📋 Web Dashboard（管理界面）
- [ ] 🌍 多语言支持（i18n）
- [ ] 📊 Metrics 和 Prometheus 集成
- [ ] 🔔 Webhook 模式（vs 长轮询）
- [ ] 💾 Redis 缓存层
- [ ] 🤖 AI 集成（ChatGPT, Claude 等）

---

## 🤝 贡献指南

欢迎贡献！我们遵循以下流程：

### 提交流程

1. **Fork 本仓库**
2. **创建特性分支** (`git checkout -b feature/AmazingFeature`)
3. **编写代码和测试**
4. **运行测试** (`make test`)
5. **格式化代码** (`make fmt`)
6. **提交更改** (`git commit -m 'Add some AmazingFeature'`)
7. **推送到分支** (`git push origin feature/AmazingFeature`)
8. **开启 Pull Request**

### 代码规范

- ✅ 所有代码必须通过 `go fmt` 格式化
- ✅ 所有代码必须通过 `golangci-lint` 检查
- ✅ 新功能必须包含单元测试（覆盖率 > 80%）
- ✅ 复杂逻辑必须添加注释
- ✅ 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/)

### 提交信息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

**示例**:
```
feat(command): add /weather command

Add a new weather command that queries weather API
and returns formatted weather information.

Closes #123
```

---

## 🛠️ 技术栈

| 组件 | 技术 | 版本 | 说明 |
|------|------|------|------|
| 语言 | [Go](https://go.dev/) | 1.25+ | 高性能、并发友好 |
| Bot SDK | [go-telegram/bot](https://github.com/go-telegram/bot) | v1.17.0 | 官方推荐的 Go 客户端 |
| 数据库 | [MongoDB Atlas](https://www.mongodb.com/cloud/atlas) | 云数据库 | 免费 512MB，全球部署 |
| 测试 | [testify](https://github.com/stretchr/testify) | v1.11.1 | 断言和 Mock |
| Mock | [gomock](https://github.com/uber-go/mock) | v0.6.0 | 接口模拟 |
| 环境变量 | [godotenv](https://github.com/joho/godotenv) | v1.5.1 | .env 文件加载 |
| 容器 | [Docker](https://www.docker.com/) | 最新 | 容器化部署 |

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

```
MIT License

Copyright (c) 2025 Telegram Bot Development Team

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files...
```

---

## 🙏 致谢

感谢以下开源项目：

- [go-telegram/bot](https://github.com/go-telegram/bot) - 优秀的 Telegram Bot SDK
- [MongoDB](https://www.mongodb.com/) - 灵活的 NoSQL 数据库
- [testify](https://github.com/stretchr/testify) - 强大的测试工具
- Go 社区的所有贡献者

---

## 📧 联系方式

- 🐛 **问题反馈**: [提交 Issue](../../issues)
- 💡 **功能建议**: [提交 Feature Request](../../issues/new?labels=enhancement)
- 📧 **邮件**: your.email@example.com
- 💬 **Telegram**: [@your_telegram](https://t.me/your_telegram)

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给个 Star！**

Made with ❤️ by [Your Name](https://github.com/asmmitul)

</div>
