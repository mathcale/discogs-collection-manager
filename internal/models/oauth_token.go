package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
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

type SQLiteOAuthTokenRepository struct {
	db *sql.DB
}

func NewSQLiteOAuthTokenRepository(db *sql.DB) *SQLiteOAuthTokenRepository {
	return &SQLiteOAuthTokenRepository{
		db: db,
	}
}

func (r *SQLiteOAuthTokenRepository) Upsert(
	ctx context.Context,
	userID, accessToken, accessTokenSecret, discogsUsername string,
) (*OAuthToken, error) {
	id := uuid.NewString()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oauth_tokens (id, user_id, access_token, access_token_secret, discogs_username, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			access_token = excluded.access_token,
			access_token_secret = excluded.access_token_secret,
			discogs_username = excluded.discogs_username`,
		id, userID, accessToken, accessTokenSecret, discogsUsername, now.Format(time.DateTime),
	)
	if err != nil {
		return nil, err
	}

	return &OAuthToken{
		ID:                id,
		UserID:            userID,
		AccessToken:       accessToken,
		AccessTokenSecret: accessTokenSecret,
		DiscogsUsername:   discogsUsername,
		CreatedAt:         now,
	}, nil
}

func (r *SQLiteOAuthTokenRepository) FindByUserID(
	ctx context.Context,
	userID string,
) (*OAuthToken, error) {
	var t OAuthToken
	var createdAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, access_token, access_token_secret, discogs_username, created_at FROM oauth_tokens WHERE user_id = ?`,
		userID,
	).Scan(&t.ID, &t.UserID, &t.AccessToken, &t.AccessTokenSecret, &t.DiscogsUsername, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &t, nil
}

func (r *SQLiteOAuthTokenRepository) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM oauth_tokens WHERE user_id = ?`, userID)

	return err
}
