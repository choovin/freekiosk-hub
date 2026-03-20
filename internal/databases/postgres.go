package databases

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// PostgresDB wraps the sqlx.DB with additional functionality
type PostgresDB struct {
	*sqlx.DB
}

// OpenPostgres creates a new PostgreSQL database connection
func OpenPostgres(cfg *PostgresConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	slog.Info("Connected to PostgreSQL",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
	)

	return &PostgresDB{db}, nil
}

// RunMigrations executes the migration files
func (db *PostgresDB) RunMigrations(migrations []string) error {
	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

// Close closes the database connection
func (db *PostgresDB) Close() error {
	return db.DB.Close()
}
