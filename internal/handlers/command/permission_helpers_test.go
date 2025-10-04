package command

import (
	"testing"

	"telegram-bot/internal/domain/user"
	"telegram-bot/internal/handler"

	"github.com/stretchr/testify/assert"
)

func TestGetPermIcon(t *testing.T) {
	tests := []struct {
		name     string
		perm     user.Permission
		expected string
	}{
		{"Owner", user.PermissionOwner, "👑"},
		{"SuperAdmin", user.PermissionSuperAdmin, "⭐"},
		{"Admin", user.PermissionAdmin, "🛡"},
		{"User", user.PermissionUser, "👤"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPermIcon(tt.perm)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatUsername(t *testing.T) {
	tests := []struct {
		name     string
		user     *user.User
		expected string
	}{
		{
			name: "user with username",
			user: &user.User{
				ID:        123,
				Username:  "testuser",
				FirstName: "Test",
			},
			expected: "@testuser",
		},
		{
			name: "user without username but with first name",
			user: &user.User{
				ID:        123,
				FirstName: "Test",
			},
			expected: "Test",
		},
		{
			name: "user without username or first name",
			user: &user.User{
				ID: 123,
			},
			expected: "User#123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatUsername(tt.user)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTargetUser_FromArgs(t *testing.T) {
	userRepo := new(MockUserRepository)
	targetUser := &user.User{
		ID:       456,
		Username: "target",
	}

	userRepo.On("FindByUsername", "target").Return(targetUser, nil).Once()

	ctx := &handler.Context{
		Text: "/command @target",
	}

	result, err := GetTargetUser(ctx, userRepo)
	assert.NoError(t, err)
	assert.Equal(t, targetUser, result)
	userRepo.AssertExpectations(t)
}

func TestGetTargetUser_FromReply(t *testing.T) {
	userRepo := new(MockUserRepository)
	targetUser := &user.User{
		ID:       456,
		Username: "target",
	}

	userRepo.On("FindByID", int64(456)).Return(targetUser, nil).Once()

	ctx := &handler.Context{
		Text: "/command",
		ReplyTo: &handler.ReplyInfo{
			UserID: 456,
		},
	}

	result, err := GetTargetUser(ctx, userRepo)
	assert.NoError(t, err)
	assert.Equal(t, targetUser, result)
	userRepo.AssertExpectations(t)
}

func TestGetTargetUser_NoTarget(t *testing.T) {
	userRepo := new(MockUserRepository)

	ctx := &handler.Context{
		Text: "/command",
	}

	result, err := GetTargetUser(ctx, userRepo)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "未指定目标用户")
}
