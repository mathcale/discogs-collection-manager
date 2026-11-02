package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

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

func (r *otpRepository) Save(ctx context.Context, otp *domain.OTPCode) error {
	var usedAt any
	if otp.UsedAt != nil {
		usedAt = otp.UsedAt.UTC().Format(time.DateTime)
	}

	err := r.db.QueryRowContext(ctx,
		`INSERT INTO otp_codes (id, user_id, code_hash, expires_at, used_at, created_at, failed_attempts, version)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     used_at = excluded.used_at,
		     failed_attempts = excluded.failed_attempts,
		     version = otp_codes.version + 1
		 WHERE otp_codes.version = excluded.version
		 RETURNING version`,
		otp.ID, otp.UserID, otp.CodeHash,
		otp.ExpiresAt.UTC().Format(time.DateTime), usedAt, otp.CreatedAt.UTC().Format(time.DateTime),
		otp.FailedAttempts, otp.Version,
	).Scan(&otp.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrConcurrentUpdate
	}

	return err
}

func (r *otpRepository) FindPendingByUserID(
	ctx context.Context,
	userID string,
) (*domain.OTPCode, error) {
	var otp domain.OTPCode
	var usedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, code_hash, expires_at, used_at, created_at, failed_attempts, version
		 FROM otp_codes
		 WHERE user_id = ? AND used_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID,
	).Scan(&otp.ID, &otp.UserID, &otp.CodeHash, &otp.ExpiresAt, &usedAt, &otp.CreatedAt, &otp.FailedAttempts, &otp.Version)
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
