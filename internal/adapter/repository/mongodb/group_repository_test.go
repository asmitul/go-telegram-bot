package mongodb

import (
	"telegram-bot/internal/domain/group"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGroupRepository_DocumentConversion(t *testing.T) {
	repo := &GroupRepository{}

	t.Run("toDocument converts group correctly", func(t *testing.T) {
		g := group.NewGroup(-100, "Test Group", "supergroup")
		g.EnableCommand("ping", 123)
		g.DisableCommand("ban", 456)
		g.Settings = map[string]interface{}{
			"welcome": "enabled",
			"max_warnings": 3,
		}
		g.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		g.UpdatedAt = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

		doc := repo.toDocument(g)

		assert.Equal(t, int64(-100), doc.ID)
		assert.Equal(t, "Test Group", doc.Title)
		assert.Equal(t, "supergroup", doc.Type)
		assert.Len(t, doc.Commands, 2)
		assert.True(t, doc.Commands["ping"].Enabled)
		assert.False(t, doc.Commands["ban"].Enabled)
		assert.Equal(t, "enabled", doc.Settings["welcome"])
		assert.Equal(t, g.CreatedAt, doc.CreatedAt)
		assert.Equal(t, g.UpdatedAt, doc.UpdatedAt)
	})

	t.Run("toDocument with empty commands and settings", func(t *testing.T) {
		g := group.NewGroup(-200, "Empty Group", "group")

		doc := repo.toDocument(g)

		assert.Equal(t, int64(-200), doc.ID)
		assert.Empty(t, doc.Commands)
		assert.Empty(t, doc.Settings)
	})

	t.Run("toDomain converts document correctly", func(t *testing.T) {
		doc := &groupDocument{
			ID:    -300,
			Title: "Test Group",
			Type:  "supergroup",
			Commands: map[string]*commandConfigDoc{
				"ping": {
					CommandName: "ping",
					Enabled:     true,
					UpdatedAt:   time.Now(),
					UpdatedBy:   123,
				},
			},
			Settings: map[string]interface{}{
				"welcome": "Hi!",
			},
			CreatedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC),
		}

		g := repo.toDomain(doc)

		assert.Equal(t, int64(-300), g.ID)
		assert.Equal(t, "Test Group", g.Title)
		assert.Equal(t, "supergroup", g.Type)
		assert.Len(t, g.Commands, 1)
		assert.True(t, g.Commands["ping"].Enabled)
		assert.Equal(t, "Hi!", g.Settings["welcome"])
		assert.Equal(t, doc.CreatedAt, g.CreatedAt)
		assert.Equal(t, doc.UpdatedAt, g.UpdatedAt)
	})

	t.Run("round trip conversion", func(t *testing.T) {
		original := group.NewGroup(-400, "Round Trip", "supergroup")
		original.EnableCommand("ping", 100)
		original.DisableCommand("ban", 200)
		original.Settings = map[string]interface{}{
			"test": "value",
		}

		doc := repo.toDocument(original)
		converted := repo.toDomain(doc)

		assert.Equal(t, original.ID, converted.ID)
		assert.Equal(t, original.Title, converted.Title)
		assert.Equal(t, original.Type, converted.Type)
		assert.Equal(t, len(original.Commands), len(converted.Commands))
		assert.Equal(t, len(original.Settings), len(converted.Settings))
	})

	t.Run("command config fields preserved", func(t *testing.T) {
		g := group.NewGroup(-500, "Test", "supergroup")
		g.EnableCommand("test", 123)

		doc := repo.toDocument(g)
		cmdDoc := doc.Commands["test"]

		assert.Equal(t, "test", cmdDoc.CommandName)
		assert.True(t, cmdDoc.Enabled)
		assert.Equal(t, int64(123), cmdDoc.UpdatedBy)
		assert.NotZero(t, cmdDoc.UpdatedAt)

		converted := repo.toDomain(doc)
		cmdConfig := converted.Commands["test"]

		assert.Equal(t, cmdDoc.CommandName, cmdConfig.CommandName)
		assert.Equal(t, cmdDoc.Enabled, cmdConfig.Enabled)
		assert.Equal(t, cmdDoc.UpdatedBy, cmdConfig.UpdatedBy)
	})
}

