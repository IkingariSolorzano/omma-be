package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
	"github.com/IkingariSolorzano/omma-be/services"
)

type AdminController struct {
	authService        *services.AuthService
	creditService      *services.CreditService
	reservationService *services.ReservationService
}

func NewAdminController() *AdminController {
	return &AdminController{
		authService:        services.NewAuthService(),
		creditService:      services.NewCreditService(),
		reservationService: services.NewReservationService(),
	}
}

type CreateUserRequest struct {
	Email       string          `json:"email" binding:"required,email"`
	Password    string          `json:"password" binding:"required,min=6"`
	Name        string          `json:"name" binding:"required"`
	Phone       string          `json:"phone"`
	Role        models.UserRole `json:"role" binding:"required"`
	Specialty   string          `json:"specialty"`
	Description string          `json:"description"`
}

type AddCreditsRequest struct {
	UserID uint `json:"user_id" binding:"required"`
	Amount int  `json:"amount" binding:"required"`
}

type CreateSpaceRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Capacity    int    `json:"capacity"`
	CostCredits int    `json:"cost_credits"`
}

type CreateScheduleRequest struct {
	SpaceID   uint   `json:"space_id" binding:"required"`
	DayOfWeek int    `json:"day_of_week" binding:"required,min=0,max=6"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}

func (ac *AdminController) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ac.authService.CreateUser(req.Email, req.Password, req.Name, req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update additional fields
	user.Phone = req.Phone
	user.Specialty = req.Specialty
	user.Description = req.Description
	config.DB.Save(user)

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user":    user,
	})
}

func (ac *AdminController) GetUsers(c *gin.Context) {
	var users []models.User
	if err := config.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (ac *AdminController) AddCredits(c *gin.Context) {
	var req AddCreditsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	credit, err := ac.creditService.AddCredits(req.UserID, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Credits added successfully",
		"credit":  credit,
	})
}

func (ac *AdminController) CreateSpace(c *gin.Context) {
	var req CreateSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	space := models.Space{
		Name:        req.Name,
		Description: req.Description,
		Capacity:    req.Capacity,
		CostCredits: req.CostCredits,
		IsActive:    true,
	}

	if space.Capacity == 0 {
		space.Capacity = 1
	}
	if space.CostCredits == 0 {
		space.CostCredits = 6
	}

	if err := config.DB.Create(&space).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Space created successfully",
		"space":   space,
	})
}

func (ac *AdminController) GetSpaces(c *gin.Context) {
	var spaces []models.Space
	if err := config.DB.Preload("Schedules").Find(&spaces).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch spaces"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"spaces": spaces})
}

func (ac *AdminController) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schedule := models.Schedule{
		SpaceID:   req.SpaceID,
		DayOfWeek: req.DayOfWeek,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		IsActive:  true,
	}

	if err := config.DB.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Schedule created successfully",
		"schedule": schedule,
	})
}

func (ac *AdminController) GetPendingReservations(c *gin.Context) {
	reservations, err := ac.reservationService.GetPendingReservations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending reservations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reservations": reservations})
}

func (ac *AdminController) ApproveReservation(c *gin.Context) {
	reservationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reservation ID"})
		return
	}

	adminID, _ := c.Get("user_id")
	
	err = ac.reservationService.ApproveReservation(uint(reservationID), adminID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reservation approved successfully"})
}
