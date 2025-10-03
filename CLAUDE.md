# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a production-grade Telegram Bot built with Go 1.21+, featuring a **unified message handling architecture** that supports commands, keyword triggers, pattern matching, and message listeners. The bot works in all chat types (private, group, supergroup, channel) with a flexible handler-based system.

## Architecture

The codebase follows a **Handler-based architecture** with clear separation of concerns:

### 1. Core Framework (`internal/handler/`)
The foundation of the message handling system:
- **Handler Interface** (`handler.go`): Unified interface for all message processors
  - `Match(ctx *Context) bool` - Determines if handler should process the message
  - `Handle(ctx *Context) error` - Processes the message
  - `Priority() int` - Handler execution priority (0-999, lower = higher priority)
  - `ContinueChain() bool` - Whether to continue executing subsequent handlers

- **Context** (`context.go`): Enhanced message context
  - Contains all message information (text, user, chat, etc.)
  - Provides helper methods: `IsPrivate()`, `IsGroup()`, `IsChannel()`
  - Built-in reply methods: `Reply()`, `ReplyMarkdown()`, `Send()`
  - Permission checking: `HasPermission()`, `RequirePermission()`
  - Context storage for inter-handler data sharing

- **Router** (`router.go`): Message routing and handler execution
  - Automatically sorts handlers by priority
  - Executes matching handlers in order
  - Supports middleware chaining
  - Thread-safe handler registration

### 2. Handlers Layer (`internal/handlers/`)
Four types of handlers, each with specific priorities:

**Command Handlers** (`handlers/command/`, Priority: 100-199)
- Match messages starting with `/`
- Support `@botname` suffix
- Per-group enable/disable configuration
- Built-in permission checking via `BaseCommand`

Example structure:
```go
type MyCommandHandler struct {
    *BaseCommand
}

func (h *MyCommandHandler) Handle(ctx *handler.Context) error {
    // Permission check (handled by BaseCommand.Match)
    return ctx.Reply("Response")
}
```

**Keyword Handlers** (`handlers/keyword/`, Priority: 200-299)
- Match messages containing specific keywords
- Case-insensitive matching
- Can specify supported chat types
- Usually continue chain for logging

**Pattern Handlers** (`handlers/pattern/`, Priority: 300-399)
- Use regex for complex pattern matching
- Extract matched groups for processing
- Support multi-language patterns
- Can stop chain after matching

**Listeners** (`handlers/listener/`, Priority: 900-999)
- Match ALL messages
- Used for logging, analytics, monitoring
- Always continue chain
- Lowest priority (execute last)

### 3. Middleware Layer (`internal/middleware/`)
Global middleware applied to all handlers:

1. **RecoveryMiddleware**: Catches panics to prevent crashes
2. **LoggingMiddleware**: Logs all message processing
3. **PermissionMiddleware**: Auto-loads user from database
4. **RateLimitMiddleware**: Token bucket rate limiting (optional)

Middleware execution order: Recovery → Logging → Permission → [Handler]

### 4. Domain Layer (`internal/domain/`)
Core business entities:
- **User** (`domain/user/`): User entity with per-group permissions
  - `Permissions map[int64]Permission` - Group ID → Permission level
  - `HasPermission(groupID, required)` - Permission check

- **Group** (`domain/group/`): Group entity with command configuration
  - `Commands map[string]*CommandConfig` - Command enable/disable
  - `IsCommandEnabled(commandName)` - Check if command is enabled

### 5. Adapter Layer (`internal/adapter/`)
External integrations:
- **MongoDB Repository** (`adapter/repository/mongodb/`): Data persistence
- **Telegram Adapter** (`adapter/telegram/`):
  - `converter.go`: Converts Telegram Update to Handler Context
  - `api.go`: Telegram API operations wrapper

## Message Flow

```
Telegram Update
    ↓
ConvertUpdate (creates Context)
    ↓
Router.Route(ctx)
    ↓
For each handler (sorted by priority):
    ↓
    Match(ctx)? → Yes
        ↓
        Middleware Chain:
            Recovery → Logging → Permission
                ↓
            Handle(ctx)
                ↓
            ContinueChain()? → Yes: Next handler
                           → No: Stop
```

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

