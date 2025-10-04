package command

import (
	"context"
	"testing"

	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/handler"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGroupRepositoryWithUpdate extends MockGroupRepository with Update method
type MockGroupRepositoryWithUpdate struct {
	MockGroupRepository
}

func (m *MockGroupRepositoryWithUpdate) Update(ctx context.Context, g *group.Group) error {
	args := m.Called(ctx, g)
	return args.Error(0)
}

func TestToggleCalcHandler_Match(t *testing.T) {
	groupRepo := new(MockGroupRepositoryWithUpdate)
	userRepo := new(MockUserRepository)
	h := NewToggleCalcHandler(groupRepo, userRepo)

	tests := []struct {
		name      string
		ctx       *handler.Context
		setupMock func()
		expected  bool
	}{
		{
			name: "matches /togglecalc in group",
			ctx: &handler.Context{
				Text:     "/togglecalc",
				ChatType: "group",
				ChatID:   -1001234567890,
			},
			setupMock: func() {
				g := &group.Group{
					ID:       -1001234567890,
					Commands: make(map[string]*group.CommandConfig),
				}
				groupRepo.On("FindByID", mock.Anything, int64(-1001234567890)).Return(g, nil).Once()
			},
			expected: true,
		},
		{
			name: "matches /togglecalc in supergroup",
			ctx: &handler.Context{
				Text:     "/togglecalc",
				ChatType: "supergroup",
				ChatID:   -1001234567890,
			},
			setupMock: func() {
				g := &group.Group{
					ID:       -1001234567890,
					Commands: make(map[string]*group.CommandConfig),
				}
				groupRepo.On("FindByID", mock.Anything, int64(-1001234567890)).Return(g, nil).Once()
			},
			expected: true,
		},
		{
			name: "does not match in private chat",
			ctx: &handler.Context{
				Text:     "/togglecalc",
				ChatType: "private",
				ChatID:   123456,
			},
			setupMock: func() {},
			expected:  false,
		},
		{
			name: "does not match different command",
			ctx: &handler.Context{
				Text:     "/ping",
				ChatType: "group",
				ChatID:   -1001234567890,
			},
			setupMock: func() {},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			result := h.Match(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestToggleCalcHandler_Handle is skipped because it requires a real Telegram Bot
// to send responses. The Handle method's core logic is tested through integration tests.
// Unit tests focus on Match(), Priority(), and ContinueChain() methods.

func TestToggleCalcHandler_Priority(t *testing.T) {
	groupRepo := new(MockGroupRepositoryWithUpdate)
	userRepo := new(MockUserRepository)
	h := NewToggleCalcHandler(groupRepo, userRepo)

	assert.Equal(t, 100, h.Priority())
}

func TestToggleCalcHandler_ContinueChain(t *testing.T) {
	groupRepo := new(MockGroupRepositoryWithUpdate)
	userRepo := new(MockUserRepository)
	h := NewToggleCalcHandler(groupRepo, userRepo)

	assert.False(t, h.ContinueChain())
}
