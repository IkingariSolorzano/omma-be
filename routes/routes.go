package routes

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/IkingariSolorzano/omma-be/controllers"
	"github.com/IkingariSolorzano/omma-be/middleware"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200", "http://localhost:3000", "http://127.0.0.1:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Initialize controllers
	authController := controllers.NewAuthController()
	adminController := controllers.NewAdminController()
	userController := controllers.NewUserController()
	dashboardController := controllers.NewDashboardController()
	calendarController := controllers.NewCalendarController()
	paymentController := controllers.NewPaymentController()

	// Public routes
	public := r.Group("/api/v1")
	{
		public.POST("/auth/login", authController.Login)
		public.POST("/auth/register", authController.Register)
		public.GET("/professionals", userController.GetProfessionalDirectory)
		public.GET("/closed-dates", controllers.GetPublicClosedDates)
	}

	// Protected routes
	protected := r.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware())
	{
		// User routes
		protected.GET("/profile", userController.GetProfile)
		protected.GET("/credits", userController.GetCredits)
		protected.GET("/spaces", userController.GetSpaces)
		protected.GET("/schedules", adminController.GetSchedules)
		protected.GET("/reservations", userController.GetReservations)
		protected.POST("/reservations", userController.CreateReservation)
		protected.DELETE("/reservations/:id", userController.CancelReservation)
		
		// Calendar routes
		protected.GET("/calendar", calendarController.GetCalendar)
		protected.GET("/calendar/available", calendarController.GetAvailableSlots)
	}

	// Admin only routes
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AuthMiddleware())
	admin.Use(middleware.AdminOnly())
	{
		// Dashboard
		admin.GET("/dashboard/stats", dashboardController.GetStats)
		admin.GET("/dashboard/activity", dashboardController.GetRecentActivity)
		
		// User management
		admin.POST("/users", adminController.CreateUser)
		admin.GET("/users", adminController.GetUsers)
		admin.GET("/users/:id/credit-lots", adminController.GetUserCreditLots)
		admin.PUT("/users/:id", adminController.UpdateUser)
		admin.PUT("/users/:id/password", adminController.ChangeUserPassword)
		
		// Credit management
		admin.POST("/credits", adminController.AddCredits)
		admin.POST("/credits/extend", adminController.ExtendCreditExpiry)
		admin.POST("/credits/reactivate", adminController.ReactivateExpiredCredits)
		admin.POST("/credits/transfer", adminController.TransferCredits)
		admin.POST("/credits/deduct", adminController.DeductCredits)
		// Credit lot (per-lot) management
		admin.POST("/credit-lots/extend", adminController.ExtendCreditLot)
		admin.POST("/credit-lots/reactivate", adminController.ReactivateCreditLot)
		admin.POST("/credit-lots/transfer", adminController.TransferFromCreditLot)
		admin.POST("/credit-lots/deduct", adminController.DeductFromCreditLot)
		
		// Payment management
		admin.POST("/payments", paymentController.RegisterPayment)
		admin.GET("/payments", paymentController.GetPaymentHistory)
		
		// Space management
		admin.POST("/spaces", adminController.CreateSpace)
		admin.GET("/spaces", adminController.GetSpaces)
		admin.POST("/schedules", adminController.CreateSchedule)
		admin.GET("/schedules", adminController.GetSchedules)
		admin.PUT("/schedules/:id", adminController.UpdateSchedule)
		admin.DELETE("/schedules/:id", adminController.DeleteSchedule)
		
		// Reservation management
		admin.GET("/reservations/pending", adminController.GetPendingReservations)
		admin.PUT("/reservations/:id/approve", adminController.ApproveReservation)
		admin.PUT("/reservations/:id/cancel", adminController.CancelReservation)
		
		// Business Hours management
		admin.GET("/business-hours", adminController.GetBusinessHours)
		admin.POST("/business-hours", adminController.CreateBusinessHour)
		admin.PUT("/business-hours/:id", adminController.UpdateBusinessHour)
		admin.DELETE("/business-hours/:id", adminController.DeleteBusinessHour)
		
		// Closed Dates management
		admin.GET("/closed-dates", adminController.GetClosedDates)
		admin.POST("/closed-dates", adminController.CreateClosedDate)
		admin.DELETE("/closed-dates/:id", adminController.DeleteClosedDate)
	}

	return r
}
