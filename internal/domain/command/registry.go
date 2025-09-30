package command

import (
	"fmt"
	"sync"
)

// DefaultRegistry 默认命令注册表实现
type DefaultRegistry struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

// NewRegistry 创建新的命令注册表
func NewRegistry() Registry {
	return &DefaultRegistry{
		handlers: make(map[string]Handler),
	}
}

// Register 注册命令处理器
func (r *DefaultRegistry) Register(handler Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := handler.Name()
	if name == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("command '%s' already registered", name)
	}

	r.handlers[name] = handler
	return nil
}

// Get 获取命令处理器
func (r *DefaultRegistry) Get(name string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, exists := r.handlers[name]
	return handler, exists
}

// GetAll 获取所有命令处理器
func (r *DefaultRegistry) GetAll() []Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := make([]Handler, 0, len(r.handlers))
	for _, handler := range r.handlers {
		handlers = append(handlers, handler)
	}

	return handlers
}

// Unregister 注销命令处理器
func (r *DefaultRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.handlers, name)
}

// Count 返回已注册的命令数量
func (r *DefaultRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.handlers)
}

// Has 检查命令是否已注册
func (r *DefaultRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.handlers[name]
	return exists
}

// Clear 清空所有注册的命令
func (r *DefaultRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers = make(map[string]Handler)
}

// GetCommandList 获取命令列表（用于 /help 命令）
func (r *DefaultRegistry) GetCommandList() []CommandInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]CommandInfo, 0, len(r.handlers))
	for name, handler := range r.handlers {
		list = append(list, CommandInfo{
			Name:        name,
			Description: handler.Description(),
			Permission:  handler.RequiredPermission(),
		})
	}

	return list
}

// CommandInfo 命令信息
type CommandInfo struct {
	Name        string
	Description string
	Permission  interface{} // 使用 interface{} 避免循环依赖
}