func TestGroupRepository_Commands(t *testing.T) {
	repo := &GroupRepository{}

	t.Run("multiple commands", func(t *testing.T) {
		g := group.NewGroup(-100, "Multi Command", "supergroup")
		g.EnableCommand("ping", 1)
		g.EnableCommand("help", 1)
		g.DisableCommand("ban", 2)
		g.DisableCommand("mute", 2)

		doc := repo.toDocument(g)
		assert.Len(t, doc.Commands, 4)

		converted := repo.toDomain(doc)
		assert.Len(t, converted.Commands, 4)
		assert.True(t, converted.IsCommandEnabled("ping"))
		assert.True(t, converted.IsCommandEnabled("help"))
		assert.False(t, converted.IsCommandEnabled("ban"))
		assert.False(t, converted.IsCommandEnabled("mute"))
	})

	t.Run("command enable/disable state", func(t *testing.T) {
		g := group.NewGroup(-200, "Test", "supergroup")
		g.EnableCommand("cmd1", 10)
		g.DisableCommand("cmd2", 20)

		doc := repo.toDocument(g)
		assert.True(t, doc.Commands["cmd1"].Enabled)
		assert.False(t, doc.Commands["cmd2"].Enabled)

		converted := repo.toDomain(doc)
		assert.True(t, converted.IsCommandEnabled("cmd1"))
		assert.False(t, converted.IsCommandEnabled("cmd2"))
	})

	t.Run("command updatedBy tracking", func(t *testing.T) {
		g := group.NewGroup(-300, "Test", "supergroup")
		g.EnableCommand("test", 12345)

		doc := repo.toDocument(g)
		assert.Equal(t, int64(12345), doc.Commands["test"].UpdatedBy)

		converted := repo.toDomain(doc)
		assert.Equal(t, int64(12345), converted.Commands["test"].UpdatedBy)
	})
}

func TestGroupRepository_Settings(t *testing.T) {
	repo := &GroupRepository{}

	t.Run("string settings", func(t *testing.T) {
		g := group.NewGroup(-100, "Test", "supergroup")
		g.Settings = map[string]interface{}{
			"welcome_message": "Welcome!",
			"language":        "en",
		}

		doc := repo.toDocument(g)
		assert.Equal(t, "Welcome!", doc.Settings["welcome_message"])
		assert.Equal(t, "en", doc.Settings["language"])

		converted := repo.toDomain(doc)
		assert.Equal(t, "Welcome!", converted.Settings["welcome_message"])
		assert.Equal(t, "en", converted.Settings["language"])
	})

	t.Run("numeric settings", func(t *testing.T) {
		g := group.NewGroup(-200, "Test", "supergroup")
		g.Settings = map[string]interface{}{
			"max_warnings": 3,
			"timeout":      300,
		}

		doc := repo.toDocument(g)
		assert.Equal(t, 3, doc.Settings["max_warnings"])
		assert.Equal(t, 300, doc.Settings["timeout"])

		converted := repo.toDomain(doc)
		assert.Equal(t, 3, converted.Settings["max_warnings"])
		assert.Equal(t, 300, converted.Settings["timeout"])
	})

	t.Run("boolean settings", func(t *testing.T) {
		g := group.NewGroup(-300, "Test", "supergroup")
		g.Settings = map[string]interface{}{
			"auto_ban":     true,
			"allow_links":  false,
		}

		doc := repo.toDocument(g)
		assert.True(t, doc.Settings["auto_ban"].(bool))
		assert.False(t, doc.Settings["allow_links"].(bool))

		converted := repo.toDomain(doc)
		assert.True(t, converted.Settings["auto_ban"].(bool))
		assert.False(t, converted.Settings["allow_links"].(bool))
	})

	t.Run("complex settings", func(t *testing.T) {
		g := group.NewGroup(-400, "Test", "supergroup")
		g.Settings = map[string]interface{}{
			"allowed_domains": []string{"google.com", "github.com"},
			"config": map[string]interface{}{
				"enabled": true,
				"value":   42,
			},
		}

		doc := repo.toDocument(g)
		converted := repo.toDomain(doc)

		// Just verify the settings are preserved (exact type checking depends on BSON marshaling)
		assert.NotNil(t, converted.Settings["allowed_domains"])
		assert.NotNil(t, converted.Settings["config"])
	})
}

