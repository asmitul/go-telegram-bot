package ban

import (
	"testing"
	"time"

	"telegram-bot/internal/domain/command"
	"telegram-bot/internal/domain/group"
	"telegram-bot/internal/domain/user"
)

// MockGroupRepository Group Repository mock
type MockGroupRepository struct {
	groups map[int64]*group.Group
}

func NewMockGroupRepository() *MockGroupRepository {
	return &MockGroupRepository{
		groups: make(map[int64]*group.Group),
	}
}

func (m *MockGroupRepository) FindByID(id int64) (*group.Group, error) {
	g, ok := m.groups[id]
	if !ok {
		return nil, group.ErrGroupNotFound
	}
	return g, nil
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
	groups := make([]*group.Group, 0, len(m.groups))
	for _, g := range m.groups {
		groups = append(groups, g)
	}
	return groups, nil
}

// MockUserRepository User Repository mock
type MockUserRepository struct {
	users map[int64]*user.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[int64]*user.User),
	}
}

func (m *MockUserRepository) FindByID(id int64) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, user.ErrUserNotFound
	}
	return u, nil
}

func (m *MockUserRepository) FindByUsername(username string) (*user.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, user.ErrUserNotFound
}

func (m *MockUserRepository) Save(u *user.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *MockUserRepository) Update(u *user.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *MockUserRepository) Delete(id int64) error {
	delete(m.users, id)
	return nil
}

func (m *MockUserRepository) FindAdminsByGroup(groupID int64) ([]*user.User, error) {
	admins := make([]*user.User, 0)
	for _, u := range m.users {
		if u.GetPermission(groupID) >= user.PermissionAdmin {
			admins = append(admins, u)
		}
	}
	return admins, nil
}

// MockTelegramAPI Telegram API mock
type MockTelegramAPI struct {
	BanCalls              []BanCall
	BanWithDurationCalls  []BanWithDurationCall
	SendMessageCalls      []SendMessageCall
	ShouldFailBan         bool
	ShouldFailSendMessage bool
}

type BanCall struct {
	ChatID int64
	UserID int64
}

type BanWithDurationCall struct {
	ChatID int64
	UserID int64
	Until  time.Time
}

type SendMessageCall struct {
	ChatID int64
	Text   string
}

func NewMockTelegramAPI() *MockTelegramAPI {
	return &MockTelegramAPI{
		BanCalls:             make([]BanCall, 0),
		BanWithDurationCalls: make([]BanWithDurationCall, 0),
		SendMessageCalls:     make([]SendMessageCall, 0),
	}
}

func (m *MockTelegramAPI) BanChatMember(chatID, userID int64) error {
	if m.ShouldFailBan {
		return ErrCannotBanAdmin
	}
	m.BanCalls = append(m.BanCalls, BanCall{ChatID: chatID, UserID: userID})
	return nil
}

func (m *MockTelegramAPI) BanChatMemberWithDuration(chatID, userID int64, until time.Time) error {
	if m.ShouldFailBan {
		return ErrCannotBanAdmin
	}
	m.BanWithDurationCalls = append(m.BanWithDurationCalls, BanWithDurationCall{
		ChatID: chatID,
		UserID: userID,
		Until:  until,
	})
	return nil
}

func (m *MockTelegramAPI) SendMessage(chatID int64, text string) error {
	if m.ShouldFailSendMessage {
		return ErrInvalidArguments
	}
	m.SendMessageCalls = append(m.SendMessageCalls, SendMessageCall{ChatID: chatID, Text: text})
	return nil
}

// Test cases

func TestHandler_Name(t *testing.T) {
	handler := &Handler{}
	if got := handler.Name(); got != "ban" {
		t.Errorf("Name() = %v, want %v", got, "ban")
	}
}

func TestHandler_Description(t *testing.T) {
	handler := &Handler{}
	if got := handler.Description(); got == "" {
		t.Error("Description() should not be empty")
	}
}

func TestHandler_RequiredPermission(t *testing.T) {
	handler := &Handler{}
	if got := handler.RequiredPermission(); got != user.PermissionAdmin {
		t.Errorf("RequiredPermission() = %v, want %v", got, user.PermissionAdmin)
	}
}

