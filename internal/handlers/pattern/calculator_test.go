package pattern

import (
	"context"
	"testing"

	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/handler"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGroupRepository is a mock for GroupRepository
type MockGroupRepository struct {
	mock.Mock
}

func (m *MockGroupRepository) FindByID(ctx context.Context, id int64) (*group.Group, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*group.Group), args.Error(1)
}

func TestCalculatorHandler_Match(t *testing.T) {
	groupRepo := new(MockGroupRepository)
	h := NewCalculatorHandler(groupRepo)

	tests := []struct {
		name         string
		ctx          *handler.Context
		setupMock    func()
		expected     bool
		mockRequired bool
	}{
		{
			name: "matches basic addition in group",
			ctx: &handler.Context{
				Text:     "1+2",
				ChatType: "group",
				ChatID:   -1001234567890,
			},
			setupMock: func() {
				g := &group.Group{
					ID:       -1001234567890,
					Settings: make(map[string]interface{}),
				}
				groupRepo.On("FindByID", mock.Anything, int64(-1001234567890)).Return(g, nil).Once()
			},
			expected:     true,
			mockRequired: true,
		},
		{
			name: "matches complex expression",
			ctx: &handler.Context{
				Text:     "(10+5)*2",
				ChatType: "supergroup",
				ChatID:   -1001234567890,
			},
			setupMock: func() {
				g := &group.Group{
					ID:       -1001234567890,
					Settings: make(map[string]interface{}),
				}
				groupRepo.On("FindByID", mock.Anything, int64(-1001234567890)).Return(g, nil).Once()
			},
			expected:     true,
			mockRequired: true,
		},
		{
			name: "matches expression with spaces",
			ctx: &handler.Context{
				Text:     "1 + 2 * 3",
				ChatType: "group",
				ChatID:   -1001234567890,
			},
			setupMock: func() {
				g := &group.Group{
					ID:       -1001234567890,
					Settings: make(map[string]interface{}),
				}
				groupRepo.On("FindByID", mock.Anything, int64(-1001234567890)).Return(g, nil).Once()
			},
			expected:     true,
			mockRequired: true,
		},
		{
			name: "does not match pure number",
			ctx: &handler.Context{
				Text:     "123",
				ChatType: "group",
				ChatID:   -1001234567890,
			},
			setupMock:    func() {},
			expected:     false,
			mockRequired: false,
		},
		{
			name: "does not match text with letters",
			ctx: &handler.Context{
				Text:     "abc + 123",
				ChatType: "group",
				ChatID:   -1001234567890,
			},
			setupMock:    func() {},
			expected:     false,
			mockRequired: false,
		},
		{
			name: "does not match in private chat",
			ctx: &handler.Context{
				Text:     "1+2",
				ChatType: "private",
				ChatID:   123456,
			},
			setupMock:    func() {},
			expected:     false,
			mockRequired: false,
		},
		{
			name: "does not match when feature is disabled",
			ctx: &handler.Context{
				Text:     "1+2",
				ChatType: "group",
				ChatID:   -1001234567890,
			},
			setupMock: func() {
				g := &group.Group{
					ID: -1001234567890,
					Settings: map[string]interface{}{
						"calculator": false,
					},
				}
				groupRepo.On("FindByID", mock.Anything, int64(-1001234567890)).Return(g, nil).Once()
			},
			expected:     false,
			mockRequired: true,
		},
		{
			name: "matches when group not found (defaults to enabled)",
			ctx: &handler.Context{
				Text:     "1+2",
				ChatType: "group",
				ChatID:   -1001234567890,
			},
			setupMock: func() {
				groupRepo.On("FindByID", mock.Anything, int64(-1001234567890)).Return(nil, group.ErrGroupNotFound).Once()
			},
			expected:     true,
			mockRequired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockRequired {
				tt.setupMock()
			}
			result := h.Match(tt.ctx)
			assert.Equal(t, tt.expected, result)
			if tt.mockRequired {
				groupRepo.AssertExpectations(t)
			}
		})
	}
}

func TestCalculatorHandler_Priority(t *testing.T) {
	h := NewCalculatorHandler(nil)
	assert.Equal(t, 310, h.Priority())
}

func TestCalculatorHandler_ContinueChain(t *testing.T) {
	h := NewCalculatorHandler(nil)
	assert.False(t, h.ContinueChain())
}

func TestEvaluateExpression(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		expected    float64
		expectError bool
	}{
		// Basic operations
		{name: "simple addition", expression: "1+2", expected: 3, expectError: false},
		{name: "simple subtraction", expression: "10-5", expected: 5, expectError: false},
		{name: "simple multiplication", expression: "3*4", expected: 12, expectError: false},
		{name: "simple division", expression: "8/2", expected: 4, expectError: false},

		// Operator precedence
		{name: "precedence: addition and multiplication", expression: "2+3*4", expected: 14, expectError: false},
		{name: "precedence: subtraction and division", expression: "10-8/2", expected: 6, expectError: false},

		// Parentheses
		{name: "parentheses: basic", expression: "(2+3)*4", expected: 20, expectError: false},
		{name: "parentheses: nested", expression: "((2+3)*4)+1", expected: 21, expectError: false},
		{name: "parentheses: complex", expression: "(10+5)*(2+3)", expected: 75, expectError: false},

		// Decimals
		{name: "decimal: addition", expression: "1.5+2.5", expected: 4, expectError: false},
		{name: "decimal: division", expression: "10/3", expected: 3.333333333333333, expectError: false},

		// Negative numbers
		{name: "negative: at start", expression: "-5+3", expected: -2, expectError: false},
		{name: "negative: after operator", expression: "10+-5", expected: 5, expectError: false},
		{name: "negative: in parentheses", expression: "(-5)*2", expected: -10, expectError: false},

		// Spaces
		{name: "with spaces", expression: "1 + 2 * 3", expected: 7, expectError: false},

		// Complex expressions
		{name: "complex 1", expression: "(10+20)/(5-2)", expected: 10, expectError: false},
		{name: "complex 2", expression: "100-50/2+10*3", expected: 105, expectError: false},

		// Error cases
		{name: "error: division by zero", expression: "1/0", expected: 0, expectError: true},
		{name: "error: unmatched left parenthesis", expression: "((1+2)", expected: 0, expectError: true},
		{name: "error: unmatched right parenthesis", expression: "(1+2))", expected: 0, expectError: true},
		{name: "error: invalid expression", expression: "1++2", expected: 0, expectError: true},
		{name: "error: empty expression", expression: "", expected: 0, expectError: true},
		{name: "error: trailing operator", expression: "1+2+", expected: 0, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateExpression(tt.expression)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expected, result, 0.000001)
			}
		})
	}
}

func TestContainsOperator(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{name: "has addition", text: "1+2", expected: true},
		{name: "has subtraction", text: "5-3", expected: true},
		{name: "has multiplication", text: "2*3", expected: true},
		{name: "has division", text: "6/2", expected: true},
		{name: "has parentheses", text: "(1+2)", expected: true},
		{name: "pure number", text: "123", expected: false},
		{name: "decimal number", text: "123.45", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsOperator(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		number   float64
		expected string
	}{
		{name: "integer", number: 3, expected: "3"},
		{name: "decimal", number: 3.5, expected: "3.5"},
		{name: "long decimal", number: 3.333333, expected: "3.33333"},
		{name: "negative", number: -5, expected: "-5"},
		{name: "large number", number: 1000000, expected: "1000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNumber(tt.number)
			assert.Equal(t, tt.expected, result)
		})
	}
}
