# Telegram Bot - 统一消息处理架构

一个基于 Go 语言的生产级 Telegram 机器人，采用统一的 Handler 架构，支持命令、关键词、正则匹配和消息监听。支持私聊、群组、超级群组和频道所有聊天类型。

## ✨ 核心特性

### 🎯 统一消息处理
- **四种处理器类型**：命令、关键词、正则、监听器
- **全聊天类型支持**：私聊、群组、超级群组、频道
- **灵活匹配机制**：每个处理器自主决定是否处理消息
- **优先级控制**：自动按优先级排序执行

### 🔐 权限系统
- **多级权限**：User、Admin、SuperAdmin、Owner
- **群组隔离**：每个用户在不同群组有不同权限
- **自动检查**：中间件自动加载用户和检查权限
- **权限管理命令**：提升/降低权限、设置权限、查看管理员列表

### 🛡️ 中间件系统
- **错误恢复**：捕获 panic 防止程序崩溃
- **日志记录**：自动记录所有消息处理
- **权限管理**：自动加载用户信息
- **限流保护**：令牌桶算法防止滥用
- **健康检查**：应用和数据库状态
- **优雅关闭**：处理中的消息不丢失

## 🏗️ 架构设计

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

## 📦 目录结构

```
internal/
├── handler/              # 核心框架
│   ├── handler.go        # Handler 接口定义
│   ├── context.go        # 消息上下文
│   ├── router.go         # 消息路由器
│   └── middleware.go     # 中间件基础
│
├── handlers/             # 处理器实现
│   ├── command/          # 命令处理器 (Priority: 100)
│   │   ├── base.go       # 命令基类
│   │   ├── ping.go       # /ping 命令
│   │   ├── help.go       # /help 命令
│   │   ├── stats.go      # /stats 命令
│   │   ├── promote.go    # /promote 提升权限
│   │   ├── demote.go     # /demote 降低权限
│   │   ├── setperm.go    # /setperm 设置权限
│   │   ├── listadmins.go # /listadmins 管理员列表
│   │   └── myperm.go     # /myperm 查看自己权限
│   │
│   ├── keyword/          # 关键词处理器 (Priority: 200)
│   │   └── greeting.go   # 问候语处理
│   │
│   ├── pattern/          # 正则处理器 (Priority: 300)
│   │   └── weather.go    # 天气查询
│   │
│   └── listener/         # 监听器 (Priority: 900+)
│       ├── message_logger.go  # 消息日志
│       └── analytics.go       # 数据分析
│
├── middleware/           # 中间件
│   ├── recovery.go       # 错误恢复
│   ├── logging.go        # 日志记录
│   ├── permission.go     # 权限检查
│   └── ratelimit.go      # 限流控制
│
├── domain/               # 领域模型
│   ├── user/             # 用户聚合根
│   └── group/            # 群组聚合根
│
└── adapter/              # 外部适配器
    ├── telegram/         # Telegram 适配
    └── repository/       # 数据持久化
```

## 🚀 快速开始

### 前置要求

- Go 1.25+
- MongoDB Atlas（推荐使用云数据库）
- Docker & Docker Compose (可选)

### 1. 克隆项目

```bash
git clone <your-repo-url>
cd go-telegram-bot
```

### 2. 配置环境

```bash
# 复制配置模板
cp .env.example .env

# 编辑 .env 文件，填入你的配置
# TELEGRAM_TOKEN=your_bot_token_here  # 从 @BotFather 获取
# MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net/  # MongoDB Atlas 连接字符串
```

### 3. 使用 Docker 运行（推荐）

```bash
# 启动所有服务
make docker-up

# 查看日志
make docker-logs

# 停止服务
make docker-down
```

### 4. 本地开发

```bash
# 安装依赖
go mod download

# 运行测试
make test

# 编译
make build

# 运行
./bin/bot
```

## 💻 开发指南

### 添加命令处理器

