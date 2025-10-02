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
- ✅ 包含监控组件（Prometheus + Grafana）
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

# MongoDB
MONGO_ROOT_USER=admin
MONGO_ROOT_PASSWORD=<强密码>

# Grafana
GRAFANA_USER=admin
GRAFANA_PASSWORD=<强密码>

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

# 检查 MongoDB
docker exec -it telegram-bot-mongo mongosh

# 访问监控
# Prometheus: http://your-server:9090
# Grafana: http://your-server:3000
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

### 3. 部署 MongoDB

```yaml
# deployments/k8s/mongodb.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mongodb
  namespace: telegram-bot
spec:
  serviceName: mongodb
  replicas: 1
  selector:
    matchLabels:
      app: mongodb
  template:
    metadata:
      labels:
        app: mongodb
    spec:
      containers:
      - name: mongodb
        image: mongo:7.0
        ports:
        - containerPort: 27017
        env:
        - name: MONGO_INITDB_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: bot-secrets
              key: mongo-password
        volumeMounts:
        - name: data
          mountPath: /data/db
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

### 4. 部署 Bot

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
| `MONGO_URI` | ✅ | `mongodb://localhost:27017` | MongoDB 连接串 |
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

## 监控告警

### Prometheus 配置

**位置**：`monitoring/prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'telegram-bot'
    static_configs:
      - targets: ['bot:9091']

  - job_name: 'mongodb'
    static_configs:
      - targets: ['mongodb-exporter:9216']
```

### 关键指标

| 指标 | 说明 | 告警阈值 |
|------|------|---------|
| `bot_messages_total` | 消息总数 | - |
| `bot_handler_errors_total` | 错误数 | > 100/min |
| `bot_handler_duration_seconds` | 处理延迟 | > 1s |
| `bot_active_users` | 活跃用户数 | - |

### Grafana 仪表板

访问：`http://your-server:3000`

默认账户：
- 用户名：`admin`
- 密码：见 `.env` 中的 `GRAFANA_PASSWORD`

**预置仪表板**：
1. Bot 概览
2. 性能监控
3. 错误追踪
4. MongoDB 状态

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

**手动备份**：
```bash
# Docker 环境
docker exec telegram-bot-mongo mongodump \
  --out=/backup/$(date +%Y%m%d) \
  --authenticationDatabase=admin

# 导出备份
docker cp telegram-bot-mongo:/backup ./backup
```

**自动备份脚本**：
```bash
#!/bin/bash
# scripts/backup.sh

BACKUP_DIR="/backup/mongodb"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份
docker exec telegram-bot-mongo mongodump \
  --out=/backup/${DATE} \
  --authenticationDatabase=admin

# 压缩
tar -czf ${BACKUP_DIR}/backup_${DATE}.tar.gz ${BACKUP_DIR}/${DATE}

# 清理 7 天前的备份
find ${BACKUP_DIR} -name "backup_*.tar.gz" -mtime +7 -delete
```

**定时备份（Cron）**：
```bash
# 每天凌晨 3 点备份
0 3 * * * /opt/telegram-bot/scripts/backup.sh
```

### 恢复数据

```bash
# 解压备份
tar -xzf backup_20251002.tar.gz

# 恢复到 MongoDB
docker exec -i telegram-bot-mongo mongorestore \
  --drop \
  /backup/20251002 \
  --authenticationDatabase=admin
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

# 3. 检查 MongoDB 连接
docker exec telegram-bot ping -c 1 mongodb

# 4. 检查 Token 是否正确
curl https://api.telegram.org/bot<TOKEN>/getMe
```

### 2. MongoDB 连接失败

**排查**：

```bash
# 检查 MongoDB 是否运行
docker ps | grep mongodb

# 检查 MongoDB 日志
docker logs telegram-bot-mongo

# 测试连接
docker exec -it telegram-bot-mongo mongosh
```

### 3. 消息无响应

**检查**：

```bash
# 查看处理器注册
docker logs telegram-bot | grep "Handlers registered"

# 查看权限
docker exec -it telegram-bot-mongo mongosh
> use telegram_bot
> db.users.find({user_id: 123456789})

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

# 监控
http://localhost:9090   # Prometheus
http://localhost:3000   # Grafana

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
