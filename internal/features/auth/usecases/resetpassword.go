package usecases

import (
	"context"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type (
	ResetPasswordInput struct {
		Token           string
		Password        string
		PasswordConfirm string
	}

	ResetPasswordOutput struct{}

	oauthTokenRevoker interface {
		DeleteByUserID(ctx context.Context, userID string) error
	}

	ResetPassword struct {
		UsersRepo       domain.UserRepository
		ResetTokensRepo domain.PasswordResetTokenRepository
		SessionsRepo    domain.SessionRepository
		OAuthTokensRepo oauthTokenRevoker
	}
)

func (uc *ResetPassword) Execute(
	ctx context.Context,
	in ResetPasswordInput,
) (ResetPasswordOutput, error) {
	if in.Password != in.PasswordConfirm {
		return ResetPasswordOutput{}, domain.ErrPasswordMismatch
	}

	record, err := findUsableResetToken(ctx, uc.ResetTokensRepo, in.Token)
	if err != nil {
		return ResetPasswordOutput{}, err
	}

	user, err := uc.UsersRepo.FindByID(ctx, record.UserID)
	if err != nil {
		return ResetPasswordOutput{}, err
	}

	if err := user.ChangePassword(in.Password); err != nil {
		return ResetPasswordOutput{}, err
	}

	if err := uc.UsersRepo.Save(ctx, user); err != nil {
		return ResetPasswordOutput{}, err
	}

	if err := uc.ResetTokensRepo.MarkAllUsedForUser(ctx, record.UserID); err != nil {
		return ResetPasswordOutput{}, err
	}

	if err := uc.SessionsRepo.DeleteAllForUser(ctx, record.UserID); err != nil {
		return ResetPasswordOutput{}, err
	}

	if err := uc.OAuthTokensRepo.DeleteByUserID(ctx, record.UserID); err != nil {
		return ResetPasswordOutput{}, err
	}

	return ResetPasswordOutput{}, nil
}
