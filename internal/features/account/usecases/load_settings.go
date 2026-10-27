package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/models"
	authmodels "github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
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
	FindByID(ctx context.Context, id string) (*authmodels.User, error)
}

type LoadSettings struct {
	Users       userFinder
	OAuthTokens models.OAuthTokenRepository
}

func (uc *LoadSettings) Execute(ctx context.Context, in LoadSettingsInput) (LoadSettingsOutput, error) {
	user, err := uc.Users.FindByID(ctx, in.UserID)
	if err != nil {
		return LoadSettingsOutput{}, err
	}
	if user == nil {
		return LoadSettingsOutput{}, ErrUserNotFound
	}

	token, err := uc.OAuthTokens.FindByUserID(ctx, in.UserID)
	if err != nil {
		return LoadSettingsOutput{}, err
	}

	out := LoadSettingsOutput{Email: user.Email}
	if token != nil {
		out.DiscogsUsername = token.DiscogsUsername
	}

	return out, nil
}
