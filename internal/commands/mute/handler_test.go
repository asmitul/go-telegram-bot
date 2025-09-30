package mute

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/user"

	"github.com/go-telegram/bot/models"
)

// MockTelegramAPI 模拟 Telegram API
type MockTelegramAPI struct {
	SendMessageFunc                      func(chatID int64, text string) error
	SendMessageWithReplyFunc             func(chatID int64, text string, replyToMessageID int) error
	RestrictChatMemberFunc               func(chatID, userID int64, permissions models.ChatPermissions) error
	RestrictChatMemberWithDurationFunc   func(chatID, userID int64, permissions models.ChatPermissions, until time.Time) error
	Messages                             []string
	RestrictCalls                        []RestrictCall
	RestrictWithDurationCalls            []RestrictWithDurationCall
}

type RestrictCall struct {
	ChatID      int64
	UserID      int64
	Permissions models.ChatPermissions
}

type RestrictWithDurationCall struct {
	ChatID      int64
	UserID      int64
	Permissions models.ChatPermissions
	Until       time.Time
}

func (m *MockTelegramAPI) SendMessage(chatID int64, text string) error {
	m.Messages = append(m.Messages, text)
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(chatID, text)
	}
	return nil
}

func (m *MockTelegramAPI) SendMessageWithReply(chatID int64, text string, replyToMessageID int) error {
	m.Messages = append(m.Messages, text)
	if m.SendMessageWithReplyFunc != nil {
		return m.SendMessageWithReplyFunc(chatID, text, replyToMessageID)
	}
	return nil
}

func (m *MockTelegramAPI) RestrictChatMember(chatID, userID int64, permissions models.ChatPermissions) error {
	m.RestrictCalls = append(m.RestrictCalls, RestrictCall{
		ChatID:      chatID,
		UserID:      userID,
		Permissions: permissions,
	})
	if m.RestrictChatMemberFunc != nil {
		return m.RestrictChatMemberFunc(chatID, userID, permissions)
	}
	return nil
}

func (m *MockTelegramAPI) RestrictChatMemberWithDuration(chatID, userID int64, permissions models.ChatPermissions, until time.Time) error {
	m.RestrictWithDurationCalls = append(m.RestrictWithDurationCalls, RestrictWithDurationCall{
		ChatID:      chatID,
		UserID:      userID,
		Permissions: permissions,
		Until:       until,
	})
	if m.RestrictChatMemberWithDurationFunc != nil {
		return m.RestrictChatMemberWithDurationFunc(chatID, userID, permissions, until)
	}
	return nil
}

// MockUserRepository 模拟用户仓储
type MockUserRepository struct {
	FindByIDFunc func(userID int64, groupID int64) (*user.User, error)
	Users        map[int64]*user.User
}

func (m *MockUserRepository) FindByID(userID int64, groupID int64) (*user.User, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(userID, groupID)
	}
	if u, ok := m.Users[userID]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func TestHandler_Name(t *testing.T) {
	h := NewHandler(nil, nil)
	if h.Name() != "mute" {
		t.Errorf("expected name 'mute', got '%s'", h.Name())
	}
}

