package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
	authdomain "github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type LoadSettingsInput struct {
	UserID string
}

type LoadSettingsOutput struct {
	Email           string
	DiscogsUsername string
}

var ErrUserNotFound = errors.New("user not found")

type userFinder interface {
	FindByID(ctx context.Context, id string) (*authdomain.User, error)
}

type LoadSettings struct {
	Users       userFinder
	OAuthTokens domain.OAuthTokenRepository
}

func (uc *LoadSettings) Execute(ctx context.Context, in LoadSettingsInput) (LoadSettingsOutput, error) {
	user, err := uc.Users.FindByID(ctx, in.UserID)
	if errors.Is(err, authdomain.ErrNotFound) {
		return LoadSettingsOutput{}, ErrUserNotFound
	}
	if err != nil {
		return LoadSettingsOutput{}, err
	}

	out := LoadSettingsOutput{Email: user.Email}

	token, err := uc.OAuthTokens.FindByUserID(ctx, in.UserID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return LoadSettingsOutput{}, err
	}
	if err == nil {
		out.DiscogsUsername = token.DiscogsUsername
	}

	return out, nil
}
