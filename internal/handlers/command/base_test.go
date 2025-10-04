package command

import (
	"testing"

	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"

	"github.com/stretchr/testify/assert"
)

func TestBaseCommand_Match(t *testing.T) {
	groupRepo := new(MockGroupRepository)
	base := NewBaseCommand(
		"test",
		"Test command",
		user.PermissionUser,
		[]string{"private", "group"},
		groupRepo,
	)

	tests := []struct {
		name     string
		ctx      *handler.Context
		expected bool
	}{
		{
			name: "matches command",
			ctx: &handler.Context{
				Text:     "/test",
				ChatType: "private",
			},
			expected: true,
		},
		{
			name: "matches command with bot name",
			ctx: &handler.Context{
				Text:     "/test@botname",
				ChatType: "private",
			},
			expected: true,
		},
		{
			name: "matches command with arguments",
			ctx: &handler.Context{
				Text:     "/test arg1 arg2",
				ChatType: "private",
			},
			expected: true,
		},
		{
			name: "does not match different command",
			ctx: &handler.Context{
				Text:     "/other",
				ChatType: "private",
			},
			expected: false,
		},
		{
			name: "does not match non-command text",
			ctx: &handler.Context{
				Text:     "test",
				ChatType: "private",
			},
			expected: false,
		},
		{
			name: "does not match empty text",
			ctx: &handler.Context{
				Text:     "",
				ChatType: "private",
			},
			expected: false,
		},
		{
			name: "does not match unsupported chat type",
			ctx: &handler.Context{
				Text:     "/test",
				ChatType: "channel",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup group mock if needed
			if (tt.ctx.ChatType == "group" || tt.ctx.ChatType == "supergroup") && tt.ctx.ChatID != 0 {
				g := &group.Group{
					ID:       tt.ctx.ChatID,
					Commands: make(map[string]*group.CommandConfig),
				}
				groupRepo.On("FindByID", tt.ctx.ChatID).Return(g, nil).Once()
			}

			result := base.Match(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseCommandName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/ping", "ping"},
		{"/ping@botname", "ping"},
		{"/ping arg1 arg2", "ping"},
		{"/help", "help"},
		{"/help@mybot", "help"},
		{"/", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseCommandName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"/command arg1 arg2", []string{"arg1", "arg2"}},
		{"/command", []string{}},
		{"/command arg1", []string{"arg1"}},
		{"/command arg1 arg2 arg3", []string{"arg1", "arg2", "arg3"}},
		{"", []string{}},
		{"/cmd @username admin", []string{"@username", "admin"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseArgs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBaseCommand_GetMethods(t *testing.T) {
	groupRepo := new(MockGroupRepository)
	base := NewBaseCommand(
		"testcmd",
		"Test command description",
		user.PermissionAdmin,
		[]string{"private"},
		groupRepo,
	)

	assert.Equal(t, "testcmd", base.GetName())
	assert.Equal(t, "Test command description", base.GetDescription())
	assert.Equal(t, user.PermissionAdmin, base.GetPermission())
	assert.Equal(t, 100, base.Priority())
	assert.False(t, base.ContinueChain())
}

func TestBaseCommand_CheckPermission(t *testing.T) {
	tests := []struct {
		name           string
		userPerm       user.Permission
		requiredPerm   user.Permission
		expectError    bool
	}{
		{
			name:         "user has sufficient permission",
			userPerm:     user.PermissionAdmin,
			requiredPerm: user.PermissionUser,
			expectError:  false,
		},
		{
			name:         "user has exact permission",
			userPerm:     user.PermissionAdmin,
			requiredPerm: user.PermissionAdmin,
			expectError:  false,
		},
		{
			name:         "user has insufficient permission",
			userPerm:     user.PermissionUser,
			requiredPerm: user.PermissionAdmin,
			expectError:  true,
		},
		{
			name:         "owner can access admin commands",
			userPerm:     user.PermissionOwner,
			requiredPerm: user.PermissionAdmin,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo := new(MockGroupRepository)
			base := NewBaseCommand(
				"test",
				"Test",
				tt.requiredPerm,
				[]string{"private"},
				groupRepo,
			)

			ctx := &handler.Context{
				ChatType: "private",
				UserID:   123,
				User: &user.User{
					ID: 123,
					// 私聊使用全局权限（groupID = 0）
					Permissions: map[int64]user.Permission{0: tt.userPerm},
				},
			}

			err := base.CheckPermission(ctx)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
