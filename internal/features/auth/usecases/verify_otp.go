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
	if errors.Is(err, domain.ErrNotFound) {
		return VerifyOTPOutput{}, ErrOTPNotFound
	}
	if err != nil {
		return VerifyOTPOutput{}, err
	}

	if otp.IsExpired() {
		return VerifyOTPOutput{}, ErrOTPExpired
	}

	if !otp.Matches(in.Code) {
		if err := uc.recordFailedAttempt(ctx, otp); err != nil {
			return VerifyOTPOutput{}, err
		}

		return VerifyOTPOutput{}, ErrOTPMismatch
	}

	if err := otp.MarkUsed(); err != nil {
		return VerifyOTPOutput{}, ErrOTPNotFound
	}

	if err := uc.OTPCodes.Save(ctx, otp); err != nil {
		if errors.Is(err, domain.ErrConcurrentUpdate) {
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

func (uc *VerifyOTP) recordFailedAttempt(ctx context.Context, otp *domain.OTPCode) error {
	attemptedCodeID := otp.ID

	for {
		if otp.ID != attemptedCodeID {
			return nil
		}

		if err := otp.RecordFailedAttempt(); err != nil {
			return nil
		}

		err := uc.OTPCodes.Save(ctx, otp)
		if !errors.Is(err, domain.ErrConcurrentUpdate) {
			return err
		}

		otp, err = uc.OTPCodes.FindPendingByUserID(ctx, otp.UserID)
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
