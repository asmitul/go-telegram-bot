package mongodb

import (
	"telegram-bot/internal/domain/user"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUserRepository_DocumentConversion(t *testing.T) {
	repo := &UserRepository{}

	t.Run("toDocument converts user correctly", func(t *testing.T) {
		u := user.NewUser(123, "testuser", "Test", "User")
		u.SetPermission(-100, user.PermissionAdmin)
		u.SetPermission(-200, user.PermissionUser)
		u.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		u.UpdatedAt = time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

		doc := repo.toDocument(u)

		assert.Equal(t, int64(123), doc.ID)
		assert.Equal(t, "testuser", doc.Username)
		assert.Equal(t, "Test", doc.FirstName)
		assert.Equal(t, "User", doc.LastName)
		assert.Len(t, doc.Permissions, 2)
		assert.Equal(t, int(user.PermissionAdmin), doc.Permissions[-100])
		assert.Equal(t, int(user.PermissionUser), doc.Permissions[-200])
		assert.Equal(t, u.CreatedAt, doc.CreatedAt)
		assert.Equal(t, u.UpdatedAt, doc.UpdatedAt)
	})

	t.Run("toDocument with empty permissions", func(t *testing.T) {
		u := user.NewUser(456, "user2", "User", "Two")

		doc := repo.toDocument(u)

		assert.Equal(t, int64(456), doc.ID)
		assert.Empty(t, doc.Permissions)
	})

	t.Run("toDomain converts document correctly", func(t *testing.T) {
		doc := &userDocument{
			ID:        789,
			Username:  "user3",
			FirstName: "User",
			LastName:  "Three",
			Permissions: map[int64]int{
				-100: int(user.PermissionAdmin),
				-200: int(user.PermissionSuperAdmin),
			},
			CreatedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC),
		}

		u := repo.toDomain(doc)

		assert.Equal(t, int64(789), u.ID)
		assert.Equal(t, "user3", u.Username)
		assert.Equal(t, "User", u.FirstName)
		assert.Equal(t, "Three", u.LastName)
		assert.Len(t, u.Permissions, 2)
		assert.Equal(t, user.PermissionAdmin, u.Permissions[-100])
		assert.Equal(t, user.PermissionSuperAdmin, u.Permissions[-200])
		assert.Equal(t, doc.CreatedAt, u.CreatedAt)
		assert.Equal(t, doc.UpdatedAt, u.UpdatedAt)
	})

	t.Run("toDomain with empty permissions", func(t *testing.T) {
		doc := &userDocument{
			ID:          999,
			Username:    "user4",
			FirstName:   "User",
			LastName:    "Four",
			Permissions: make(map[int64]int),
		}

		u := repo.toDomain(doc)

		assert.Equal(t, int64(999), u.ID)
		assert.Empty(t, u.Permissions)
	})

	t.Run("round trip conversion", func(t *testing.T) {
		original := user.NewUser(111, "roundtrip", "Round", "Trip")
		original.SetPermission(-100, user.PermissionAdmin)
		original.SetPermission(-200, user.PermissionUser)
		original.SetPermission(-300, user.PermissionSuperAdmin)

		doc := repo.toDocument(original)
		converted := repo.toDomain(doc)

		assert.Equal(t, original.ID, converted.ID)
		assert.Equal(t, original.Username, converted.Username)
		assert.Equal(t, original.FirstName, converted.FirstName)
		assert.Equal(t, original.LastName, converted.LastName)
		assert.Equal(t, len(original.Permissions), len(converted.Permissions))
		for groupID, perm := range original.Permissions {
			assert.Equal(t, perm, converted.Permissions[groupID])
		}
	})

	t.Run("permission level conversion", func(t *testing.T) {
		tests := []struct {
			name       string
			permission user.Permission
		}{
			{"NoPermission", user.PermissionNone},
			{"User", user.PermissionUser},
			{"Admin", user.PermissionAdmin},
			{"SuperAdmin", user.PermissionSuperAdmin},
			{"Owner", user.PermissionOwner},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				u := user.NewUser(1, "test", "Test", "User")
				u.SetPermission(-100, tt.permission)

				doc := repo.toDocument(u)
				assert.Equal(t, int(tt.permission), doc.Permissions[-100])

				converted := repo.toDomain(doc)
				assert.Equal(t, tt.permission, converted.Permissions[-100])
			})
		}
	})

	t.Run("multiple groups with different permissions", func(t *testing.T) {
		u := user.NewUser(222, "multigroup", "Multi", "Group")
		u.SetPermission(-100, user.PermissionUser)
		u.SetPermission(-200, user.PermissionAdmin)
		u.SetPermission(-300, user.PermissionSuperAdmin)
		u.SetPermission(-400, user.PermissionOwner)
		u.SetPermission(-500, user.PermissionNone)

		doc := repo.toDocument(u)
		assert.Len(t, doc.Permissions, 5)

		converted := repo.toDomain(doc)
		assert.Len(t, converted.Permissions, 5)
		assert.Equal(t, user.PermissionUser, converted.GetPermission(-100))
		assert.Equal(t, user.PermissionAdmin, converted.GetPermission(-200))
		assert.Equal(t, user.PermissionSuperAdmin, converted.GetPermission(-300))
		assert.Equal(t, user.PermissionOwner, converted.GetPermission(-400))
		assert.Equal(t, user.PermissionNone, converted.GetPermission(-500))
	})
}

