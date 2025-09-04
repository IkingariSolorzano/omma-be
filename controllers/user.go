package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
	"github.com/IkingariSolorzano/omma-be/services"
)

type UserController struct {
	creditService      *services.CreditService
	reservationService *services.ReservationService
}

func NewUserController() *UserController {
	return &UserController{
		creditService:      services.NewCreditService(),
		reservationService: services.NewReservationService(),
	}
}

type CreateReservationRequest struct {
	SpaceID   uint      `json:"space_id" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
}

func (uc *UserController) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (uc *UserController) GetCredits(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	credits, err := uc.creditService.GetUserCredits(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los créditos"})
		return
	}

	activeCredits, err := uc.creditService.GetActiveCredits(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los créditos activos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"credits":        credits,
		"active_credits": activeCredits,
	})
}

func (uc *UserController) CreateReservation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	var req CreateReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert times to local timezone (GMT-6) if they come as UTC
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		loc = time.Local
	}
	
	// Ensure times are interpreted in local timezone
	startTime := req.StartTime.In(loc)
	endTime := req.EndTime.In(loc)
	
	// If the times came as UTC but should be local, adjust them
	if req.StartTime.Location() == time.UTC {
		// Parse as if it were local time instead of UTC
		startTime = time.Date(req.StartTime.Year(), req.StartTime.Month(), req.StartTime.Day(),
			req.StartTime.Hour(), req.StartTime.Minute(), req.StartTime.Second(), 
			req.StartTime.Nanosecond(), loc)
		endTime = time.Date(req.EndTime.Year(), req.EndTime.Month(), req.EndTime.Day(),
			req.EndTime.Hour(), req.EndTime.Minute(), req.EndTime.Second(), 
			req.EndTime.Nanosecond(), loc)
	}

	reservation, err := uc.reservationService.CreateReservation(
		userID.(uint), req.SpaceID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Reservación creada exitosamente",
		"reservation": reservation,
	})
}

func (uc *UserController) GetReservations(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	reservations, err := uc.reservationService.GetUserReservations(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener las reservaciones"})
		return
	}

	// Transform reservations to include space_name and cost_credits
	var transformedReservations []map[string]interface{}
	for _, reservation := range reservations {
		transformed := map[string]interface{}{
			"id":           reservation.ID,
			"user_id":      reservation.UserID,
			"space_id":     reservation.SpaceID,
			"space_name":   reservation.Space.Name,
			"start_time":   reservation.StartTime,
			"end_time":     reservation.EndTime,
			"status":       reservation.Status,
			"cost_credits": reservation.CreditsUsed,
			"created_at":   reservation.CreatedAt,
			"updated_at":   reservation.UpdatedAt,
		}
		transformedReservations = append(transformedReservations, transformed)
	}

	c.JSON(http.StatusOK, transformedReservations)
}

func (uc *UserController) CancelReservation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	reservationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reservation ID"})
		return
	}

	err = uc.reservationService.CancelReservation(uint(reservationID), userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reservación cancelada exitosamente"})
}

func (uc *UserController) GetSpaces(c *gin.Context) {
	var spaces []models.Space
	if err := config.DB.Where("is_active = ?", true).Preload("Schedules").Find(&spaces).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los espacios"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"spaces": spaces})
}

func (uc *UserController) GetProfessionalDirectory(c *gin.Context) {
	var users []models.User
	
	// Get users with active credits who are professionals
	subquery := config.DB.Model(&models.Credit{}).
		Select("user_id").
		Where("is_active = ? AND expiry_date > ?", true, time.Now()).
		Group("user_id")

	err := config.DB.Where("role = ? AND is_active = ? AND id IN (?)", 
		models.RoleProfessional, true, subquery).
		Select("id, name, email, phone, specialty, description, profile_image").
		Find(&users).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los profesionales"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"professionals": users})
}
