package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/mailer"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/i18n"
)

type RequestPasswordResetInput struct {
	Email string
}

type RequestPasswordResetOutput struct{}

type RequestPasswordReset struct {
	Users       domain.UserRepository
	ResetTokens domain.PasswordResetTokenRepository
	Mailer      mailer.Mailer
	BaseURL     string
}

func (uc *RequestPasswordReset) Execute(ctx context.Context, in RequestPasswordResetInput) (RequestPasswordResetOutput, error) {
	email := domain.NormalizeEmail(in.Email)

	user, err := uc.Users.FindByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return RequestPasswordResetOutput{}, nil
	}
	if err != nil {
		return RequestPasswordResetOutput{}, err
	}

	rawToken, err := domain.GenerateToken()
	if err != nil {
		return RequestPasswordResetOutput{}, err
	}

	if err := uc.ResetTokens.MarkAllUsedForUser(ctx, user.ID); err != nil {
		return RequestPasswordResetOutput{}, err
	}

	tokenHash := domain.HashToken(rawToken)
	if err := uc.ResetTokens.Save(ctx, domain.NewPasswordResetToken(user.ID, tokenHash)); err != nil {
		return RequestPasswordResetOutput{}, err
	}

	resetLink := uc.BaseURL + "/auth/reset-password?token=" + rawToken
	body := i18n.T(ctx, "Click the link below to reset your password:<br><a href=\"%s\">%s</a><br>This link expires in 1 hour.", resetLink, resetLink)
	if err := uc.Mailer.Send(user.Email, i18n.T(ctx, "Reset your password"), body); err != nil {
		return RequestPasswordResetOutput{}, err
	}

	return RequestPasswordResetOutput{}, nil
}
