package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server Metrics HTTP 服务器
type Server struct {
	server *http.Server
	addr   string
}

// NewServer 创建新的 Metrics 服务器
func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", healthHandler)

	return &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
		addr: addr,
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

// healthHandler 健康检查处理器
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
