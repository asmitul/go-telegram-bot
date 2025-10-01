package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"telegram-bot/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLogger 用于测试的简单 logger
type MockLogger struct {
	logs []string
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.logs = append(m.logs, msg)
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.logs = append(m.logs, msg)
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.logs = append(m.logs, msg)
}

func (m *MockLogger) Error(msg string, args ...interface{}) {
	m.logs = append(m.logs, msg)
}

func (m *MockLogger) WithField(key string, value interface{}) logger.Logger {
	return m
}

func (m *MockLogger) WithFields(fields map[string]interface{}) logger.Logger {
	return m
}

func (m *MockLogger) WithContext(ctx context.Context) logger.Logger {
	return m
}

func (m *MockLogger) SetLevel(level logger.Level) {}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		want     time.Duration
		wantErr  bool
	}{
		{
			name:     "parse seconds",
			schedule: "30s",
			want:     30 * time.Second,
			wantErr:  false,
		},
		{
			name:     "parse minutes",
			schedule: "5m",
			want:     5 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "parse hours",
			schedule: "1h",
			want:     1 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "parse days",
			schedule: "1d",
			want:     24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "parse multiple days",
			schedule: "7d",
			want:     7 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "invalid format",
			schedule: "invalid",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "empty schedule",
			schedule: "",
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.schedule)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSimpleJob(t *testing.T) {
	t.Run("creates simple job correctly", func(t *testing.T) {
		called := false
		fn := func(ctx context.Context) error {
			called = true
			return nil
		}

		job := NewSimpleJob("test-job", "5m", fn)

		assert.Equal(t, "test-job", job.Name())
		assert.Equal(t, "5m", job.Schedule())

		err := job.Run(context.Background())
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("propagates error from function", func(t *testing.T) {
		expectedErr := errors.New("job failed")
		fn := func(ctx context.Context) error {
			return expectedErr
		}

		job := NewSimpleJob("failing-job", "5m", fn)

		err := job.Run(context.Background())
		assert.Equal(t, expectedErr, err)
	})
}

func TestScheduler_AddJob(t *testing.T) {
	log := &MockLogger{}
	scheduler := NewScheduler(log)

	assert.Equal(t, 0, len(scheduler.GetJobs()))

	job := NewSimpleJob("test-job", "5m", func(ctx context.Context) error { return nil })
	scheduler.AddJob(job)

	assert.Equal(t, 1, len(scheduler.GetJobs()))
	assert.Equal(t, "test-job", scheduler.GetJobs()[0].Name())
}

func TestScheduler_StartStop(t *testing.T) {
	log := &MockLogger{}
	scheduler := NewScheduler(log)

	var counter int32
	job := NewSimpleJob("test-job", "100ms", func(ctx context.Context) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})

	scheduler.AddJob(job)
	scheduler.Start()

	// 等待任务执行几次
	time.Sleep(350 * time.Millisecond)

	scheduler.Stop()

	// 验证任务至少执行了2次（立即执行1次 + ticker触发至少1次）
	count := atomic.LoadInt32(&counter)
	assert.GreaterOrEqual(t, count, int32(2), "job should have run at least twice")
}

func TestScheduler_MultipleJobs(t *testing.T) {
	log := &MockLogger{}
	scheduler := NewScheduler(log)

	var counter1, counter2 int32

	job1 := NewSimpleJob("job1", "100ms", func(ctx context.Context) error {
		atomic.AddInt32(&counter1, 1)
		return nil
	})

	job2 := NewSimpleJob("job2", "150ms", func(ctx context.Context) error {
		atomic.AddInt32(&counter2, 1)
		return nil
	})

	scheduler.AddJob(job1)
	scheduler.AddJob(job2)

	scheduler.Start()
	time.Sleep(400 * time.Millisecond)
	scheduler.Stop()

	// job1 应该执行更多次（因为间隔更短）
	count1 := atomic.LoadInt32(&counter1)
	count2 := atomic.LoadInt32(&counter2)

	assert.GreaterOrEqual(t, count1, int32(2))
	assert.GreaterOrEqual(t, count2, int32(1))
}

func TestScheduler_JobError(t *testing.T) {
	log := &MockLogger{}
	scheduler := NewScheduler(log)

	var callCount int32
	job := NewSimpleJob("failing-job", "100ms", func(ctx context.Context) error {
		atomic.AddInt32(&callCount, 1)
		return errors.New("job error")
	})

	scheduler.AddJob(job)
	scheduler.Start()

	time.Sleep(250 * time.Millisecond)
	scheduler.Stop()

	// 即使任务失败，也应该继续执行
	count := atomic.LoadInt32(&callCount)
	assert.GreaterOrEqual(t, count, int32(2), "job should continue running even after errors")
}

func TestScheduler_ContextCancellation(t *testing.T) {
	log := &MockLogger{}
	scheduler := NewScheduler(log)

	jobStarted := make(chan struct{})
	jobCancelled := false

	job := NewSimpleJob("long-job", "100ms", func(ctx context.Context) error {
		close(jobStarted)
		select {
		case <-ctx.Done():
			jobCancelled = true
			return ctx.Err()
		case <-time.After(10 * time.Second):
			return nil
		}
	})

	scheduler.AddJob(job)
	scheduler.Start()

	// 等待任务启动
	<-jobStarted

	// 立即停止调度器
	scheduler.Stop()

	// 验证任务接收到取消信号
	assert.True(t, jobCancelled, "job should be cancelled when scheduler stops")
}

func TestScheduler_GetJobs(t *testing.T) {
	log := &MockLogger{}
	scheduler := NewScheduler(log)

	job1 := NewSimpleJob("job1", "5m", func(ctx context.Context) error { return nil })
	job2 := NewSimpleJob("job2", "10m", func(ctx context.Context) error { return nil })

	scheduler.AddJob(job1)
	scheduler.AddJob(job2)

	jobs := scheduler.GetJobs()
	require.Equal(t, 2, len(jobs))

	// 验证返回的是副本
	jobs[0] = nil
	assert.NotNil(t, scheduler.GetJobs()[0], "GetJobs should return a copy")
}

func TestScheduler_InvalidSchedule(t *testing.T) {
	log := &MockLogger{}
	scheduler := NewScheduler(log)

	var called bool
	job := NewSimpleJob("invalid-job", "invalid-schedule", func(ctx context.Context) error {
		called = true
		return nil
	})

	scheduler.AddJob(job)
	scheduler.Start()

	time.Sleep(100 * time.Millisecond)
	scheduler.Stop()

	// 无效的调度格式应该导致任务不执行
	assert.False(t, called, "job with invalid schedule should not run")
}

func TestScheduler_ImmediateExecution(t *testing.T) {
	log := &MockLogger{}
	scheduler := NewScheduler(log)

	executed := make(chan struct{}, 1)
	job := NewSimpleJob("immediate-job", "1h", func(ctx context.Context) error {
		select {
		case executed <- struct{}{}:
		default:
		}
		return nil
	})

	scheduler.AddJob(job)
	scheduler.Start()

	// 任务应该立即执行一次
	select {
	case <-executed:
		// 成功
	case <-time.After(1 * time.Second):
		t.Fatal("job should execute immediately after start")
	}

	scheduler.Stop()
}

func TestScheduler_ConcurrentSafety(t *testing.T) {
	log := &MockLogger{}
	scheduler := NewScheduler(log)

	// 并发添加任务
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			job := NewSimpleJob(
				"job-"+string(rune('0'+n)),
				"100ms",
				func(ctx context.Context) error { return nil },
			)
			scheduler.AddJob(job)
			done <- struct{}{}
		}(i)
	}

	// 等待所有任务添加完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 并发获取任务列表
	for i := 0; i < 10; i++ {
		go func() {
			_ = scheduler.GetJobs()
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, len(scheduler.GetJobs()))
}
