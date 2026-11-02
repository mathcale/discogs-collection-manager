package usecases

import (
	"context"
	"errors"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
)

type RegisterInput struct {
	Email    string
	Password string
}

type RegisterOutput struct{}

var ErrEmailTaken = errors.New("email already registered")

type Register struct {
	Users domain.UserRepository
}

func (uc *Register) Execute(ctx context.Context, in RegisterInput) (RegisterOutput, error) {
	email := domain.NormalizeEmail(in.Email)

	_, err := uc.Users.FindByEmail(ctx, email)
	if err == nil {
		return RegisterOutput{}, ErrEmailTaken
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return RegisterOutput{}, err
	}

	user, err := domain.NewUser(email, in.Password)
	if err != nil {
		return RegisterOutput{}, err
	}

	if err := uc.Users.Save(ctx, user); err != nil {
		return RegisterOutput{}, err
	}

	return RegisterOutput{}, nil
}
