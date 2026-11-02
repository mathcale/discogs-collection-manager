package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type otpRepository struct {
	db *sql.DB
}

func NewOTPRepository(db *sql.DB) domain.OTPRepository {
	return &otpRepository{
		db: db,
	}
}

func (r *otpRepository) Create(
	ctx context.Context,
	userID, codeHash string,
	expiresAt time.Time,
) (*domain.OTPCode, error) {
	id := uuid.NewString()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO otp_codes (id, user_id, code_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, codeHash, expiresAt.UTC().Format(time.DateTime), now.Format(time.DateTime),
	)
	if err != nil {
		return nil, err
	}

	return &domain.OTPCode{
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
) (*domain.OTPCode, error) {
	var otp domain.OTPCode
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
		return nil, domain.ErrNotFound
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
		return domain.ErrOTPNotPending
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
