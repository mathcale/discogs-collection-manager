package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/mailer"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
)

type LoginInput struct {
	Email        string
	Password     string
	StayLoggedIn bool
}

type LoginOutput struct {
	UserID           string
	OTPRequired      bool
	SessionToken     string
	SessionExpiresAt time.Time
}

var ErrInvalidCredentials = errors.New("invalid credentials")

type Login struct {
	Users              models.UserRepository
	Sessions           models.SessionRepository
	OTPCodes           models.OTPRepository
	Mailer             mailer.Mailer
	TwoStepAuthEnabled bool
}

func (uc *Login) Execute(ctx context.Context, in LoginInput) (LoginOutput, error) {
	email := models.NormalizeEmail(in.Email)

	user, err := uc.Users.FindByEmail(ctx, email)
	if err != nil {
		return LoginOutput{}, err
	}
	if user == nil || !user.CheckPassword(in.Password) {
		return LoginOutput{}, ErrInvalidCredentials
	}

	if !uc.TwoStepAuthEnabled {
		token, expiresAt, err := createSession(ctx, uc.Sessions, user.ID, in.StayLoggedIn)
		if err != nil {
			return LoginOutput{}, err
		}

		return LoginOutput{UserID: user.ID, SessionToken: token, SessionExpiresAt: expiresAt}, nil
	}

	code, err := models.GenerateOTPCode()
	if err != nil {
		return LoginOutput{}, err
	}

	codeHash := models.HashToken(code)
	if _, err := uc.OTPCodes.Create(ctx, user.ID, codeHash, time.Now().Add(models.OTPExpiry)); err != nil {
		return LoginOutput{}, err
	}

	if err := uc.Mailer.Send(user.Email, "Your login code", fmt.Sprintf("Your verification code is: <strong>%s</strong>. It expires in 10 minutes.", code)); err != nil {
		return LoginOutput{}, err
	}

	return LoginOutput{UserID: user.ID, OTPRequired: true}, nil
}
