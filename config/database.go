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
	)
	if err != nil {
		log.Fatal("Error al migrar la base de datos:", err)
	}

	log.Println("Base de datos conectada y migrada exitosamente")
}
