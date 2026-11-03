package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type sessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) domain.SessionRepository {
	return &sessionRepository{
		db: db,
	}
}

func (r *sessionRepository) Save(ctx context.Context, session *domain.Session) error {
	persistentInt := 0

	if session.Persistent {
		persistentInt = 1
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, token, expires_at, persistent, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.Token, session.ExpiresAt.UTC().Format(time.DateTime), persistentInt, session.CreatedAt.UTC().Format(time.DateTime),
	)

	return err
}

func (r *sessionRepository) FindByToken(
	ctx context.Context,
	token string,
) (*domain.Session, error) {
	var s domain.Session
	var expiresAt, createdAt string
	var persistentInt int

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, token, expires_at, persistent, created_at FROM sessions WHERE token = ?`, token,
	).Scan(&s.ID, &s.UserID, &s.Token, &expiresAt, &persistentInt, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	s.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	s.Persistent = persistentInt == 1

	return &s, nil
}

func (r *sessionRepository) Delete(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)

	return err
}

func (r *sessionRepository) DeleteAllForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)

	return err
}
