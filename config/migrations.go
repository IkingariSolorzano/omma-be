package config

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/IkingariSolorzano/omma-be/migrations"
	"github.com/pressly/goose/v3"
)

// RunMigrations executes all pending database migrations
func RunMigrations(db *sql.DB) error {
	// Set the dialect for goose
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Run migrations from the migrations directory
	// Note: For Go migrations, the directory path is still needed for goose to track versions
	if err := goose.Up(db, "./migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// GetSQLDB returns a *sql.DB from the gorm.DB instance
func GetSQLDB() (*sql.DB, error) {
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}
	return sqlDB, nil
}
