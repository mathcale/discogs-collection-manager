package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	OTPExpiry            = 10 * time.Minute
	MaxOTPFailedAttempts = 5
)

type OTPCode struct {
	ID        string
	UserID    string
	CodeHash  string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

var ErrOTPNotPending = errors.New("otp code is not pending")

func (o *OTPCode) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}

func (o *OTPCode) Matches(code string) bool {
	return HashToken(digitsOnly(code)) == o.CodeHash
}

func digitsOnly(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}

		return -1
	}, value)
}

type OTPRepository interface {
	Create(ctx context.Context, userID, codeHash string, expiresAt time.Time) (*OTPCode, error)
	FindPendingByUserID(ctx context.Context, userID string) (*OTPCode, error)
	MarkUsed(ctx context.Context, id string) error
	RecordFailedAttempt(ctx context.Context, id string, maxAttempts int) error
}

type otpRepository struct {
	db *sql.DB
}

func NewOTPRepository(db *sql.DB) OTPRepository {
	return &otpRepository{
		db: db,
	}
}

func (r *otpRepository) Create(
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

func (r *otpRepository) FindPendingByUserID(
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

func (r *otpRepository) MarkUsed(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE otp_codes SET used_at = ? WHERE id = ? AND used_at IS NULL`,
		time.Now().UTC().Format(time.DateTime), id,
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrOTPNotPending
	}

	return nil
}

func (r *otpRepository) RecordFailedAttempt(ctx context.Context, id string, maxAttempts int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE otp_codes
		 SET failed_attempts = failed_attempts + 1,
		     used_at = CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE used_at END
		 WHERE id = ? AND used_at IS NULL`,
		maxAttempts, time.Now().UTC().Format(time.DateTime), id,
	)

	return err
}
