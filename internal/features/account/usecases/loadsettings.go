package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
	authdomain "github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type (
	LoadSettingsInput struct {
		UserID string
	}

	LoadSettingsOutput struct {
		Email           string
		DiscogsUsername string
	}

	userFinder interface {
		FindByID(ctx context.Context, id string) (*authdomain.User, error)
	}

	LoadSettings struct {
		UsersRepo       userFinder
		OAuthTokensRepo domain.OAuthTokenRepository
	}
)

func (uc *LoadSettings) Execute(
	ctx context.Context,
	in LoadSettingsInput,
) (LoadSettingsOutput, error) {
	user, err := uc.UsersRepo.FindByID(ctx, in.UserID)

	if err != nil {
		if errors.Is(err, authdomain.ErrNotFound) {
			return LoadSettingsOutput{}, domain.ErrUserNotFound
		}

		return LoadSettingsOutput{}, err
	}

	out := LoadSettingsOutput{
		Email: user.Email,
	}

	token, err := uc.OAuthTokensRepo.FindByUserID(ctx, in.UserID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return LoadSettingsOutput{}, err
	}

	if err == nil {
		out.DiscogsUsername = token.DiscogsUsername
	}

	return out, nil
}
