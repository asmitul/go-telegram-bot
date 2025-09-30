package ping

import (
	"context"
	"errors"
	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockGroupRepository 模拟群组仓储
type MockGroupRepository struct {
	groups map[int64]*group.Group
}

func NewMockGroupRepository() *MockGroupRepository {
	return &MockGroupRepository{
		groups: make(map[int64]*group.Group),
	}
}

func (m *MockGroupRepository) FindByID(id int64) (*group.Group, error) {
	if g, ok := m.groups[id]; ok {
		return g, nil
	}
	return nil, errors.New("group not found")
}

func (m *MockGroupRepository) Save(g *group.Group) error {
	m.groups[g.ID] = g
	return nil
}

func (m *MockGroupRepository) Update(g *group.Group) error {
	m.groups[g.ID] = g
	return nil
}

func (m *MockGroupRepository) Delete(id int64) error {
	delete(m.groups, id)
	return nil
}

func (m *MockGroupRepository) FindAll() ([]*group.Group, error) {
	var groups []*group.Group
	for _, g := range m.groups {
		groups = append(groups, g)
	}
	return groups, nil
}

func TestPingHandler_Name(t *testing.T) {
	repo := NewMockGroupRepository()
	handler := NewHandler(repo)

	assert.Equal(t, "ping", handler.Name())
}

func TestPingHandler_Description(t *testing.T) {
	repo := NewMockGroupRepository()
	handler := NewHandler(repo)

	assert.NotEmpty(t, handler.Description())
}

func TestPingHandler_RequiredPermission(t *testing.T) {
	repo := NewMockGroupRepository()
	handler := NewHandler(repo)

	// Ping 命令应该允许所有用户使用
	assert.Equal(t, user.PermissionUser, handler.RequiredPermission())
}

func TestPingHandler_IsEnabled_Default(t *testing.T) {
	repo := NewMockGroupRepository()
	handler := NewHandler(repo)

	// 群组不存在时，默认启用
	enabled := handler.IsEnabled(123456)
	assert.True(t, enabled)
}

func TestPingHandler_IsEnabled_ExplicitlyDisabled(t *testing.T) {
	repo := NewMockGroupRepository()

	// 创建一个禁用了 ping 命令的群组
	g := group.NewGroup(123456, "Test Group", "supergroup")
	g.DisableCommand("ping", 12345)
	repo.Save(g)

	handler := NewHandler(repo)

	// 命令应该被禁用
	enabled := handler.IsEnabled(123456)
	assert.False(t, enabled)
}

func TestPingHandler_IsEnabled_ExplicitlyEnabled(t *testing.T) {
	repo := NewMockGroupRepository()

	// 创建一个启用了 ping 命令的群组
	g := group.NewGroup(123456, "Test Group", "supergroup")
	g.EnableCommand("ping", 12345)
	repo.Save(g)

	handler := NewHandler(repo)

	// 命令应该被启用
	enabled := handler.IsEnabled(123456)
	assert.True(t, enabled)
}

func TestPingHandler_Handle(t *testing.T) {
	repo := NewMockGroupRepository()
	handler := NewHandler(repo)

	ctx := &command.Context{
		Ctx:       context.Background(),
		UserID:    12345,
		GroupID:   123456,
		MessageID: 1,
		Text:      "/ping",
		Args:      []string{},
	}

	// 执行命令
	err := handler.Handle(ctx)

	// 应该成功执行
	assert.NoError(t, err)
}

func TestPingHandler_Handle_WithGroup(t *testing.T) {
	repo := NewMockGroupRepository()

	// 创建群组
	g := group.NewGroup(123456, "Test Group", "supergroup")
	repo.Save(g)

	handler := NewHandler(repo)

	ctx := &command.Context{
		Ctx:       context.Background(),
		UserID:    12345,
		GroupID:   123456,
		MessageID: 1,
		Text:      "/ping",
		Args:      []string{},
	}

	// 执行命令
	err := handler.Handle(ctx)

	// 应该成功执行
	assert.NoError(t, err)
}

// 基准测试
func BenchmarkPingHandler_Handle(b *testing.B) {
	repo := NewMockGroupRepository()
	handler := NewHandler(repo)

	ctx := &command.Context{
		Ctx:       context.Background(),
		UserID:    12345,
		GroupID:   123456,
		MessageID: 1,
		Text:      "/ping",
		Args:      []string{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.Handle(ctx)
	}
}

// 表驱动测试
func TestPingHandler_IsEnabled_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		setupGroup     func(*MockGroupRepository)
		groupID        int64
		expectedResult bool
	}{
		{
			name: "群组不存在，默认启用",
			setupGroup: func(repo *MockGroupRepository) {
				// 不创建群组
			},
			groupID:        999999,
			expectedResult: true,
		},
		{
			name: "群组存在，命令未配置，默认启用",
			setupGroup: func(repo *MockGroupRepository) {
				g := group.NewGroup(111111, "Group 1", "supergroup")
				repo.Save(g)
			},
			groupID:        111111,
			expectedResult: true,
		},
		{
			name: "群组存在，命令被禁用",
			setupGroup: func(repo *MockGroupRepository) {
				g := group.NewGroup(222222, "Group 2", "supergroup")
				g.DisableCommand("ping", 12345)
				repo.Save(g)
			},
			groupID:        222222,
			expectedResult: false,
		},
		{
			name: "群组存在，命令被启用",
			setupGroup: func(repo *MockGroupRepository) {
				g := group.NewGroup(333333, "Group 3", "supergroup")
				g.EnableCommand("ping", 12345)
				repo.Save(g)
			},
			groupID:        333333,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockGroupRepository()
			tt.setupGroup(repo)
			handler := NewHandler(repo)

			result := handler.IsEnabled(tt.groupID)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
