package kernel

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/dghubble/oauth1"
	_ "github.com/mattn/go-sqlite3"

	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
	"github.com/mathcale/discogs-collection-manager/internal/repository"
)

type Kernel struct {
	DB          *sql.DB
	Users       repository.UserRepository
	Sessions    repository.SessionRepository
	OAuthTokens repository.OAuthTokenRepository
	DiscogsOAuth *oauth1.Config
}

func New() (*Kernel, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/discogs.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()

		return nil, fmt.Errorf("connect to database: %w", err)
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "./migrations"
	}

	if err := migrate.Up(dbPath, migrationsPath); err != nil {
		db.Close()

		return nil, fmt.Errorf("run migrations: %w", err)
	}

	discogsOAuth := &oauth1.Config{
		ConsumerKey:    os.Getenv("DISCOGS_CONSUMER_KEY"),
		ConsumerSecret: os.Getenv("DISCOGS_CONSUMER_SECRET"),
		CallbackURL:    os.Getenv("DISCOGS_CALLBACK_URL"),
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: "https://api.discogs.com/oauth/request_token",
			AuthorizeURL:    "https://www.discogs.com/oauth/authorize",
			AccessTokenURL:  "https://api.discogs.com/oauth/access_token",
		},
	}

	return &Kernel{
		DB:           db,
		Users:        repository.NewSQLiteUserRepository(db),
		Sessions:     repository.NewSQLiteSessionRepository(db),
		OAuthTokens:  repository.NewSQLiteOAuthTokenRepository(db),
		DiscogsOAuth: discogsOAuth,
	}, nil
}

func (k *Kernel) Close() {
	k.DB.Close()
}
