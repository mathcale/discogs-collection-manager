package domain

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type (
	UserRepository interface {
		FindByEmail(ctx context.Context, email string) (*User, error)
		FindByID(ctx context.Context, id string) (*User, error)
		Save(ctx context.Context, user *User) error
	}

	User struct {
		ID           string
		Email        string
		PasswordHash string
		CreatedAt    time.Time
	}
)

func NewUser(email, password string) (*User, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           uuid.NewString(),
		Email:        NormalizeEmail(email),
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func (u *User) ChangePassword(password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}

	u.PasswordHash = hash

	return nil
}

func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}