func TestHandler_Description(t *testing.T) {
	h := NewHandler(nil, nil)
	if h.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestHandler_RequiredPermission(t *testing.T) {
	h := NewHandler(nil, nil)
	if h.RequiredPermission() != user.PermissionAdmin {
		t.Errorf("expected Admin permission, got %v", h.RequiredPermission())
	}
}

func TestHandler_HandleMute(t *testing.T) {
	tests := []struct {
		name          string
		ctx           *command.Context
		setupMocks    func(*MockTelegramAPI, *MockUserRepository)
		wantErr       bool
		wantMsgCount  int
		checkRestrict func(*testing.T, *MockTelegramAPI)
	}{
		{
			name: "永久禁言 - 指定用户ID",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"123456789"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkRestrict: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.RestrictCalls) != 1 {
					t.Errorf("expected 1 RestrictChatMember call, got %d", len(api.RestrictCalls))
					return
				}
				call := api.RestrictCalls[0]
				if call.UserID != 123456789 {
					t.Errorf("expected userID 123456789, got %d", call.UserID)
				}
				if call.Permissions.CanSendMessages {
					t.Error("expected CanSendMessages to be false")
				}
			},
		},
		{
			name: "临时禁言 - 1小时",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"123456789", "1h"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkRestrict: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.RestrictWithDurationCalls) != 1 {
					t.Errorf("expected 1 RestrictChatMemberWithDuration call, got %d", len(api.RestrictWithDurationCalls))
					return
				}
				call := api.RestrictWithDurationCalls[0]
				if call.UserID != 123456789 {
					t.Errorf("expected userID 123456789, got %d", call.UserID)
				}
				// 检查时间是否在合理范围内（约1小时后）
				expectedTime := time.Now().Add(1 * time.Hour)
				diff := call.Until.Sub(expectedTime)
				if diff < -time.Minute || diff > time.Minute {
					t.Errorf("expected until time around %v, got %v", expectedTime, call.Until)
				}
			},
		},
		{
			name: "临时禁言带原因 - 30分钟",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"123456789", "30m", "刷屏"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkRestrict: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.RestrictWithDurationCalls) != 1 {
					t.Errorf("expected 1 RestrictChatMemberWithDuration call, got %d", len(api.RestrictWithDurationCalls))
					return
				}
				// 检查消息是否包含原因
				if len(api.Messages) == 0 {
					t.Error("expected at least one message")
					return
				}
				msg := api.Messages[0]
				if !contains(msg, "刷屏") {
					t.Errorf("expected message to contain reason '刷屏', got: %s", msg)
				}
			},
		},
		{
			name: "永久禁言带原因",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"123456789", "违规发言"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkRestrict: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.RestrictCalls) != 1 {
					t.Errorf("expected 1 RestrictChatMember call, got %d", len(api.RestrictCalls))
					return
				}
				// 检查消息是否包含原因
				if len(api.Messages) == 0 {
					t.Error("expected at least one message")
					return
				}
				msg := api.Messages[0]
				if !contains(msg, "违规发言") {
					t.Errorf("expected message to contain reason '违规发言', got: %s", msg)
				}
			},
		},
		{
			name: "回复消息禁言",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{},
				ReplyToMessage: &command.ReplyToMessage{
					MessageID: 100,
					UserID:    123456789,
				},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkRestrict: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.RestrictCalls) != 1 {
					t.Errorf("expected 1 RestrictChatMember call, got %d", len(api.RestrictCalls))
				}
			},
		},
		{
			name: "回复消息禁言 - 带时长",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"2h"},
				ReplyToMessage: &command.ReplyToMessage{
					MessageID: 100,
					UserID:    123456789,
				},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkRestrict: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.RestrictWithDurationCalls) != 1 {
					t.Errorf("expected 1 RestrictChatMemberWithDuration call, got %d", len(api.RestrictWithDurationCalls))
					return
				}
				call := api.RestrictWithDurationCalls[0]
				expectedTime := time.Now().Add(2 * time.Hour)
				diff := call.Until.Sub(expectedTime)
				if diff < -time.Minute || diff > time.Minute {
					t.Errorf("expected until time around %v, got %v", expectedTime, call.Until)
				}
			},
		},
		{
			name: "回复消息禁言 - 带时长和原因",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"1h", "多次警告无效"},
				ReplyToMessage: &command.ReplyToMessage{
					MessageID: 100,
					UserID:    123456789,
				},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkRestrict: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.RestrictWithDurationCalls) != 1 {
					t.Errorf("expected 1 RestrictChatMemberWithDuration call, got %d", len(api.RestrictWithDurationCalls))
					return
				}
				if len(api.Messages) == 0 {
					t.Error("expected at least one message")
					return
				}
				msg := api.Messages[0]
				if !contains(msg, "多次警告无效") {
					t.Errorf("expected message to contain reason, got: %s", msg)
				}
			},
		},
		{
			name: "缺少用户ID - 显示帮助",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{},
			},
			setupMocks:   func(api *MockTelegramAPI, repo *MockUserRepository) {},
			wantErr:      false,
			wantMsgCount: 1,
		},
		{
			name: "无效的用户ID",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"invalid"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {},
			wantErr:    true,
		},
		{
			name: "用户不存在",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"999999999"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				// 不添加任何用户
			},
			wantErr: true,
		},
		{
			name: "Telegram API 失败",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"123456789"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
				api.RestrictChatMemberFunc = func(chatID, userID int64, permissions models.ChatPermissions) error {
					return errors.New("telegram API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := &MockTelegramAPI{
				Messages:                  []string{},
				RestrictCalls:             []RestrictCall{},
				RestrictWithDurationCalls: []RestrictWithDurationCall{},
			}
			mockRepo := &MockUserRepository{
				Users: make(map[int64]*user.User),
			}

			tt.setupMocks(mockAPI, mockRepo)

			h := NewHandler(mockAPI, mockRepo)
			err := h.Handle(tt.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(mockAPI.Messages) != tt.wantMsgCount {
					t.Errorf("expected %d messages, got %d", tt.wantMsgCount, len(mockAPI.Messages))
				}

				if tt.checkRestrict != nil {
					tt.checkRestrict(t, mockAPI)
				}
			}
		})
	}
}

