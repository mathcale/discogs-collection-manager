package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

func TestNewPasswordResetToken_SetsIdentityAndExpiry(t *testing.T) {
	before := time.Now()

	token := domain.NewPasswordResetToken("user-1", "hash-1")

	require.NotEmpty(t, token.ID)
	require.Equal(t, "user-1", token.UserID)
	require.Equal(t, "hash-1", token.TokenHash)
	require.Nil(t, token.UsedAt)
	require.False(t, token.CreatedAt.Before(before))
	require.Equal(t, domain.ResetTokenExpiry, token.ExpiresAt.Sub(token.CreatedAt))
	require.True(t, token.IsUsable())
}

func TestNewPasswordResetToken_GeneratesUniqueIDs(t *testing.T) {
	first := domain.NewPasswordResetToken("user-1", "hash-1")
	second := domain.NewPasswordResetToken("user-1", "hash-2")

	require.NotEqual(t, first.ID, second.ID)
}
