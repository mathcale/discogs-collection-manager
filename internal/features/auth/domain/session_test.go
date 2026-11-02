package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

func TestNewSession_ShortExpiryWithoutRememberMe(t *testing.T) {
	before := time.Now()

	session, err := domain.NewSession("user-1", false)

	require.NoError(t, err)
	require.NotEmpty(t, session.ID)
	require.NotEmpty(t, session.Token)
	require.Equal(t, "user-1", session.UserID)
	require.False(t, session.Persistent)
	require.False(t, session.CreatedAt.Before(before))
	require.WithinDuration(t, before.Add(6*time.Hour), session.ExpiresAt, time.Minute)
}

func TestNewSession_LongExpiryWithRememberMe(t *testing.T) {
	before := time.Now()

	session, err := domain.NewSession("user-1", true)

	require.NoError(t, err)
	require.True(t, session.Persistent)
	require.WithinDuration(t, before.Add(30*24*time.Hour), session.ExpiresAt, time.Minute)
}

func TestNewSession_GeneratesUniqueTokensAndIDs(t *testing.T) {
	first, err := domain.NewSession("user-1", false)
	require.NoError(t, err)

	second, err := domain.NewSession("user-1", false)
	require.NoError(t, err)

	require.NotEqual(t, first.Token, second.Token)
	require.NotEqual(t, first.ID, second.ID)
}
