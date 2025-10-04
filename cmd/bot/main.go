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
	"telegram-bot/internal/config"
	"telegram-bot/internal/handler"
	"telegram-bot/internal/handlers/command"
	"telegram-bot/internal/handlers/keyword"
	"telegram-bot/internal/handlers/listener"
	"telegram-bot/internal/handlers/pattern"
	"telegram-bot/internal/middleware"
	"telegram-bot/internal/scheduler"
	"telegram-bot/pkg/logger"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// 记录启动时间
	startTime := time.Now()

	// 0. 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

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
	appLogger.Info("🚀 Bot starting...", "version", "2.0.0")
	appLogger.Info("Logger initialized", "level", cfg.LogLevel, "format", cfg.LogFormat)

	// 3. 初始化 MongoDB
	mongoClient, err := initMongoDB(cfg.MongoURI)
	if err != nil {
		appLogger.Error("Failed to connect to MongoDB", "error", err)
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	appLogger.Info("✅ MongoDB connected successfully")

	db := mongoClient.Database(cfg.DatabaseName)

	// 3.1. 创建数据库索引（性能优化）
	indexManager := mongodb.NewIndexManager(db, appLogger)
	if err := indexManager.EnsureIndexes(context.Background()); err != nil {
		appLogger.Warn("Failed to create indexes (continuing anyway)", "error", err)
	} else {
		appLogger.Info("✅ Database indexes created")
	}

	// 4. 初始化仓储
	userRepo := mongodb.NewUserRepository(db)
	groupRepo := mongodb.NewGroupRepository(db)

	// 5. 创建路由器
	router := handler.NewRouter()

	// 6. 注册全局中间件（按执行顺序）
	router.Use(middleware.NewRecoveryMiddleware(appLogger).Middleware())
	router.Use(middleware.NewLoggingMiddleware(appLogger).Middleware())
	router.Use(middleware.NewPermissionMiddleware(userRepo, cfg.OwnerUserIDs, appLogger).Middleware())
	router.Use(middleware.NewGroupMiddleware(groupRepo, appLogger).Middleware())
	// 可选：添加限流中间件
	// rateLimiter := middleware.NewSimpleRateLimiter(time.Second, 5)
	// router.Use(middleware.NewRateLimitMiddleware(rateLimiter).Middleware())

	appLogger.Info("✅ Middlewares registered")

	// 7. 注册处理器
	registerHandlers(router, groupRepo, userRepo, appLogger)
	appLogger.Info("✅ Handlers registered", "count", router.Count())

	// 8. 初始化 WaitGroup 用于追踪正在处理的消息
	var wg sync.WaitGroup

	// 9. 初始化 Telegram Bot
	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			// 增加计数器
			wg.Add(1)
			defer wg.Done()

			// 转换为 Handler Context
			handlerCtx := telegram.ConvertUpdate(ctx, b, update)
			if handlerCtx == nil {
				return // 不是消息更新，忽略
			}

			// 路由消息
			if err := router.Route(handlerCtx); err != nil {
				appLogger.Error("route_error", "error", err)
				handlerCtx.Reply("❌ 处理消息时出错，请稍后再试")
			}
		}),
	}

	telegramBot, err := bot.New(cfg.TelegramToken, opts...)
	if err != nil {
		appLogger.Error("Failed to create bot", "error", err)
		log.Fatalf("Failed to create bot: %v", err)
	}

	appLogger.Info("✅ Telegram Bot initialized successfully")

	// 10. 初始化定时任务调度器
	taskScheduler := scheduler.NewScheduler(appLogger)

	// 添加定时任务
	taskScheduler.AddJob(scheduler.NewCleanupExpiredDataJob(db, appLogger))
	taskScheduler.AddJob(scheduler.NewStatisticsReportJob(userRepo, groupRepo, appLogger))

	appLogger.Info("✅ Scheduler initialized", "jobs", len(taskScheduler.GetJobs()))

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

	// 13. 启动定时任务调度器
	taskScheduler.Start()
	appLogger.Info("✅ Scheduler started")

	// 14. 等待退出信号
	sig := <-sigChan
	appLogger.Info("📥 Received shutdown signal", "signal", sig.String())

	// 15. 开始优雅关闭
	shutdown(appLogger, mongoClient, taskScheduler, &wg, cancel, startTime)
}

