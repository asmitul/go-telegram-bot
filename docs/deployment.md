# 部署文档 (Deployment Guide)

本文档详细说明如何在本地开发环境和生产环境中部署 Telegram Bot。

## 目录

1. [本地开发环境搭建](#1-本地开发环境搭建)
2. [Linode 生产环境部署](#2-linode-生产环境部署)
3. [GitHub Actions 自动部署](#3-github-actions-自动部署)
4. [环境变量配置](#4-环境变量配置)
5. [日常更新流程](#5-日常更新流程)
6. [故障排查](#6-故障排查)

---

## 1. 本地开发环境搭建

### 1.1 前置要求

- Docker 20.10+
- Docker Compose 2.0+
- Git
- 一个有效的 Telegram Bot Token

### 1.2 快速启动

```bash
# 1. 克隆仓库
git clone https://github.com/your-username/go-telegram-bot.git
cd go-telegram-bot

# 2. 复制环境变量模板
cp .env.example .env

# 3. 编辑 .env 文件，填入你的 Telegram Bot Token
# TELEGRAM_TOKEN=your_bot_token_here

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

## 2. Linode 生产环境部署

### 2.1 服务器配置要求

- **推荐配置**: Linode 2GB 或更高
- **操作系统**: Ubuntu 22.04 LTS
- **开放端口**: 22 (SSH), 80 (HTTP), 443 (HTTPS)

### 2.2 服务器初始化

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

### 2.3 配置 SSH 密钥（用于 GitHub Actions）

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

### 2.4 服务器环境配置

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
EOF
```

### 2.5 创建生产环境 Docker Compose 配置

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

## 3. GitHub Actions 自动部署

项目已配置 GitHub Actions 自动部署流程（`.github/workflows/cd.yml`）。

### 3.1 工作流程说明

当代码推送到 `main` 分支时，自动执行以下步骤：

1. **Test**: 运行单元测试，生成覆盖率报告
2. **Build and Push**: 构建 Docker 镜像，推送到 GitHub Container Registry (GHCR)
3. **Deploy**: SSH 连接到生产服务器，拉取最新镜像，重启服务
4. **Rollback**: 如果部署失败，自动回滚到上一版本

### 3.2 配置 GitHub Secrets

在 GitHub 仓库中配置以下 Secrets：

**Settings → Secrets and variables → Actions → New repository secret**

| Secret Name | 说明 | 示例值 |
|-------------|------|--------|
| `PROD_HOST` | 生产服务器 IP 地址 | `123.45.67.89` |
| `PROD_USER` | SSH 用户名 | `root` |
| `PROD_PORT` | SSH 端口 | `22` |
| `PROD_SSH_KEY` | SSH 私钥（完整内容） | `-----BEGIN OPENSSH PRIVATE KEY-----\n...` |
| `TELEGRAM_TOKEN` | Telegram Bot Token | `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz` |
| `MONGO_URI` | MongoDB Atlas 连接字符串 | `mongodb+srv://user:pass@cluster.mongodb.net/` |

### 3.3 配置 GHCR 访问权限

```bash
# 在服务器上登录 GitHub Container Registry
echo "YOUR_GITHUB_PAT" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

**生成 GitHub Personal Access Token (PAT)**:
1. GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Generate new token
3. 勾选 `read:packages` 权限
4. 保存 Token，在服务器上使用

### 3.4 首次部署

```bash
# 在服务器上手动执行首次部署
cd /opt/telegram-bot
docker-compose pull
docker-compose up -d
```

---

## 4. 环境变量配置

### 4.1 必需的环境变量

| 变量名 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `TELEGRAM_TOKEN` | Telegram Bot API Token | - | ✅ |
| `MONGO_URI` | MongoDB Atlas 连接字符串 | - | ✅ |
| `DATABASE_NAME` | 数据库名称 | `telegram_bot` | ✅ |

### 4.2 可选的环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DEBUG` | 调试模式 | `false` |
| `ENVIRONMENT` | 运行环境 | `production` |
| `LOG_LEVEL` | 日志级别 | `info` |
| `LOG_FORMAT` | 日志格式 | `text` |
| `PORT` | 应用端口 | `8080` |
| `MONGO_TIMEOUT` | MongoDB 连接超时 | `10s` |

### 4.3 环境变量优先级

1. 系统环境变量（最高优先级）
2. `.env` 文件
3. 代码默认值（最低优先级）

---

## 5. 日常更新流程

### 5.1 自动更新（推荐）

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

### 5.2 手动更新

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

### 5.3 回滚到上一版本

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

## 6. 故障排查

### 6.1 Bot 无法启动

**检查日志**:
```bash
docker-compose logs bot
```

**常见问题**:
- ❌ `Invalid token`: 检查 `TELEGRAM_TOKEN` 是否正确
- ❌ `Cannot connect to MongoDB`: 检查 `MONGO_URI` 和 Atlas IP 白名单
- ❌ `Permission denied`: 检查文件权限

### 6.2 MongoDB 连接失败

**排查步骤**：

1. **检查连接字符串**
   ```bash
   echo $MONGO_URI
   ```

2. **检查 Atlas IP 白名单**
   - 登录 https://cloud.mongodb.com/
   - Network Access → 添加服务器 IP

3. **测试连接**
   ```bash
   mongosh "mongodb+srv://user:pass@cluster.mongodb.net/telegram_bot"
   ```

### 6.3 GitHub Actions 部署失败

**检查以下配置**:
1. GitHub Secrets 是否正确配置
2. SSH 密钥是否有效
3. 服务器 SSH 端口是否开放
4. GHCR 镜像是否成功推送

**查看 Actions 日志**:
GitHub → Actions → 选择失败的 workflow → 查看详细日志

### 6.4 性能问题

```bash
# 查看容器资源使用
docker stats

# 查看 Bot 内存使用
docker exec telegram-bot ps aux

# 查看 MongoDB 慢查询（使用 Atlas Performance Advisor）
# 或使用 mongosh:
mongosh "mongodb+srv://user:pass@cluster.mongodb.net/telegram_bot" --eval "db.setProfilingLevel(2); db.system.profile.find().sort({ts:-1}).limit(10)"
```

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
- [ ] 配置监控告警

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

### C. 性能优化建议

1. **MongoDB Atlas**:
   - 创建适当的索引（使用 Performance Advisor）
   - 升级到更高层级集群（M10+）
   - 配置连接池大小
   - 启用复制集（高可用）

2. **Bot 应用**:
   - 启用请求缓存
   - 使用消息队列处理耗时任务
   - 配置合理的超时时间

3. **监控**:
   - 启用日志聚合分析
   - 监控系统资源使用情况

---

## 联系和支持

如有问题，请：
1. 查看本文档的故障排查部分
2. 查看项目 README.md
3. 在 GitHub 上提交 Issue
4. 查看 Telegram Bot API 官方文档: https://core.telegram.org/bots/api
