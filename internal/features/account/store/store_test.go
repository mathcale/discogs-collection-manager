package store_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/mathcale/discogs-collection-manager/internal/features/account/domain"
	"github.com/mathcale/discogs-collection-manager/internal/features/account/store"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
)

func TestFindByUserID_ReturnsErrNotFoundWhenNoRowMatches(t *testing.T) {
	f, err := os.CreateTemp("", "account-store-test-*.db")
	require.NoError(t, err)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	require.NoError(t, migrate.Up(f.Name(), "../../../../migrations"))

	db, err := sql.Open("sqlite3", f.Name())
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	token, err := store.NewOAuthTokenRepository(db).FindByUserID(context.Background(), "missing")

	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Nil(t, token)
}
