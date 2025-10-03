# 部署运维指南 (Deployment Guide)

本文档详细说明如何在本地开发环境和生产环境中部署 Telegram Bot。

## 📚 目录

1. [本地开发环境](#1-本地开发环境搭建)
2. [部署方式对比](#2-部署方式对比)
3. [Docker Compose 部署](#3-docker-compose-部署推荐)
4. [Linode 生产环境部署](#4-linode-生产环境部署)
5. [Kubernetes 部署](#5-kubernetes-部署)
6. [Systemd 部署](#6-systemd-部署)
7. [GitHub Actions 自动部署](#7-github-actions-自动部署)
8. [环境变量配置](#8-环境变量配置)
9. [日常更新流程](#9-日常更新流程)
10. [日志管理](#10-日志管理)
11. [备份恢复](#11-备份恢复)
12. [故障排查](#12-故障排查)
13. [性能优化](#13-性能优化)

---

## 1. 本地开发环境搭建

### 1.1 前置要求

- Docker 20.10+
- Docker Compose 2.0+
- Git
- Go 1.25+ (本地开发)
- 一个有效的 Telegram Bot Token

### 1.2 快速启动

```bash
# 1. 克隆仓库
git clone https://github.com/your-username/go-telegram-bot.git
cd go-telegram-bot

# 2. 复制环境变量模板
cp .env.example .env

# 3. 编辑 .env 文件，填入你的配置
# TELEGRAM_TOKEN=your_bot_token_here
# MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net/

# 4. 启动所有服务
make docker-up

# 或者直接使用 docker-compose
cd deployments/docker
docker-compose up -d
```

### 1.3 验证部署

```bash
# 查看服务状态
docker-compose ps

# 查看 Bot 日志
make docker-logs
# 或
docker-compose logs -f bot

# 测试 Bot
# 在 Telegram 中向你的 Bot 发送 /ping 命令
```

### 1.4 常用命令

```bash
# 启动服务
make docker-up

# 停止服务
make docker-down

# 重启服务
make docker-restart

# 查看日志
make docker-logs

# 运行测试
make test

# 构建二进制文件
make build
```

---

## 2. 部署方式对比

### 部署方式选择

| 方式 | 难度 | 适用场景 | 推荐度 |
|------|------|---------|--------|
| **Docker Compose** | ⭐ | 单机部署，快速上手 | ⭐⭐⭐⭐⭐ |
| **Kubernetes** | ⭐⭐⭐ | 集群部署，高可用 | ⭐⭐⭐⭐ |
| **Systemd** | ⭐⭐ | 传统服务器 | ⭐⭐⭐ |
| **Binary** | ⭐ | 测试环境 | ⭐⭐ |

### Docker Compose（推荐）

**优点**：
- ✅ 一键部署所有服务
- ✅ 易于管理和维护
- ✅ 资源隔离
- ✅ 配合 GitHub Actions 实现 CI/CD

**缺点**：
- ❌ 单机部署，不支持高可用
- ❌ 需要 Docker 环境

### Kubernetes

**优点**：
- ✅ 高可用
- ✅ 自动扩缩容
- ✅ 滚动更新
- ✅ 健康检查

**缺点**：
- ❌ 学习曲线陡峭
- ❌ 资源开销大

### Systemd

**优点**：
- ✅ 原生系统集成
- ✅ 自动重启
- ✅ 资源占用小

**缺点**：
- ❌ 手动管理依赖
- ❌ 需要手动配置 MongoDB Atlas

---

## 3. Docker Compose 部署（推荐）

### 3.1 准备配置文件

```bash
cd /opt/telegram-bot
cp .env.example .env
nano .env
```

**编辑 `.env` 文件**：
```bash
# Telegram Bot
TELEGRAM_TOKEN=<你的_bot_token>

# MongoDB Atlas (推荐使用云数据库)
MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net/
DATABASE_NAME=telegram_bot

# 日志
LOG_LEVEL=info
LOG_FORMAT=json

# Bot Owner (可选)
BOT_OWNER_IDS=your_user_id
```

### 3.2 启动服务

```bash
# 启动所有服务
docker-compose -f deployments/docker/docker-compose.yml up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f bot
```

### 3.3 验证部署

```bash
# 检查 Bot 容器
docker logs telegram-bot

# 检查 MongoDB Atlas 连接
# 使用 Atlas Web UI: https://cloud.mongodb.com/

# 测试 Bot 命令
# 在 Telegram 中发送: /ping
```

### 3.4 服务管理

```bash
# 停止服务
docker-compose down

# 重启服务
docker-compose restart bot

# 更新服务
git pull
docker-compose build bot
docker-compose up -d bot

# 查看资源使用
docker stats
```

---

## 4. Linode 生产环境部署

### 4.1 服务器配置要求

- **推荐配置**: Linode 2GB 或更高
- **操作系统**: Ubuntu 22.04 LTS
- **开放端口**: 22 (SSH), 80 (HTTP), 443 (HTTPS)

### 4.2 服务器初始化

```bash
# SSH 登录到 Linode 服务器
ssh root@your-server-ip

# 更新系统
apt update && apt upgrade -y

# 安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# 安装 Docker Compose
apt install docker-compose-plugin -y

# 启动 Docker 服务
systemctl enable docker
systemctl start docker

# 创建部署目录
mkdir -p /opt/telegram-bot
cd /opt/telegram-bot
```

### 4.3 配置 SSH 密钥（用于 GitHub Actions）

```bash
# 在本地生成 SSH 密钥对
ssh-keygen -t ed25519 -C "github-actions" -f ~/.ssh/github_actions_key

# 将公钥添加到服务器
ssh-copy-id -i ~/.ssh/github_actions_key.pub root@your-server-ip

# 测试连接
ssh -i ~/.ssh/github_actions_key root@your-server-ip
```

**保存私钥内容**，稍后需要添加到 GitHub Secrets：
```bash
cat ~/.ssh/github_actions_key
```

### 4.4 服务器环境配置

在服务器上创建 `/opt/telegram-bot/.env` 文件：

```bash
cat > /opt/telegram-bot/.env << 'EOF'
# Telegram Bot Configuration
TELEGRAM_TOKEN=your_production_bot_token
DEBUG=false

# MongoDB Configuration (MongoDB Atlas)
MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net/
DATABASE_NAME=telegram_bot
MONGO_TIMEOUT=10s

# Application Configuration
ENVIRONMENT=production
LOG_LEVEL=info
LOG_FORMAT=json
PORT=8080

# Bot Owner
BOT_OWNER_IDS=your_user_id
EOF
```

### 4.5 创建生产环境 Docker Compose 配置

在服务器上创建 `/opt/telegram-bot/docker-compose.yml`：

```bash
cat > /opt/telegram-bot/docker-compose.yml << 'EOF'
version: '3.8'

services:
  bot:
    image: ghcr.io/your-username/go-telegram-bot:main
    container_name: telegram-bot
    restart: unless-stopped
    env_file: .env
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
EOF
```

---

## 5. Kubernetes 部署

### 5.1 创建 Namespace

```bash
kubectl create namespace telegram-bot
```

### 5.2 创建 Secret

```bash
kubectl create secret generic bot-secrets \
  --from-literal=telegram-token=<your_token> \
  --from-literal=mongo-uri=<your_atlas_uri> \
  -n telegram-bot
```

**注意**: 推荐使用 MongoDB Atlas 云数据库，无需在 Kubernetes 中部署 MongoDB。

### 5.3 部署 Bot

```bash
kubectl apply -f deployments/k8s/deployment.yaml
kubectl apply -f deployments/k8s/service.yaml
```

### 5.4 查看状态

```bash
# 查看 Pods
kubectl get pods -n telegram-bot

# 查看日志
kubectl logs -f deployment/telegram-bot -n telegram-bot

# 进入容器
kubectl exec -it <pod-name> -n telegram-bot -- /bin/sh
```

---

## 6. Systemd 部署

### 6.1 编译程序

```bash
make build-linux
```

### 6.2 复制文件

```bash
sudo cp bin/bot-linux /usr/local/bin/telegram-bot
sudo chmod +x /usr/local/bin/telegram-bot
```

### 6.3 创建配置文件

```bash
sudo mkdir -p /etc/telegram-bot
sudo cp .env /etc/telegram-bot/config.env
sudo chown -R telegram-bot:telegram-bot /etc/telegram-bot
```

### 6.4 创建 Systemd 服务

```bash
sudo cp deployments/systemd/telegram-bot.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable telegram-bot
sudo systemctl start telegram-bot
```

### 6.5 管理服务

```bash
# 查看状态
sudo systemctl status telegram-bot

# 查看日志
sudo journalctl -u telegram-bot -f

# 重启服务
sudo systemctl restart telegram-bot
```

---

## 7. GitHub Actions 自动部署

项目已配置 GitHub Actions 自动部署流程（`.github/workflows/cd.yml` 和 `ci.yml`）。

### 7.1 工作流程说明

#### CI Workflow (`ci.yml`)
在 Pull Request 或推送到 develop 分支时触发：
1. **Lint**: 代码风格检查
2. **Test**: 单元测试（Go 1.24 和 1.25 矩阵测试）
3. **Integration Test**: 集成测试（使用 MongoDB）
4. **Build**: 编译二进制文件
5. **Docker Build**: Docker 镜像构建测试
6. **Security Scan**: 安全扫描（Gosec + Trivy）

#### CD Workflow (`cd.yml`)
当代码推送到 `main` 分支时自动执行：
1. **Test**: 运行单元测试，生成覆盖率报告
2. **Build and Push**: 构建 Docker 镜像，推送到 GitHub Container Registry (GHCR)
3. **Deploy**: SSH 连接到生产服务器，拉取最新镜像，重启服务
4. **Rollback**: 如果部署失败，自动回滚到上一版本

### 7.2 配置 GitHub Secrets

在 GitHub 仓库中配置以下 Secrets：

**Settings → Secrets and variables → Actions → New repository secret**

| Secret Name | 说明 | 示例值 |
|-------------|------|--------|
| `PROD_HOST` | 生产服务器 IP 地址 | `123.45.67.89` |
| `PROD_USER` | SSH 用户名 | `root` |
| `PROD_PORT` | SSH 端口 | `22` |
| `PROD_SSH_KEY` | SSH 私钥（完整内容） | `-----BEGIN OPENSSH PRIVATE KEY-----\n...` |

**环境变量在服务器的 `.env` 文件中配置，无需添加到 GitHub Secrets。**

### 7.3 配置 GHCR 访问权限

```bash
# 在服务器上登录 GitHub Container Registry
echo "YOUR_GITHUB_PAT" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

**生成 GitHub Personal Access Token (PAT)**:
1. GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate new token
3. 勾选 `read:packages` 权限
4. 保存 Token，在服务器上使用

### 7.4 首次部署

```bash
# 在服务器上手动执行首次部署
cd /opt/telegram-bot
docker-compose pull
docker-compose up -d
```

---

## 8. 环境变量配置

### 8.1 必需的环境变量

| 变量名 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `TELEGRAM_TOKEN` | Telegram Bot API Token | - | ✅ |
| `MONGO_URI` | MongoDB Atlas 连接字符串 | - | ✅ |
| `DATABASE_NAME` | 数据库名称 | `telegram_bot` | ✅ |

### 8.2 可选的环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DEBUG` | 调试模式 | `false` |
| `ENVIRONMENT` | 运行环境 | `production` |
| `LOG_LEVEL` | 日志级别 (debug/info/warn/error) | `info` |
| `LOG_FORMAT` | 日志格式 (json/text) | `text` |
| `PORT` | 应用端口 | `8080` |
| `MONGO_TIMEOUT` | MongoDB 连接超时 | `10s` |
| `BOT_OWNER_IDS` | Bot Owner 用户ID（逗号分隔） | - |
| `RATE_LIMIT_ENABLED` | 是否启用限流 | `true` |
| `RATE_LIMIT_PER_MIN` | 每分钟最大请求数 | `20` |

### 8.3 环境变量优先级

1. 系统环境变量（最高优先级）
2. `.env` 文件
3. 代码默认值（最低优先级）

---

## 9. 日常更新流程

### 9.1 自动更新（推荐）

```bash
# 本地开发
git add .
git commit -m "feat: add new feature"
git push origin main

# GitHub Actions 自动执行：
# 1. 运行测试
# 2. 构建 Docker 镜像
# 3. 推送到 GHCR
# 4. 部署到生产服务器
# 5. 自动重启服务
```

### 9.2 手动更新

如果需要手动更新：

```bash
# SSH 登录到服务器
ssh root@your-server-ip

# 进入部署目录
cd /opt/telegram-bot

# 拉取最新镜像
docker-compose pull

# 重启服务（零停机时间）
docker-compose up -d

# 查看日志确认启动成功
docker-compose logs -f bot
```

### 9.3 回滚到上一版本

```bash
# 在服务器上
cd /opt/telegram-bot

# 使用之前的镜像标签
docker-compose down
docker pull ghcr.io/your-username/go-telegram-bot:previous-tag
docker tag ghcr.io/your-username/go-telegram-bot:previous-tag ghcr.io/your-username/go-telegram-bot:main
docker-compose up -d
```

---

## 10. 日志管理

### 10.1 日志格式

```json
{
  "level": "info",
  "msg": "message_received",
  "chat_type": "private",
  "user_id": 123456789,
  "text": "/ping",
  "timestamp": "2025-10-02T10:30:00Z"
}
```

### 10.2 日志查看

**Docker Compose**：
```bash
# 实时查看
docker-compose logs -f bot

# 查看最近 100 行
docker-compose logs --tail=100 bot

# 保存到文件
docker-compose logs bot > bot.log
```

**Kubernetes**：
```bash
kubectl logs -f deployment/telegram-bot -n telegram-bot
```

**Systemd**：
```bash
journalctl -u telegram-bot -f
```

### 10.3 日志轮转

Docker Compose 已配置日志轮转：

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"  # 单文件最大 10MB
    max-file: "3"    # 保留 3 个文件
```

---

## 11. 备份恢复

### 11.1 MongoDB Atlas 备份

**自动备份（M10+ 集群）**：

1. 登录 [MongoDB Atlas](https://cloud.mongodb.com/)
2. 选择你的集群
3. 进入 "Backup" 标签
4. 配置备份策略（快照频率、保留时间）

**免费 M0 集群**：
- 不支持自动备份
- 建议使用 mongodump 手动导出：

```bash
# 使用 mongodump 导出（需要本地安装 MongoDB Tools）
mongodump --uri="mongodb+srv://user:pass@cluster.mongodb.net/telegram_bot" --out=./backup
```

### 11.2 恢复数据

```bash
# 使用 mongorestore 恢复
mongorestore --uri="mongodb+srv://user:pass@cluster.mongodb.net/" --drop ./backup
```

---

## 12. 故障排查

### 12.1 Bot 无法启动

**检查日志**:
```bash
docker-compose logs bot
```

**常见问题**:
- ❌ `Invalid token`: 检查 `TELEGRAM_TOKEN` 是否正确
- ❌ `Cannot connect to MongoDB`: 检查 `MONGO_URI` 和 Atlas IP 白名单
- ❌ `Permission denied`: 检查文件权限

**检查步骤**：
```bash
# 1. 查看日志
docker logs telegram-bot

# 2. 检查环境变量
docker exec telegram-bot env | grep TELEGRAM
docker exec telegram-bot env | grep MONGO_URI

# 3. 检查 Token 是否正确
curl https://api.telegram.org/bot<TOKEN>/getMe
```

### 12.2 MongoDB Atlas 连接失败

**排查步骤**：

1. **检查连接字符串**
   ```bash
   echo $MONGO_URI
   docker exec telegram-bot env | grep MONGO_URI
   ```

2. **检查 Atlas IP 白名单**
   - 登录 Atlas → Network Access
   - 确保服务器 IP 在白名单中
   - 或添加 `0.0.0.0/0` 允许所有 IP（仅开发环境）

3. **测试连接**
   ```bash
   # 使用 mongosh 测试（需要本地安装）
   mongosh "mongodb+srv://user:pass@cluster.mongodb.net/telegram_bot"
   ```

4. **检查 Bot 日志中的错误**
   ```bash
   docker logs telegram-bot | grep -i "mongo\|database"
   ```

### 12.3 GitHub Actions 部署失败

**检查以下配置**:
1. GitHub Secrets 是否正确配置
2. SSH 密钥是否有效
3. 服务器 SSH 端口是否开放
4. GHCR 镜像是否成功推送

**查看 Actions 日志**:
GitHub → Actions → 选择失败的 workflow → 查看详细日志

### 12.4 消息无响应

**检查**：
```bash
# 查看处理器注册
docker logs telegram-bot | grep "Handlers registered"

# 查看权限（使用 Atlas Web UI 或 mongosh）
# 1. Atlas Web UI: Collections → users → 搜索 user_id
# 2. 或使用 mongosh:
mongosh "mongodb+srv://user:pass@cluster.mongodb.net/telegram_bot" --eval "db.users.find({user_id: 123456789}).pretty()"

# 测试命令
/ping
/help
```

### 12.5 内存占用过高

**优化**：
```bash
# 查看内存使用
docker stats telegram-bot

# 限制内存（在 docker-compose.yml 中添加）
deploy:
  resources:
    limits:
      memory: 512M
```

---

## 13. 性能优化

### 13.1 MongoDB Atlas 优化

- 创建适当的索引（使用 Performance Advisor）
- 升级到更高层级集群（M10+）
- 配置连接池大小
- 启用复制集（高可用）

### 13.2 连接池配置

```go
// cmd/bot/main.go
clientOpts := options.Client().
    SetMaxPoolSize(100).      // 最大连接数
    SetMinPoolSize(10).        // 最小连接数
    SetMaxConnIdleTime(30 * time.Second)
```

### 13.3 限流配置

```bash
# .env
RATE_LIMIT_ENABLED=true
RATE_LIMIT_PER_MIN=20
```

### 13.4 Bot 应用优化

- 启用请求缓存
- 使用消息队列处理耗时任务
- 配置合理的超时时间

---

## 附录

### A. 完整部署检查清单

#### 本地开发环境
- [ ] 安装 Docker 和 Docker Compose
- [ ] 克隆代码仓库
- [ ] 配置 `.env` 文件
- [ ] 启动服务 `make docker-up`
- [ ] 测试 Bot 功能

#### 生产环境
- [ ] 配置 Linode 服务器
- [ ] 安装 Docker 和 Docker Compose
- [ ] 生成 SSH 密钥对
- [ ] 配置服务器环境变量
- [ ] 配置 GitHub Secrets
- [ ] 首次手动部署
- [ ] 验证自动部署流程

### B. 安全建议

1. **密码安全**:
   - 使用强密码（至少 16 字符）
   - 定期更新密码
   - 不要在代码中硬编码密码

2. **SSH 安全**:
   - 使用 SSH 密钥认证，禁用密码登录
   - 更改默认 SSH 端口
   - 配置防火墙规则

3. **Docker 安全**:
   - 定期更新镜像
   - 使用非 root 用户运行容器
   - 限制容器资源使用

4. **MongoDB Atlas 安全**:
   - 使用强密码
   - 配置 IP 白名单
   - 启用数据库审计日志（付费功能）

### C. 快速命令参考

```bash
# Docker Compose
make docker-up          # 启动
make docker-down        # 停止
make docker-logs        # 查看日志
make docker-restart     # 重启
make docker-clean       # 清理

# 健康检查
curl http://localhost:8080/health
```

### D. 相关文档

- [项目快速入门](./getting-started.md)
- [命令参考](./commands-reference.md)
- [开发者 API](./developer-api.md)
- [架构总览](../CLAUDE.md)

---

## 联系和支持

如有问题，请：
1. 查看本文档的故障排查部分
2. 查看项目 README.md
3. 在 GitHub 上提交 Issue
4. 查看 Telegram Bot API 官方文档: https://core.telegram.org/bots/api

---

**最后更新**: 2025-10-03
**文档版本**: v2.0
**维护者**: Telegram Bot Development Team
