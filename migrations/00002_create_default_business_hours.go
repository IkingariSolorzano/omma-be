package migrations

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upCreateDefaultBusinessHours, downCreateDefaultBusinessHours)
}

func upCreateDefaultBusinessHours(tx *sql.Tx) error {
	// Define default business hours
	// Monday to Friday: 10:00 - 20:00
	// Saturday: 09:00 - 18:00
	// Sunday: Closed
	businessHours := []struct {
		dayOfWeek int
		startTime string
		endTime   string
		isClosed  bool
	}{
		{1, "10:00", "20:00", false}, // Monday - Open
		{2, "10:00", "20:00", false}, // Tuesday - Open
		{3, "10:00", "20:00", false}, // Wednesday - Open
		{4, "10:00", "20:00", false}, // Thursday - Open
		{5, "10:00", "20:00", false}, // Friday - Open
		{6, "09:00", "18:00", false}, // Saturday - Open
		{0, "00:00", "00:00", true},  // Sunday - Closed
	}

	// Insert business hours if they don't exist
	for _, bh := range businessHours {
		var count int
		err := tx.QueryRow("SELECT COUNT(*) FROM business_hours WHERE day_of_week = $1", bh.dayOfWeek).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check existing business hours for day %d: %w", bh.dayOfWeek, err)
		}

		if count == 0 {
			query := `
				INSERT INTO business_hours (day_of_week, start_time, end_time, is_closed, created_at, updated_at)
				VALUES ($1, $2, $3, $4, NOW(), NOW())
			`
			_, err = tx.Exec(query, bh.dayOfWeek, bh.startTime, bh.endTime, bh.isClosed)
			if err != nil {
				return fmt.Errorf("failed to create business hours for day %d: %w", bh.dayOfWeek, err)
			}
		}
	}

	return nil
}

func downCreateDefaultBusinessHours(tx *sql.Tx) error {
	// Delete all default business hours
	query := "DELETE FROM business_hours WHERE day_of_week IN (0, 1, 2, 3, 4, 5, 6)"
	_, err := tx.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to delete business hours: %w", err)
	}

	return nil
}
