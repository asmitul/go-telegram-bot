package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatus(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{"healthy", StatusHealthy, "healthy"},
		{"unhealthy", StatusUnhealthy, "unhealthy"},
		{"degraded", StatusDegraded, "degraded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("expected %s, got %s", tt.want, tt.status)
			}
		})
	}
}

func TestSimpleChecker_Success(t *testing.T) {
	checker := NewSimpleChecker("test", func(ctx context.Context) error {
		return nil
	})

	if checker.Name() != "test" {
		t.Errorf("expected name 'test', got %s", checker.Name())
	}

	ctx := context.Background()
	status := checker.Check(ctx)

	if status.Status != StatusHealthy {
		t.Errorf("expected healthy status, got %s", status.Status)
	}

	if status.Message != "OK" {
		t.Errorf("expected message 'OK', got %s", status.Message)
	}

	if status.Error != "" {
		t.Errorf("expected no error, got %s", status.Error)
	}

	if status.Duration == 0 {
		t.Error("expected duration > 0")
	}
}

func TestSimpleChecker_Failure(t *testing.T) {
	testErr := errors.New("test error")
	checker := NewSimpleChecker("test", func(ctx context.Context) error {
		return testErr
	})

	ctx := context.Background()
	status := checker.Check(ctx)

	if status.Status != StatusUnhealthy {
		t.Errorf("expected unhealthy status, got %s", status.Status)
	}

	if status.Error != testErr.Error() {
		t.Errorf("expected error '%s', got '%s'", testErr.Error(), status.Error)
	}

	if status.Duration == 0 {
		t.Error("expected duration > 0")
	}
}

func TestSimpleChecker_ContextTimeout(t *testing.T) {
	checker := NewSimpleChecker("test", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	status := checker.Check(ctx)

	if status.Status != StatusUnhealthy {
		t.Errorf("expected unhealthy status, got %s", status.Status)
	}

	if status.Error == "" {
		t.Error("expected timeout error")
	}
}

func TestMongoDBChecker_Success(t *testing.T) {
	checker := NewMongoDBChecker(func(ctx context.Context) error {
		return nil
	})

	if checker.Name() != "mongodb" {
		t.Errorf("expected name 'mongodb', got %s", checker.Name())
	}

	ctx := context.Background()
	status := checker.Check(ctx)

	if status.Status != StatusHealthy {
		t.Errorf("expected healthy status, got %s", status.Status)
	}

	if status.Message != "Connected" {
		t.Errorf("expected message 'Connected', got %s", status.Message)
	}

	if status.Error != "" {
		t.Errorf("expected no error, got %s", status.Error)
	}
}

func TestMongoDBChecker_Failure(t *testing.T) {
	testErr := errors.New("connection failed")
	checker := NewMongoDBChecker(func(ctx context.Context) error {
		return testErr
	})

	ctx := context.Background()
	status := checker.Check(ctx)

	if status.Status != StatusUnhealthy {
		t.Errorf("expected unhealthy status, got %s", status.Status)
	}

	if status.Message != "MongoDB connection failed" {
		t.Errorf("expected message 'MongoDB connection failed', got %s", status.Message)
	}

	if status.Error != testErr.Error() {
		t.Errorf("expected error '%s', got '%s'", testErr.Error(), status.Error)
	}
}

func TestTelegramChecker_Success(t *testing.T) {
	checker := NewTelegramChecker(func(ctx context.Context) error {
		return nil
	})

	if checker.Name() != "telegram" {
		t.Errorf("expected name 'telegram', got %s", checker.Name())
	}

	ctx := context.Background()
	status := checker.Check(ctx)

	if status.Status != StatusHealthy {
		t.Errorf("expected healthy status, got %s", status.Status)
	}

	if status.Message != "Connected" {
		t.Errorf("expected message 'Connected', got %s", status.Message)
	}

	if status.Error != "" {
		t.Errorf("expected no error, got %s", status.Error)
	}
}

func TestTelegramChecker_Failure(t *testing.T) {
	testErr := errors.New("API error")
	checker := NewTelegramChecker(func(ctx context.Context) error {
		return testErr
	})

	ctx := context.Background()
	status := checker.Check(ctx)

	if status.Status != StatusUnhealthy {
		t.Errorf("expected unhealthy status, got %s", status.Status)
	}

	if status.Message != "Telegram API connection failed" {
		t.Errorf("expected message 'Telegram API connection failed', got %s", status.Message)
	}

	if status.Error != testErr.Error() {
		t.Errorf("expected error '%s', got '%s'", testErr.Error(), status.Error)
	}
}

func TestService_NoCheckers(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	response := service.Check(ctx)

	if response.Status != StatusHealthy {
		t.Errorf("expected healthy status with no checkers, got %s", response.Status)
	}

	if len(response.Components) != 0 {
		t.Errorf("expected no components, got %d", len(response.Components))
	}

	if response.Uptime == 0 {
		t.Error("expected uptime > 0")
	}
}

func TestService_AllHealthy(t *testing.T) {
	service := NewService()

	checker1 := NewSimpleChecker("check1", func(ctx context.Context) error {
		return nil
	})
	checker2 := NewSimpleChecker("check2", func(ctx context.Context) error {
		return nil
	})

	service.RegisterChecker(checker1)
	service.RegisterChecker(checker2)

	ctx := context.Background()
	response := service.Check(ctx)

	if response.Status != StatusHealthy {
		t.Errorf("expected healthy status, got %s", response.Status)
	}

	if len(response.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(response.Components))
	}

	for name, status := range response.Components {
		if status.Status != StatusHealthy {
			t.Errorf("expected %s to be healthy, got %s", name, status.Status)
		}
	}
}

