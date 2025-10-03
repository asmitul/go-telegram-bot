# 部署运维指南

## 📚 目录

- [概述](#概述)
- [部署方式对比](#部署方式对比)
- [Docker Compose 部署](#docker-compose-部署)
- [Kubernetes 部署](#kubernetes-部署)
- [Systemd 部署](#systemd-部署)
- [环境变量配置](#环境变量配置)
- [监控告警](#监控告警)
- [日志管理](#日志管理)
- [备份恢复](#备份恢复)
- [常见问题排查](#常见问题排查)
- [性能优化](#性能优化)

---

## 概述

本文档介绍如何将 Telegram Bot 部署到生产环境。

### 支持的部署方式

| 方式 | 难度 | 适用场景 | 推荐度 |
|------|------|---------|--------|
| **Docker Compose** | ⭐ | 单机部署，快速上手 | ⭐⭐⭐⭐⭐ |
| **Kubernetes** | ⭐⭐⭐ | 集群部署，高可用 | ⭐⭐⭐⭐ |
| **Systemd** | ⭐⭐ | 传统服务器 | ⭐⭐⭐ |
| **Binary** | ⭐ | 测试环境 | ⭐⭐ |

---

## 部署方式对比

### Docker Compose（推荐）

**优点**：
- ✅ 一键部署所有服务
- ✅ 易于管理和维护
- ✅ 资源隔离

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
- ❌ 需要手动配置 MongoDB

---

## Docker Compose 部署

### 前置要求

- Docker 20.10+
- Docker Compose 2.0+
- 2GB+ RAM
- 10GB+ 磁盘空间

### 1. 准备配置文件

```bash
cd /opt/telegram-bot
cp .env.example .env
nano .env
```

**编辑 `.env` 文件**：
```bash
# Telegram Bot
TELEGRAM_TOKEN=<你的_bot_token>

# MongoDB Atlas
MONGO_URI=mongodb+srv://user:pass@cluster.mongodb.net/
DATABASE_NAME=telegram_bot

# 日志
LOG_LEVEL=info
LOG_FORMAT=json
```

### 2. 启动服务

```bash
# 启动所有服务
docker-compose -f deployments/docker/docker-compose.yml up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f bot
```

### 3. 验证部署

```bash
# 检查 Bot 容器
docker logs telegram-bot

# 检查 MongoDB Atlas 连接
# 使用 Atlas Web UI: https://cloud.mongodb.com/
```

### 4. 服务管理

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

## Kubernetes 部署

### 1. 创建 Namespace

```bash
kubectl create namespace telegram-bot
```

### 2. 创建 Secret

```bash
kubectl create secret generic bot-secrets \
  --from-literal=telegram-token=<your_token> \
  --from-literal=mongo-password=<password> \
  -n telegram-bot
```

**注意**: 推荐使用 MongoDB Atlas 云数据库，无需在 Kubernetes 中部署 MongoDB。

### 3. 部署 Bot

```bash
kubectl apply -f deployments/k8s/deployment.yaml
kubectl apply -f deployments/k8s/service.yaml
```

### 5. 查看状态

```bash
# 查看 Pods
kubectl get pods -n telegram-bot

# 查看日志
kubectl logs -f deployment/telegram-bot -n telegram-bot

# 进入容器
kubectl exec -it <pod-name> -n telegram-bot -- /bin/sh
```

---

## Systemd 部署

### 1. 编译程序

```bash
make build-linux
```

### 2. 复制文件

```bash
sudo cp bin/bot-linux /usr/local/bin/telegram-bot
sudo chmod +x /usr/local/bin/telegram-bot
```

### 3. 创建配置文件

```bash
sudo mkdir -p /etc/telegram-bot
sudo cp .env /etc/telegram-bot/config.env
sudo chown -R telegram-bot:telegram-bot /etc/telegram-bot
```

### 4. 创建 Systemd 服务

```bash
sudo cp deployments/systemd/telegram-bot.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable telegram-bot
sudo systemctl start telegram-bot
```

### 5. 管理服务

```bash
# 查看状态
sudo systemctl status telegram-bot

# 查看日志
sudo journalctl -u telegram-bot -f

# 重启服务
sudo systemctl restart telegram-bot
```

---

## 环境变量配置

### 核心配置

| 变量 | 必需 | 默认值 | 说明 |
|------|-----|--------|------|
| `TELEGRAM_TOKEN` | ✅ | 无 | Bot API Token |
| `MONGO_URI` | ✅ | 无 | MongoDB 连接串 (Atlas) |
| `DATABASE_NAME` | ❌ | `telegram_bot` | 数据库名称 |

### 日志配置

| 变量 | 默认值 | 可选值 |
|------|--------|--------|
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json`, `text` |

### 性能配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `RATE_LIMIT_ENABLED` | `false` | 是否启用限流 |
| `RATE_LIMIT_PER_MIN` | `20` | 每分钟最大请求数 |
| `MAX_WORKERS` | `10` | 最大并发数 |

---

## 日志管理

### 日志格式

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

### 日志查看

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

### 日志轮转

Docker Compose 已配置日志轮转（`docker-compose.yml:26-30`）：

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"  # 单文件最大 10MB
    max-file: "3"    # 保留 3 个文件
```

---

## 备份恢复

### MongoDB 备份

**MongoDB Atlas 自动备份**：

Atlas 提供自动备份功能（M10+ 集群）：

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

### 恢复数据

```bash
# 使用 mongorestore 恢复
mongorestore --uri="mongodb+srv://user:pass@cluster.mongodb.net/" --drop ./backup
```

---

## 常见问题排查

### 1. Bot 无法启动

**检查步骤**：

```bash
# 1. 查看日志
docker logs telegram-bot

# 2. 检查环境变量
docker exec telegram-bot env | grep TELEGRAM

# 3. 检查 MongoDB Atlas 连接
docker exec telegram-bot env | grep MONGO_URI

# 4. 检查 Token 是否正确
curl https://api.telegram.org/bot<TOKEN>/getMe
```

### 2. MongoDB 连接失败

**排查**：

1. **检查 MONGO_URI 配置**
   ```bash
   # 确认环境变量正确
   docker exec telegram-bot env | grep MONGO_URI
   ```

2. **检查 Atlas IP 白名单**
   - 登录 Atlas → Network Access
   - 确保服务器 IP 在白名单中
   - 或添加 `0.0.0.0/0` 允许所有 IP（开发环境）

3. **测试连接**
   ```bash
   # 使用 mongosh 测试（需要本地安装）
   mongosh "mongodb+srv://user:pass@cluster.mongodb.net/telegram_bot"
   ```

4. **检查 Bot 日志中的错误**
   ```bash
   docker logs telegram-bot | grep -i "mongo\|database"
   ```

### 3. 消息无响应

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

### 4. 内存占用过高

**优化**：

```bash
# 查看内存使用
docker stats telegram-bot

# 限制内存
docker-compose.yml 中添加：
    deploy:
      resources:
        limits:
          memory: 512M
```

---

## 性能优化

### 1. MongoDB 优化

```javascript
// 创建索引
db.users.createIndex({user_id: 1})
db.groups.createIndex({group_id: 1})

// 查看慢查询
db.setProfilingLevel(1, {slowms: 100})
db.system.profile.find().limit(10).sort({ts: -1})
```

### 2. 连接池配置

```go
// cmd/bot/main.go
clientOpts := options.Client().
    SetMaxPoolSize(100).      // 最大连接数
    SetMinPoolSize(10).        // 最小连接数
    SetMaxConnIdleTime(30 * time.Second)
```

### 3. 限流配置

```bash
# .env
RATE_LIMIT_ENABLED=true
RATE_LIMIT_PER_MIN=20
```

### 4. 并发控制

```go
// 限制最大并发
sem := make(chan struct{}, 10)

for update := range updates {
    sem <- struct{}{}
    go func(u Update) {
        defer func() { <-sem }()
        handleUpdate(u)
    }(update)
}
```

---

## 附录

### 快速命令参考

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

### 相关文档

- [项目快速入门](./getting-started.md)
- [中间件开发指南](./middleware-guide.md)
- [Repository 开发指南](./repository-guide.md)
- [架构总览](../CLAUDE.md)

---

**编写日期**: 2025-10-02
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team