### Code Quality
```bash
make fmt            # Format code with gofmt and goimports
make lint           # Run golangci-lint
make vet            # Run go vet
make ci-check       # Run all CI checks (fmt + lint + test)
```

### Docker Development
```bash
make docker-up      # Start all services (bot, MongoDB)
make docker-down    # Stop all services
make docker-logs    # Follow bot logs
make docker-clean   # Remove all containers and volumes
```

## Adding New Handlers

### 1. Command Handler

```go
// internal/handlers/command/mycmd.go
package command

import (
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"
)

type MyCommandHandler struct {
	*BaseCommand
}

func NewMyCommandHandler(groupRepo GroupRepository) *MyCommandHandler {
	return &MyCommandHandler{
		BaseCommand: NewBaseCommand(
			"mycmd",                       // Command name
			"My command description",      // Description
			user.PermissionUser,           // Required permission
			[]string{"private", "group"},  // Supported chat types
			groupRepo,
		),
	}
}

func (h *MyCommandHandler) Handle(ctx *handler.Context) error {
	// Permission is already checked by BaseCommand.Match
	// Access user: ctx.User
	// Access chat type: ctx.ChatType, ctx.IsGroup(), etc.

	return ctx.Reply("Command executed!")
}
```

**Register in `cmd/bot/main.go`:**
```go
router.Register(command.NewMyCommandHandler(groupRepo))
```

### 2. Keyword Handler

```go
// internal/handlers/keyword/thanks.go
package keyword

import (
	"strings"
	"telegram-bot/internal/handler"
)

type ThanksHandler struct {
	keywords  []string
	chatTypes []string
}

func NewThanksHandler() *ThanksHandler {
	return &ThanksHandler{
		keywords:  []string{"谢谢", "thanks", "thank you"},
		chatTypes: []string{"private"},
	}
}

func (h *ThanksHandler) Match(ctx *handler.Context) bool {
	// Check chat type
	if !contains(h.chatTypes, ctx.ChatType) {
		return false
	}

	// Check keywords
	text := strings.ToLower(ctx.Text)
	for _, kw := range h.keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

func (h *ThanksHandler) Handle(ctx *handler.Context) error {
	return ctx.Reply("You're welcome! 😊")
}

func (h *ThanksHandler) Priority() int { return 200 }
func (h *ThanksHandler) ContinueChain() bool { return true }
```

### 3. Pattern Handler (Regex)

```go
// internal/handlers/pattern/weather.go
package pattern

import (
	"fmt"
	"regexp"
	"telegram-bot/internal/handler"
)

type WeatherHandler struct {
	pattern *regexp.Regexp
}

func NewWeatherHandler() *WeatherHandler {
	return &WeatherHandler{
		pattern: regexp.MustCompile(`(?i)天气\s+(.+)`),
	}
}

func (h *WeatherHandler) Match(ctx *handler.Context) bool {
	return h.pattern.MatchString(ctx.Text)
}

func (h *WeatherHandler) Handle(ctx *handler.Context) error {
	matches := h.pattern.FindStringSubmatch(ctx.Text)
	if len(matches) < 2 {
		return nil
	}

	city := matches[1]
	// Call weather API here

	return ctx.Reply(fmt.Sprintf("天气查询: %s", city))
}

func (h *WeatherHandler) Priority() int { return 300 }
func (h *WeatherHandler) ContinueChain() bool { return false }
```

### 4. Listener (Message Logger)

```go
// internal/handlers/listener/audit.go
package listener

import (
	"telegram-bot/internal/handler"
)

type AuditHandler struct {
	logger Logger
}

func NewAuditHandler(logger Logger) *AuditHandler {
	return &AuditHandler{logger: logger}
}

func (h *AuditHandler) Match(ctx *handler.Context) bool {
	return true // Match all messages
}

func (h *AuditHandler) Handle(ctx *handler.Context) error {
	h.logger.Info("message_audit",
		"user_id", ctx.UserID,
		"chat_id", ctx.ChatID,
		"text", ctx.Text,
	)
	return nil
}

func (h *AuditHandler) Priority() int { return 900 }
func (h *ThanksHandler) ContinueChain() bool { return true }
```

## Permission System

Four permission levels (defined in `domain/user/user.go`):
- `PermissionUser`: Basic user (default)
- `PermissionAdmin`: Can execute admin commands
- `PermissionSuperAdmin`: Can configure commands, manage admins
- `PermissionOwner`: Full control

