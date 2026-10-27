package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
)

type ResetPasswordInput struct {
	Token           string
	Password        string
	PasswordConfirm string
}

type ResetPasswordOutput struct{}

var ErrPasswordMismatch = errors.New("passwords do not match")

type oauthTokenRevoker interface {
	DeleteByUserID(ctx context.Context, userID string) error
}

type ResetPassword struct {
	Users       models.UserRepository
	ResetTokens models.PasswordResetTokenRepository
	Sessions    models.SessionRepository
	OAuthTokens oauthTokenRevoker
}

func (uc *ResetPassword) Execute(ctx context.Context, in ResetPasswordInput) (ResetPasswordOutput, error) {
	if in.Password != in.PasswordConfirm {
		return ResetPasswordOutput{}, ErrPasswordMismatch
	}

	record, err := findUsableResetToken(ctx, uc.ResetTokens, in.Token)
	if err != nil {
		return ResetPasswordOutput{}, err
	}

	hash, err := models.HashPassword(in.Password)
	if err != nil {
		return ResetPasswordOutput{}, err
	}

	if err := uc.Users.UpdatePassword(ctx, record.UserID, hash); err != nil {
		return ResetPasswordOutput{}, err
	}

	if err := uc.ResetTokens.MarkAllUsedForUser(ctx, record.UserID); err != nil {
		return ResetPasswordOutput{}, err
	}

	if err := uc.Sessions.DeleteAllForUser(ctx, record.UserID); err != nil {
		return ResetPasswordOutput{}, err
	}

	if err := uc.OAuthTokens.DeleteByUserID(ctx, record.UserID); err != nil {
		return ResetPasswordOutput{}, err
	}

	return ResetPasswordOutput{}, nil
}
