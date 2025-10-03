# GitHub Actions 自动部署配置指南

本指南将帮助你配置 GitHub Actions，实现自动化部署到 VPS。

## 📋 前置要求

- ✅ 一台 VPS 服务器（已安装 Docker）
- ✅ GitHub 仓库已推送代码
- ✅ Telegram Bot Token
- ✅ MongoDB Atlas 连接字符串

---

## 🔧 步骤 1: 准备 SSH 密钥

### 1.1 生成 SSH 密钥（如果还没有）

在**本地电脑**上运行：

```bash
# 生成 SSH 密钥对（ED25519 算法，更安全）
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/github_deploy_key

# 不要设置密码，直接回车（GitHub Actions 无法输入密码）
```

这会生成两个文件：
- `~/.ssh/github_deploy_key` - 私钥（稍后添加到 GitHub Secrets）
- `~/.ssh/github_deploy_key.pub` - 公钥（添加到 VPS）

### 1.2 添加公钥到 VPS

```bash
# 方法 1: 使用 ssh-copy-id（推荐）
ssh-copy-id -i ~/.ssh/github_deploy_key.pub root@你的VPS_IP

# 方法 2: 手动添加
cat ~/.ssh/github_deploy_key.pub
# 复制输出内容，然后 SSH 登录到 VPS：
ssh root@你的VPS_IP
mkdir -p ~/.ssh
echo "粘贴公钥内容" >> ~/.ssh/authorized_keys
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

### 1.3 测试 SSH 连接

```bash
# 使用新生成的密钥测试连接
ssh -i ~/.ssh/github_deploy_key root@你的VPS_IP

# 如果能成功登录，说明配置正确
```

### 1.4 获取私钥内容

```bash
# 查看私钥内容（稍后需要复制到 GitHub Secrets）
cat ~/.ssh/github_deploy_key
```

**复制完整输出**，包括：
```
-----BEGIN OPENSSH PRIVATE KEY-----
...
-----END OPENSSH PRIVATE KEY-----
```

---

## 🔐 步骤 2: 配置 GitHub Secrets

### 2.1 进入 GitHub Secrets 设置页面

1. 打开你的 GitHub 仓库
2. 点击 **Settings**（设置）
3. 左侧菜单选择 **Secrets and variables** → **Actions**
4. 点击 **New repository secret** 按钮

### 2.2 添加以下 7 个 Secrets

| Secret Name | 说明 | 示例值 |
|------------|------|--------|
| `PROD_HOST` | VPS IP 地址 | `123.45.67.89` |
| `PROD_USER` | SSH 用户名 | `root` |
| `PROD_PORT` | SSH 端口 | `22` |
| `PROD_SSH_KEY` | SSH 私钥（完整内容） | `-----BEGIN OPENSSH PRIVATE KEY-----\n...` |
| `TELEGRAM_TOKEN` | Telegram Bot Token | `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz` |
| `MONGO_URI` | MongoDB Atlas 连接字符串 | `mongodb+srv://user:pass@cluster.mongodb.net/` |
| `BOT_OWNER_IDS` | Bot Owner 用户 ID（可选） | `123456789` 或 `123456789,987654321` |

### 2.3 详细配置说明

#### `PROD_HOST`
```
你的 VPS 公网 IP 地址
例如: 123.45.67.89
```

#### `PROD_USER`
```
SSH 登录用户名，通常是 root
如果你使用其他用户，确保该用户有 sudo 权限和 docker 权限
```

#### `PROD_PORT`
```
SSH 端口，默认是 22
如果你修改过 SSH 端口，填写实际端口号
```

#### `PROD_SSH_KEY`
```
完整的 SSH 私钥内容，包括开头和结尾
格式:
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
...（中间省略）...
-----END OPENSSH PRIVATE KEY-----

⚠️ 注意：必须包含完整的开头和结尾行
```

#### `TELEGRAM_TOKEN`
```
从 @BotFather 获取的 Bot Token
格式: 1234567890:ABCdefGHIjklMNOpqrsTUVwxyz
```

#### `MONGO_URI`
```
MongoDB Atlas 连接字符串
格式: mongodb+srv://username:password@cluster.xxxxx.mongodb.net/

获取方式:
1. 登录 https://cloud.mongodb.com/
2. 选择你的集群
3. 点击 "Connect" → "Connect your application"
4. 复制连接字符串，替换 <password> 为实际密码
```

#### `BOT_OWNER_IDS`（可选）
```
Bot Owner 的 Telegram 用户 ID
单个用户: 123456789
多个用户: 123456789,987654321（逗号分隔，无空格）

获取用户 ID:
在 Telegram 中向 @userinfobot 发送消息，它会告诉你你的用户 ID
```

---

## 🚀 步骤 3: 首次部署

### 3.1 启用 GitHub Actions

1. 进入仓库的 **Actions** 标签
2. 如果提示启用 Workflows，点击 **I understand my workflows, go ahead and enable them**

### 3.2 推送代码触发部署

```bash
# 在本地仓库中
git add .
git commit -m "feat: configure GitHub Actions auto-deployment"
git push origin main
```

### 3.3 查看部署进度

1. 进入 GitHub 仓库的 **Actions** 标签
2. 点击最新的 workflow 运行
3. 查看每个步骤的执行情况

**部署流程**：
```
1. Test - 运行单元测试
   ↓
2. Build and Push - 构建 Docker 镜像并推送到 GHCR
   ↓
3. Deploy - SSH 到 VPS，部署服务
   ↓
4. Verify - 验证部署成功
```

