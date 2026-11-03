package domain

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	OTPExpiry            = 10 * time.Minute
	MaxOTPFailedAttempts = 5
)

type (
	OTPRepository interface {
		FindPendingByUserID(ctx context.Context, userID string) (*OTPCode, error)
		Save(ctx context.Context, otp *OTPCode) error
	}

	OTPCode struct {
		ID        string
		UserID    string
		CodeHash  string
		ExpiresAt time.Time
		UsedAt    *time.Time
		CreatedAt time.Time

		FailedAttempts int
		Version        int
	}
)

func NewOTPCode(userID, codeHash string) *OTPCode {
	now := time.Now().UTC()

	return &OTPCode{
		ID:        uuid.NewString(),
		UserID:    userID,
		CodeHash:  codeHash,
		ExpiresAt: now.Add(OTPExpiry),
		CreatedAt: now,
	}
}

func (o *OTPCode) MarkUsed() error {
	if o.UsedAt != nil {
		return ErrOTPNotPending
	}

	now := time.Now().UTC()
	o.UsedAt = &now

	return nil
}

func (o *OTPCode) RecordFailedAttempt() error {
	if o.UsedAt != nil {
		return ErrOTPNotPending
	}

	o.FailedAttempts++

	if o.FailedAttempts >= MaxOTPFailedAttempts {
		return o.MarkUsed()
	}

	return nil
}

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
