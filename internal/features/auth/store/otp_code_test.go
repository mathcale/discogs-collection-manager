package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/store"
)

func TestOTPRepository_Save_RejectsStaleVersion(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()
	otpCodes := store.NewOTPRepository(db)

	created := domain.NewOTPCode("user-1", domain.HashToken("123456"))
	require.NoError(t, otpCodes.Save(ctx, created))

	first, err := otpCodes.FindPendingByUserID(ctx, "user-1")
	require.NoError(t, err)
	second, err := otpCodes.FindPendingByUserID(ctx, "user-1")
	require.NoError(t, err)

	require.NoError(t, first.RecordFailedAttempt())
	require.NoError(t, otpCodes.Save(ctx, first))
	require.Equal(t, second.Version+1, first.Version)

	require.NoError(t, second.RecordFailedAttempt())
	require.ErrorIs(t, otpCodes.Save(ctx, second), domain.ErrConcurrentUpdate)

	stored, err := otpCodes.FindPendingByUserID(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, 1, stored.FailedAttempts)
}
