package warn

import (
	"context"
	"errors"
	"testing"
	"time"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/user"
)

// MockTelegramAPI 模拟 Telegram API
type MockTelegramAPI struct {
	SendMessageFunc          func(chatID int64, text string) error
	SendMessageWithReplyFunc func(chatID int64, text string, replyToMessageID int) error
	BanChatMemberFunc        func(chatID, userID int64) error
	Messages                 []string
	BanCalls                 []BanCall
}

type BanCall struct {
	ChatID int64
	UserID int64
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

func (m *MockTelegramAPI) BanChatMember(chatID, userID int64) error {
	m.BanCalls = append(m.BanCalls, BanCall{
		ChatID: chatID,
		UserID: userID,
	})
	if m.BanChatMemberFunc != nil {
		return m.BanChatMemberFunc(chatID, userID)
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

// MockWarningRepository 模拟警告仓储
type MockWarningRepository struct {
	SaveFunc                func(warning *user.Warning) error
	FindByUserAndGroupFunc  func(userID, groupID int64) ([]*user.Warning, error)
	CountActiveWarningsFunc func(userID, groupID int64) (int, error)
	ClearWarningsFunc       func(userID, groupID int64) error
	Warnings                map[int64][]*user.Warning // userID -> warnings
	SaveCalls               int
	ClearCalls              int
}

func (m *MockWarningRepository) Save(warning *user.Warning) error {
	m.SaveCalls++
	if m.SaveFunc != nil {
		return m.SaveFunc(warning)
	}
	if m.Warnings == nil {
		m.Warnings = make(map[int64][]*user.Warning)
	}
	m.Warnings[warning.UserID] = append(m.Warnings[warning.UserID], warning)
	return nil
}

func (m *MockWarningRepository) FindByUserAndGroup(userID, groupID int64) ([]*user.Warning, error) {
	if m.FindByUserAndGroupFunc != nil {
		return m.FindByUserAndGroupFunc(userID, groupID)
	}
	return m.Warnings[userID], nil
}

func (m *MockWarningRepository) CountActiveWarnings(userID, groupID int64) (int, error) {
	if m.CountActiveWarningsFunc != nil {
		return m.CountActiveWarningsFunc(userID, groupID)
	}
	warnings := m.Warnings[userID]
	count := 0
	for _, w := range warnings {
		if !w.IsCleared {
			count++
		}
	}
	return count, nil
}

func (m *MockWarningRepository) ClearWarnings(userID, groupID int64) error {
	m.ClearCalls++
	if m.ClearWarningsFunc != nil {
		return m.ClearWarningsFunc(userID, groupID)
	}
	warnings := m.Warnings[userID]
	for _, w := range warnings {
		w.Clear()
	}
	return nil
}

func (m *MockWarningRepository) Delete(id int64) error {
	return nil
}

func TestHandler_Name(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	if h.Name() != "warn" {
		t.Errorf("expected name 'warn', got '%s'", h.Name())
	}
}

func TestHandler_Description(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	if h.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestHandler_RequiredPermission(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	if h.RequiredPermission() != user.PermissionAdmin {
		t.Errorf("expected Admin permission, got %v", h.RequiredPermission())
	}
}

func TestHandler_HandleWarn(t *testing.T) {
	tests := []struct {
		name          string
		ctx           *command.Context
		setupMocks    func(*MockTelegramAPI, *MockUserRepository, *MockWarningRepository)
		wantErr       bool
		wantMsgCount  int
		checkWarning  func(*testing.T, *MockWarningRepository)
		checkBan      func(*testing.T, *MockTelegramAPI)
	}{
		{
			name: "警告用户 - 指定用户ID",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"123456789", "发送广告"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkWarning: func(t *testing.T, repo *MockWarningRepository) {
				if repo.SaveCalls != 1 {
					t.Errorf("expected 1 Save call, got %d", repo.SaveCalls)
				}
				warnings := repo.Warnings[123456789]
				if len(warnings) != 1 {
					t.Errorf("expected 1 warning, got %d", len(warnings))
					return
				}
				if warnings[0].Reason != "发送广告" {
					t.Errorf("expected reason '发送广告', got '%s'", warnings[0].Reason)
				}
			},
		},
		{
			name: "警告用户 - 回复消息",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"刷屏"},
				ReplyToMessage: &command.ReplyToMessage{
					MessageID: 100,
					UserID:    123456789,
				},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkWarning: func(t *testing.T, repo *MockWarningRepository) {
				warnings := repo.Warnings[123456789]
				if len(warnings) != 1 {
					t.Errorf("expected 1 warning, got %d", len(warnings))
					return
				}
				if warnings[0].Reason != "刷屏" {
					t.Errorf("expected reason '刷屏', got '%s'", warnings[0].Reason)
				}
			},
		},
		{
			name: "达到警告上限 - 自动踢出",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"123456789", "第三次警告"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
				// 已经有2次警告
				warnRepo.Warnings = map[int64][]*user.Warning{
					123456789: {
						{UserID: 123456789, GroupID: -1, Reason: "第一次", IsCleared: false},
						{UserID: 123456789, GroupID: -1, Reason: "第二次", IsCleared: false},
					},
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkWarning: func(t *testing.T, repo *MockWarningRepository) {
				warnings := repo.Warnings[123456789]
				if len(warnings) != 3 {
					t.Errorf("expected 3 warnings, got %d", len(warnings))
				}
			},
			checkBan: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.BanCalls) != 1 {
					t.Errorf("expected 1 ban call, got %d", len(api.BanCalls))
					return
				}
				if api.BanCalls[0].UserID != 123456789 {
					t.Errorf("expected ban userID 123456789, got %d", api.BanCalls[0].UserID)
				}
				// 检查消息包含"已被踢出"
				if len(api.Messages) == 0 {
					t.Error("expected at least one message")
					return
				}
				msg := api.Messages[0]
				if !contains(msg, "踢出") {
					t.Errorf("expected message to contain '踢出', got: %s", msg)
				}
			},
		},
		{
			name: "缺少原因 - 显示帮助",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"123456789"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
		},
		{
			name: "回复消息缺少原因",
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
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {},
			wantErr:    true,
		},
		{
			name: "无效的用户ID",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"invalid", "原因"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {},
			wantErr:    true,
		},
		{
			name: "用户不存在",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"999999999", "原因"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				// 不添加用户
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := &MockTelegramAPI{
				Messages: []string{},
				BanCalls: []BanCall{},
			}
			mockRepo := &MockUserRepository{
				Users: make(map[int64]*user.User),
			}
			mockWarnRepo := &MockWarningRepository{
				Warnings: make(map[int64][]*user.Warning),
			}

			tt.setupMocks(mockAPI, mockRepo, mockWarnRepo)

			h := NewHandler(mockAPI, mockRepo, mockWarnRepo)
			err := h.Handle(tt.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(mockAPI.Messages) != tt.wantMsgCount {
					t.Errorf("expected %d messages, got %d", tt.wantMsgCount, len(mockAPI.Messages))
				}

				if tt.checkWarning != nil {
					tt.checkWarning(t, mockWarnRepo)
				}

				if tt.checkBan != nil {
					tt.checkBan(t, mockAPI)
				}
			}
		})
	}
}

func TestHandler_HandleWarnings(t *testing.T) {
	tests := []struct {
		name         string
		ctx          *command.Context
		setupMocks   func(*MockTelegramAPI, *MockUserRepository, *MockWarningRepository)
		wantErr      bool
		wantMsgCount int
		checkMessage func(*testing.T, string)
	}{
		{
			name: "查看警告记录 - 有警告",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"warnings", "123456789"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
				warnRepo.Warnings[123456789] = []*user.Warning{
					{UserID: 123456789, GroupID: -1, Reason: "第一次警告", IssuedAt: time.Now(), IsCleared: false},
					{UserID: 123456789, GroupID: -1, Reason: "第二次警告", IssuedAt: time.Now(), IsCleared: false},
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkMessage: func(t *testing.T, msg string) {
				if !contains(msg, "2/3") {
					t.Errorf("expected message to contain '2/3', got: %s", msg)
				}
				if !contains(msg, "第一次警告") {
					t.Errorf("expected message to contain '第一次警告', got: %s", msg)
				}
			},
		},
		{
			name: "查看警告记录 - 无警告",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"list", "123456789"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
				warnRepo.Warnings[123456789] = []*user.Warning{}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkMessage: func(t *testing.T, msg string) {
				if !contains(msg, "没有警告记录") {
					t.Errorf("expected message to contain '没有警告记录', got: %s", msg)
				}
			},
		},
		{
			name: "回复消息查看警告",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"warnings"},
				ReplyToMessage: &command.ReplyToMessage{
					MessageID: 100,
					UserID:    123456789,
				},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
				warnRepo.Warnings[123456789] = []*user.Warning{
					{UserID: 123456789, GroupID: -1, Reason: "警告", IssuedAt: time.Now(), IsCleared: false},
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
		},
		{
			name: "缺少用户ID",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"warnings"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := &MockTelegramAPI{
				Messages: []string{},
			}
			mockRepo := &MockUserRepository{
				Users: make(map[int64]*user.User),
			}
			mockWarnRepo := &MockWarningRepository{
				Warnings: make(map[int64][]*user.Warning),
			}

			tt.setupMocks(mockAPI, mockRepo, mockWarnRepo)

			h := NewHandler(mockAPI, mockRepo, mockWarnRepo)
			err := h.handleWarnings(tt.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleWarnings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(mockAPI.Messages) != tt.wantMsgCount {
					t.Errorf("expected %d messages, got %d", tt.wantMsgCount, len(mockAPI.Messages))
				}

				if tt.checkMessage != nil && len(mockAPI.Messages) > 0 {
					tt.checkMessage(t, mockAPI.Messages[0])
				}
			}
		})
	}
}

func TestHandler_HandleClearWarn(t *testing.T) {
	tests := []struct {
		name         string
		ctx          *command.Context
		setupMocks   func(*MockTelegramAPI, *MockUserRepository, *MockWarningRepository)
		wantErr      bool
		wantMsgCount int
		checkClear   func(*testing.T, *MockWarningRepository)
	}{
		{
			name: "清除警告 - 有警告",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"clear", "123456789"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
				warnRepo.Warnings[123456789] = []*user.Warning{
					{UserID: 123456789, GroupID: -1, Reason: "警告1", IsCleared: false},
					{UserID: 123456789, GroupID: -1, Reason: "警告2", IsCleared: false},
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
			checkClear: func(t *testing.T, repo *MockWarningRepository) {
				if repo.ClearCalls != 1 {
					t.Errorf("expected 1 Clear call, got %d", repo.ClearCalls)
				}
			},
		},
		{
			name: "清除警告 - 无警告",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"clearwarn", "123456789"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
				warnRepo.Warnings[123456789] = []*user.Warning{}
			},
			wantErr:      false,
			wantMsgCount: 1,
		},
		{
			name: "回复消息清除警告",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"clear"},
				ReplyToMessage: &command.ReplyToMessage{
					MessageID: 100,
					UserID:    123456789,
				},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {
				repo.Users[123456789] = &user.User{
					ID:       123456789,
					Username: "testuser",
				}
				warnRepo.Warnings[123456789] = []*user.Warning{
					{UserID: 123456789, GroupID: -1, Reason: "警告", IsCleared: false},
				}
			},
			wantErr:      false,
			wantMsgCount: 1,
		},
		{
			name: "缺少用户ID",
			ctx: &command.Context{
				Ctx:     context.Background(),
				UserID:  1,
				GroupID: -1,
				Args:    []string{"clear"},
			},
			setupMocks: func(api *MockTelegramAPI, repo *MockUserRepository, warnRepo *MockWarningRepository) {},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := &MockTelegramAPI{
				Messages: []string{},
			}
			mockRepo := &MockUserRepository{
				Users: make(map[int64]*user.User),
			}
			mockWarnRepo := &MockWarningRepository{
				Warnings: make(map[int64][]*user.Warning),
			}

			tt.setupMocks(mockAPI, mockRepo, mockWarnRepo)

			h := NewHandler(mockAPI, mockRepo, mockWarnRepo)
			err := h.handleClearWarn(tt.ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleClearWarn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(mockAPI.Messages) != tt.wantMsgCount {
					t.Errorf("expected %d messages, got %d", tt.wantMsgCount, len(mockAPI.Messages))
				}

				if tt.checkClear != nil {
					tt.checkClear(t, mockWarnRepo)
				}
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
	mockWarnRepo := &MockWarningRepository{
		Warnings: make(map[int64][]*user.Warning),
	}

	h := NewHandler(mockAPI, mockRepo, mockWarnRepo)

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
	expectedKeywords := []string{"警告", "warnings", "clear", "示例"}
	for _, keyword := range expectedKeywords {
		if !contains(helpMsg, keyword) {
			t.Errorf("expected help message to contain '%s', got: %s", keyword, helpMsg)
		}
	}
}

func TestWarning_NewWarning(t *testing.T) {
	warning := user.NewWarning(123, -1, "test reason", 456)

	if warning.UserID != 123 {
		t.Errorf("expected UserID 123, got %d", warning.UserID)
	}
	if warning.GroupID != -1 {
		t.Errorf("expected GroupID -1, got %d", warning.GroupID)
	}
	if warning.Reason != "test reason" {
		t.Errorf("expected Reason 'test reason', got '%s'", warning.Reason)
	}
	if warning.IssuedBy != 456 {
		t.Errorf("expected IssuedBy 456, got %d", warning.IssuedBy)
	}
	if warning.IsCleared {
		t.Error("expected IsCleared to be false")
	}
}

func TestWarning_Clear(t *testing.T) {
	warning := user.NewWarning(123, -1, "test", 456)
	warning.Clear()

	if !warning.IsCleared {
		t.Error("expected IsCleared to be true")
	}
}

func TestHandler_Integration(t *testing.T) {
	mockAPI := &MockTelegramAPI{
		Messages: []string{},
		BanCalls: []BanCall{},
	}
	mockRepo := &MockUserRepository{
		Users: map[int64]*user.User{
			123456789: {
				ID:       123456789,
				Username: "testuser",
			},
		},
	}
	mockWarnRepo := &MockWarningRepository{
		Warnings: make(map[int64][]*user.Warning),
	}

	h := NewHandler(mockAPI, mockRepo, mockWarnRepo)

	// 第一次警告
	ctx1 := &command.Context{
		Ctx:     context.Background(),
		UserID:  1,
		GroupID: -1,
		Args:    []string{"123456789", "第一次违规"},
	}
	if err := h.Handle(ctx1); err != nil {
		t.Errorf("first warn failed: %v", err)
	}

	// 第二次警告
	ctx2 := &command.Context{
		Ctx:     context.Background(),
		UserID:  1,
		GroupID: -1,
		Args:    []string{"123456789", "第二次违规"},
	}
	if err := h.Handle(ctx2); err != nil {
		t.Errorf("second warn failed: %v", err)
	}

	// 查看警告
	ctx3 := &command.Context{
		Ctx:     context.Background(),
		UserID:  1,
		GroupID: -1,
		Args:    []string{"warnings", "123456789"},
	}
	if err := h.Handle(ctx3); err != nil {
		t.Errorf("view warnings failed: %v", err)
	}

	// 应该有2次警告
	count, _ := mockWarnRepo.CountActiveWarnings(123456789, -1)
	if count != 2 {
		t.Errorf("expected 2 active warnings, got %d", count)
	}

	// 第三次警告（达到上限）
	ctx4 := &command.Context{
		Ctx:     context.Background(),
		UserID:  1,
		GroupID: -1,
		Args:    []string{"123456789", "第三次违规"},
	}
	if err := h.Handle(ctx4); err != nil {
		t.Errorf("third warn failed: %v", err)
	}

	// 应该被踢出
	if len(mockAPI.BanCalls) != 1 {
		t.Errorf("expected 1 ban call, got %d", len(mockAPI.BanCalls))
	}

	// 清除警告
	ctx5 := &command.Context{
		Ctx:     context.Background(),
		UserID:  1,
		GroupID: -1,
		Args:    []string{"clear", "123456789"},
	}
	if err := h.Handle(ctx5); err != nil {
		t.Errorf("clear warnings failed: %v", err)
	}

	// 警告应该被清除
	count, _ = mockWarnRepo.CountActiveWarnings(123456789, -1)
	if count != 0 {
		t.Errorf("expected 0 active warnings after clear, got %d", count)
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
