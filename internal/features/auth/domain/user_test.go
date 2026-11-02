package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

func TestNewUser_NormalizesEmailHashesPasswordAndSetsIdentity(t *testing.T) {
	before := time.Now().UTC().Add(-time.Second)

	user, err := domain.NewUser("  Foo@Example.COM ", "secret123")

	require.NoError(t, err)
	require.Equal(t, "foo@example.com", user.Email)
	require.NotEmpty(t, user.ID)
	require.NotEqual(t, "secret123", user.PasswordHash)
	require.True(t, user.CheckPassword("secret123"))
	require.True(t, user.CreatedAt.After(before))
}

func TestNewUser_GeneratesDistinctIDs(t *testing.T) {
	first, err := domain.NewUser("a@example.com", "secret123")
	require.NoError(t, err)
	second, err := domain.NewUser("b@example.com", "secret123")
	require.NoError(t, err)

	require.NotEqual(t, first.ID, second.ID)
}

func TestUser_ChangePassword_ReplacesHash(t *testing.T) {
	user, err := domain.NewUser("a@example.com", "old-password")
	require.NoError(t, err)

	require.NoError(t, user.ChangePassword("new-password"))

	require.True(t, user.CheckPassword("new-password"))
	require.False(t, user.CheckPassword("old-password"))
}
