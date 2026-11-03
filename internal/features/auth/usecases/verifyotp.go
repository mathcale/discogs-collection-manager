package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type (
	VerifyOTPInput struct {
		UserID       string
		Code         string
		StayLoggedIn bool
	}

	VerifyOTPOutput struct {
		SessionToken     string
		SessionExpiresAt time.Time
	}

	VerifyOTP struct {
		OTPCodesRepo domain.OTPRepository
		SessionsRepo domain.SessionRepository
	}
)

func (uc *VerifyOTP) Execute(ctx context.Context, in VerifyOTPInput) (VerifyOTPOutput, error) {
	otp, err := uc.OTPCodesRepo.FindPendingByUserID(ctx, in.UserID)

	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return VerifyOTPOutput{}, domain.ErrOTPNotFound
		}

		return VerifyOTPOutput{}, err
	}

	if otp.IsExpired() {
		return VerifyOTPOutput{}, domain.ErrOTPExpired
	}

	if !otp.Matches(in.Code) {
		if err := uc.recordFailedAttempt(ctx, otp); err != nil {
			return VerifyOTPOutput{}, err
		}

		return VerifyOTPOutput{}, domain.ErrOTPMismatch
	}

	if err := otp.MarkUsed(); err != nil {
		return VerifyOTPOutput{}, domain.ErrOTPNotFound
	}

	if err := uc.OTPCodesRepo.Save(ctx, otp); err != nil {
		if errors.Is(err, domain.ErrConcurrentUpdate) {
			return VerifyOTPOutput{}, domain.ErrOTPNotFound
		}

		return VerifyOTPOutput{}, err
	}

	token, expiresAt, err := createSession(ctx, uc.SessionsRepo, in.UserID, in.StayLoggedIn)
	if err != nil {
		return VerifyOTPOutput{}, err
	}

	return VerifyOTPOutput{
		SessionToken:     token,
		SessionExpiresAt: expiresAt,
	}, nil
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

		err := uc.OTPCodesRepo.Save(ctx, otp)
		if !errors.Is(err, domain.ErrConcurrentUpdate) {
			return err
		}

		otp, err = uc.OTPCodesRepo.FindPendingByUserID(ctx, otp.UserID)
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}

		if err != nil {
			return err
		}
	}
}
