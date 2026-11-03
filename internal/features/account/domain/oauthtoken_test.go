package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
)

func TestNewOAuthToken_SetsFieldsIDAndCreatedAt(t *testing.T) {
	before := time.Now().UTC().Add(-time.Second)

	token := domain.NewOAuthToken("user-1", "access", "secret", "discogsman")

	require.NotEmpty(t, token.ID)
	require.Equal(t, "user-1", token.UserID)
	require.Equal(t, "access", token.AccessToken)
	require.Equal(t, "secret", token.AccessTokenSecret)
	require.Equal(t, "discogsman", token.DiscogsUsername)
	require.True(t, token.CreatedAt.After(before))
}

func TestNewOAuthToken_GeneratesDistinctIDs(t *testing.T) {
	first := domain.NewOAuthToken("user-1", "a", "b", "c")
	second := domain.NewOAuthToken("user-1", "a", "b", "c")

	require.NotEqual(t, first.ID, second.ID)
}
