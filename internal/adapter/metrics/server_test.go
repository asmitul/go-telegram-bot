package metrics

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	addr := ":9090"
	server := NewServer(addr)

	if server == nil {
		t.Fatal("expected server to be created")
	}

	if server.Addr() != addr {
		t.Errorf("expected addr %s, got %s", addr, server.Addr())
	}
}

func TestServer_StartAndShutdown(t *testing.T) {
	addr := ":19090" // 使用不同的端口避免冲突
	server := NewServer(addr)

	// 启动服务器
	go func() {
		err := server.Start()
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 测试 /health 端点
	resp, err := http.Get("http://localhost" + addr + "/health")
	if err != nil {
		t.Fatalf("failed to get /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// 测试 /metrics 端点
	resp, err = http.Get("http://localhost" + addr + "/metrics")
	if err != nil {
		t.Fatalf("failed to get /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("failed to shutdown server: %v", err)
	}
}

func TestHealthHandler(t *testing.T) {
	addr := ":19091" // 使用不同的端口
	server := NewServer(addr)

	// 启动服务器
	go func() {
		server.Start()
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 测试健康检查
	resp, err := http.Get("http://localhost" + addr + "/health")
	if err != nil {
		t.Fatalf("failed to get /health: %v", err)
	}
	defer resp.Body.Close()

	// 验证状态码
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// 验证 Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// 读取响应体
	body := make([]byte, 1024)
	n, err := resp.Body.Read(body)
	if err != nil && err.Error() != "EOF" {
		t.Errorf("failed to read body: %v", err)
	}

	bodyStr := string(body[:n])
	expected := `{"status":"ok"}`
	if bodyStr != expected {
		t.Errorf("expected body %s, got %s", expected, bodyStr)
	}

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func TestMetricsEndpoint(t *testing.T) {
	addr := ":19092" // 使用不同的端口
	server := NewServer(addr)

	// 启动服务器
	go func() {
		server.Start()
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 请求 /metrics 端点
	resp, err := http.Get("http://localhost" + addr + "/metrics")
	if err != nil {
		t.Fatalf("failed to get /metrics: %v", err)
	}
	defer resp.Body.Close()

	// 验证状态码
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// 验证响应体不为空
	body := make([]byte, 1024)
	n, err := resp.Body.Read(body)
	if err != nil && err.Error() != "EOF" {
		t.Errorf("failed to read body: %v", err)
	}

	if n == 0 {
		t.Error("expected non-empty metrics response")
	}

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func TestServer_MultipleRequests(t *testing.T) {
	addr := ":19093" // 使用不同的端口
	server := NewServer(addr)

	// 启动服务器
	go func() {
		server.Start()
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 发送多个并发请求
	numRequests := 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := http.Get("http://localhost" + addr + "/health")
			if err != nil {
				t.Errorf("request failed: %v", err)
				done <- false
				return
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
				done <- false
				return
			}

			done <- true
		}()
	}

	// 等待所有请求完成
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}
