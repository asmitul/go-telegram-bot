# Telegram Bot - 生产级清洁架构

一个基于 Go 语言的生产级 Telegram 机器人，采用清洁架构设计，支持模块化命令、权限管理和完整的 DevOps 流程。

## ✨ 特性

- 🏗️ **清洁架构**：领域驱动设计，层次分明，易于测试和维护
- 🔐 **多级权限系统**：超级管理员、高级管理员、普通管理员三级权限
- 🧩 **模块化命令**：每个命令独立模块，可单独开关
- 🔄 **可配置性**：每个群组可独立配置命令开关
- 🧪 **高测试覆盖率**：单元测试、集成测试、Mock 支持
- 🐳 **Docker 支持**：完整的容器化方案
- 📊 **监控告警**：Prometheus + Grafana 监控体系
- 🚀 **CI/CD**：GitHub Actions 自动化部署
- 📝 **完整文档**：代码注释、API 文档齐全

## 🏛️ 架构设计

```
├── Domain Layer（领域层）
│   ├── User Aggregate（用户聚合根）
│   ├── Group Aggregate（群组聚合根）
│   └── Command Interface（命令接口）
│
├── Use Case Layer（用例层）
│   ├── Permission Check（权限检查）
│   ├── Command Configuration（命令配置）
│   └── User Management（用户管理）
│
├── Adapter Layer（适配器层）
│   ├── MongoDB Repository（数据持久化）
│   ├── Telegram API（消息收发）
│   └── Logger（日志记录）
│
└── Commands（命令模块）
    ├── Ping（测试命令）
    ├── Ban（封禁命令）
    ├── Stats（统计命令）
    └── ... （更多命令）
```

## 📦 快速开始

### 前置要求

- Go 1.21+
- Docker & Docker Compose
- Make

### 1. 克隆项目

```bash
git clone https://github.com/yourusername/telegram-bot.git
cd telegram-bot
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，填入你的 Telegram Bot Token
```

### 3. 本地开发（使用 Docker）

```bash
# 启动所有服务（Bot + MongoDB + Prometheus + Grafana）
make docker-up

# 查看日志
make docker-logs

# 停止服务
make docker-down
```

### 4. 本地开发（不使用 Docker）

```bash
# 安装依赖
make deps

# 运行测试
make test

# 构建应用
make build

# 运行应用
make run
```

## 🧪 测试

```bash
# 运行所有测试
make test

# 运行单元测试
make test-unit

# 运行集成测试
make test-integration

# 生成覆盖率报告
make test-coverage

# 查看覆盖率报告
open coverage.html
```

## 🎯 权限系统

### 权限等级

1. **PermissionUser（普通用户）** - 可使用基础命令
2. **PermissionAdmin（普通管理员）** - 可使用管理命令
3. **PermissionSuperAdmin（超级管理员）** - 可配置命令开关、管理其他管理员
4. **PermissionOwner（群主）** - 最高权限

### 权限检查流程

```go
// 1. 命令定义所需权限
func (h *BanHandler) RequiredPermission() user.Permission {
    return user.PermissionAdmin
}

// 2. 中间件自动检查权限
middleware := telegram.NewPermissionMiddleware(userRepo, groupRepo)

// 3. 权限不足自动拒绝
// ❌ 权限不足！需要权限: Admin，当前权限: User
```

## 🔧 添加新命令

### 1. 创建命令模块

```bash
mkdir -p internal/commands/mycommand
cd internal/commands/mycommand
```

### 2. 实现命令接口

```go
// handler.go
package mycommand

import (
    "telegram-bot/internal/domain/command"
    "telegram-bot/internal/domain/user"
)

type Handler struct {
    // 注入依赖
}

func NewHandler(...) *Handler {
    return &Handler{...}
}

// 命令名称
func (h *Handler) Name() string {
    return "mycommand"
}

// 命令描述
func (h *Handler) Description() string {
    return "我的新命令"
}

// 所需权限
func (h *Handler) RequiredPermission() user.Permission {
    return user.PermissionUser
}

// 检查是否启用
func (h *Handler) IsEnabled(groupID int64) bool {
    // 从数据库检查配置
    return true
}

// 处理命令
func (h *Handler) Handle(ctx *command.Context) error {
    // 实现命令逻辑
    return nil
}
```

### 3. 注册命令

```go
// cmd/bot/main.go
func registerCommands(...) {
    // ... 其他命令
    registry.Register(mycommand.NewHandler(...))
}
```

### 4. 编写测试

```go
// handler_test.go
package mycommand

import "testing"

func TestHandler_Name(t *testing.T) {
    handler := NewHandler(...)
    if handler.Name() != "mycommand" {
        t.Errorf("expected mycommand, got %s", handler.Name())
    }
}
```

## 🎮 命令开关管理

### 在群组中禁用命令

```go
// 管理员可以在群组中禁用特定命令
/disable_command ping

// 或在代码中操作
group.DisableCommand("ping", adminUserID)
groupRepo.Update(group)
```

### 在群组中启用命令

```go
// 重新启用命令
/enable_command ping

// 或在代码中操作
group.EnableCommand("ping", adminUserID)
groupRepo.Update(group)
```

## 📊 监控与告警

### 访问监控面板

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000（用户名/密码: admin/admin）

### 关键指标

- `bot_command_total` - 命令执行总数
- `bot_command_duration_seconds` - 命令执行时间
- `bot_command_errors_total` - 命令错误总数
- `bot_active_users` - 活跃用户数
- `mongodb_connections` - MongoDB 连接数

