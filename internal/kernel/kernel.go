package kernel

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mathcale/discogs-collection-manager/internal/migrate"
)

// Kernel holds all application dependencies and is passed to the router.
type Kernel struct {
	DB *sql.DB
}

// New initialises all dependencies: opens the database, runs pending migrations,
// and returns a ready-to-use Kernel.
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

	return &Kernel{DB: db}, nil
}

// Close releases all resources held by the kernel.
func (k *Kernel) Close() {
	k.DB.Close()
}
