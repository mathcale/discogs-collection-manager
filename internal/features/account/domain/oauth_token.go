package domain

import (
	"context"
	"time"
)

type OAuthToken struct {
	ID                string
	UserID            string
	AccessToken       string
	AccessTokenSecret string
	DiscogsUsername   string
	CreatedAt         time.Time
}

type OAuthTokenRepository interface {
	Upsert(ctx context.Context, userID, accessToken, accessTokenSecret, discogsUsername string) (*OAuthToken, error)
	FindByUserID(ctx context.Context, userID string) (*OAuthToken, error)
	DeleteByUserID(ctx context.Context, userID string) error
}
