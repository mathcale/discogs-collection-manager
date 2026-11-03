package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/mailer"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
)

type (
	LoginInput struct {
		Email        string
		Password     string
		StayLoggedIn bool
	}

	LoginOutput struct {
		UserID           string
		OTPRequired      bool
		SessionToken     string
		SessionExpiresAt time.Time
		OTPExpiresIn     time.Duration
	}

	Login struct {
		UsersRepo          domain.UserRepository
		SessionsRepo       domain.SessionRepository
		OTPCodesRepo       domain.OTPRepository
		Mailer             mailer.Mailer
		TwoStepAuthEnabled bool
	}
)

func (uc *Login) Execute(ctx context.Context, in LoginInput) (LoginOutput, error) {
	email := domain.NormalizeEmail(in.Email)

	user, err := uc.UsersRepo.FindByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return LoginOutput{}, domain.ErrInvalidCredentials
	}

	if err != nil {
		return LoginOutput{}, err
	}

	if !user.CheckPassword(in.Password) {
		return LoginOutput{}, domain.ErrInvalidCredentials
	}

	if !uc.TwoStepAuthEnabled {
		token, expiresAt, err := createSession(ctx, uc.SessionsRepo, user.ID, in.StayLoggedIn)
		if err != nil {
			return LoginOutput{}, err
		}

		return LoginOutput{
			UserID:           user.ID,
			SessionToken:     token,
			SessionExpiresAt: expiresAt,
		}, nil
	}

	code, err := domain.GenerateOTPCode()
	if err != nil {
		return LoginOutput{}, err
	}

	codeHash := domain.HashToken(code)
	if err := uc.OTPCodesRepo.Save(ctx, domain.NewOTPCode(user.ID, codeHash)); err != nil {
		return LoginOutput{}, err
	}

	if err := uc.Mailer.Send(
		user.Email,
		i18n.T(ctx, "Your login code"),
		i18n.T(ctx, "Your verification code is: <strong>%s</strong>. It expires in 10 minutes.", code),
	); err != nil {
		return LoginOutput{}, err
	}

	return LoginOutput{
		UserID:       user.ID,
		OTPRequired:  true,
		OTPExpiresIn: domain.OTPExpiry,
	}, nil
}
