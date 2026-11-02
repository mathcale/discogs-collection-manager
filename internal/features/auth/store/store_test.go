package store_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/features/auth/domain"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth/store"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()

	f, err := os.CreateTemp("", "auth-store-test-*.db")
	require.NoError(t, err)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	require.NoError(t, migrate.Up(f.Name(), "../../../../migrations"))

	db, err := sql.Open("sqlite3", f.Name())
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	return db
}

func TestFindMethods_ReturnErrNotFoundWhenNoRowMatches(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()

	users := store.NewUserRepository(db)
	sessions := store.NewSessionRepository(db)
	otpCodes := store.NewOTPRepository(db)
	resetTokens := store.NewPasswordResetTokenRepository(db)

	userByEmail, err := users.FindByEmail(ctx, "missing@example.com")
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Nil(t, userByEmail)

	userByID, err := users.FindByID(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Nil(t, userByID)

	session, err := sessions.FindByToken(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Nil(t, session)

	otp, err := otpCodes.FindPendingByUserID(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Nil(t, otp)

	resetToken, err := resetTokens.FindByTokenHash(ctx, "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Nil(t, resetToken)
}

func TestUserRepository_SaveInsertsThenUpdatesWholeEntityByID(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()
	users := store.NewUserRepository(db)

	user, err := domain.NewUser("Save@Example.com", "first-password")
	require.NoError(t, err)

	require.NoError(t, users.Save(ctx, user))

	found, err := users.FindByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, user.ID, found.ID)
	require.Equal(t, "save@example.com", found.Email)
	require.True(t, found.CheckPassword("first-password"))

	require.NoError(t, user.ChangePassword("second-password"))
	require.NoError(t, users.Save(ctx, user))

	updated, err := users.FindByEmail(ctx, "save@example.com")
	require.NoError(t, err)
	require.Equal(t, user.ID, updated.ID)
	require.True(t, updated.CheckPassword("second-password"))
	require.False(t, updated.CheckPassword("first-password"))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count))
	require.Equal(t, 1, count)
}
