package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status 健康状态
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// ComponentStatus 组件状态
type ComponentStatus struct {
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration_ms"`
	Error     string        `json:"error,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status     Status                     `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Uptime     time.Duration              `json:"uptime_seconds"`
	Components map[string]ComponentStatus `json:"components"`
}

// Checker 健康检查器接口
type Checker interface {
	Check(ctx context.Context) ComponentStatus
	Name() string
}

// CheckerFunc 检查函数类型
type CheckerFunc func(ctx context.Context) error

// SimpleChecker 简单检查器
type SimpleChecker struct {
	name string
	fn   CheckerFunc
}

// NewSimpleChecker 创建简单检查器
func NewSimpleChecker(name string, fn CheckerFunc) *SimpleChecker {
	return &SimpleChecker{
		name: name,
		fn:   fn,
	}
}

// Name 返回检查器名称
func (c *SimpleChecker) Name() string {
	return c.name
}

// Check 执行检查
func (c *SimpleChecker) Check(ctx context.Context) ComponentStatus {
	start := time.Now()
	status := ComponentStatus{
		Timestamp: start,
	}

	err := c.fn(ctx)
	status.Duration = time.Since(start)

	if err != nil {
		status.Status = StatusUnhealthy
		status.Error = err.Error()
	} else {
		status.Status = StatusHealthy
		status.Message = "OK"
	}

	return status
}

// Service 健康检查服务
type Service struct {
	checkers  []Checker
	startTime time.Time
	mu        sync.RWMutex
}

// NewService 创建健康检查服务
func NewService() *Service {
	return &Service{
		checkers:  make([]Checker, 0),
		startTime: time.Now(),
	}
}

// RegisterChecker 注册检查器
func (s *Service) RegisterChecker(checker Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers = append(s.checkers, checker)
}

// Check 执行所有健康检查
func (s *Service) Check(ctx context.Context) HealthResponse {
	s.mu.RLock()
	checkers := s.checkers
	s.mu.RUnlock()

	response := HealthResponse{
		Timestamp:  time.Now(),
		Uptime:     time.Since(s.startTime),
		Components: make(map[string]ComponentStatus),
		Status:     StatusHealthy,
	}

	// 并发执行所有检查
	type result struct {
		name   string
		status ComponentStatus
	}

	results := make(chan result, len(checkers))
	var wg sync.WaitGroup

	for _, checker := range checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			status := c.Check(ctx)
			results <- result{
				name:   c.Name(),
				status: status,
			}
		}(checker)
	}

	// 等待所有检查完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	healthyCount := 0
	totalCount := 0

	for res := range results {
		response.Components[res.name] = res.status
		totalCount++

		if res.status.Status == StatusHealthy {
			healthyCount++
		}
	}

	// 计算总体状态
	if healthyCount == totalCount {
		response.Status = StatusHealthy
	} else if healthyCount > 0 {
		response.Status = StatusDegraded
	} else {
		response.Status = StatusUnhealthy
	}

	return response
}

// Handler 返回 HTTP 处理器
func (s *Service) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		response := s.Check(ctx)

		// 设置响应头
		w.Header().Set("Content-Type", "application/json")

		// 根据状态设置 HTTP 状态码
		switch response.Status {
		case StatusHealthy:
			w.WriteHeader(http.StatusOK)
		case StatusDegraded:
			w.WriteHeader(http.StatusOK) // 部分降级仍返回 200
		case StatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		// 编码 JSON 响应
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.Encode(response)
	}
}

// MongoDBChecker MongoDB 健康检查器
type MongoDBChecker struct {
	pingFunc func(ctx context.Context) error
}

// NewMongoDBChecker 创建 MongoDB 检查器
func NewMongoDBChecker(pingFunc func(ctx context.Context) error) *MongoDBChecker {
	return &MongoDBChecker{
		pingFunc: pingFunc,
	}
}

// Name 返回检查器名称
func (c *MongoDBChecker) Name() string {
	return "mongodb"
}

// Check 执行检查
func (c *MongoDBChecker) Check(ctx context.Context) ComponentStatus {
	start := time.Now()
	status := ComponentStatus{
		Timestamp: start,
	}

	// 执行 ping
	err := c.pingFunc(ctx)
	status.Duration = time.Since(start)

	if err != nil {
		status.Status = StatusUnhealthy
		status.Error = err.Error()
		status.Message = "MongoDB connection failed"
	} else {
		status.Status = StatusHealthy
		status.Message = "Connected"
	}

	return status
}

// TelegramChecker Telegram API 健康检查器
type TelegramChecker struct {
	checkFunc func(ctx context.Context) error
}

// NewTelegramChecker 创建 Telegram 检查器
func NewTelegramChecker(checkFunc func(ctx context.Context) error) *TelegramChecker {
	return &TelegramChecker{
		checkFunc: checkFunc,
	}
}

// Name 返回检查器名称
func (c *TelegramChecker) Name() string {
	return "telegram"
}

// Check 执行检查
func (c *TelegramChecker) Check(ctx context.Context) ComponentStatus {
	start := time.Now()
	status := ComponentStatus{
		Timestamp: start,
	}

	// 执行检查
	err := c.checkFunc(ctx)
	status.Duration = time.Since(start)

	if err != nil {
		status.Status = StatusUnhealthy
		status.Error = err.Error()
		status.Message = "Telegram API connection failed"
	} else {
		status.Status = StatusHealthy
		status.Message = "Connected"
	}

	return status
}

// Server 健康检查 HTTP 服务器
type Server struct {
	server  *http.Server
	service *Service
	addr    string
}

// NewServer 创建健康检查服务器
func NewServer(addr string, service *Service) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", service.Handler())
	mux.HandleFunc("/health/ready", service.Handler()) // Kubernetes readiness probe
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		// Liveness probe - 简单检查进程是否存活
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	})

	return &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
		service: service,
		addr:    addr,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Addr 返回服务器地址
func (s *Server) Addr() string {
	return s.addr
}
