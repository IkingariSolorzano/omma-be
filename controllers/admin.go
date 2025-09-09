package controllers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
	"github.com/IkingariSolorzano/omma-be/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminController struct {
	authService        *services.AuthService
	creditService      *services.CreditService
	reservationService *services.ReservationService
}

// Per-lot handlers
func (ac *AdminController) ExtendCreditLot(c *gin.Context) {
	var req ExtendCreditLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ac.creditService.ExtendCreditLot(req.CreditID, req.Days); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Fecha de expiración del lote extendida"})
}

func (ac *AdminController) ReactivateCreditLot(c *gin.Context) {
	var req ReactivateCreditLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newExpiry, err := time.Parse("2006-01-02", req.NewExpiry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido. Use YYYY-MM-DD"})
		return
	}
	if err := ac.creditService.ReactivateCreditLot(req.CreditID, newExpiry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Lote reactivado"})
}

func (ac *AdminController) TransferFromCreditLot(c *gin.Context) {
	var req TransferFromLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ac.creditService.TransferFromLot(req.CreditID, req.ToUserID, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Créditos transferidos desde el lote"})
}

func (ac *AdminController) DeductFromCreditLot(c *gin.Context) {
	var req DeductFromLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ac.creditService.AdminDeductFromLot(req.CreditID, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Créditos deducidos del lote"})
}

func NewAdminController() *AdminController {
	return &AdminController{
		authService:        services.NewAuthService(),
		creditService:      services.NewCreditService(),
		reservationService: services.NewReservationService(),
	}
}

// GetUserCreditLots returns all credit lots for a specific user (admin-only)
func (ac *AdminController) GetUserCreditLots(c *gin.Context) {
	userIDParam := c.Param("id")
	uid, err := strconv.ParseUint(userIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario invalido"})
		return
	}

	credits, err := ac.creditService.GetUserCredits(uint(uid))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los créditos del usuario"})
		return
	}

	activeCredits, err := ac.creditService.GetActiveCredits(uint(uid))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los créditos activos del usuario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"credits":        credits,
		"active_credits": activeCredits,
	})
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

type ExtendExpiryRequest struct {
	UserID uint `json:"user_id" binding:"required"`
	Days   int  `json:"days" binding:"required"`
}

type ReactivateExpiredRequest struct {
	UserID    uint   `json:"user_id" binding:"required"`
	NewExpiry string `json:"new_expiry" binding:"required"` // YYYY-MM-DD
}

type TransferCreditsRequest struct {
	FromUserID uint `json:"from_user_id" binding:"required"`
	ToUserID   uint `json:"to_user_id" binding:"required"`
	Amount     int  `json:"amount" binding:"required"`
}

type DeductCreditsRequest struct {
	UserID uint `json:"user_id" binding:"required"`
	Amount int  `json:"amount" binding:"required"`
}

type ExtendCreditLotRequest struct {
	CreditID uint `json:"credit_id" binding:"required"`
	Days     int  `json:"days" binding:"required"`
}

type ReactivateCreditLotRequest struct {
	CreditID  uint   `json:"credit_id" binding:"required"`
	NewExpiry string `json:"new_expiry" binding:"required"` // YYYY-MM-DD
}

type TransferFromLotRequest struct {
	CreditID uint `json:"credit_id" binding:"required"`
	ToUserID uint `json:"to_user_id" binding:"required"`
	Amount   int  `json:"amount" binding:"required"`
}

type DeductFromLotRequest struct {
	CreditID uint `json:"credit_id" binding:"required"`
	Amount   int  `json:"amount" binding:"required"`
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

type UpdateUserRequest struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	Specialty   string `json:"specialty"`
	Description string `json:"description"`
}

type ChangePasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type CreateBusinessHourRequest struct {
	DayOfWeek int    `json:"day_of_week" binding:"gte=0,lte=6"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	IsClosed  bool   `json:"is_closed"`
}

type CreateClosedDateRequest struct {
	Date     string `json:"date" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
	IsActive bool   `json:"is_active"`
}

type CancelReservationRequest struct {
	Reason  string  `json:"reason" binding:"required"`
	Penalty float64 `json:"penalty"`
	Notes   string  `json:"notes"`
}

type UpdateReservationRequest struct {
	SpaceID   *uint   `json:"space_id"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Notes     *string `json:"notes"`
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
		"message": "Usuario creado exitosamente",
		"user":    user,
	})
}

func (ac *AdminController) GetUsers(c *gin.Context) {
	var users []models.User
	if err := config.DB.Preload("Credits").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los usuarios"})
		return
	}

	// Calculate active credits for each user
	type UserWithCredits struct {
		models.User
		ActiveCredits int `json:"active_credits"`
		TotalCredits  int `json:"total_credits"`
	}

	var usersWithCredits []UserWithCredits
	for _, user := range users {
		activeCredits, totalCredits := ac.creditService.GetUserCreditCounts(user.ID)
		userWithCredits := UserWithCredits{
			User:          user,
			ActiveCredits: activeCredits,
			TotalCredits:  totalCredits,
		}
		usersWithCredits = append(usersWithCredits, userWithCredits)
	}

	c.JSON(http.StatusOK, gin.H{"users": usersWithCredits})
}

func (ac *AdminController) AddCredits(c *gin.Context) {
	var req AddCreditsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := c.Get("user_id")
	notes := fmt.Sprintf("Créditos agregados por administrador ID: %d", adminID)

	credit, err := ac.creditService.AddCredits(req.UserID, req.Amount, "Créditos agregados por administrador", 0, notes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Record in credit history for dashboard statistics
	creditHistory := models.CreditHistory{
		UserID:      req.UserID,
		AdminID:     adminID.(uint),
		Amount:      req.Amount,
		Action:      "granted",
		Description: "Créditos agregados por administrador",
		Notes:       notes,
	}
	config.DB.Create(&creditHistory)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Creditos agregados exitosamente",
		"credit":  credit,
	})
}

func (ac *AdminController) ExtendCreditExpiry(c *gin.Context) {
	var req ExtendExpiryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ac.creditService.ExtendExpiry(req.UserID, req.Days); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Fecha de expiración extendida"})
}

func (ac *AdminController) ReactivateExpiredCredits(c *gin.Context) {
	var req ReactivateExpiredRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newExpiry, err := time.Parse("2006-01-02", req.NewExpiry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido. Use YYYY-MM-DD"})
		return
	}
	_, err = ac.creditService.ReactivateExpired(req.UserID, newExpiry)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Créditos reactivados"})
}

func (ac *AdminController) TransferCredits(c *gin.Context) {
	var req TransferCreditsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ac.creditService.TransferCredits(req.FromUserID, req.ToUserID, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Créditos transferidos"})
}

func (ac *AdminController) DeductCredits(c *gin.Context) {
	var req DeductCreditsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ac.creditService.AdminDeduct(req.UserID, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Créditos deducidos"})
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
		"message": "Espacio creado exitosamente",
		"space":   space,
	})
}

func (ac *AdminController) GetSpaces(c *gin.Context) {
	var spaces []models.Space
	if err := config.DB.Preload("Schedules").Find(&spaces).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los espacios"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"spaces": spaces})
}

func (ac *AdminController) UpdateSpace(c *gin.Context) {
	spaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de espacio invalido"})
		return
	}

	var req CreateSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var space models.Space
	if err := config.DB.First(&space, spaceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Espacio no encontrado"})
		return
	}

	// Update fields
	space.Name = req.Name
	space.Description = req.Description
	space.Capacity = req.Capacity
	space.CostCredits = req.CostCredits

	// Set default values if not provided
	if space.Capacity == 0 {
		space.Capacity = 1
	}
	if space.CostCredits == 0 {
		space.CostCredits = 6
	}

	if err := config.DB.Save(&space).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar el espacio"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Espacio actualizado exitosamente",
		"space":   space,
	})
}

func (ac *AdminController) DeleteSpace(c *gin.Context) {
	spaceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de espacio invalido"})
		return
	}

	// Check if space exists and get it for response
	var space models.Space
	if err := config.DB.First(&space, spaceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Espacio no encontrado"})
		return
	}

	// Check if space has active reservations
	var reservationCount int64
	config.DB.Model(&models.Reservation{}).Where("space_id = ? AND status != ?", spaceID, models.StatusCancelled).Count(&reservationCount)
	if reservationCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se puede eliminar el espacio porque tiene reservas activas"})
		return
	}

	if err := config.DB.Delete(&models.Space{}, spaceID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el espacio"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Espacio eliminado exitosamente"})
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
		"message":  "Horario creado exitosamente",
		"schedule": schedule,
	})
}

func (ac *AdminController) GetSchedules(c *gin.Context) {
	var schedules []models.Schedule
	query := config.DB.Preload("Space")

	// Filter by space_id if provided
	if spaceID := c.Query("space_id"); spaceID != "" {
		query = query.Where("space_id = ?", spaceID)
	}

	if err := query.Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los horarios"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

func (ac *AdminController) UpdateSchedule(c *gin.Context) {
	scheduleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de horario invalido"})
		return
	}

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var schedule models.Schedule
	if err := config.DB.First(&schedule, scheduleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Horario no encontrado"})
		return
	}

	schedule.SpaceID = req.SpaceID
	schedule.DayOfWeek = req.DayOfWeek
	schedule.StartTime = req.StartTime
	schedule.EndTime = req.EndTime

	if err := config.DB.Save(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar el horario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Horario actualizado exitosamente",
		"schedule": schedule,
	})
}

func (ac *AdminController) DeleteSchedule(c *gin.Context) {
	scheduleID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de horario invalido"})
		return
	}

	if err := config.DB.Delete(&models.Schedule{}, scheduleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el horario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Horario eliminado exitosamente"})
}

func (ac *AdminController) GetPendingReservations(c *gin.Context) {
	reservations, err := ac.reservationService.GetPendingReservations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener las reservas pendientes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reservations": reservations})
}

// GetAllReservations returns all reservations with optional filters
func (ac *AdminController) GetAllReservations(c *gin.Context) {
	// Parse query parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	status := c.Query("status")
	spaceIDStr := c.Query("space_id")

	var spaceID *uint
	if spaceIDStr != "" {
		if id, err := strconv.ParseUint(spaceIDStr, 10, 32); err == nil {
			spaceIDUint := uint(id)
			spaceID = &spaceIDUint
		}
	}

	reservations := []models.Reservation{}
	query := config.DB.Preload("User").Preload("ExternalClient").Preload("Space").Preload("CreatedByUser")

	// Apply filters
	if startDate != "" {
		if startTime, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("start_time >= ?", startTime)
		}
	}
	if endDate != "" {
		if endTime, err := time.Parse("2006-01-02", endDate); err == nil {
			endTime = endTime.Add(24 * time.Hour) // Include the entire end date
			query = query.Where("start_time < ?", endTime)
		}
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if spaceID != nil {
		query = query.Where("space_id = ?", *spaceID)
	}

	if err := query.Order("start_time DESC").Find(&reservations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener las reservas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reservations": reservations})
}

// GetReservationDetails returns detailed information about a specific reservation
func (ac *AdminController) GetReservationDetails(c *gin.Context) {
	reservationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de reserva inválido"})
		return
	}

	var reservation models.Reservation
	if err := config.DB.Preload("User").Preload("ExternalClient").Preload("Space").Preload("CreatedByUser").First(&reservation, reservationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reserva no encontrada"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reservation": reservation})
}

func (ac *AdminController) CancelReservation(c *gin.Context) {
	reservationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de la reserva invalido"})
		return
	}

	var req CancelReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := c.Get("user_id")

	err = ac.reservationService.AdminCancelReservation(uint(reservationID), adminID.(uint), req.Reason, req.Penalty, req.Notes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reserva cancelada exitosamente"})
}

func (ac *AdminController) ApproveReservation(c *gin.Context) {
	reservationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de la reserva invalido"})
		return
	}

	adminID, _ := c.Get("user_id")

	err = ac.reservationService.ApproveReservation(uint(reservationID), adminID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reserva aprobada exitosamente"})
}

func (ac *AdminController) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario invalido"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	// Update fields if provided
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Specialty != "" {
		user.Specialty = req.Specialty
	}
	if req.Description != "" {
		user.Description = req.Description
	}

	if err := config.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar el usuario"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Usuario actualizado exitosamente",
		"user":    user,
	})
}

func (ac *AdminController) ChangeUserPassword(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de usuario invalido"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	// Hash the new password
	hashedPassword, err := ac.authService.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al hashear la contraseña"})
		return
	}

	user.Password = hashedPassword
	if err := config.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar la contraseña"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contraseña cambiada exitosamente"})
}

// Business Hours Management
func (ac *AdminController) GetBusinessHours(c *gin.Context) {
	var businessHours []models.BusinessHour
	if err := config.DB.Find(&businessHours).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los horarios de negocio"})
		return
	}

	c.JSON(http.StatusOK, businessHours)
}

func (ac *AdminController) CreateBusinessHour(c *gin.Context) {
	var req CreateBusinessHourRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.IsClosed {
		if req.StartTime == "" || req.EndTime == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Start time and end time are required when not closed"})
			return
		}
	}

	var businessHour models.BusinessHour
	db := config.DB.Unscoped().Where("day_of_week = ?", req.DayOfWeek).First(&businessHour)

	if db.Error == nil {
		// Record exists, update and restore it
		updates := map[string]interface{}{
			"start_time": req.StartTime,
			"end_time":   req.EndTime,
			"is_closed":  req.IsClosed,
			"deleted_at": nil,
		}
		if err := config.DB.Model(&businessHour).Unscoped().Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar el horario de negocio"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Horario de negocio actualizado exitosamente", "business_hour": businessHour})
	} else if errors.Is(db.Error, gorm.ErrRecordNotFound) {
		// Record does not exist, create it
		newBusinessHour := models.BusinessHour{
			DayOfWeek: req.DayOfWeek,
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
			IsClosed:  req.IsClosed,
		}
		if err := config.DB.Create(&newBusinessHour).Error; err != nil {
			log.Printf("Error creating business hour: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error al crear el horario de negocio"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "Horario de negocio creado exitosamente", "business_hour": newBusinessHour})
	} else {
		// Another database error occurred
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en la base de datos al buscar el horario de negocio"})
	}
}

func (ac *AdminController) UpdateBusinessHour(c *gin.Context) {
	businessHourID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de horario de negocio invalido"})
		return
	}

	var req CreateBusinessHourRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var businessHour models.BusinessHour
	if err := config.DB.First(&businessHour, businessHourID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Horario de negocio no encontrado"})
		return
	}

	// Validate time format if not closed
	if !req.IsClosed {
		if req.StartTime == "" || req.EndTime == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Start time and end time are required when not closed"})
			return
		}
	}

	businessHour.DayOfWeek = req.DayOfWeek
	businessHour.StartTime = req.StartTime
	businessHour.EndTime = req.EndTime
	businessHour.IsClosed = req.IsClosed

	if err := config.DB.Save(&businessHour).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar el horario de negocio"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Horario de negocio actualizado exitosamente",
		"business_hour": businessHour,
	})
}

func (ac *AdminController) DeleteBusinessHour(c *gin.Context) {
	businessHourID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de horario de negocio invalido"})
		return
	}

	if err := config.DB.Delete(&models.BusinessHour{}, businessHourID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el horario de negocio"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Horario de negocio eliminado exitosamente"})
}

// Closed Dates Management
func (ac *AdminController) GetClosedDates(c *gin.Context) {
	var closedDates []models.ClosedDate
	if err := config.DB.Find(&closedDates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener las fechas cerradas"})
		return
	}

	c.JSON(http.StatusOK, closedDates)
}

// Public endpoint for closed dates (no auth required)
func GetPublicClosedDates(c *gin.Context) {
	var closedDates []models.ClosedDate
	if err := config.DB.Where("is_active = ?", true).Find(&closedDates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener las fechas cerradas"})
		return
	}

	// We need to return dates in YYYY-MM-DD format to match frontend expectations
	type PublicClosedDate struct {
		ID       uint   `json:"id"`
		Date     string `json:"date"`
		Reason   string `json:"reason"`
		IsActive bool   `json:"is_active"`
	}

	publicDates := make([]PublicClosedDate, len(closedDates))
	for i, cd := range closedDates {
		publicDates[i] = PublicClosedDate{
			ID:       cd.ID,
			Date:     cd.Date.Format("2006-01-02"),
			Reason:   cd.Reason,
			IsActive: cd.IsActive,
		}
	}

	c.JSON(http.StatusOK, publicDates)
}

func (ac *AdminController) CreateClosedDate(c *gin.Context) {
	var req CreateClosedDateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse date in local timezone to avoid timezone conversion issues
	date, err := time.ParseInLocation("2006-01-02", req.Date, time.Local)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	var closedDate models.ClosedDate
	db := config.DB.Unscoped().Where("date = ?", date).First(&closedDate)

	if db.Error == nil {
		// Record exists, update and restore it
		updates := map[string]interface{}{
			"reason":     req.Reason,
			"is_active":  req.IsActive,
			"deleted_at": nil,
		}
		if err := config.DB.Model(&closedDate).Unscoped().Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar la fecha cerrada"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Fecha cerrada actualizada exitosamente", "closed_date": closedDate})
	} else if errors.Is(db.Error, gorm.ErrRecordNotFound) {
		// Record does not exist, create it
		newClosedDate := models.ClosedDate{
			Date:     date,
			Reason:   req.Reason,
			IsActive: req.IsActive,
		}
		if err := config.DB.Create(&newClosedDate).Error; err != nil {
			log.Printf("Error creating closed date: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear la fecha cerrada"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "Fecha cerrada creada exitosamente", "closed_date": newClosedDate})
	} else {
		log.Printf("Database error: %v", db.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error de base de datos"})
	}
}

func (ac *AdminController) DeleteClosedDate(c *gin.Context) {
	closedDateID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de fecha cerrada invalido"})
		return
	}

	if err := config.DB.Delete(&models.ClosedDate{}, closedDateID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar la fecha cerrada"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fecha cerrada eliminada exitosamente"})
}

// CreateExternalReservation creates a reservation for external clients (without user accounts)
type CreateExternalReservationRequest struct {
	// Client info
	ClientName  string `json:"client_name" binding:"required"`
	ClientPhone string `json:"client_phone" binding:"required"`
	ClientEmail string `json:"client_email"`

	// Reservation details
	SpaceID   uint   `json:"space_id" binding:"required"`
	StartTime string `json:"start_time" binding:"required"` // Format: "2024-01-15T14:00:00Z"
	Duration  int    `json:"duration" binding:"required"`   // Hours
	Status    string `json:"status"`                        // "confirmed" or "pending"
	Notes     string `json:"notes"`
}

func (ac *AdminController) CreateExternalReservation(c *gin.Context) {
	var req CreateExternalReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get admin user ID from context
	adminID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return
	}

	// Parse start time
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha inválido"})
		return
	}

	// Calculate end time
	endTime := startTime.Add(time.Duration(req.Duration) * time.Hour)

	// Create or find external client
	var externalClient models.ExternalClient
	if err := config.DB.Where("phone = ?", req.ClientPhone).First(&externalClient).Error; err != nil {
		// Client doesn't exist, create new one
		externalClient = models.ExternalClient{
			Name:  req.ClientName,
			Phone: req.ClientPhone,
			Email: req.ClientEmail,
			Notes: req.Notes,
		}
		if err := config.DB.Create(&externalClient).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear cliente"})
			return
		}
	} else {
		// Update existing client info
		externalClient.Name = req.ClientName
		externalClient.Email = req.ClientEmail
		if req.Notes != "" {
			externalClient.Notes = req.Notes
		}
		config.DB.Save(&externalClient)
	}

	// Set default status
	status := req.Status
	if status == "" {
		status = "confirmed"
	}

	// Create reservation
	adminIDUint := adminID.(uint)
	reservation := models.Reservation{
		ExternalClientID: &externalClient.ID,
		SpaceID:          req.SpaceID,
		StartTime:        startTime,
		EndTime:          endTime,
		Status:           models.ReservationStatus(status),
		CreditsUsed:      0, // External clients don't use credits
		CreatedBy:        &adminIDUint,
		Notes:            req.Notes,
	}

	if err := config.DB.Create(&reservation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear la reserva"})
		return
	}

	// Load relations for response
	config.DB.Preload("ExternalClient").Preload("Space").First(&reservation, reservation.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Reserva creada exitosamente",
		"reservation": reservation,
	})
}

// UpdateReservation allows admin to update reservation details
func (ac *AdminController) UpdateReservation(c *gin.Context) {
	reservationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de reserva inválido"})
		return
	}

	var req UpdateReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Debug logging
	log.Printf("Update request for reservation %d: %+v", reservationID, req)

	// Get the existing reservation
	var reservation models.Reservation
	if err := config.DB.Preload("User").Preload("ExternalClient").Preload("Space").First(&reservation, reservationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reserva no encontrada"})
		return
	}

	// Check if reservation can be updated (not cancelled)
	if reservation.Status == models.StatusCancelled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se puede actualizar una reserva cancelada"})
		return
	}

	// Log original values
	log.Printf("Original reservation - SpaceID: %d, StartTime: %v, EndTime: %v",
		reservation.SpaceID, reservation.StartTime, reservation.EndTime)

	// Update fields if provided
	if req.SpaceID != nil {
		// Verify the space exists
		var space models.Space
		if err := config.DB.First(&space, *req.SpaceID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Espacio no encontrado"})
			return
		}
		log.Printf("Updating SpaceID from %d to %d", reservation.SpaceID, *req.SpaceID)
		reservation.SpaceID = *req.SpaceID
	}

	if req.StartTime != nil {
		startTime, err := time.Parse(time.RFC3339, *req.StartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha de inicio inválido"})
			return
		}
		reservation.StartTime = startTime
	}

	if req.EndTime != nil {
		endTime, err := time.Parse(time.RFC3339, *req.EndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de fecha de fin inválido"})
			return
		}
		reservation.EndTime = endTime
	}

	if req.Notes != nil {
		reservation.Notes = *req.Notes
	}

	// Validate that start time is before end time
	if reservation.StartTime.After(reservation.EndTime) || reservation.StartTime.Equal(reservation.EndTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La hora de inicio debe ser anterior a la hora de fin"})
		return
	}

	// Check for conflicts with other reservations (excluding current one)
	var conflictCount int64
	config.DB.Model(&models.Reservation{}).
		Where("space_id = ? AND id != ? AND status != ? AND ((start_time < ? AND end_time > ?) OR (start_time < ? AND end_time > ?))",
			reservation.SpaceID, reservation.ID, models.StatusCancelled,
			reservation.EndTime, reservation.StartTime,
			reservation.StartTime, reservation.EndTime).
		Count(&conflictCount)

	if conflictCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ya existe una reserva en ese horario para el espacio seleccionado"})
		return
	}

	// Log values before save
	log.Printf("Before save - SpaceID: %d, StartTime: %v, EndTime: %v",
		reservation.SpaceID, reservation.StartTime, reservation.EndTime)

	// Use Updates instead of Save to ensure changes persist
	updates := make(map[string]interface{})
	if req.SpaceID != nil {
		log.Printf("Adding space_id to updates: %d", *req.SpaceID)
		updates["space_id"] = *req.SpaceID
	}
	if req.StartTime != nil {
		startTime, _ := time.Parse(time.RFC3339, *req.StartTime)
		log.Printf("Adding start_time to updates: %v", startTime)
		updates["start_time"] = startTime
	}
	if req.EndTime != nil {
		endTime, _ := time.Parse(time.RFC3339, *req.EndTime)
		log.Printf("Adding end_time to updates: %v", endTime)
		updates["end_time"] = endTime
	}
	if req.Notes != nil {
		log.Printf("Adding notes to updates: %s", *req.Notes)
		updates["notes"] = *req.Notes
	}

	log.Printf("Final updates map before DB call: %+v", updates)

	// Use direct SQL update to bypass any GORM hooks that might be interfering
	result := config.DB.Model(&models.Reservation{}).Where("id = ?", reservation.ID).Updates(updates)
	if result.Error != nil {
		log.Printf("Error updating reservation: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar la reserva"})
		return
	}

	log.Printf("Rows affected: %d", result.RowsAffected)

	log.Printf("Reservation updated successfully with: %+v", updates)

	// Reload with relations for response
	var updatedReservation models.Reservation
	if err := config.DB.Preload("User").Preload("ExternalClient").Preload("Space").Preload("CreatedByUser").First(&updatedReservation, reservation.ID).Error; err != nil {
		log.Printf("Error reloading reservation: %v", err)
	}

	log.Printf("After reload - SpaceID: %d, StartTime: %v, EndTime: %v",
		updatedReservation.SpaceID, updatedReservation.StartTime, updatedReservation.EndTime)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Reserva actualizada exitosamente",
		"reservation": updatedReservation,
	})
}
