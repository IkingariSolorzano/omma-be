package config

import (
	"fmt"
	"log"
	"os"

	"github.com/IkingariSolorzano/omma-be/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Error al conectar a la base de datos:", err)
	}

	DB = database

	// Auto migrate the schema
	err = DB.AutoMigrate(
		&models.User{},
		&models.Credit{},
		&models.Space{},
		&models.Schedule{},
		&models.Reservation{},
		&models.Penalty{},
		&models.Payment{},
		&models.Cancellation{},
		&models.BusinessHour{},
		&models.ClosedDate{},
		&models.ExternalClient{},
	)
	if err != nil {
		log.Fatal("Error al migrar la base de datos:", err)
	}

	// Run manual migration to make user_id nullable in reservations table
	err = migrateReservationsUserID(DB)
	if err != nil {
		log.Printf("Warning: Could not migrate reservations user_id column: %v", err)
	}

	log.Println("Base de datos conectada y migrada exitosamente")
}

// migrateReservationsUserID makes the user_id column nullable in reservations table
func migrateReservationsUserID(db *gorm.DB) error {
	// Check if the column exists and is not nullable
	var count int64
	err := db.Raw(`
		SELECT COUNT(*) 
		FROM information_schema.columns 
		WHERE table_name = 'reservations' 
		AND column_name = 'user_id' 
		AND is_nullable = 'NO'
	`).Scan(&count).Error
	
	if err != nil {
		return err
	}
	
	// If the column exists and is not nullable, make it nullable
	if count > 0 {
		err = db.Exec("ALTER TABLE reservations ALTER COLUMN user_id DROP NOT NULL").Error
		if err != nil {
			return err
		}
		log.Println("Successfully made user_id column nullable in reservations table")
	}
	
	return nil
}
