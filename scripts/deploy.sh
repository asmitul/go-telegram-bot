#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 检查参数
if [ $# -eq 0 ]; then
    print_error "请指定部署环境: prod 或 staging"
    echo "用法: $0 <prod|staging>"
    exit 1
fi

ENVIRONMENT=$1

# 加载环境配置
if [ "$ENVIRONMENT" == "prod" ]; then
    print_info "部署到生产环境..."
    SERVER=${PROD_HOST}
    USER=${PROD_USER}
    PORT=${PROD_PORT:-22}
    DEPLOY_PATH="/opt/telegram-bot"
elif [ "$ENVIRONMENT" == "staging" ]; then
    print_info "部署到测试环境..."
    SERVER=${STAGING_HOST}
    USER=${STAGING_USER}
    PORT=${STAGING_PORT:-22}
    DEPLOY_PATH="/opt/telegram-bot-staging"
else
    print_error "未知环境: $ENVIRONMENT"
    exit 1
fi

# 检查必要变量
if [ -z "$SERVER" ] || [ -z "$USER" ]; then
    print_error "缺少必要的环境变量"
    print_error "请设置: ${ENVIRONMENT^^}_HOST, ${ENVIRONMENT^^}_USER"
    exit 1
fi

# 1. 构建应用
print_info "构建应用..."
make build-linux

# 2. 创建部署包
print_info "创建部署包..."
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DEPLOY_PACKAGE="telegram-bot-${ENVIRONMENT}-${TIMESTAMP}.tar.gz"

tar czf "$DEPLOY_PACKAGE" \
    bin/bot-linux \
    deployments/docker/docker-compose.yml \
    .env.example

# 3. 上传到服务器
print_info "上传到服务器: $SERVER..."
scp -P "$PORT" "$DEPLOY_PACKAGE" "$USER@$SERVER:/tmp/"

# 4. 在服务器上部署
print_info "在服务器上部署..."
ssh -p "$PORT" "$USER@$SERVER" << EOF
    set -e
    
    # 创建部署目录
    sudo mkdir -p $DEPLOY_PATH
    cd $DEPLOY_PATH
    
    # 备份当前版本
    if [ -f "bot-linux" ]; then
        echo "备份当前版本..."
        sudo mv bot-linux bot-linux.backup.\$(date +%Y%m%d_%H%M%S)
    fi
    
    # 解压新版本
    echo "解压新版本..."
    sudo tar xzf /tmp/$DEPLOY_PACKAGE -C $DEPLOY_PATH
    sudo chmod +x $DEPLOY_PATH/bin/bot-linux
    sudo mv $DEPLOY_PATH/bin/bot-linux $DEPLOY_PATH/bot-linux
    
    # 检查环境变量文件
    if [ ! -f ".env" ]; then
        echo "创建 .env 文件..."
        sudo cp .env.example .env
        echo "请编辑 .env 文件并设置正确的配置"
    fi
    
    # 重启服务
    if command -v systemctl &> /dev/null; then
        echo "使用 systemd 重启服务..."
        sudo systemctl restart telegram-bot || true
    else
        echo "使用 Docker Compose 重启..."
        cd $DEPLOY_PATH
        sudo docker-compose down
        sudo docker-compose up -d
    fi
    
    # 清理
    rm /tmp/$DEPLOY_PACKAGE
    
    echo "部署完成！"
EOF

# 5. 验证部署
print_info "验证部署..."
sleep 5

ssh -p "$PORT" "$USER@$SERVER" << EOF
    if pgrep -f "bot-linux" > /dev/null || sudo docker ps | grep -q telegram-bot; then
        echo "✅ 服务运行正常"
        exit 0
    else
        echo "❌ 服务未运行"
        exit 1
    fi
EOF

if [ $? -eq 0 ]; then
    print_info "✅ 部署成功！"
    print_info "环境: $ENVIRONMENT"
    print_info "服务器: $SERVER"
    print_info "部署路径: $DEPLOY_PATH"
    
    # 清理本地部署包
    rm "$DEPLOY_PACKAGE"
else
    print_error "❌ 部署失败！请检查服务器日志"
    exit 1
fi

# 6. 发送通知（如果配置了 Slack Webhook）
if [ -n "$SLACK_WEBHOOK" ]; then
    print_info "发送 Slack 通知..."
    curl -X POST "$SLACK_WEBHOOK" \
        -H 'Content-Type: application/json' \
        -d "{
            \"text\": \"✅ Telegram Bot 部署成功\",
            \"attachments\": [{
                \"color\": \"good\",
                \"fields\": [
                    {\"title\": \"Environment\", \"value\": \"$ENVIRONMENT\", \"short\": true},
                    {\"title\": \"Server\", \"value\": \"$SERVER\", \"short\": true},
                    {\"title\": \"Time\", \"value\": \"$(date)\", \"short\": false}
                ]
            }]
        }" || print_warning "Slack 通知发送失败"
fi

print_info "部署流程完成！"