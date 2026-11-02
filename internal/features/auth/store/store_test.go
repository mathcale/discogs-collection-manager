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
