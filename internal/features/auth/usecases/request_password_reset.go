package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/mailer"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
)

type RequestPasswordResetInput struct {
	Email string
}

type RequestPasswordResetOutput struct{}

type RequestPasswordReset struct {
	Users       models.UserRepository
	ResetTokens models.PasswordResetTokenRepository
	Mailer      mailer.Mailer
	BaseURL     string
}

func (uc *RequestPasswordReset) Execute(ctx context.Context, in RequestPasswordResetInput) (RequestPasswordResetOutput, error) {
	email := models.NormalizeEmail(in.Email)

	user, err := uc.Users.FindByEmail(ctx, email)
	if err != nil {
		return RequestPasswordResetOutput{}, err
	}
	if user == nil {
		return RequestPasswordResetOutput{}, nil
	}

	rawToken, err := models.GenerateToken()
	if err != nil {
		return RequestPasswordResetOutput{}, err
	}

	if err := uc.ResetTokens.MarkAllUsedForUser(ctx, user.ID); err != nil {
		return RequestPasswordResetOutput{}, err
	}

	tokenHash := models.HashToken(rawToken)
	if _, err := uc.ResetTokens.Create(ctx, user.ID, tokenHash, time.Now().Add(models.ResetTokenExpiry)); err != nil {
		return RequestPasswordResetOutput{}, err
	}

	resetLink := uc.BaseURL + "/auth/reset-password?token=" + rawToken
	body := fmt.Sprintf("Click the link below to reset your password:<br><a href=\"%s\">%s</a><br>This link expires in 1 hour.", resetLink, resetLink)
	if err := uc.Mailer.Send(user.Email, "Reset your password", body); err != nil {
		return RequestPasswordResetOutput{}, err
	}

	return RequestPasswordResetOutput{}, nil
}
