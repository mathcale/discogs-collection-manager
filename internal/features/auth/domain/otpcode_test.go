package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

func TestOTPCode_MarkUsed_FailsWhenAlreadyUsed(t *testing.T) {
	otp := domain.NewOTPCode("user-1", domain.HashToken("123456"))

	require.NoError(t, otp.MarkUsed())
	require.ErrorIs(t, otp.MarkUsed(), domain.ErrOTPNotPending)
}

func TestOTPCode_RecordFailedAttempt_MarksUsedAtMaximum(t *testing.T) {
	otp := domain.NewOTPCode("user-1", domain.HashToken("123456"))

	for range domain.MaxOTPFailedAttempts - 1 {
		require.NoError(t, otp.RecordFailedAttempt())
		require.Nil(t, otp.UsedAt)
	}

	require.NoError(t, otp.RecordFailedAttempt())
	require.NotNil(t, otp.UsedAt)
	require.Equal(t, domain.MaxOTPFailedAttempts, otp.FailedAttempts)
}