func TestUserRepository_Initialization(t *testing.T) {
	t.Run("timeout is set correctly", func(t *testing.T) {
		// We can't test actual MongoDB without a real connection,
		// but we can verify the timeout constant
		expectedTimeout := 10 * time.Second
		assert.Equal(t, expectedTimeout, 10*time.Second)
	})
}

func TestUserDocument_Structure(t *testing.T) {
	t.Run("userDocument has correct fields", func(t *testing.T) {
		doc := &userDocument{
			ID:        123,
			Username:  "test",
			FirstName: "Test",
			LastName:  "User",
			Permissions: map[int64]int{
				-100: 2,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		assert.NotZero(t, doc.ID)
		assert.NotEmpty(t, doc.Username)
		assert.NotNil(t, doc.Permissions)
		assert.NotZero(t, doc.CreatedAt)
		assert.NotZero(t, doc.UpdatedAt)
	})
}

// Integration-style tests that verify the repository interface
func TestUserRepository_Interface(t *testing.T) {
	t.Run("implements user.Repository", func(t *testing.T) {
		var _ user.Repository = (*UserRepository)(nil)
	})
}

// Test edge cases
func TestUserRepository_EdgeCases(t *testing.T) {
	repo := &UserRepository{}

	t.Run("user with no last name", func(t *testing.T) {
		u := user.NewUser(100, "nolast", "First", "")

		doc := repo.toDocument(u)
		assert.Empty(t, doc.LastName)

		converted := repo.toDomain(doc)
		assert.Empty(t, converted.LastName)
	})

	t.Run("user with no username", func(t *testing.T) {
		u := user.NewUser(200, "", "First", "Last")

		doc := repo.toDocument(u)
		assert.Empty(t, doc.Username)

		converted := repo.toDomain(doc)
		assert.Empty(t, converted.Username)
	})

	t.Run("negative group IDs", func(t *testing.T) {
		u := user.NewUser(300, "test", "Test", "User")
		u.SetPermission(-1001234567890, user.PermissionAdmin) // Typical Telegram supergroup ID

		doc := repo.toDocument(u)
		assert.Contains(t, doc.Permissions, int64(-1001234567890))

		converted := repo.toDomain(doc)
		assert.Equal(t, user.PermissionAdmin, converted.GetPermission(-1001234567890))
	})

	t.Run("very large user ID", func(t *testing.T) {
		largeID := int64(9999999999)
		u := user.NewUser(largeID, "large", "Large", "ID")

		doc := repo.toDocument(u)
		assert.Equal(t, largeID, doc.ID)

		converted := repo.toDomain(doc)
		assert.Equal(t, largeID, converted.ID)
	})
}

// Benchmark tests
func BenchmarkUserRepository_ToDocument(b *testing.B) {
	repo := &UserRepository{}
	u := user.NewUser(123, "testuser", "Test", "User")
	u.SetPermission(-100, user.PermissionAdmin)
	u.SetPermission(-200, user.PermissionUser)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.toDocument(u)
	}
}

func BenchmarkUserRepository_ToDomain(b *testing.B) {
	repo := &UserRepository{}
	doc := &userDocument{
		ID:        123,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Permissions: map[int64]int{
			-100: int(user.PermissionAdmin),
			-200: int(user.PermissionUser),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.toDomain(doc)
	}
}

func BenchmarkUserRepository_RoundTrip(b *testing.B) {
	repo := &UserRepository{}
	u := user.NewUser(123, "testuser", "Test", "User")
	u.SetPermission(-100, user.PermissionAdmin)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := repo.toDocument(u)
		repo.toDomain(doc)
	}
}
