package app

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/dghubble/oauth1"
	_ "github.com/mattn/go-sqlite3"

	"github.com/mathcale/discogs-collection-manager/internal/app/clients/discogs"
	"github.com/mathcale/discogs-collection-manager/internal/app/clients/mailer"
	"github.com/mathcale/discogs-collection-manager/internal/models"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/logger"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
)

type App struct {
	DB           *sql.DB
	Logger       logger.Logger
	Users        models.UserRepository
	Sessions     models.SessionRepository
	OAuthTokens  models.OAuthTokenRepository
	OTPCodes     models.OTPRepository
	ResetTokens  models.PasswordResetTokenRepository
	Mailer       mailer.Mailer
	BaseURL      string
	DiscogsOAuth *oauth1.Config
	Discogs      discogs.DiscogsClient
}

func New() (*App, error) {
	env := os.Getenv("ENVIRONMENT")
	log := logger.New(env)

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

	oauthTokens := models.NewSQLiteOAuthTokenRepository(db)

	resendAPIKey := os.Getenv("RESEND_API_KEY")
	resendFrom := os.Getenv("RESEND_FROM")
	if resendFrom == "" {
		resendFrom = "noreply@example.com"
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &App{
		DB:           db,
		Logger:       log,
		Users:        models.NewSQLiteUserRepository(db),
		Sessions:     models.NewSQLiteSessionRepository(db),
		OAuthTokens:  oauthTokens,
		OTPCodes:     models.NewSQLiteOTPRepository(db),
		ResetTokens:  models.NewSQLitePasswordResetTokenRepository(db),
		Mailer:       mailer.NewResendMailer(resendAPIKey, resendFrom),
		BaseURL:      baseURL,
		DiscogsOAuth: discogsOAuth,
		Discogs:      discogs.NewHTTPDiscogsClient(discogsOAuth, oauthTokens),
	}, nil
}

func (a *App) Close() {
	a.DB.Close()
}
