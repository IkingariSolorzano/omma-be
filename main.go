package main

import (
	"log"
	"os"

	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/routes"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Connect to database
	config.ConnectDatabase()

	// Run database migrations
	sqlDB, err := config.GetSQLDB()
	if err != nil {
		log.Fatal("Error al obtener conexión SQL:", err)
	}

	if err := config.RunMigrations(sqlDB); err != nil {
		log.Fatal("Error al ejecutar migraciones:", err)
	}

	// Initialize WebSocket hub
	config.InitializeWebSocketHub()
	log.Println("WebSocket hub started")

	// Setup routes with WebSocket hub
	r := routes.SetupRoutes(config.WSHub)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Error al iniciar el servidor:", err)
	}
}
