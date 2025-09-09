package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
	"github.com/IkingariSolorzano/omma-be/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	type CancelRequest struct {
		CreditsToRefund *int `json:"credits_to_refund"` // Use pointer to check if value was provided
	}

	reservationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reservation ID"})
		return
	}

	var req CancelRequest
	// Bind JSON, but ignore errors if body is empty for backward compatibility
	_ = c.ShouldBindJSON(&req)

	// The service will handle the logic if CreditsToRefund is nil
	err = uc.reservationService.CancelReservation(uint(reservationID), userID.(uint), req.CreditsToRefund)
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

func (uc *UserController) UploadProfilePicture(c *gin.Context) {
	fmt.Printf("[UPLOAD] Profile picture upload handler called\n")
	userID, exists := c.Get("user_id")
	if !exists {
		fmt.Printf("[UPLOAD] Error: user_id not found in context\n")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	fmt.Printf("[UPLOAD] User ID from context: %v\n", userID)

	// Parse multipart form
	file, header, err := c.Request.FormFile("profile_picture")
	if err != nil {
		fmt.Printf("[UPLOAD] Error parsing form file: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	fmt.Printf("[UPLOAD] File received: %s, Size: %d bytes, Type: %s\n",
		header.Filename, header.Size, header.Header.Get("Content-Type"))

	// Validate file type
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		fmt.Printf("[UPLOAD] Invalid file type: %s\n", contentType)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JPEG, PNG and WebP images are allowed"})
		return
	}

	// Validate file size (max 5MB)
	if header.Size > 5*1024*1024 {
		fmt.Printf("[UPLOAD] File too large: %d bytes\n", header.Size)
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size must be less than 5MB"})
		return
	}

	// Production paths - relative to project directory (portable)
	uploadsBaseDir := "uploads"
	profilePicturesDir := filepath.Join(uploadsBaseDir, "profile_pictures")

	// Create uploads directories if they don't exist
	if err := os.MkdirAll(profilePicturesDir, 0755); err != nil {
		fmt.Printf("[UPLOAD] Error creating upload directory: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}
	fmt.Printf("[UPLOAD] Upload directory ready: %s\n", profilePicturesDir)

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		// Determine extension from content type
		switch contentType {
		case "image/jpeg", "image/jpg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		}
	}

	filename := fmt.Sprintf("%s_%s%s", uuid.New().String(), strconv.FormatUint(uint64(userID.(uint)), 10), ext)
	filePath := filepath.Join(profilePicturesDir, filename)
	fmt.Printf("[UPLOAD] Generated filename: %s\n", filename)
	fmt.Printf("[UPLOAD] Full file path: %s\n", filePath)

	// Save file to disk
	dst, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("[UPLOAD] Error creating file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		fmt.Printf("[UPLOAD] Error saving file: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	fmt.Printf("[UPLOAD] File saved successfully\n")

	// Get current user to delete old profile picture
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		fmt.Printf("[UPLOAD] Error finding user: %v\n", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	fmt.Printf("[UPLOAD] Current user profile image: %s\n", user.ProfileImage)

	// Delete old profile picture if exists
	if user.ProfileImage != "" {
		// Construct path relative to current directory
		oldImagePath := strings.TrimPrefix(user.ProfileImage, "/")
		// If it starts with "uploads/", it's already relative
		if !strings.HasPrefix(oldImagePath, "uploads/") {
			oldImagePath = filepath.Join("uploads", oldImagePath)
		}
		fmt.Printf("[UPLOAD] Attempting to delete old image: %s\n", oldImagePath)

		// Check if file exists and delete it
		if _, err := os.Stat(oldImagePath); err == nil {
			if err := os.Remove(oldImagePath); err != nil {
				fmt.Printf("[UPLOAD] Warning: Failed to delete old profile picture: %v\n", err)
				// Don't return error, just log the warning
			} else {
				fmt.Printf("[UPLOAD] Old profile picture deleted successfully\n")
			}
		} else {
			fmt.Printf("[UPLOAD] Old profile picture not found (already deleted or moved): %v\n", err)
		}
	}

	// Update user profile image path in database
	// Store relative path for serving: /uploads/profile_pictures/filename
	relativePath := "/uploads/profile_pictures/" + filename
	fmt.Printf("[UPLOAD] Updating database with path: %s\n", relativePath)

	if err := config.DB.Model(&user).Update("profile_image", relativePath).Error; err != nil {
		// If database update fails, clean up the uploaded file
		os.Remove(filePath)
		fmt.Printf("[UPLOAD] Error updating database: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	fmt.Printf("[UPLOAD] Profile picture uploaded successfully: %s\n", relativePath)
	c.JSON(http.StatusOK, gin.H{
		"message":       "Profile picture uploaded successfully",
		"profile_image": relativePath,
	})
}

func (uc *UserController) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")

	type ChangePasswordRequest struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// This assumes you have a method in your authService to check passwords
	if !services.NewAuthService().CheckPassword(req.CurrentPassword, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "La contraseña actual es incorrecta"})
		return
	}

	hashedPassword, err := services.NewAuthService().HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar la contraseña"})
		return
	}

	if err := config.DB.Model(&user).Update("password", hashedPassword).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar la contraseña"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contraseña actualizada exitosamente"})
}

func (uc *UserController) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name        string `json:"name"`
		Phone       string `json:"phone"`
		Specialty   string `json:"specialty"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update only provided fields
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Specialty != "" {
		updates["specialty"] = req.Specialty
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if len(updates) > 0 {
		if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}
	}

	// Fetch updated user
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user":    user,
	})
}