```go
// internal/handlers/command/hello.go
package command

import (
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

type HelloHandler struct {
	*BaseCommand
}

func NewHelloHandler(groupRepo GroupRepository) *HelloHandler {
	return &HelloHandler{
		BaseCommand: NewBaseCommand(
			"hello",                       // 命令名
			"Say hello",                   // 描述
			user.PermissionUser,           // 所需权限
			[]string{"private", "group"},  // 支持的聊天类型
			groupRepo,
		),
	}
}

func (h *HelloHandler) Handle(ctx *handler.Context) error {
	// 权限已由 BaseCommand 检查
	return ctx.Reply("Hello, " + ctx.FirstName + "!")
}
```

**注册到 main.go:**
```go
router.Register(command.NewHelloHandler(groupRepo))
```

### 添加关键词处理器

```go
// internal/handlers/keyword/thanks.go
package keyword

type ThanksHandler struct {
	keywords []string
}

func NewThanksHandler() *ThanksHandler {
	return &ThanksHandler{
		keywords: []string{"谢谢", "thanks"},
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
	return ctx.Reply("不客气！")
}

func (h *ThanksHandler) Priority() int { return 200 }
func (h *ThanksHandler) ContinueChain() bool { return true }
```

### 添加正则处理器

```go
// internal/handlers/pattern/url.go
package pattern

type URLHandler struct {
	pattern *regexp.Regexp
}

func NewURLHandler() *URLHandler {
	return &URLHandler{
		pattern: regexp.MustCompile(`https?://[^\s]+`),
	}
}

func (h *URLHandler) Match(ctx *handler.Context) bool {
	return h.pattern.MatchString(ctx.Text)
}

func (h *URLHandler) Handle(ctx *handler.Context) error {
	urls := h.pattern.FindAllString(ctx.Text, -1)
	return ctx.Reply(fmt.Sprintf("检测到 %d 个链接", len(urls)))
}

func (h *URLHandler) Priority() int { return 300 }
func (h *URLHandler) ContinueChain() bool { return false }
```

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
```

## 🔧 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `TELEGRAM_TOKEN` | Bot API Token | 必填 |
| `MONGO_URI` | MongoDB Atlas 连接字符串 | 必填 |
| `DATABASE_NAME` | 数据库名称 | `telegram_bot` |
| `LOG_LEVEL` | 日志级别 | `info` |
| `LOG_FORMAT` | 日志格式 (json/text) | `json` |

### 权限级别

| 级别 | 值 | 说明 |
|------|----|----|
| User | 1 | 普通用户（默认） |
| Admin | 2 | 管理员 |
| SuperAdmin | 3 | 超级管理员 |
| Owner | 4 | 群主 |

### 处理器优先级

| 范围 | 类型 | 说明 |
|------|------|------|
| 0-99 | 系统级 | 系统保留 |
| 100-199 | 命令 | 以 / 开头的命令 |
| 200-299 | 关键词 | 关键词匹配 |
| 300-399 | 正则 | 正则表达式匹配 |
| 400-499 | 交互 | 按钮、表单等 |
| 900-999 | 监听器 | 日志、统计等 |

## 🐳 Docker 部署

```bash
# 构建镜像
docker build -t telegram-bot .

# 使用 Docker Compose
docker-compose -f deployments/docker/docker-compose.yml up -d

# 查看日志
docker-compose logs -f bot
```

## 📝 Make 命令

```bash
make help           # 查看所有可用命令
make build          # 编译
make run            # 运行
make test           # 测试
make lint           # 代码检查
make fmt            # 格式化代码
make docker-up      # 启动 Docker 环境
make docker-down    # 停止 Docker 环境
make docker-logs    # 查看 Docker 日志
make clean          # 清理构建产物
```

## 🤝 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

- [go-telegram/bot](https://github.com/go-telegram/bot) - Telegram Bot API 客户端
- [MongoDB](https://www.mongodb.com/) - 数据库

## 📧 联系方式

有问题或建议？欢迎 [提交 Issue](../../issues)
