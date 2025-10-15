package config

import (
	"fmt"
	"log"
	"os"
	"time"

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

	// Configurar el pool de conexiones
	sqlDB, err := database.DB()
	if err != nil {
		log.Fatal("Error al obtener la instancia de SQL DB:", err)
	}

	// Configuración del pool de conexiones
	sqlDB.SetMaxOpenConns(25)                  // Máximo de conexiones abiertas
	sqlDB.SetMaxIdleConns(5)                   // Máximo de conexiones inactivas
	sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Tiempo de vida máximo de una conexión
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // Tiempo máximo de inactividad

	log.Println("Pool de conexiones configurado: MaxOpen=25, MaxIdle=5, MaxLifetime=5m")

	DB = database

	// Auto migrate the schema
	err = DB.AutoMigrate(
		&models.User{},
		&models.Credit{},
		&models.CreditHistory{},
		&models.Space{},
		&models.Schedule{},
		&models.Reservation{},
		&models.Penalty{},
		&models.Payment{},
		&models.Cancellation{},
		&models.BusinessHour{},
		&models.ClosedDate{},
		&models.CreditTransaction{},
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
