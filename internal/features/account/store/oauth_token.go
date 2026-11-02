package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
)

type oauthTokenRepository struct {
	db *sql.DB
}

func NewOAuthTokenRepository(db *sql.DB) domain.OAuthTokenRepository {
	return &oauthTokenRepository{
		db: db,
	}
}

func (r *oauthTokenRepository) Save(ctx context.Context, token *domain.OAuthToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oauth_tokens (id, user_id, access_token, access_token_secret, discogs_username, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			id = excluded.id,
			access_token = excluded.access_token,
			access_token_secret = excluded.access_token_secret,
			discogs_username = excluded.discogs_username,
			created_at = excluded.created_at`,
		token.ID, token.UserID, token.AccessToken, token.AccessTokenSecret, token.DiscogsUsername, token.CreatedAt.Format(time.DateTime),
	)

	return err
}

func (r *oauthTokenRepository) FindByUserID(
	ctx context.Context,
	userID string,
) (*domain.OAuthToken, error) {
	var t domain.OAuthToken
	var createdAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, access_token, access_token_secret, discogs_username, created_at FROM oauth_tokens WHERE user_id = ?`,
		userID,
	).Scan(&t.ID, &t.UserID, &t.AccessToken, &t.AccessTokenSecret, &t.DiscogsUsername, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &t, nil
}

func (r *oauthTokenRepository) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM oauth_tokens WHERE user_id = ?`, userID)

	return err
}
