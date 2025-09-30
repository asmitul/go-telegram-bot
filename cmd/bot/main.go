package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"telegram-bot/internal/adapter/repository/mongodb"
	"telegram-bot/internal/adapter/telegram"
	"telegram-bot/internal/commands/ban"
	"telegram-bot/internal/commands/ping"
	"telegram-bot/internal/config"
	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"telegram-bot/pkg/logger"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
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

	// 4. 初始化中间件和注册表（需要在 bot 之前）
	// 初始化 Logger
	appLogger := logger.New(logger.Config{
		Level:  logger.ParseLevel(cfg.LogLevel),
		Format: cfg.LogFormat,
	})
	appLogger.Info("Logger initialized", "level", cfg.LogLevel, "format", cfg.LogFormat)

	permMiddleware := telegram.NewPermissionMiddleware(userRepo, groupRepo)
	logMiddleware := telegram.NewLoggingMiddleware(appLogger)

	// 初始化命令注册表
	registry := command.NewRegistry()

	// 5. 初始化 Telegram Bot
	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			// 使用我们的 handler 处理更新
			telegram.HandleUpdate(ctx, b, update, registry, permMiddleware, logMiddleware)
		}),
	}

	telegramBot, err := bot.New(cfg.TelegramToken, opts...)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	log.Printf("Bot initialized successfully")

	// 6. 初始化 Telegram API 适配器
	telegramAPI := telegram.NewAPI(telegramBot)

	// 7. 注册命令
	registerCommands(registry, groupRepo, userRepo, telegramAPI)

	// 8. 启动 Bot
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Println("Bot is running... Press Ctrl+C to stop")
	telegramBot.Start(ctx)
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
	groupRepo group.Repository,
	userRepo user.Repository,
	api *telegram.API,
) {
	// Ping 命令
	registry.Register(ping.NewHandler(groupRepo))

	// Ban 命令
	registry.Register(ban.NewHandler(groupRepo, userRepo, api))

	// TODO: 在这里注册更多命令
	// registry.Register(stats.NewHandler(groupRepo, userRepo))
	// registry.Register(welcome.NewHandler(...))
	// registry.Register(mute.NewHandler(...))
}
