package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type (
	OAuthTokenRepository interface {
		FindByUserID(ctx context.Context, userID string) (*OAuthToken, error)
		Save(ctx context.Context, token *OAuthToken) error
		DeleteByUserID(ctx context.Context, userID string) error
	}

	OAuthToken struct {
		ID                string
		UserID            string
		AccessToken       string
		AccessTokenSecret string
		DiscogsUsername   string
		CreatedAt         time.Time
	}
)

func NewOAuthToken(userID, accessToken, accessTokenSecret, discogsUsername string) *OAuthToken {
	return &OAuthToken{
		ID:                uuid.NewString(),
		UserID:            userID,
		AccessToken:       accessToken,
		AccessTokenSecret: accessTokenSecret,
		DiscogsUsername:   discogsUsername,
		CreatedAt:         time.Now().UTC(),
	}
}
