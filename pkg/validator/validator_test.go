package validator

import (
	"testing"

	"telegram-bot/pkg/errors"
)

func TestRequired(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		wantErr   bool
	}{
		{"empty string", "", "字段", true},
		{"whitespace only", "   ", "字段", true},
		{"valid value", "test", "字段", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Required(tt.value, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Required() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		min       int
		fieldName string
		wantErr   bool
	}{
		{"too short", "ab", 3, "字段", true},
		{"exact length", "abc", 3, "字段", false},
		{"longer", "abcd", 3, "字段", false},
		{"unicode", "你好", 3, "字段", true},
		{"unicode valid", "你好啊", 3, "字段", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MinLength(tt.value, tt.min, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		max       int
		fieldName string
		wantErr   bool
	}{
		{"within limit", "ab", 3, "字段", false},
		{"exact length", "abc", 3, "字段", false},
		{"too long", "abcd", 3, "字段", true},
		{"unicode", "你好啊呀", 3, "字段", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MaxLength(tt.value, tt.max, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxLength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLengthRange(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		min       int
		max       int
		fieldName string
		wantErr   bool
	}{
		{"too short", "ab", 3, 5, "字段", true},
		{"valid min", "abc", 3, 5, "字段", false},
		{"valid max", "abcde", 3, 5, "字段", false},
		{"too long", "abcdef", 3, 5, "字段", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := LengthRange(tt.value, tt.min, tt.max, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("LengthRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPattern(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		pattern   string
		fieldName string
		wantErr   bool
	}{
		{"valid email", "test@example.com", `^[a-z]+@[a-z]+\.[a-z]+$`, "邮箱", false},
		{"invalid email", "test@", `^[a-z]+@[a-z]+\.[a-z]+$`, "邮箱", true},
		{"valid digits", "12345", `^\d+$`, "数字", false},
		{"invalid digits", "12a45", `^\d+$`, "数字", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Pattern(tt.value, tt.pattern, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Pattern() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserID(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{"valid id", 12345, false},
		{"zero id", 0, true},
		{"negative id", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UserID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGroupID(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{"valid group id", -100123456789, false},
		{"zero id", 0, true},
		{"positive id", 12345, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GroupID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GroupID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid username", "test_user", false},
		{"valid with numbers", "user123", false},
		{"too short", "test", true},
		{"too long", "this_is_a_very_long_username_that_exceeds_limit", true},
		{"invalid chars", "test-user", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Username(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("Username() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommandName(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{"valid command", "/start", false},
		{"valid with underscore", "/my_command", false},
		{"missing slash", "start", true},
		{"only slash", "/", true},
		{"invalid chars", "/test-command", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CommandName(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("CommandName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTextMessage(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		minLen  int
		maxLen  int
		wantErr bool
	}{
		{"valid message", "Hello world", 1, 100, false},
		{"too short", "Hi", 5, 100, true},
		{"too long", "This is a very long message", 1, 10, true},
		{"empty", "", 1, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TextMessage(tt.text, tt.minLen, tt.maxLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("TextMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInSlice(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		slice     []string
		fieldName string
		wantErr   bool
	}{
		{"value exists", "admin", []string{"admin", "user", "guest"}, "角色", false},
		{"value not exists", "superadmin", []string{"admin", "user", "guest"}, "角色", true},
		{"empty slice", "admin", []string{}, "角色", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InSlice(tt.value, tt.slice, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("InSlice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNotInSlice(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		slice     []string
		fieldName string
		wantErr   bool
	}{
		{"value not exists", "superadmin", []string{"admin", "user", "guest"}, "角色", false},
		{"value exists", "admin", []string{"admin", "user", "guest"}, "角色", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NotInSlice(tt.value, tt.slice, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NotInSlice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid with subdomain", "user@mail.example.com", false},
		{"invalid missing @", "testexample.com", true},
		{"invalid missing domain", "test@", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Email(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("Email() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid http", "http://example.com", false},
		{"valid https", "https://example.com", false},
		{"invalid protocol", "ftp://example.com", true},
		{"invalid no protocol", "example.com", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := URL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("URL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain(t *testing.T) {
	t.Run("all valid", func(t *testing.T) {
		chain := NewChain().
			Add(UserID(123)).
			Add(Username("valid_user")).
			Add(CommandName("/start"))

		if !chain.IsValid() {
			t.Error("expected chain to be valid")
		}

		if chain.Error() != nil {
			t.Errorf("expected no error, got %v", chain.Error())
		}
	})

	t.Run("with errors", func(t *testing.T) {
		chain := NewChain().
			Add(UserID(0)).
			Add(Username("ab")).
			Add(CommandName("start"))

		if chain.IsValid() {
			t.Error("expected chain to be invalid")
		}

		if chain.Error() == nil {
			t.Error("expected error")
		}

		result := chain.Result()
		if len(result.AllErrors()) != 3 {
			t.Errorf("expected 3 errors, got %d", len(result.AllErrors()))
		}
	})

	t.Run("mixed valid and invalid", func(t *testing.T) {
		chain := NewChain().
			Add(UserID(123)).
			Add(Username("ab")).
			Add(CommandName("/start"))

		if chain.IsValid() {
			t.Error("expected chain to be invalid")
		}

		result := chain.Result()
		if len(result.AllErrors()) != 1 {
			t.Errorf("expected 1 error, got %d", len(result.AllErrors()))
		}
	})
}

func TestResult(t *testing.T) {
	t.Run("add multiple errors", func(t *testing.T) {
		result := &Result{Valid: true, Errors: []error{}}
		result.AddError(errors.Validation("ERR1", "error 1"))
		result.AddError(errors.Validation("ERR2", "error 2"))

		if result.Valid {
			t.Error("expected result to be invalid")
		}

		if len(result.AllErrors()) != 2 {
			t.Errorf("expected 2 errors, got %d", len(result.AllErrors()))
		}

		if result.Error() == nil {
			t.Error("expected first error to be returned")
		}
	})

	t.Run("no errors", func(t *testing.T) {
		result := &Result{Valid: true, Errors: []error{}}

		if !result.Valid {
			t.Error("expected result to be valid")
		}

		if result.Error() != nil {
			t.Error("expected no error")
		}
	})
}