func TestHandler_IsEnabled(t *testing.T) {
	groupRepo := NewMockGroupRepository()
	userRepo := NewMockUserRepository()
	api := NewMockTelegramAPI()
	handler := NewHandler(groupRepo, userRepo, api)

	tests := []struct {
		name    string
		setup   func()
		groupID int64
		want    bool
	}{
		{
			name: "default enabled",
			setup: func() {
				// no groups added
			},
			groupID: -1,
			want:    true,
		},
		{
			name: "enabled in group",
			setup: func() {
				g := group.NewGroup(-1, "Test Group", "supergroup")
				g.EnableCommand("ban", 1)
				groupRepo.Save(g)
			},
			groupID: -1,
			want:    true,
		},
		{
			name: "disabled in group",
			setup: func() {
				g := group.NewGroup(-2, "Test Group 2", "supergroup")
				g.DisableCommand("ban", 1)
				groupRepo.Save(g)
			},
			groupID: -2,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			if got := handler.IsEnabled(tt.groupID); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_Handle(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*MockGroupRepository, *MockUserRepository, *MockTelegramAPI)
		ctx       *command.Context
		wantErr   bool
		checkAPI  func(*testing.T, *MockTelegramAPI)
	}{
		{
			name: "ban user by ID - permanent",
			setup: func(gr *MockGroupRepository, ur *MockUserRepository, api *MockTelegramAPI) {
				u := user.NewUser(2, "target", "Target", "User")
				u.SetPermission(-1, user.PermissionUser)
				ur.Save(u)
			},
			ctx: &command.Context{
				GroupID: -1,
				UserID:  1,
				Args:    []string{"2"},
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.BanCalls) != 1 {
					t.Errorf("Expected 1 ban call, got %d", len(api.BanCalls))
				}
				if len(api.BanCalls) > 0 && api.BanCalls[0].UserID != 2 {
					t.Errorf("Expected to ban user 2, got %d", api.BanCalls[0].UserID)
				}
			},
		},
		{
			name: "ban user by ID - with duration",
			setup: func(gr *MockGroupRepository, ur *MockUserRepository, api *MockTelegramAPI) {
				u := user.NewUser(2, "target", "Target", "User")
				u.SetPermission(-1, user.PermissionUser)
				ur.Save(u)
			},
			ctx: &command.Context{
				GroupID: -1,
				UserID:  1,
				Args:    []string{"2", "1h"},
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.BanWithDurationCalls) != 1 {
					t.Errorf("Expected 1 ban with duration call, got %d", len(api.BanWithDurationCalls))
				}
			},
		},
		{
			name: "ban user by ID - with reason",
			setup: func(gr *MockGroupRepository, ur *MockUserRepository, api *MockTelegramAPI) {
				u := user.NewUser(2, "target", "Target", "User")
				u.SetPermission(-1, user.PermissionUser)
				ur.Save(u)
			},
			ctx: &command.Context{
				GroupID: -1,
				UserID:  1,
				Args:    []string{"2", "spam"},
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.BanCalls) != 1 {
					t.Errorf("Expected 1 ban call, got %d", len(api.BanCalls))
				}
			},
		},
		{
			name: "ban user by ID - with duration and reason",
			setup: func(gr *MockGroupRepository, ur *MockUserRepository, api *MockTelegramAPI) {
				u := user.NewUser(2, "target", "Target", "User")
				u.SetPermission(-1, user.PermissionUser)
				ur.Save(u)
			},
			ctx: &command.Context{
				GroupID: -1,
				UserID:  1,
				Args:    []string{"2", "1h", "spam"},
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.BanWithDurationCalls) != 1 {
					t.Errorf("Expected 1 ban with duration call, got %d", len(api.BanWithDurationCalls))
				}
			},
		},
		{
			name: "ban user via reply - permanent",
			setup: func(gr *MockGroupRepository, ur *MockUserRepository, api *MockTelegramAPI) {
				u := user.NewUser(2, "target", "Target", "User")
				u.SetPermission(-1, user.PermissionUser)
				ur.Save(u)
			},
			ctx: &command.Context{
				GroupID: -1,
				UserID:  1,
				Args:    []string{},
				ReplyToMessage: &command.ReplyToMessage{
					UserID: 2,
				},
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.BanCalls) != 1 {
					t.Errorf("Expected 1 ban call, got %d", len(api.BanCalls))
				}
			},
		},
		{
			name: "ban user via reply - with duration",
			setup: func(gr *MockGroupRepository, ur *MockUserRepository, api *MockTelegramAPI) {
				u := user.NewUser(2, "target", "Target", "User")
				u.SetPermission(-1, user.PermissionUser)
				ur.Save(u)
			},
			ctx: &command.Context{
				GroupID: -1,
				UserID:  1,
				Args:    []string{"30m"},
				ReplyToMessage: &command.ReplyToMessage{
					UserID: 2,
				},
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.BanWithDurationCalls) != 1 {
					t.Errorf("Expected 1 ban with duration call, got %d", len(api.BanWithDurationCalls))
				}
			},
		},
		{
			name: "ban user via reply - with reason",
			setup: func(gr *MockGroupRepository, ur *MockUserRepository, api *MockTelegramAPI) {
				u := user.NewUser(2, "target", "Target", "User")
				u.SetPermission(-1, user.PermissionUser)
				ur.Save(u)
			},
			ctx: &command.Context{
				GroupID: -1,
				UserID:  1,
				Args:    []string{"spamming", "ads"},
				ReplyToMessage: &command.ReplyToMessage{
					UserID: 2,
				},
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.BanCalls) != 1 {
					t.Errorf("Expected 1 ban call, got %d", len(api.BanCalls))
				}
			},
		},
		{
			name: "cannot ban admin",
			setup: func(gr *MockGroupRepository, ur *MockUserRepository, api *MockTelegramAPI) {
				u := user.NewUser(2, "admin", "Admin", "User")
				u.SetPermission(-1, user.PermissionAdmin)
				ur.Save(u)
			},
			ctx: &command.Context{
				GroupID: -1,
				UserID:  1,
				Args:    []string{"2"},
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.BanCalls) != 0 {
					t.Error("Should not call ban for admin")
				}
			},
		},
		{
			name: "no arguments and no reply",
			setup: func(gr *MockGroupRepository, ur *MockUserRepository, api *MockTelegramAPI) {
			},
			ctx: &command.Context{
				GroupID: -1,
				UserID:  1,
				Args:    []string{},
			},
			wantErr: false,
			checkAPI: func(t *testing.T, api *MockTelegramAPI) {
				if len(api.BanCalls) != 0 {
					t.Error("Should not call ban with invalid arguments")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupRepo := NewMockGroupRepository()
			userRepo := NewMockUserRepository()
			api := NewMockTelegramAPI()
			handler := NewHandler(groupRepo, userRepo, api)

			tt.setup(groupRepo, userRepo, api)

			err := handler.Handle(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.checkAPI != nil {
				tt.checkAPI(t, api)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "30 minutes",
			input:   "30m",
			want:    30 * time.Minute,
			wantErr: false,
		},
		{
			name:    "1 hour",
			input:   "1h",
			want:    1 * time.Hour,
			wantErr: false,
		},
		{
			name:    "2 hours 30 minutes",
			input:   "2h30m",
			want:    2*time.Hour + 30*time.Minute,
			wantErr: false,
		},
		{
			name:    "1 day",
			input:   "1d",
			want:    24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "7 days",
			input:   "7d",
			want:    7 * 24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "uppercase D",
			input:   "1D",
			want:    24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "invalid",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
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
		{
			name:     "30 minutes",
			duration: 30 * time.Minute,
			want:     "30 分钟",
		},
		{
			name:     "1 hour",
			duration: 1 * time.Hour,
			want:     "1 小时",
		},
		{
			name:     "2 hours 30 minutes",
			duration: 2*time.Hour + 30*time.Minute,
			want:     "2 小时 30 分钟",
		},
		{
			name:     "1 day",
			duration: 24 * time.Hour,
			want:     "1 天",
		},
		{
			name:     "2 days 3 hours",
			duration: 2*24*time.Hour + 3*time.Hour,
			want:     "2 天 3 小时",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDuration(tt.duration); got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSuccessMessage(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name     string
		userID   int64
		duration time.Duration
		reason   string
		contains []string
	}{
		{
			name:     "permanent ban",
			userID:   123,
			duration: 0,
			reason:   "",
			contains: []string{"123", "永久封禁"},
		},
		{
			name:     "permanent ban with reason",
			userID:   123,
			duration: 0,
			reason:   "spam",
			contains: []string{"123", "永久封禁", "spam"},
		},
		{
			name:     "temporary ban",
			userID:   123,
			duration: 1 * time.Hour,
			reason:   "",
			contains: []string{"123", "临时封禁", "1 小时"},
		},
		{
			name:     "temporary ban with reason",
			userID:   123,
			duration: 1 * time.Hour,
			reason:   "spam",
			contains: []string{"123", "临时封禁", "1 小时", "spam"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.buildSuccessMessage(tt.userID, tt.duration, tt.reason)
			for _, substr := range tt.contains {
				if !contains(got, substr) {
					t.Errorf("buildSuccessMessage() should contain %q, got %q", substr, got)
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Verify Handler implements command.Handler interface
var _ command.Handler = (*Handler)(nil)