func TestService_SomeFailed(t *testing.T) {
	service := NewService()

	checker1 := NewSimpleChecker("check1", func(ctx context.Context) error {
		return nil
	})
	checker2 := NewSimpleChecker("check2", func(ctx context.Context) error {
		return errors.New("failed")
	})

	service.RegisterChecker(checker1)
	service.RegisterChecker(checker2)

	ctx := context.Background()
	response := service.Check(ctx)

	if response.Status != StatusDegraded {
		t.Errorf("expected degraded status, got %s", response.Status)
	}

	if len(response.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(response.Components))
	}

	if response.Components["check1"].Status != StatusHealthy {
		t.Error("expected check1 to be healthy")
	}

	if response.Components["check2"].Status != StatusUnhealthy {
		t.Error("expected check2 to be unhealthy")
	}
}

func TestService_AllFailed(t *testing.T) {
	service := NewService()

	checker1 := NewSimpleChecker("check1", func(ctx context.Context) error {
		return errors.New("failed")
	})
	checker2 := NewSimpleChecker("check2", func(ctx context.Context) error {
		return errors.New("failed")
	})

	service.RegisterChecker(checker1)
	service.RegisterChecker(checker2)

	ctx := context.Background()
	response := service.Check(ctx)

	if response.Status != StatusUnhealthy {
		t.Errorf("expected unhealthy status, got %s", response.Status)
	}

	for name, status := range response.Components {
		if status.Status != StatusUnhealthy {
			t.Errorf("expected %s to be unhealthy, got %s", name, status.Status)
		}
	}
}

func TestService_ConcurrentExecution(t *testing.T) {
	service := NewService()

	// Add multiple checkers that take time
	for i := 0; i < 5; i++ {
		name := string(rune('a' + i))
		checker := NewSimpleChecker(name, func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
		service.RegisterChecker(checker)
	}

	ctx := context.Background()
	start := time.Now()
	response := service.Check(ctx)
	duration := time.Since(start)

	// If sequential, would take 250ms+, concurrent should be ~50ms
	if duration > 150*time.Millisecond {
		t.Errorf("expected concurrent execution to be fast, took %v", duration)
	}

	if len(response.Components) != 5 {
		t.Errorf("expected 5 components, got %d", len(response.Components))
	}
}

func TestService_ContextCancellation(t *testing.T) {
	service := NewService()

	checker := NewSimpleChecker("slow", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	})

	service.RegisterChecker(checker)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	response := service.Check(ctx)

	if response.Status != StatusUnhealthy {
		t.Errorf("expected unhealthy status due to timeout, got %s", response.Status)
	}
}