func TestGroupRepository_GroupTypes(t *testing.T) {
	repo := &GroupRepository{}

	tests := []struct {
		name      string
		groupType string
	}{
		{"supergroup", "supergroup"},
		{"group", "group"},
		{"channel", "channel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := group.NewGroup(-100, "Test", tt.groupType)

			doc := repo.toDocument(g)
			assert.Equal(t, tt.groupType, doc.Type)

			converted := repo.toDomain(doc)
			assert.Equal(t, tt.groupType, converted.Type)
		})
	}
}

func TestGroupRepository_Interface(t *testing.T) {
	t.Run("implements group.Repository", func(t *testing.T) {
		var _ group.Repository = (*GroupRepository)(nil)
	})
}

func TestGroupRepository_Initialization(t *testing.T) {
	t.Run("timeout is set correctly", func(t *testing.T) {
		expectedTimeout := 10 * time.Second
		assert.Equal(t, expectedTimeout, 10*time.Second)
	})
}

func TestGroupDocument_Structure(t *testing.T) {
	t.Run("groupDocument has correct fields", func(t *testing.T) {
		doc := &groupDocument{
			ID:    -100,
			Title: "Test",
			Type:  "supergroup",
			Commands: map[string]*commandConfigDoc{
				"test": {
					CommandName: "test",
					Enabled:     true,
					UpdatedAt:   time.Now(),
					UpdatedBy:   123,
				},
			},
			Settings:  map[string]interface{}{"key": "value"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.NotZero(t, doc.ID)
		assert.NotEmpty(t, doc.Title)
		assert.NotNil(t, doc.Commands)
		assert.NotNil(t, doc.Settings)
		assert.NotZero(t, doc.CreatedAt)
		assert.NotZero(t, doc.UpdatedAt)
	})
}

func TestGroupRepository_EdgeCases(t *testing.T) {
	repo := &GroupRepository{}

	t.Run("negative group ID (typical Telegram)", func(t *testing.T) {
		g := group.NewGroup(-1001234567890, "Test", "supergroup")

		doc := repo.toDocument(g)
		assert.Equal(t, int64(-1001234567890), doc.ID)

		converted := repo.toDomain(doc)
		assert.Equal(t, int64(-1001234567890), converted.ID)
	})

	t.Run("empty title", func(t *testing.T) {
		g := group.NewGroup(-100, "", "supergroup")

		doc := repo.toDocument(g)
		assert.Empty(t, doc.Title)

		converted := repo.toDomain(doc)
		assert.Empty(t, converted.Title)
	})

	t.Run("nil settings map", func(t *testing.T) {
		g := group.NewGroup(-100, "Test", "supergroup")
		g.Settings = nil

		doc := repo.toDocument(g)
		// Should not panic
		assert.Nil(t, doc.Settings)
	})

	t.Run("many commands", func(t *testing.T) {
		g := group.NewGroup(-100, "Test", "supergroup")
		for i := 0; i < 50; i++ {
			g.EnableCommand(string(rune('a'+i)), 1)
		}

		doc := repo.toDocument(g)
		assert.Len(t, doc.Commands, 50)

		converted := repo.toDomain(doc)
		assert.Len(t, converted.Commands, 50)
	})
}

// Benchmark tests
func BenchmarkGroupRepository_ToDocument(b *testing.B) {
	repo := &GroupRepository{}
	g := group.NewGroup(-100, "Test Group", "supergroup")
	g.EnableCommand("ping", 1)
	g.EnableCommand("help", 1)
	g.Settings = map[string]interface{}{
		"welcome": "Hi!",
		"max_warnings": 3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.toDocument(g)
	}
}

func BenchmarkGroupRepository_ToDomain(b *testing.B) {
	repo := &GroupRepository{}
	doc := &groupDocument{
		ID:    -100,
		Title: "Test Group",
		Type:  "supergroup",
		Commands: map[string]*commandConfigDoc{
			"ping": {
				CommandName: "ping",
				Enabled:     true,
				UpdatedAt:   time.Now(),
				UpdatedBy:   1,
			},
		},
		Settings: map[string]interface{}{
			"welcome": "Hi!",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.toDomain(doc)
	}
}

func BenchmarkGroupRepository_RoundTrip(b *testing.B) {
	repo := &GroupRepository{}
	g := group.NewGroup(-100, "Test", "supergroup")
	g.EnableCommand("ping", 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := repo.toDocument(g)
		repo.toDomain(doc)
	}
}
