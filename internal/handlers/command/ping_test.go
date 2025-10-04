package command

import (
	"testing"

	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGroupRepository is a mock for GroupRepository
type MockGroupRepository struct {
	mock.Mock
}

func (m *MockGroupRepository) FindByID(id int64) (*group.Group, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*group.Group), args.Error(1)
}

// MockUserRepository is a mock for UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByID(id int64) (*user.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(username string) (*user.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) Save(u *user.User) error {
	args := m.Called(u)
	return args.Error(0)
}

func (m *MockUserRepository) Update(u *user.User) error {
	args := m.Called(u)
	return args.Error(0)
}

func (m *MockUserRepository) FindAdminsByGroup(groupID int64) ([]*user.User, error) {
	args := m.Called(groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func TestPingHandler_Match(t *testing.T) {
	groupRepo := new(MockGroupRepository)
	h := NewPingHandler(groupRepo)

	tests := []struct {
		name     string
		ctx      *handler.Context
		expected bool
	}{
		{
			name: "matches /ping command",
			ctx: &handler.Context{
				Text:     "/ping",
				ChatType: "private",
			},
			expected: true,
		},
		{
			name: "matches /ping@botname command",
			ctx: &handler.Context{
				Text:     "/ping@testbot",
				ChatType: "private",
			},
			expected: true,
		},
		{
			name: "matches /ping with arguments",
			ctx: &handler.Context{
				Text:     "/ping arg1 arg2",
				ChatType: "private",
			},
			expected: true,
		},
		{
			name: "does not match non-command text",
			ctx: &handler.Context{
				Text:     "ping",
				ChatType: "private",
			},
			expected: false,
		},
		{
			name: "does not match different command",
			ctx: &handler.Context{
				Text:     "/help",
				ChatType: "private",
			},
			expected: false,
		},
		{
			name: "matches in group",
			ctx: &handler.Context{
				Text:     "/ping",
				ChatType: "group",
				ChatID:   -1001234567890,
			},
			expected: true,
		},
		{
			name: "matches in supergroup",
			ctx: &handler.Context{
				Text:     "/ping",
				ChatType: "supergroup",
				ChatID:   -1001234567890,
			},
			expected: true,
		},
		{
			name: "does not match in channel",
			ctx: &handler.Context{
				Text:     "/ping",
				ChatType: "channel",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock for group commands if needed
			if tt.ctx.ChatType == "group" || tt.ctx.ChatType == "supergroup" {
				g := &group.Group{
					ID:       tt.ctx.ChatID,
					Commands: make(map[string]*group.CommandConfig),
				}
				groupRepo.On("FindByID", tt.ctx.ChatID).Return(g, nil).Once()
			}

			result := h.Match(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPingHandler_Handle is skipped because it requires a real Telegram Bot
// Integration tests should cover the full Handle() behavior

func TestPingHandler_Priority(t *testing.T) {
	groupRepo := new(MockGroupRepository)
	h := NewPingHandler(groupRepo)

	assert.Equal(t, 100, h.Priority())
}

func TestPingHandler_ContinueChain(t *testing.T) {
	groupRepo := new(MockGroupRepository)
	h := NewPingHandler(groupRepo)

	assert.False(t, h.ContinueChain())
}

func TestPingHandler_GetName(t *testing.T) {
	groupRepo := new(MockGroupRepository)
	h := NewPingHandler(groupRepo)

	assert.Equal(t, "ping", h.GetName())
}

func TestPingHandler_GetDescription(t *testing.T) {
	groupRepo := new(MockGroupRepository)
	h := NewPingHandler(groupRepo)

	assert.Equal(t, "测试机器人是否在线", h.GetDescription())
}

func TestPingHandler_GetPermission(t *testing.T) {
	groupRepo := new(MockGroupRepository)
	h := NewPingHandler(groupRepo)

	assert.Equal(t, user.PermissionUser, h.GetPermission())
}
