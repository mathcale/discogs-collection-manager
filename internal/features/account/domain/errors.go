package domain

import "errors"

var (
	ErrNotFound               = errors.New("not found")
	ErrUserNotFound           = errors.New("user not found")
	ErrRequestTokenFailed     = errors.New("discogs request token failed")
	ErrAuthorizationURLFailed = errors.New("discogs authorization url failed")
)
