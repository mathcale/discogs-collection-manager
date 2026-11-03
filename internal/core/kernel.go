package core

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"

	"github.com/dghubble/oauth1"
	_ "github.com/mattn/go-sqlite3"

	"github.com/mathcale/discogs-collection-manager/internal/core/clients/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/core/clients/mailer"
	"github.com/mathcale/discogs-collection-manager/internal/features/account"
	"github.com/mathcale/discogs-collection-manager/internal/features/account/store"
	"github.com/mathcale/discogs-collection-manager/internal/features/auth"
	"github.com/mathcale/discogs-collection-manager/internal/features/landing"
	"github.com/mathcale/discogs-collection-manager/internal/features/scanner"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/logger"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
)

type Kernel struct {
	DB      *sql.DB
	Logger  logger.Logger
	Landing *landing.Module
	Auth    *auth.Module
	Account *account.Module
	Scanner *scanner.Module
}

func New() (*Kernel, error) {
	env := os.Getenv("ENVIRONMENT")

	l := logger.New(env)

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	twoStepAuthEnabled, err := envBool("TWO_STEP_AUTH_ENABLED", true)
	if err != nil {
		return nil, err
	}

	signupsEnabled, err := envBool("SIGNUPS_ENABLED", true)
	if err != nil {
		return nil, err
	}

	resendAPIKey := os.Getenv("RESEND_API_KEY")
	resendFrom := os.Getenv("RESEND_FROM")
	if resendFrom == "" {
		resendFrom = "noreply@example.com"
	}

	db, err := connectToDatabase(l)
	if err != nil {
		return nil, err
	}

	discogsOAuthConfig := &oauth1.Config{
		ConsumerKey:    os.Getenv("DISCOGS_CONSUMER_KEY"),
		ConsumerSecret: os.Getenv("DISCOGS_CONSUMER_SECRET"),
		CallbackURL:    os.Getenv("DISCOGS_CALLBACK_URL"),
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: "https://api.discogs.com/oauth/request_token",
			AuthorizeURL:    "https://www.discogs.com/oauth/authorize",
			AccessTokenURL:  "https://api.discogs.com/oauth/access_token",
		},
	}

	oauthTokensRepo := store.NewOAuthTokenRepository(db)
	discogsClient := discogs.NewHTTPDiscogsClient(discogsOAuthConfig, oauthTokensRepo)

	return &Kernel{
		DB:     db,
		Logger: l,
		Landing: landing.New(landing.Deps{
			SignupsEnabled: signupsEnabled,
		}),
		Auth: auth.New(auth.Deps{
			DB:                 db,
			Mailer:             mailer.NewResendMailer(resendAPIKey, resendFrom),
			BaseURL:            baseURL,
			TwoStepAuthEnabled: twoStepAuthEnabled,
			SignupsEnabled:     signupsEnabled,
		}),
		Account: account.New(account.Deps{
			DB:                 db,
			DiscogsOAuthConfig: discogsOAuthConfig,
		}),
		Scanner: scanner.New(scanner.Deps{
			DB:            db,
			DiscogsClient: discogsClient,
		}),
	}, nil
}

func (k *Kernel) Close() {
	k.DB.Close()
}

func connectToDatabase(log logger.Logger) (*sql.DB, error) {
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

	log.Info("database connected", "path", dbPath)

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "./migrations"
	}

	if err := migrate.Up(dbPath, migrationsPath); err != nil {
		db.Close()

		return nil, fmt.Errorf("run migrations: %w", err)
	}

	log.Info("migrations applied", "path", migrationsPath)

	return db, nil
}

func envBool(name string, defaultValue bool) (bool, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}

	return value, nil
}
