package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"telegram-bot/pkg/logger"
)

// Job 定时任务接口
type Job interface {
	// Name 返回任务名称
	Name() string
	// Run 执行任务
	Run(ctx context.Context) error
	// Schedule 返回任务调度时间表达式（cron 格式或间隔时间）
	Schedule() string
}

// TaskFunc 任务函数类型
type TaskFunc func(ctx context.Context) error

// SimpleJob 简单任务实现
type SimpleJob struct {
	name     string
	schedule string
	fn       TaskFunc
}

// NewSimpleJob 创建简单任务
func NewSimpleJob(name, schedule string, fn TaskFunc) *SimpleJob {
	return &SimpleJob{
		name:     name,
		schedule: schedule,
		fn:       fn,
	}
}

func (j *SimpleJob) Name() string {
	return j.name
}

func (j *SimpleJob) Run(ctx context.Context) error {
	return j.fn(ctx)
}

func (j *SimpleJob) Schedule() string {
	return j.schedule
}

// Scheduler 任务调度器
type Scheduler struct {
	jobs   []Job
	logger logger.Logger
	mu     sync.RWMutex
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewScheduler 创建调度器
func NewScheduler(log logger.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		jobs:   make([]Job, 0),
		logger: log,
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddJob 添加任务
func (s *Scheduler) AddJob(job Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
	s.logger.Info("Job added", "name", job.Name(), "schedule", job.Schedule())
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.logger.Info("Scheduler starting", "jobs", len(s.jobs))

	for _, job := range s.jobs {
		s.wg.Add(1)
		go s.runJob(job)
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.logger.Info("Scheduler stopping...")
	s.cancel()

	// 等待所有任务完成（最多30秒）
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("All scheduled jobs stopped")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Scheduler stop timeout: some jobs may not have completed")
	}
}

// runJob 运行单个任务
func (s *Scheduler) runJob(job Job) {
	defer s.wg.Done()

	interval, err := parseDuration(job.Schedule())
	if err != nil {
		s.logger.Error("Invalid schedule format", "job", job.Name(), "schedule", job.Schedule(), "error", err)
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Info("Job started", "name", job.Name(), "interval", interval)

	// 立即执行一次（同步）
	// 注意：如果任务执行时间较长，会阻塞定时器启动
	// 但可以确保 context 取消信号正确传递
	s.executeJob(job)

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Job stopped", "name", job.Name())
			return
		case <-ticker.C:
			s.executeJob(job)
		}
	}
}

// executeJob 执行任务
func (s *Scheduler) executeJob(job Job) {
	startTime := time.Now()
	s.logger.Info("Job executing", "name", job.Name())

	// 创建带超时的 context（任务最多执行5分钟）
	// 继承 s.ctx，这样当 scheduler 停止时，长时间运行的任务会被取消
	// 同时保证正常情况下任务有足够时间完成
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	err := job.Run(ctx)
	duration := time.Since(startTime)

	if err != nil {
		s.logger.Error("Job failed",
			"name", job.Name(),
			"error", err,
			"duration", duration,
		)
	} else {
		s.logger.Info("Job completed",
			"name", job.Name(),
			"duration", duration,
		)
	}
}

// GetJobs 获取所有任务
func (s *Scheduler) GetJobs() []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]Job, len(s.jobs))
	copy(jobs, s.jobs)
	return jobs
}

// parseDuration 解析时间间隔
// 支持格式：
// - "30s" - 30秒
// - "5m" - 5分钟
// - "1h" - 1小时
// - "1d" - 1天
func parseDuration(schedule string) (time.Duration, error) {
	if schedule == "" {
		return 0, fmt.Errorf("empty schedule")
	}

	// 尝试直接解析（支持 s, m, h）
	duration, err := time.ParseDuration(schedule)
	if err == nil {
		return duration, nil
	}

	// 支持 "d" (天) 格式
	if len(schedule) > 1 && schedule[len(schedule)-1] == 'd' {
		days := schedule[:len(schedule)-1]
		var d int
		_, err := fmt.Sscanf(days, "%d", &d)
		if err != nil {
			return 0, fmt.Errorf("invalid schedule format: %s", schedule)
		}
		return time.Duration(d) * 24 * time.Hour, nil
	}

	return 0, fmt.Errorf("invalid schedule format: %s (supported: 30s, 5m, 1h, 1d)", schedule)
}
