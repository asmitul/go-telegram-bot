package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
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
	// 记录启动时间
	startTime := time.Now()

	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 初始化 Logger
	appLogger := logger.New(logger.Config{
		Level:  logger.ParseLevel(cfg.LogLevel),
		Format: cfg.LogFormat,
	})
	appLogger.Info("🚀 Bot starting...", "version", "1.0.0")
	appLogger.Info("Logger initialized", "level", cfg.LogLevel, "format", cfg.LogFormat)

	// 3. 初始化 MongoDB
	mongoClient, err := initMongoDB(cfg.MongoURI)
	if err != nil {
		appLogger.Error("Failed to connect to MongoDB", "error", err)
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	appLogger.Info("✅ MongoDB connected successfully")

	db := mongoClient.Database(cfg.DatabaseName)

	// 4. 初始化仓储
	userRepo := mongodb.NewUserRepository(db)
	groupRepo := mongodb.NewGroupRepository(db)

	// 5. 初始化中间件
	permMiddleware := telegram.NewPermissionMiddleware(userRepo, groupRepo)
	logMiddleware := telegram.NewLoggingMiddleware(appLogger)

	// 6. 初始化命令注册表
	registry := command.NewRegistry()

	// 7. 初始化 WaitGroup 用于追踪正在处理的命令
	var wg sync.WaitGroup

	// 8. 初始化 Telegram Bot
	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			// 增加计数器
			wg.Add(1)
			defer wg.Done()

			// 使用我们的 handler 处理更新
			telegram.HandleUpdate(ctx, b, update, registry, permMiddleware, logMiddleware)
		}),
	}

	telegramBot, err := bot.New(cfg.TelegramToken, opts...)
	if err != nil {
		appLogger.Error("Failed to create bot", "error", err)
		log.Fatalf("Failed to create bot: %v", err)
	}

	appLogger.Info("✅ Telegram Bot initialized successfully")

	// 9. 初始化 Telegram API 适配器
	telegramAPI := telegram.NewAPI(telegramBot)

	// 10. 注册命令
	registerCommands(registry, groupRepo, userRepo, telegramAPI)
	appLogger.Info("✅ Commands registered", "count", len(registry.GetAll()))

	// 11. 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// 12. 启动 Bot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 在 goroutine 中启动 bot
	go func() {
		appLogger.Info("✅ Bot is running", "uptime", time.Since(startTime))
		telegramBot.Start(ctx)
	}()

	// 13. 等待退出信号
	sig := <-sigChan
	appLogger.Info("📥 Received shutdown signal", "signal", sig.String())

	// 14. 开始优雅关闭
	shutdown(appLogger, mongoClient, &wg, cancel, startTime)
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

// shutdown 优雅关闭
func shutdown(appLogger logger.Logger, mongoClient *mongo.Client, wg *sync.WaitGroup, cancel context.CancelFunc, startTime time.Time) {
	appLogger.Info("🛑 Starting graceful shutdown...")

	// 1. 停止接收新的更新
	cancel()
	appLogger.Info("✅ Stopped accepting new updates")

	// 2. 等待正在处理的命令完成（最多30秒）
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		appLogger.Info("✅ All pending commands completed")
	case <-time.After(30 * time.Second):
		appLogger.Warn("⚠️ Shutdown timeout: some commands may not have completed")
	}

	// 3. 关闭数据库连接
	appLogger.Info("Closing database connection...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mongoClient.Disconnect(ctx); err != nil {
		appLogger.Error("Failed to close database connection", "error", err)
	} else {
		appLogger.Info("✅ Database connection closed")
	}

	// 4. 输出运行统计
	uptime := time.Since(startTime)
	appLogger.Info("📊 Bot Statistics",
		"total_uptime", uptime.String(),
		"uptime_seconds", int(uptime.Seconds()),
	)

	// 5. 最终关闭日志
	appLogger.Info("👋 Bot shutdown complete. Goodbye!")
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
