package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RotationConfig 日志轮转配置
type RotationConfig struct {
	// Filename 日志文件路径
	Filename string
	// MaxSize 单个日志文件最大大小（MB）
	MaxSize int
	// MaxAge 日志文件保留天数
	MaxAge int
	// MaxBackups 最多保留的旧日志文件数量
	MaxBackups int
	// Compress 是否压缩旧日志文件
	Compress bool
}

// RotatingWriter 支持轮转的日志写入器
type RotatingWriter struct {
	config    RotationConfig
	file      *os.File
	size      int64
	mu        sync.Mutex
	startTime time.Time
}

// NewRotatingWriter 创建支持轮转的写入器
func NewRotatingWriter(config RotationConfig) (*RotatingWriter, error) {
	// 设置默认值
	if config.MaxSize == 0 {
		config.MaxSize = 100 // 默认 100MB
	}
	if config.MaxAge == 0 {
		config.MaxAge = 7 // 默认保留 7 天
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = 10 // 默认保留 10 个备份
	}

	w := &RotatingWriter{
		config:    config,
		startTime: time.Now(),
	}

	// 创建或打开日志文件
	if err := w.openFile(); err != nil {
		return nil, err
	}

	// 启动清理协程
	go w.cleanupOldFiles()

	return w, nil
}

// Write 实现 io.Writer 接口
func (w *RotatingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查是否需要轮转
	if w.shouldRotate(len(p)) {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	// 写入数据
	n, err = w.file.Write(p)
	w.size += int64(n)
	return n, err
}

// Close 关闭文件
func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// openFile 打开日志文件
func (w *RotatingWriter) openFile() error {
	// 确保目录存在
	dir := filepath.Dir(w.config.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// 打开文件
	file, err := os.OpenFile(w.config.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// 获取文件信息
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat log file: %v", err)
	}

	w.file = file
	w.size = info.Size()
	return nil
}

// shouldRotate 判断是否需要轮转
func (w *RotatingWriter) shouldRotate(writeSize int) bool {
	maxBytes := int64(w.config.MaxSize) * 1024 * 1024
	return w.size+int64(writeSize) >= maxBytes
}

// rotate 执行日志轮转
func (w *RotatingWriter) rotate() error {
	// 关闭当前文件
	if w.file != nil {
		w.file.Close()
	}

	// 重命名当前文件
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s.%s", w.config.Filename, timestamp)

	if err := os.Rename(w.config.Filename, backupName); err != nil {
		return fmt.Errorf("failed to rotate log file: %v", err)
	}

	// 打开新文件
	return w.openFile()
}

// cleanupOldFiles 清理过期的日志文件
func (w *RotatingWriter) cleanupOldFiles() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		w.cleanup()
	}
}

// cleanup 执行清理
func (w *RotatingWriter) cleanup() {
	w.mu.Lock()
	defer w.mu.Unlock()

	dir := filepath.Dir(w.config.Filename)
	basename := filepath.Base(w.config.Filename)

	// 查找所有备份文件
	pattern := filepath.Join(dir, basename+".*")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	// 按修改时间排序并删除过期文件
	cutoff := time.Now().AddDate(0, 0, -w.config.MaxAge)
	backupCount := 0

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// 检查是否过期
		if info.ModTime().Before(cutoff) {
			os.Remove(file)
			continue
		}

		// 检查备份数量
		backupCount++
		if backupCount > w.config.MaxBackups {
			os.Remove(file)
		}
	}
}

// MultiWriter 多写入器（同时写入多个目标）
type MultiWriter struct {
	writers []io.Writer
}

// NewMultiWriter 创建多写入器
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write 实现 io.Writer 接口
func (m *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range m.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, io.ErrShortWrite
		}
	}
	return len(p), nil
}