// initMongoDB 初始化 MongoDB 连接（优化连接池配置）
func initMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 优化的连接池配置
	clientOpts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(100).                                        // 最大连接数
		SetMinPoolSize(10).                                         // 最小连接数
		SetMaxConnIdleTime(30 * time.Second).                       // 空闲连接超时
		SetServerSelectionTimeout(5 * time.Second).                 // 服务器选择超时
		SetSocketTimeout(10 * time.Second).                         // Socket 超时
		SetConnectTimeout(5 * time.Second).                         // 连接超时
		SetHeartbeatInterval(10 * time.Second).                     // 心跳间隔
		SetCompressors([]string{"zstd", "zlib", "snappy"}).         // 压缩算法
		SetRetryWrites(true).                                       // 自动重试写入
		SetRetryReads(true)                                         // 自动重试读取

	client, err := mongo.Connect(ctx, clientOpts)
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
func shutdown(appLogger logger.Logger, mongoClient *mongo.Client, taskScheduler *scheduler.Scheduler, wg *sync.WaitGroup, cancel context.CancelFunc, startTime time.Time) {
	appLogger.Info("🛑 Starting graceful shutdown...")

	// 1. 停止接收新的更新
	cancel()
	appLogger.Info("✅ Stopped accepting new updates")

	// 2. 停止定时任务调度器
	appLogger.Info("Stopping scheduler...")
	taskScheduler.Stop()
	appLogger.Info("✅ Scheduler stopped")

	// 3. 等待正在处理的命令完成（最多30秒）
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		appLogger.Info("✅ All pending messages completed")
	case <-time.After(30 * time.Second):
		appLogger.Warn("⚠️ Shutdown timeout: some messages may not have completed")
	}

	// 4. 关闭数据库连接
	appLogger.Info("Closing database connection...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := mongoClient.Disconnect(shutdownCtx); err != nil {
		appLogger.Error("Failed to close database connection", "error", err)
	} else {
		appLogger.Info("✅ Database connection closed")
	}

	// 5. 输出运行统计
	uptime := time.Since(startTime)
	appLogger.Info("📊 Bot Statistics",
		"total_uptime", uptime.String(),
		"uptime_seconds", int(uptime.Seconds()),
	)

	// 6. 最终关闭日志
	appLogger.Info("👋 Bot shutdown complete. Goodbye!")
}

// registerHandlers 注册所有处理器
func registerHandlers(
	router *handler.Router,
	groupRepo *mongodb.GroupRepository,
	userRepo *mongodb.UserRepository,
	appLogger logger.Logger,
) {
	// 1. 命令处理器（优先级 100）
	router.Register(command.NewPingHandler(groupRepo))
	router.Register(command.NewHelpHandler(groupRepo, router))
	router.Register(command.NewStatsHandler(groupRepo, userRepo))

	// 权限管理命令
	router.Register(command.NewPromoteHandler(groupRepo, userRepo))
	router.Register(command.NewDemoteHandler(groupRepo, userRepo))
	router.Register(command.NewSetPermHandler(groupRepo, userRepo))
	router.Register(command.NewListAdminsHandler(groupRepo, userRepo))
	router.Register(command.NewMyPermHandler(groupRepo))

	// 2. 关键词处理器（优先级 200）
	router.Register(keyword.NewGreetingHandler())

	// 3. 正则处理器（优先级 300）
	router.Register(pattern.NewWeatherHandler())

	// 4. 监听器（优先级 900+）
	router.Register(listener.NewMessageLoggerHandler(appLogger))
	router.Register(listener.NewAnalyticsHandler())

	appLogger.Info("Registered handlers breakdown",
		"commands", 8,
		"keywords", 1,
		"patterns", 1,
		"listeners", 2,
	)
}
