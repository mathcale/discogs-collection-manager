package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type VerifyOTPInput struct {
	UserID       string
	Code         string
	StayLoggedIn bool
}

type VerifyOTPOutput struct {
	SessionToken     string
	SessionExpiresAt time.Time
}

var (
	ErrOTPNotFound = errors.New("otp code not found")
	ErrOTPExpired  = errors.New("otp code expired")
	ErrOTPMismatch = errors.New("otp code mismatch")
)

type VerifyOTP struct {
	OTPCodes domain.OTPRepository
	Sessions domain.SessionRepository
}

func (uc *VerifyOTP) Execute(ctx context.Context, in VerifyOTPInput) (VerifyOTPOutput, error) {
	otp, err := uc.OTPCodes.FindPendingByUserID(ctx, in.UserID)
	if err != nil {
		return VerifyOTPOutput{}, err
	}

	if otp == nil {
		return VerifyOTPOutput{}, ErrOTPNotFound
	}

	if otp.IsExpired() {
		return VerifyOTPOutput{}, ErrOTPExpired
	}

	if !otp.Matches(in.Code) {
		if err := uc.OTPCodes.RecordFailedAttempt(ctx, otp.ID, domain.MaxOTPFailedAttempts); err != nil {
			return VerifyOTPOutput{}, err
		}

		return VerifyOTPOutput{}, ErrOTPMismatch
	}

	if err := uc.OTPCodes.MarkUsed(ctx, otp.ID); err != nil {
		if errors.Is(err, domain.ErrOTPNotPending) {
			return VerifyOTPOutput{}, ErrOTPNotFound
		}

		return VerifyOTPOutput{}, err
	}

	token, expiresAt, err := createSession(ctx, uc.Sessions, in.UserID, in.StayLoggedIn)
	if err != nil {
		return VerifyOTPOutput{}, err
	}

	return VerifyOTPOutput{SessionToken: token, SessionExpiresAt: expiresAt}, nil
}
