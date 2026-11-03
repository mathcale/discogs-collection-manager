package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrOTPNotPending      = errors.New("otp code is not pending")
	ErrOTPNotFound        = errors.New("otp code not found")
	ErrOTPExpired         = errors.New("otp code expired")
	ErrOTPMismatch        = errors.New("otp code mismatch")
	ErrConcurrentUpdate   = errors.New("concurrent update")
	ErrSessionInvalid     = errors.New("session invalid or expired")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrResetTokenInvalid  = errors.New("reset token invalid or expired")
	ErrPasswordMismatch   = errors.New("passwords do not match")
)
