package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram-bot/internal/adapter/repository/mongodb"
	"telegram-bot/internal/adapter/telegram"
	"telegram-bot/internal/commands/ban"
	"telegram-bot/internal/commands/ping"
	"telegram-bot/internal/commands/stats"
	"telegram-bot/internal/config"
	"telegram-bot/internal/domain/command"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化 MongoDB
	mongoClient, err := initMongoDB(cfg.MongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database(cfg.DatabaseName)

	// 3. 初始化仓储
	userRepo := mongodb.NewUserRepository(db)
	groupRepo := mongodb.NewGroupRepository(db)

	// 4. 初始化 Telegram Bot
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	bot.Debug = cfg.Debug

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// 5. 初始化 Telegram API 适配器
	telegramAPI := telegram.NewAPI(bot)

	// 6. 初始化命令注册表
	registry := command.NewRegistry()

	// 7. 注册命令
	registerCommands(registry, groupRepo, userRepo, telegramAPI)

	// 8. 初始化中间件
	logger := &SimpleLogger{}
	permMiddleware := telegram.NewPermissionMiddleware(userRepo, groupRepo)
	logMiddleware := telegram.NewLoggingMiddleware(logger)

	// 9. 初始化 Bot Handler
	botHandler := telegram.NewBotHandler(
		bot,
		registry,
		permMiddleware,
		logMiddleware,
	)

	// 10. 启动 Bot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 优雅关闭
	go handleShutdown(cancel)

	log.Println("Bot is running... Press Ctrl+C to stop")
	if err := botHandler.Start(ctx); err != nil {
		log.Fatalf("Bot stopped with error: %v", err)
	}
}

// initMongoDB 初始化 MongoDB 连接
func initMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

// registerCommands 注册所有命令
func registerCommands(
	registry command.Registry,
	groupRepo groupRepository,
	userRepo userRepository,
	api telegram.API,
) {
	// Ping 命令
	registry.Register(ping.NewHandler(groupRepo))

	// Ban 命令
	registry.Register(ban.NewHandler(groupRepo, userRepo, api))

	// Stats 命令
	registry.Register(stats.NewHandler(groupRepo, userRepo))

	// TODO: 在这里注册更多命令
	// registry.Register(welcome.NewHandler(...))
	// registry.Register(mute.NewHandler(...))
}

// handleShutdown 处理优雅关闭
func handleShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal %v, shutting down gracefully...", sig)
	cancel()
}

// SimpleLogger 简单的日志实现
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)
}

func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	log.Printf("[ERROR] %s %v", msg, fields)
}
