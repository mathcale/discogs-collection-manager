package domain

import (
	"context"
	"errors"
	"strings"
	"time"
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
