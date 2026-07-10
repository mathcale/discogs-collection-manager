package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type OTPCode struct {
	ID        string
	UserID    string
	CodeHash  string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type OTPRepository interface {
	Create(ctx context.Context, userID, codeHash string, expiresAt time.Time) (*OTPCode, error)
	FindPendingByUserID(ctx context.Context, userID string) (*OTPCode, error)
	MarkUsed(ctx context.Context, id string) error
}

type SQLiteOTPRepository struct {
	db *sql.DB
}

func NewSQLiteOTPRepository(db *sql.DB) *SQLiteOTPRepository {
	return &SQLiteOTPRepository{
		db: db,
	}
}

func (r *SQLiteOTPRepository) Create(
	ctx context.Context,
	userID, codeHash string,
	expiresAt time.Time,
) (*OTPCode, error) {
	id := uuid.NewString()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO otp_codes (id, user_id, code_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, codeHash, expiresAt.UTC().Format(time.DateTime), now.Format(time.DateTime),
	)
	if err != nil {
		return nil, err
	}

	return &OTPCode{
		ID:        id,
		UserID:    userID,
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

func (r *SQLiteOTPRepository) FindPendingByUserID(
	ctx context.Context,
	userID string,
) (*OTPCode, error) {
	var otp OTPCode
	var usedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, code_hash, expires_at, used_at, created_at
		 FROM otp_codes
		 WHERE user_id = ? AND used_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID,
	).Scan(&otp.ID, &otp.UserID, &otp.CodeHash, &otp.ExpiresAt, &usedAt, &otp.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if usedAt.Valid {
		otp.UsedAt = &usedAt.Time
	}

	return &otp, nil
}

func (r *SQLiteOTPRepository) MarkUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE otp_codes SET used_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.DateTime), id,
	)

	return err
}
