# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a production-grade Telegram Bot built with Go 1.21+, following Clean Architecture principles with Domain-Driven Design (DDD). The bot features a modular command system, multi-level permission management, and complete DevOps infrastructure.

## Architecture

The codebase follows Clean Architecture with clear separation of concerns:

### 1. Domain Layer (`internal/domain/`)
The core business logic with no external dependencies:
- **User Aggregate** (`domain/user/`): User entity with group-specific permissions stored in `map[int64]Permission`
- **Group Aggregate** (`domain/group/`): Group entity with per-command enable/disable configuration
- **Command Interface** (`domain/command/`): Defines the `Handler` interface that all commands must implement

Key architectural decisions:
- Permissions are **group-scoped**: each user has different permission levels per group
- Commands are **pluggable**: all commands implement the same `Handler` interface
- Groups control command availability: commands can be enabled/disabled per group by admins

### 2. Use Case Layer (`internal/usecase/`)
Application-specific business logic that orchestrates domain objects. Use cases depend on repository interfaces defined in the domain layer, not concrete implementations.

### 3. Adapter Layer (`internal/adapter/`)
External integrations implementing domain interfaces:
- **Repository Adapters** (`adapter/repository/`):
  - `mongodb/`: Production MongoDB implementations
  - `memory/`: In-memory implementations for testing
- **Telegram Adapter** (`adapter/telegram/`): Bot API integration with permission middleware
- **Logger Adapter** (`adapter/logger/`): Logging abstraction

### 4. Commands Layer (`internal/commands/`)
Each command is a self-contained module in its own directory with:
- `handler.go`: Implements the `command.Handler` interface
- `handler_test.go`: Unit tests for the command

Commands must implement:
```go
Name() string                           // Command name without "/"
Description() string                    // Human-readable description
RequiredPermission() user.Permission    // Minimum permission required
Handle(ctx *command.Context) error      // Command execution logic
IsEnabled(groupID int64) bool           // Check if enabled in group
```

## Dependency Injection Pattern

The application uses constructor-based dependency injection:
1. Initialize concrete adapters (MongoDB repositories, Telegram client)
2. Initialize use cases with repository interfaces
3. Initialize command handlers with required dependencies
4. Register commands in the registry
5. Wire everything together in `cmd/bot/main.go`

## Common Development Commands

### Building and Running
```bash
make build          # Build binary to bin/bot
make run            # Run locally
make run-dev        # Run with hot reload (requires air)
make build-linux    # Cross-compile for Linux deployment
```

### Testing
```bash
make test                  # Run all tests with race detector
make test-unit            # Unit tests only
make test-integration     # Integration tests (requires MongoDB)
make test-coverage        # Generate coverage.html report
```

Integration tests use the build tag `//go:build integration` and require MongoDB.

### Code Quality
```bash
make fmt            # Format code with gofmt and goimports
make lint           # Run golangci-lint
make vet            # Run go vet
make ci-check       # Run all CI checks (fmt + lint + test)
```

### Mock Generation
```bash
make mock           # Generate mocks for domain interfaces
```

Mocks are generated in `test/mocks/` using mockgen for repository and command interfaces.

### Docker Development
```bash
make docker-up      # Start all services (bot, MongoDB, Prometheus, Grafana)
make docker-down    # Stop all services
make docker-logs    # Follow bot logs
make docker-clean   # Remove all containers and volumes
```

### Tools Installation
```bash
make install-tools  # Install golangci-lint, goimports, mockgen, air
```

## Adding a New Command

1. **Create command directory**: `mkdir -p internal/commands/mycommand`

2. **Implement Handler** in `handler.go`:
   - Inject dependencies via constructor
   - Implement all 5 methods of `command.Handler` interface
   - Use `groupRepo` to check `IsEnabled(groupID)`

3. **Register in main**: Add to `cmd/bot/main.go` in the command registration section

