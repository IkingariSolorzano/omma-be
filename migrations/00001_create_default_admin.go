package migrations

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	goose.AddMigration(upCreateDefaultAdmin, downCreateDefaultAdmin)
}

func upCreateDefaultAdmin(tx *sql.Tx) error {
	// Get admin credentials from environment or use defaults
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@omma.com"
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123"
	}

	// Hash the password with bcrypt cost 14 (same as in the app)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 14)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Check if admin already exists
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing admin: %w", err)
	}

	// Only create admin if none exists
	if count == 0 {
		query := `
			INSERT INTO users (email, password, name, role, is_active, created_at, updated_at)
			VALUES ($1, $2, 'Administrator', 'admin', true, NOW(), NOW())
		`
		_, err = tx.Exec(query, adminEmail, string(hashedPassword))
		if err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
	}

	return nil
}

func downCreateDefaultAdmin(tx *sql.Tx) error {
	// Get admin email from environment or use default
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@omma.com"
	}

	query := "DELETE FROM users WHERE email = $1 AND role = 'admin'"
	_, err := tx.Exec(query, adminEmail)
	if err != nil {
		return fmt.Errorf("failed to delete admin user: %w", err)
	}

	return nil
}
