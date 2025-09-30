.PHONY: help build run test clean docker-build docker-up docker-down lint fmt mock migrate

# 变量
APP_NAME=telegram-bot
BINARY_DIR=bin
DOCKER_COMPOSE=docker-compose -f deployments/docker/docker-compose.yml
GO_FILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")

# 默认目标
help: ## 显示帮助信息
	@echo "可用命令："
	@grep -E '^[a-zA-Z_-]+:.*?## .*$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $1, $2}'

# 构建
build: ## 构建应用
	@echo "构建 $(APP_NAME)..."
	@mkdir -p $(BINARY_DIR)
	@go build -o $(BINARY_DIR)/bot ./cmd/bot
	@echo "✅ 构建完成: $(BINARY_DIR)/bot"

build-linux: ## 构建 Linux 版本
	@echo "构建 Linux 版本..."
	@mkdir -p $(BINARY_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/bot-linux ./cmd/bot
	@echo "✅ 构建完成: $(BINARY_DIR)/bot-linux"

# 运行
run: ## 本地运行应用
	@echo "启动 $(APP_NAME)..."
	@go run ./cmd/bot

run-dev: ## 使用热重载运行（需要安装 air）
	@air -c .air.toml

# 测试
test: ## 运行所有测试
	@echo "运行单元测试..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "✅ 测试完成"

test-coverage: ## 生成测试覆盖率报告
	@echo "生成覆盖率报告..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告: coverage.html"

test-integration: ## 运行集成测试
	@echo "运行集成测试..."
	@go test -v -tags=integration ./test/integration/...

test-unit: ## 只运行单元测试
	@echo "运行单元测试..."
	@go test -v -short ./...

# Docker
docker-build: ## 构建 Docker 镜像
	@echo "构建 Docker 镜像..."
	@docker build -f deployments/docker/Dockerfile -t $(APP_NAME):latest .
	@echo "✅ 镜像构建完成"

docker-up: ## 启动 Docker Compose
	@echo "启动服务..."
	@$(DOCKER_COMPOSE) up -d
	@echo "✅ 服务已启动"
	@$(DOCKER_COMPOSE) ps

docker-down: ## 停止 Docker Compose
	@echo "停止服务..."
	@$(DOCKER_COMPOSE) down
	@echo "✅ 服务已停止"

docker-logs: ## 查看 Docker 日志
	@$(DOCKER_COMPOSE) logs -f bot

docker-restart: ## 重启 Docker 服务
	@$(DOCKER_COMPOSE) restart bot

docker-clean: ## 清理 Docker 资源
	@echo "清理 Docker 资源..."
	@$(DOCKER_COMPOSE) down -v
	@docker system prune -f
	@echo "✅ 清理完成"

# 代码质量
lint: ## 运行代码检查
	@echo "运行 golangci-lint..."
	@golangci-lint run --timeout=5m ./...

fmt: ## 格式化代码
	@echo "格式化代码..."
	@go fmt ./...
	@goimports -w $(GO_FILES)
	@echo "✅ 代码格式化完成"

vet: ## 运行 go vet
	@echo "运行 go vet..."
	@go vet ./...

# Mock 生成
mock: ## 生成 mock 文件
	@echo "生成 mock 文件..."
	@mockgen -source=internal/domain/user/repository.go -destination=test/mocks/user_repository_mock.go -package=mocks
	@mockgen -source=internal/domain/group/repository.go -destination=test/mocks/group_repository_mock.go -package=mocks
	@mockgen -source=internal/domain/command/command.go -destination=test/mocks/command_mock.go -package=mocks
	@echo "✅ Mock 文件生成完成"

# 依赖管理
deps: ## 下载依赖
	@echo "下载依赖..."
	@go mod download
	@go mod tidy
	@echo "✅ 依赖下载完成"

deps-update: ## 更新依赖
	@echo "更新依赖..."
	@go get -u ./...
	@go mod tidy
	@echo "✅ 依赖更新完成"

# 数据库
migrate-up: ## 运行数据库迁移
	@echo "执行数据库迁移..."
	@./scripts/migrate.sh up

migrate-down: ## 回滚数据库迁移
	@echo "回滚数据库迁移..."
	@./scripts/migrate.sh down

# 清理
clean: ## 清理构建文件
	@echo "清理构建文件..."
	@rm -rf $(BINARY_DIR)
	@rm -f coverage.out coverage.html
	@echo "✅ 清理完成"

clean-all: clean docker-clean ## 清理所有文件和 Docker 资源

# 开发工具安装
install-tools: ## 安装开发工具
	@echo "安装开发工具..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golang/mock/mockgen@latest
	@go install github.com/cosmtrek/air@latest
	@echo "✅ 工具安装完成"

# 生产部署
deploy-prod: ## 部署到生产环境（需要配置 SSH）
	@echo "部署到生产环境..."
	@./scripts/deploy.sh prod

deploy-staging: ## 部署到测试环境
	@echo "部署到测试环境..."
	@./scripts/deploy.sh staging

# 监控
metrics: ## 查看应用指标
	@echo "打开 Prometheus: http://localhost:9090"
	@echo "打开 Grafana: http://localhost:3000"

logs: ## 查看应用日志
	@$(DOCKER_COMPOSE) logs -f --tail=100 bot

# 安全扫描
security-scan: ## 运行安全扫描
	@echo "运行安全扫描..."
	@gosec ./...
	@echo "✅ 安全扫描完成"

# 全面检查（CI 前运行）
ci-check: fmt lint test ## 运行所有 CI 检查
	@echo "✅ 所有检查通过"