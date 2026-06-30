package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID         string
	UserID     string
	Token      string
	ExpiresAt  time.Time
	Persistent bool
	CreatedAt  time.Time
}

type SessionRepository interface {
	Create(ctx context.Context, userID, token string, expiresAt time.Time, persistent bool) (*Session, error)
	FindByToken(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
}

type SQLiteSessionRepository struct {
	db *sql.DB
}

func NewSQLiteSessionRepository(db *sql.DB) *SQLiteSessionRepository {
	return &SQLiteSessionRepository{db: db}
}

func (r *SQLiteSessionRepository) Create(ctx context.Context, userID, token string, expiresAt time.Time, persistent bool) (*Session, error) {
	id := uuid.NewString()
	now := time.Now().UTC()

	persistentInt := 0
	if persistent {
		persistentInt = 1
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, token, expires_at, persistent, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, token, expiresAt.UTC().Format(time.DateTime), persistentInt, now.Format(time.DateTime),
	)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:         id,
		UserID:     userID,
		Token:      token,
		ExpiresAt:  expiresAt,
		Persistent: persistent,
		CreatedAt:  now,
	}, nil
}

func (r *SQLiteSessionRepository) FindByToken(ctx context.Context, token string) (*Session, error) {
	var s Session
	var expiresAt, createdAt string
	var persistentInt int

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, token, expires_at, persistent, created_at FROM sessions WHERE token = ?`, token,
	).Scan(&s.ID, &s.UserID, &s.Token, &expiresAt, &persistentInt, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	s.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	s.Persistent = persistentInt == 1

	return &s, nil
}

func (r *SQLiteSessionRepository) Delete(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	return err
}
