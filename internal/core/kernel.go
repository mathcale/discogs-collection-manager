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
	accountmodels "github.com/mathcale/discogs-collection-manager/internal/features/account/models"
	authmodels "github.com/mathcale/discogs-collection-manager/internal/features/auth/models"
	"github.com/mathcale/discogs-collection-manager/internal/features/landing"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/logger"
	"github.com/mathcale/discogs-collection-manager/internal/pkg/migrate"
)

type Kernel struct {
	DB           *sql.DB
	Logger       logger.Logger
	Users        authmodels.UserRepository
	Sessions     authmodels.SessionRepository
	OAuthTokens  accountmodels.OAuthTokenRepository
	OTPCodes     authmodels.OTPRepository
	ResetTokens  authmodels.PasswordResetTokenRepository
	Mailer       mailer.Mailer
	BaseURL      string
	DiscogsOAuth *oauth1.Config
	Discogs      discogs.DiscogsClient
	Landing      *landing.Module

	TwoStepAuthEnabled bool
	SignupsEnabled     bool
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

func New() (*Kernel, error) {
	twoStepAuthEnabled, err := envBool("TWO_STEP_AUTH_ENABLED", true)
	if err != nil {
		return nil, err
	}

	signupsEnabled, err := envBool("SIGNUPS_ENABLED", true)
	if err != nil {
		return nil, err
	}

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

	oauthTokens := accountmodels.NewOAuthTokenRepository(db)

	resendAPIKey := os.Getenv("RESEND_API_KEY")
	resendFrom := os.Getenv("RESEND_FROM")
	if resendFrom == "" {
		resendFrom = "noreply@example.com"
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &Kernel{
		DB:           db,
		Logger:       log,
		Users:        authmodels.NewUserRepository(db),
		Sessions:     authmodels.NewSessionRepository(db),
		OAuthTokens:  oauthTokens,
		OTPCodes:     authmodels.NewOTPRepository(db),
		ResetTokens:  authmodels.NewPasswordResetTokenRepository(db),
		Mailer:       mailer.NewResendMailer(resendAPIKey, resendFrom),
		BaseURL:      baseURL,
		DiscogsOAuth: discogsOAuth,
		Discogs:      discogs.NewHTTPDiscogsClient(discogsOAuth, oauthTokens),
		Landing:      landing.New(landing.Deps{SignupsEnabled: signupsEnabled}),

		TwoStepAuthEnabled: twoStepAuthEnabled,
		SignupsEnabled:     signupsEnabled,
	}, nil
}

func (k *Kernel) Close() {
	k.DB.Close()
}