**Key concept**: Permissions are **per-group**. The `User.Permissions` field is `map[int64]Permission` where keys are group IDs.

**Checking permissions in handlers:**
```go
func (h *MyHandler) Handle(ctx *handler.Context) error {
	// Method 1: Use helper
	if err := ctx.RequirePermission(user.PermissionAdmin); err != nil {
		return err
	}

	// Method 2: Manual check
	if !ctx.HasPermission(user.PermissionAdmin) {
		return fmt.Errorf("insufficient permission")
	}

	// Business logic
	return ctx.Reply("Success")
}
```

## Permission Management Commands

The bot includes built-in commands for managing user permissions:

### 1. `/promote` - Promote User Permission
- **Permission Required**: SuperAdmin
- **Usage**: `/promote @username` or reply to a message with `/promote`
- **Function**: Promotes user permission by one level (User → Admin → SuperAdmin → Owner)
- **Protection**: Cannot promote to a level higher than your own

### 2. `/demote` - Demote User Permission
- **Permission Required**: SuperAdmin
- **Usage**: `/demote @username` or reply to a message with `/demote`
- **Function**: Demotes user permission by one level
- **Protection**: Cannot demote users with equal or higher permission

### 3. `/setperm` - Set User Permission
- **Permission Required**: Owner
- **Usage**: `/setperm @username <user|admin|superadmin|owner>`
- **Function**: Directly sets user permission to the specified level
- **Example**: `/setperm @alice admin`

### 4. `/listadmins` - List All Admins
- **Permission Required**: User (everyone can view)
- **Usage**: `/listadmins`
- **Function**: Displays all admins grouped by permission level (Owner, SuperAdmin, Admin)

### 5. `/myperm` - View Own Permission
- **Permission Required**: User (everyone can view)
- **Usage**: `/myperm`
- **Function**: Shows your current permission level and capabilities in the current group

**Example usage:**
```
User: /promote @bob
Bot: ✅ User @bob permission promoted: User → Admin 🛡

User: /listadmins
Bot:
👥 Current Group Admin List:

👑 Owner (1):
  • @alice

🛡 Admin (2):
  • @bob
  • @charlie

Total: 3 admins
```

## Command Enable/Disable System

Groups can disable specific commands:
```go
group.DisableCommand("commandname", adminUserID)
groupRepo.Update(group)
```

Commands automatically check this via `BaseCommand.Match()`.

## Configuration

Environment variables (`.env` file):
- `TELEGRAM_TOKEN`: Bot API token (required)
- `MONGO_URI`: MongoDB connection string
- `DATABASE_NAME`: MongoDB database name
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

## Priority Guidelines

- **0-99**: System-level handlers
- **100-199**: Command handlers
- **200-299**: Keyword handlers
- **300-399**: Pattern/regex handlers
- **400-499**: Interactive handlers (buttons, forms)
- **900-999**: Listeners (logging, analytics)

Lower numbers = higher priority (execute first)

## Module Name

The Go module is `telegram-bot`. Import paths:
```go
import (
    "telegram-bot/internal/handler"
    "telegram-bot/internal/handlers/command"
    "telegram-bot/internal/domain/user"
)
```

## Key Dependencies

- `github.com/go-telegram/bot`: Telegram Bot API client
- `go.mongodb.org/mongo-driver`: MongoDB driver
- `github.com/sirupsen/logrus`: Structured logging
- `github.com/joho/godotenv`: Environment variable loading
- `github.com/stretchr/testify`: Testing utilities

## Testing Strategy

1. **Unit tests**: Test handlers in isolation
   - Mock Context for testing
   - Test Match() and Handle() separately
   - Use `internal/handler/router_test.go` as reference

2. **Integration tests**: Test with real MongoDB
   - Located in `test/integration/`
   - Use build tag `//go:build integration`

## Important Notes

- **Chat Type Support**: All handlers can specify supported chat types
- **Middleware Auto-applies**: Permission middleware auto-loads `ctx.User`
- **Context Helpers**: Use `ctx.Reply()`, `ctx.IsGroup()`, etc. for convenience
- **Handler Chain**: Set `ContinueChain() = false` to stop processing after match
- **Thread Safety**: Router is thread-safe, handlers should be stateless
