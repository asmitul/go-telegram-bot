package group

import (
	"errors"
	"time"
)

var (
	ErrGroupNotFound = errors.New("group not found")
)

// CommandConfig 命令配置
type CommandConfig struct {
	CommandName string
	Enabled     bool
	UpdatedAt   time.Time
	UpdatedBy   int64
}

// Group 群组聚合根
type Group struct {
	ID        int64
	Title     string
	Type      string                    // "group", "supergroup", "channel"
	Commands  map[string]*CommandConfig // commandName -> config
	Settings  map[string]interface{}    // 其他配置
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewGroup 创建新群组
func NewGroup(id int64, title, groupType string) *Group {
	now := time.Now()
	return &Group{
		ID:        id,
		Title:     title,
		Type:      groupType,
		Commands:  make(map[string]*CommandConfig),
		Settings:  make(map[string]interface{}),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsCommandEnabled 检查命令是否启用
func (g *Group) IsCommandEnabled(commandName string) bool {
	if config, ok := g.Commands[commandName]; ok {
		return config.Enabled
	}
	// 默认所有命令都启用
	return true
}

// EnableCommand 启用命令
func (g *Group) EnableCommand(commandName string, userID int64) {
	if config, ok := g.Commands[commandName]; ok {
		config.Enabled = true
		config.UpdatedAt = time.Now()
		config.UpdatedBy = userID
	} else {
		g.Commands[commandName] = &CommandConfig{
			CommandName: commandName,
			Enabled:     true,
			UpdatedAt:   time.Now(),
			UpdatedBy:   userID,
		}
	}
	g.UpdatedAt = time.Now()
}

// DisableCommand 禁用命令
func (g *Group) DisableCommand(commandName string, userID int64) {
	if config, ok := g.Commands[commandName]; ok {
		config.Enabled = false
		config.UpdatedAt = time.Now()
		config.UpdatedBy = userID
	} else {
		g.Commands[commandName] = &CommandConfig{
			CommandName: commandName,
			Enabled:     false,
			UpdatedAt:   time.Now(),
			UpdatedBy:   userID,
		}
	}
	g.UpdatedAt = time.Now()
}

// GetCommandConfig 获取命令配置
func (g *Group) GetCommandConfig(commandName string) *CommandConfig {
	if config, ok := g.Commands[commandName]; ok {
		return config
	}
	return &CommandConfig{
		CommandName: commandName,
		Enabled:     true,
		UpdatedAt:   time.Now(),
	}
}

// SetSetting 设置群组配置项
func (g *Group) SetSetting(key string, value interface{}) {
	g.Settings[key] = value
	g.UpdatedAt = time.Now()
}

// GetSetting 获取群组配置项
func (g *Group) GetSetting(key string) (interface{}, bool) {
	value, ok := g.Settings[key]
	return value, ok
}

// Repository 群组仓储接口
type Repository interface {
	FindByID(id int64) (*Group, error)
	Save(group *Group) error
	Update(group *Group) error
	Delete(id int64) error
	FindAll() ([]*Group, error)
}
