package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/IkingariSolorzano/omma-be/controllers"
	"github.com/IkingariSolorzano/omma-be/middleware"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	// Initialize controllers
	authController := controllers.NewAuthController()
	adminController := controllers.NewAdminController()
	userController := controllers.NewUserController()

	// Public routes
	public := r.Group("/api/v1")
	{
		public.POST("/auth/login", authController.Login)
		public.POST("/auth/register", authController.Register)
		public.GET("/professionals", userController.GetProfessionalDirectory)
	}

	// Protected routes
	protected := r.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware())
	{
		// User routes
		protected.GET("/profile", userController.GetProfile)
		protected.GET("/credits", userController.GetCredits)
		protected.GET("/spaces", userController.GetSpaces)
		protected.GET("/reservations", userController.GetReservations)
		protected.POST("/reservations", userController.CreateReservation)
		protected.DELETE("/reservations/:id", userController.CancelReservation)
	}

	// Admin only routes
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.AdminOnly())
	{
		// User management
		admin.POST("/users", adminController.CreateUser)
		admin.GET("/users", adminController.GetUsers)
		
		// Credit management
		admin.POST("/credits", adminController.AddCredits)
		
		// Space management
		admin.POST("/spaces", adminController.CreateSpace)
		admin.GET("/spaces", adminController.GetSpaces)
		admin.POST("/schedules", adminController.CreateSchedule)
		
		// Reservation management
		admin.GET("/reservations/pending", adminController.GetPendingReservations)
		admin.PUT("/reservations/:id/approve", adminController.ApproveReservation)
	}

	return r
}
