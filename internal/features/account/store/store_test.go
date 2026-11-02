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

func TestSave_TwiceForSameUser_LeavesOneRowWithLatestCredentials(t *testing.T) {
	f, err := os.CreateTemp("", "account-store-test-*.db")
	require.NoError(t, err)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	require.NoError(t, migrate.Up(f.Name(), "../../../../migrations"))

	db, err := sql.Open("sqlite3", f.Name())
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`INSERT INTO users (id, email, password_hash) VALUES ('user-1', 'a@example.com', 'hash')`)
	require.NoError(t, err)

	repo := store.NewOAuthTokenRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, domain.NewOAuthToken("user-1", "old-access", "old-secret", "olduser")))
	require.NoError(t, repo.Save(ctx, domain.NewOAuthToken("user-1", "new-access", "new-secret", "newuser")))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM oauth_tokens WHERE user_id = 'user-1'`).Scan(&count))
	require.Equal(t, 1, count)

	found, err := repo.FindByUserID(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, "new-access", found.AccessToken)
	require.Equal(t, "new-secret", found.AccessTokenSecret)
	require.Equal(t, "newuser", found.DiscogsUsername)
}
