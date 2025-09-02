package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/routes"
	"github.com/IkingariSolorzano/omma-be/services"
	"github.com/IkingariSolorzano/omma-be/models"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Connect to database
	config.ConnectDatabase()

	// Create default admin user if it doesn't exist
	createDefaultAdmin()

	// Setup routes
	r := routes.SetupRoutes()

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func createDefaultAdmin() {
	var count int64
	config.DB.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&count)
	
	if count == 0 {
		authService := services.NewAuthService()
		adminEmail := os.Getenv("ADMIN_EMAIL")
		adminPassword := os.Getenv("ADMIN_PASSWORD")
		
		if adminEmail == "" {
			adminEmail = "admin@omma.com"
		}
		if adminPassword == "" {
			adminPassword = "admin123"
		}

		_, err := authService.CreateUser(adminEmail, adminPassword, "Administrator", models.RoleAdmin)
		if err != nil {
			log.Printf("Failed to create default admin: %v", err)
		} else {
			log.Printf("Default admin created with email: %s", adminEmail)
		}
	}
}