### 告警规则

- Bot 服务宕机 > 1 分钟
- 命令错误率 > 10%
- 响应时间 P95 > 2 秒
- 内存使用 > 512MB
- MongoDB 连接数 > 100

## 🚀 部署

### 使用 Docker Compose（推荐）

```bash
# 1. 在服务器上克隆代码
git clone https://github.com/yourusername/telegram-bot.git
cd telegram-bot

# 2. 配置环境变量
cp .env.example .env
vim .env

# 3. 启动服务
docker-compose -f deployments/docker/docker-compose.yml up -d

# 4. 查看状态
docker-compose ps
```

### 使用 GitHub Actions 自动部署

1. **配置 GitHub Secrets**：
   - `PROD_HOST` - 生产服务器 IP
   - `PROD_USER` - SSH 用户名
   - `PROD_SSH_KEY` - SSH 私钥
   - `PROD_PORT` - SSH 端口
   - `TELEGRAM_TOKEN` - Bot Token
   - `SLACK_WEBHOOK`（可选）- Slack 通知

2. **推送到 main 分支自动部署**：
```bash
git push origin main
```

3. **查看部署状态**：
   访问 GitHub Actions 页面查看部署进度

### 手动部署

```bash
# 构建
make build-linux

# 上传到服务器
scp bin/bot-linux user@server:/opt/telegram-bot/

# SSH 到服务器
ssh user@server

# 启动服务
cd /opt/telegram-bot
./bot-linux
```

## 🛠️ 开发工具

### Makefile 命令

```bash
make help              # 显示所有可用命令
make build             # 构建应用
make run               # 运行应用
make test              # 运行测试
make test-coverage     # 生成覆盖率报告
make docker-up         # 启动 Docker 服务
make docker-down       # 停止 Docker 服务
make docker-logs       # 查看日志
make lint              # 代码检查
make fmt               # 格式化代码
make mock              # 生成 Mock 文件
make clean             # 清理构建文件
make ci-check          # 运行 CI 检查
```

### 安装开发工具

```bash
make install-tools
```

包含：
- `golangci-lint` - 代码检查
- `goimports` - 导入排序
- `mockgen` - Mock 生成
- `air` - 热重载

## 📁 项目结构详解

```
internal/
├── domain/              # 领域层（业务核心）
│   ├── user/           # 用户聚合根
│   ├── group/          # 群组聚合根
│   └── command/        # 命令接口
│
├── usecase/            # 用例层（业务逻辑）
│   ├── user/          # 用户相关用例
│   └── group/         # 群组相关用例
│
├── adapter/            # 适配器层（外部依赖）
│   ├── repository/    # 数据持久化
│   │   ├── mongodb/   # MongoDB 实现
│   │   └── memory/    # 内存实现（测试）
│   ├── telegram/      # Telegram API
│   └── logger/        # 日志
│
├── commands/           # 命令模块（独立插件）
│   ├── ping/
│   ├── ban/
│   ├── stats/
│   └── welcome/
│
└── config/            # 配置管理
```

## 🧩 依赖注入示例

```go
// 初始化仓储
userRepo := mongodb.NewUserRepository(db)
groupRepo := mongodb.NewGroupRepository(db)

// 初始化用例
permCheck := user.NewCheckPermissionUseCase(userRepo)

// 初始化命令（注入依赖）
banHandler := ban.NewHandler(groupRepo, userRepo, telegramAPI)

// 注册命令
registry.Register(banHandler)
```

## 🔒 安全最佳实践

1. **永远不要硬编码 Token**
   - 使用环境变量
   - 使用密钥管理服务（如 HashiCorp Vault）

2. **权限检查**
   - 每个命令都通过中间件检查权限
   - 数据库中存储用户权限

3. **输入验证**
   - 验证所有用户输入
   - 防止注入攻击

4. **限流**
   - 实现速率限制防止滥用
   - 按用户 ID 限流

5. **日志审计**
   - 记录所有管理操作
   - 敏感信息脱敏

## 📝 贡献指南

1. Fork 项目
2. 创建特性分支（`git checkout -b feature/AmazingFeature`）
3. 提交更改（`git commit -m 'Add some AmazingFeature'`）
4. 推送到分支（`git push origin feature/AmazingFeature`）
5. 开启 Pull Request

### 代码规范

- 遵循 Go 官方风格指南
- 使用 `golangci-lint` 检查代码
- 测试覆盖率 > 80%
- 所有公共函数必须有注释

## 🐛 常见问题

### Q: Bot 无法接收消息？
A: 检查 Bot Token 是否正确，确保 Bot 已添加到群组

### Q: MongoDB 连接失败？
A: 检查 MongoDB 服务是否运行，URI 配置是否正确

### Q: 权限检查失败？
A: 确保用户在数据库中有记录，检查群组 ID 是否正确

### Q: Docker 部署失败？
A: 检查端口是否被占用，查看 `docker-compose logs`

## 📄 许可证

MIT License

## 🤝 联系方式

- 项目地址: [https://github.com/yourusername/telegram-bot](https://github.com/yourusername/telegram-bot)
- 问题反馈: [Issues](https://github.com/yourusername/telegram-bot/issues)

## 🙏 致谢

- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) - Telegram Bot API 库
- [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver) - MongoDB 驱动
- [Prometheus](https://prometheus.io/) - 监控系统
- [Grafana](https://grafana.com/) - 可视化平台