func TestHandler_Healthy(t *testing.T) {
	service := NewService()
	service.RegisterChecker(NewSimpleChecker("test", func(ctx context.Context) error {
		return nil
	}))

	handler := service.Handler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != StatusHealthy {
		t.Errorf("expected healthy status, got %s", response.Status)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestHandler_Degraded(t *testing.T) {
	service := NewService()
	service.RegisterChecker(NewSimpleChecker("ok", func(ctx context.Context) error {
		return nil
	}))
	service.RegisterChecker(NewSimpleChecker("fail", func(ctx context.Context) error {
		return errors.New("failed")
	}))

	handler := service.Handler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Degraded still returns 200
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != StatusDegraded {
		t.Errorf("expected degraded status, got %s", response.Status)
	}
}

func TestHandler_Unhealthy(t *testing.T) {
	service := NewService()
	service.RegisterChecker(NewSimpleChecker("fail", func(ctx context.Context) error {
		return errors.New("failed")
	}))

	handler := service.Handler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != StatusUnhealthy {
		t.Errorf("expected unhealthy status, got %s", response.Status)
	}
}

func TestServer_Endpoints(t *testing.T) {
	service := NewService()
	service.RegisterChecker(NewSimpleChecker("test", func(ctx context.Context) error {
		return nil
	}))

	server := NewServer(":8080", service)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"health endpoint", "/health", http.StatusOK},
		{"ready endpoint", "/health/ready", http.StatusOK},
		{"live endpoint", "/health/live", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			server.server.Handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestServer_LivenessProbe(t *testing.T) {
	service := NewService()
	server := NewServer(":8080", service)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "alive" {
		t.Errorf("expected status 'alive', got %s", response["status"])
	}
}

func TestServer_Addr(t *testing.T) {
	service := NewService()
	addr := ":9090"
	server := NewServer(addr, service)

	if server.Addr() != addr {
		t.Errorf("expected addr %s, got %s", addr, server.Addr())
	}
}

func TestServer_Shutdown(t *testing.T) {
	service := NewService()
	server := NewServer(":0", service)

	// Start server in background
	go func() {
		server.Start()
	}()

	time.Sleep(10 * time.Millisecond)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("unexpected shutdown error: %v", err)
	}
}

func TestService_RegisterChecker_Concurrent(t *testing.T) {
	service := NewService()

	// Register checkers concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			name := string(rune('a' + idx))
			checker := NewSimpleChecker(name, func(ctx context.Context) error {
				return nil
			})
			service.RegisterChecker(checker)
			done <- true
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < 10; i++ {
		<-done
	}

	ctx := context.Background()
	response := service.Check(ctx)

	if len(response.Components) != 10 {
		t.Errorf("expected 10 components, got %d", len(response.Components))
	}
}

func TestComponentStatus_JSONSerialization(t *testing.T) {
	status := ComponentStatus{
		Status:    StatusHealthy,
		Message:   "Test message",
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
		Error:     "test error",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ComponentStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Status != status.Status {
		t.Errorf("expected status %s, got %s", status.Status, decoded.Status)
	}

	if decoded.Message != status.Message {
		t.Errorf("expected message %s, got %s", status.Message, decoded.Message)
	}
}

func TestHealthResponse_JSONSerialization(t *testing.T) {
	response := HealthResponse{
		Status:    StatusDegraded,
		Timestamp: time.Now(),
		Uptime:    10 * time.Minute,
		Components: map[string]ComponentStatus{
			"test": {
				Status:  StatusHealthy,
				Message: "OK",
			},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded HealthResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Status != response.Status {
		t.Errorf("expected status %s, got %s", response.Status, decoded.Status)
	}

	if len(decoded.Components) != len(response.Components) {
		t.Errorf("expected %d components, got %d", len(response.Components), len(decoded.Components))
	}
}