func TestHandler_HandleUnmute(t *testing.T) {
	tests := []struct {
		name          string
		ctx           *command.Context
		setupMocks    func(*MockTelegramAPI, *MockUserRepository)
		wantErr       bool
		wantMsgCount  int
		checkRestrict func(*testing.T, *MockTelegramAPI)
	}{
		{
			name: "解除禁言 - 指定用户ID",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Text:    "/unmute 123456789",
				Args:    []string{"123456789"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkRestrict: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.RestrictCalls) != 1 {
					t.Errorf("expected 1 RestrictChatMember call, got %d", len(api.RestrictCalls))
					return
				}
				call := api.RestrictCalls[0]
				if call.UserID != 123456789 {
					t.Errorf("expected userID 123456789, got %d", call.UserID)
				}
				if !call.Permissions.CanSendMessages {
					t.Error("expected CanSendMessages to be true")
				}
			},
		},
		{
			name: "解除禁言 - 回复消息",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Text:    "/unmute",
				Args:    []string{},
				ReplyToMessage: &command.ReplyToMessage{
					MessageID: 100,
					UserID:    123456789,
				},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkRestrict: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.RestrictCalls) != 1 {
					t.Errorf("expected 1 RestrictChatMember call, got %d", len(api.RestrictCalls))
				}
			},
		},
		{
			name: "缺少用户ID",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Text:    "/unmute",
				Args:    []string{},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {},
			wantErr:    true,
		},
		{
			name: "无效的用户ID",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Text:    "/unmute invalid",
				Args:    []string{"invalid"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository) {},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := &MockTelegramAPI{
				Messages:                  []string{},
				RestrictCalls:             []RestrictCall{},
				RestrictWithDurationCalls: []RestrictWithDurationCall{},
			}
			mockRepo := &MockUserRepository{
				Users: make(map[int64]*user.User),
			}

			tt.setupMocks(mockAPI, mockRepo)

			h := NewHandler(mockAPI, mockRepo)
			err := h.handleUnmute(tt.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleUnmute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(mockAPI.Messages) != tt.wantMsgCount {
					t.Errorf("expected %d messages, got %d", tt.wantMsgCount, len(mockAPI.Messages))
				}

				if tt.checkRestrict != nil {
					tt.checkRestrict(t, mockAPI)
				}
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     time.Duration
		wantErr  bool
	}{
		{"30分钟", "30m", 30 * time.Minute, false},
		{"1小时", "1h", 1 * time.Hour, false},
		{"2小时30分钟", "2h30m", 2*time.Hour + 30*time.Minute, false},
		{"1天", "1d", 24 * time.Hour, false},
		{"7天", "7d", 7 * 24 * time.Hour, false},
		{"无效格式", "invalid", 0, true},
		{"空字符串", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"30秒", 30 * time.Second, "30秒"},
		{"30分钟", 30 * time.Minute, "30分钟"},
		{"1小时", 1 * time.Hour, "1小时"},
		{"2小时30分钟", 2*time.Hour + 30*time.Minute, "2小时30分钟"},
		{"1天", 24 * time.Hour, "1天"},
		{"7天", 7 * 24 * time.Hour, "7天"},
		{"1天2小时", 26 * time.Hour, "1天2小时"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_ShowHelp(t *testing.T) {
	mockAPI := &MockTelegramAPI{
		Messages: []string{},
	}
	mockRepo := &MockUserRepository{
		Users: make(map[int64]*user.User),
	}

	h := NewHandler(mockAPI, mockRepo)

	ctx := &command.Context{
		Ctx:     context.Background(),
		UserID:  1,
		GroupID: -1,
		Args:    []string{},
	}

	err := h.Handle(ctx)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
		return
	}

	if len(mockAPI.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(mockAPI.Messages))
		return
	}

	helpMsg := mockAPI.Messages[0]
	expectedKeywords := []string{"禁言", "unmute", "时长", "示例"}
	for _, keyword := range expectedKeywords {
		if !contains(helpMsg, keyword) {
			t.Errorf("expected help message to contain '%s', got: %s", keyword, helpMsg)
		}
	}
}

func TestHandler_IntegrationMuteAndUnmute(t *testing.T) {
	mockAPI := &MockTelegramAPI{
		Messages:                  []string{},
		RestrictCalls:             []RestrictCall{},
		RestrictWithDurationCalls: []RestrictWithDurationCall{},
	}
	mockRepo := &MockUserRepository{
		Users: map[int64]*user.User{
			123456789: {
				ID:       123456789,
				Username: "testuser",
			},
		},
	}

	h := NewHandler(mockAPI, mockRepo)

	// 先禁言
	muteCtx := &command.Context{
		Ctx:     context.Background(),
		UserID:  1,
		GroupID: -1,
		Args:    []string{"123456789", "1h"},
	}

	err := h.Handle(muteCtx)
	if err != nil {
		t.Errorf("mute failed: %v", err)
		return
	}

	if len(mockAPI.RestrictWithDurationCalls) != 1 {
		t.Errorf("expected 1 mute call, got %d", len(mockAPI.RestrictWithDurationCalls))
	}

	// 再解除禁言
	unmuteCtx := &command.Context{
		Ctx:     context.Background(),
		UserID:  1,
		GroupID: -1,
		Text:    "/unmute 123456789",
		Args:    []string{"123456789"},
	}

	err = h.handleUnmute(unmuteCtx)
	if err != nil {
		t.Errorf("unmute failed: %v", err)
		return
	}

	if len(mockAPI.RestrictCalls) != 1 {
		t.Errorf("expected 1 unmute call, got %d", len(mockAPI.RestrictCalls))
	}

	// 验证最后一次调用恢复了权限
	lastCall := mockAPI.RestrictCalls[0]
	if !lastCall.Permissions.CanSendMessages {
		t.Error("expected CanSendMessages to be true after unmute")
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && fmt.Sprintf("%s", s)[0:len(s)] != "" &&
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