4. **Write tests** in `handler_test.go`:
   - Test each interface method
   - Use mocks from `test/mocks/` for dependencies
   - Follow existing test patterns in `commands/ping/handler_test.go`

## Permission System

Four permission levels (defined in `domain/user/user.go`):
- `PermissionUser`: Basic user (default)
- `PermissionAdmin`: Can execute admin commands
- `PermissionSuperAdmin`: Can configure command enable/disable, manage admins
- `PermissionOwner`: Full control (reserved for group owners)

**Key concept**: Permissions are **per-group**. A user can be an admin in Group A but a regular user in Group B. The `User.Permissions` field is a `map[int64]Permission` where keys are group IDs.

The permission middleware in `adapter/telegram/middleware.go` automatically:
1. Fetches user from repository
2. Checks their permission level in the specific group
3. Compares against command's `RequiredPermission()`
4. Blocks execution if insufficient

## Command Enable/Disable System

Groups can disable specific commands via the `Group.Commands` map. By default, all commands are enabled. To disable:

```go
group.DisableCommand("commandname", adminUserID)
groupRepo.Update(group)
```

Commands check this in their `IsEnabled()` implementation by querying the group repository.

## Configuration

Environment variables are loaded via `.env` file (see `.env.example`):
- `TELEGRAM_TOKEN`: Bot API token (required)
- `MONGO_URI`: MongoDB connection string
- `DATABASE_NAME`: MongoDB database name
- `METRICS_ENABLED`: Enable Prometheus metrics on port 9091
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

Config is managed in `internal/config/config.go` using `godotenv`.

## Monitoring

The bot exposes Prometheus metrics on port 9091 (`/metrics`):
- `bot_command_total`: Total commands executed (by command name)
- `bot_command_duration_seconds`: Command execution time histogram
- `bot_command_errors_total`: Total errors (by command name)
- `bot_active_users`: Active user count

Access monitoring at:
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)

Alert rules are defined in `monitoring/alerts/alert-rules.yml`.

## Deployment

### Docker Compose (Recommended)
```bash
cp .env.example .env
# Edit .env with production values
docker-compose -f deployments/docker/docker-compose.yml up -d
```

### Manual Deployment
```bash
make deploy-prod      # Requires SSH configuration
make deploy-staging   # Deploy to staging environment
```

Deployment script: `scripts/deploy.sh` handles building, uploading, and restarting services.

### CI/CD
GitHub Actions workflows:
- `.github/workflows/ci.yml`: Runs on PRs (lint, test, build, security scan)
- `.github/workflows/cd.yml`: Runs on main branch push (test, build Docker image, deploy to production)

Required GitHub Secrets:
- `PROD_HOST`, `PROD_USER`, `PROD_SSH_KEY`, `PROD_PORT`
- `TELEGRAM_TOKEN`
- `SLACK_WEBHOOK` (optional)

## Testing Strategy

1. **Unit tests**: Test individual components in isolation using mocks
   - Located next to source files (`*_test.go`)
   - Use `github.com/stretchr/testify` for assertions
   - Mock dependencies with `github.com/golang/mock`

2. **Integration tests**: Test full stack with real MongoDB
   - Located in `test/integration/`
   - Use build tag `//go:build integration`
   - Require running MongoDB instance
   - See `test/integration/bot_test.go` for examples

3. **Test coverage goal**: >80% as specified in README

## Module Name

The Go module is named `telegram-bot` (see `go.mod`). All internal imports use this as the base:
```go
import (
    "telegram-bot/internal/domain/command"
    "telegram-bot/internal/domain/user"
)
```

## Key Dependencies

- `github.com/go-telegram-bot-api/telegram-bot-api/v5`: Telegram Bot API
- `go.mongodb.org/mongo-driver`: MongoDB driver
- `github.com/prometheus/client_golang`: Prometheus metrics
- `github.com/sirupsen/logrus`: Structured logging
- `github.com/joho/godotenv`: Environment variable loading
- `github.com/stretchr/testify`: Testing utilities
- `github.com/golang/mock`: Mock generation