### 3.4 查看部署日志

在 **Deploy to Production** 步骤中，可以看到详细的部署日志：

```
✓ Creating deployment directory
✓ Creating .env file
✓ Creating docker-compose.yml
✓ Logging in to GHCR
✓ Pulling latest image
✓ Starting container
✓ Container is running
✓ Bot logs: [最近 50 行日志]
```

---

## ✅ 步骤 4: 验证部署

### 4.1 检查 GitHub Actions 状态

确保所有步骤都显示 ✅ 绿色对勾。

### 4.2 SSH 登录 VPS 检查

```bash
# SSH 登录到 VPS
ssh root@你的VPS_IP

# 检查容器状态
cd /opt/telegram-bot
docker ps

# 应该看到类似输出:
# CONTAINER ID   IMAGE                              STATUS
# abc123def456   ghcr.io/user/go-telegram-bot:main  Up 2 minutes

# 查看日志
docker logs telegram-bot

# 或使用 docker-compose
docker-compose logs -f bot
```

### 4.3 测试 Bot 功能

在 Telegram 中向你的 Bot 发送测试命令：

```
/ping
/help
/myperm
```

如果 Bot 正常响应，说明部署成功！

---

## 🔄 日常使用流程

配置完成后，每次更新只需要：

```bash
# 本地开发
git add .
git commit -m "feat: add new feature"
git push origin main
```

GitHub Actions 会**自动**：
1. ✅ 运行测试
2. ✅ 构建 Docker 镜像
3. ✅ 推送到 GHCR
4. ✅ 部署到 VPS
5. ✅ 重启服务
6. ✅ 验证运行状态

**零停机部署**！整个过程约 3-5 分钟。

---

## 🐛 故障排查

### 问题 1: SSH 连接失败

**错误信息**:
```
ssh: connect to host xxx.xxx.xxx.xxx port 22: Connection refused
```

**解决方法**:
1. 检查 `PROD_HOST` 是否正确
2. 检查 `PROD_PORT` 是否正确
3. 检查 VPS 防火墙是否允许 SSH 连接
4. 检查 SSH 服务是否运行: `systemctl status sshd`

### 问题 2: SSH 权限被拒绝

**错误信息**:
```
Permission denied (publickey)
```

**解决方法**:
1. 检查 `PROD_SSH_KEY` 是否完整（包括开头和结尾）
2. 检查公钥是否正确添加到 VPS 的 `~/.ssh/authorized_keys`
3. 检查 VPS 的文件权限:
   ```bash
   chmod 700 ~/.ssh
   chmod 600 ~/.ssh/authorized_keys
   ```

### 问题 3: Docker 镜像拉取失败

**错误信息**:
```
Error response from daemon: pull access denied
```

**解决方法**:
1. 检查 GitHub Container Registry (GHCR) 是否启用
2. 确保镜像已成功推送（查看 **Build and Push** 步骤）
3. 检查镜像名称是否正确（`ghcr.io/用户名/go-telegram-bot:main`）

### 问题 4: Bot 启动失败

**解决方法**:
```bash
# SSH 登录到 VPS
ssh root@你的VPS_IP

# 查看容器日志
cd /opt/telegram-bot
docker logs telegram-bot

# 检查 .env 文件
cat .env

# 常见错误:
# - TELEGRAM_TOKEN 无效: 检查 Token 是否正确
# - MONGO_URI 连接失败: 检查 Atlas IP 白名单
# - 权限错误: 检查 BOT_OWNER_IDS 格式
```

### 问题 5: 部署成功但 Bot 无响应

**检查清单**:
```bash
# 1. 检查容器是否运行
docker ps | grep telegram-bot

# 2. 查看日志中是否有错误
docker logs telegram-bot | grep -i error

# 3. 测试 MongoDB 连接
docker exec telegram-bot env | grep MONGO_URI

# 4. 测试 Telegram Token
curl https://api.telegram.org/bot<你的TOKEN>/getMe
```

---

## 🔒 安全建议

### 1. SSH 安全
- ✅ 使用 SSH 密钥，禁用密码登录
- ✅ 修改默认 SSH 端口（22 → 其他端口）
- ✅ 配置防火墙，只允许必要的端口

### 2. Secrets 安全
- ✅ 永远不要在代码中硬编码敏感信息
- ✅ 定期轮换密钥和 Token
- ✅ 使用强密码（至少 16 位）

### 3. MongoDB 安全
- ✅ 使用 MongoDB Atlas（比自建更安全）
- ✅ 配置 IP 白名单
- ✅ 使用强密码

### 4. Docker 安全
- ✅ 定期更新镜像
- ✅ 使用官方基础镜像
- ✅ 限制容器资源使用

---

## 📚 相关文档

- [部署运维指南](./deployment.md) - 其他部署方式（Kubernetes、Systemd）
- [命令参考](./commands-reference.md) - Bot 可用命令
- [项目架构](../CLAUDE.md) - 代码架构说明

---

## 🆘 需要帮助？

如果遇到问题：
1. 查看本文档的故障排查部分
2. 查看 GitHub Actions 日志
3. 查看服务器日志 (`docker logs telegram-bot`)
4. 在 GitHub 上提交 Issue

---

**最后更新**: 2025-10-03
**文档版本**: v1.0
**维护者**: Telegram Bot Development Team
