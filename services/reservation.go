package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
)

type ReservationService struct {
	creditService *CreditService
}

func NewReservationService() *ReservationService {
	return &ReservationService{
		creditService: NewCreditService(),
	}
}

func (s *ReservationService) CreateReservation(userID, spaceID uint, startTime, endTime time.Time) (*models.Reservation, error) {
	// Get space details
	var space models.Space
	if err := config.DB.First(&space, spaceID).Error; err != nil {
		return nil, errors.New("Espacio no encontrado")
	}

	// Check if user has enough credits
	availableCredits, err := s.creditService.GetActiveCredits(userID)
	if err != nil {
		return nil, err
	}

	if availableCredits < space.CostCredits {
		return nil, errors.New("Creditos insuficientes")
	}

	// Check for conflicts
	if err := s.checkReservationConflicts(spaceID, startTime, endTime, 0); err != nil {
		return nil, err
	}

	// Check if reservation is within allowed schedule and business hours
	requiresApproval := s.requiresApproval(spaceID, startTime, endTime)

	// Calculate cost: add +1 credit surcharge for special reservations
	totalCredits := space.CostCredits
	if requiresApproval {
		totalCredits += 1 // Special reservation surcharge
	}

	reservation := models.Reservation{
		UserID:           &userID,
		SpaceID:          spaceID,
		StartTime:        startTime,
		EndTime:          endTime,
		Status:           models.StatusPending,
		CreditsUsed:      totalCredits,
		RequiresApproval: requiresApproval,
	}

	if !requiresApproval {
		reservation.Status = models.StatusConfirmed
		// Deduct credits immediately for confirmed reservations
		if err := s.creditService.DeductCredits(userID, totalCredits); err != nil {
			return nil, err
		}
	}

	if err := config.DB.Create(&reservation).Error; err != nil {
		return nil, err
	}

	return &reservation, nil
}

func (s *ReservationService) checkReservationConflicts(spaceID uint, startTime, endTime time.Time, excludeID uint) error {
	var count int64
	query := config.DB.Model(&models.Reservation{}).
		Where("space_id = ? AND status IN (?, ?) AND ((start_time <= ? AND end_time > ?) OR (start_time < ? AND end_time >= ?))",
			spaceID, models.StatusPending, models.StatusConfirmed,
			startTime, startTime, endTime, endTime)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	query.Count(&count)

	if count > 0 {
		return errors.New("Periodo ya reservado")
	}

	return nil
}

func (s *ReservationService) requiresApproval(spaceID uint, startTime, endTime time.Time) bool {
	// DEBUG: Add logging to understand timezone handling
	fmt.Printf("DEBUG requiresApproval - spaceID: %d, startTime: %s (location: %s), endTime: %s (location: %s)\n", 
		spaceID, startTime.Format("2006-01-02 15:04"), startTime.Location().String(), 
		endTime.Format("2006-01-02 15:04"), endTime.Location().String())
	
	// Convert to local timezone for time comparisons
	loc, err := time.LoadLocation("America/Mexico_City") // GMT-6
	if err != nil {
		loc = time.Local
	}
	
	localStartTime := startTime.In(loc)
	localEndTime := endTime.In(loc)
	
	// Check if date is a closed date
	if s.isClosedDate(startTime) {
		fmt.Printf("DEBUG: Closed date detected\n")
		return true // Closed date, requires approval
	}

	// Check business hours
	if !s.isWithinBusinessHours(startTime, endTime) {
		fmt.Printf("DEBUG: Outside business hours\n")
		return true // Outside business hours, requires approval
	}

	// Check space-specific schedules
	var schedules []models.Schedule
	dayOfWeek := int(localStartTime.Weekday())

	config.DB.Where("space_id = ? AND day_of_week = ? AND is_active = ?", spaceID, dayOfWeek, true).Find(&schedules)
	fmt.Printf("DEBUG: Found %d schedules for space %d on day %d\n", len(schedules), spaceID, dayOfWeek)

	if len(schedules) == 0 {
		fmt.Printf("DEBUG: No schedule defined for space\n")
		return true // No schedule defined, requires approval
	}

	// Use local times for schedule comparisons
	startTimeStr := localStartTime.Format("15:04")
	endTimeStr := localEndTime.Format("15:04")
	fmt.Printf("DEBUG: Checking time range %s-%s in local timezone\n", startTimeStr, endTimeStr)

	for i, schedule := range schedules {
		fmt.Printf("DEBUG: Schedule %d: %s-%s\n", i, schedule.StartTime, schedule.EndTime)
		if startTimeStr >= schedule.StartTime && endTimeStr <= schedule.EndTime {
			fmt.Printf("DEBUG: Time is within schedule - NO approval required\n")
			return false // Within allowed schedule
		}
	}

	fmt.Printf("DEBUG: Outside allowed schedule - approval required\n")
	return true // Outside allowed schedule, requires approval
}

// isClosedDate checks if the given date is marked as closed
func (s *ReservationService) isClosedDate(date time.Time) bool {
	// Convert to local timezone for date comparison
	loc, err := time.LoadLocation("America/Mexico_City") // GMT-6
	if err != nil {
		loc = time.Local
	}
	
	localDate := date.In(loc)
	var count int64
	config.DB.Model(&models.ClosedDate{}).
		Where("date = ? AND is_active = ?", localDate.Format("2006-01-02"), true).
		Count(&count)
	return count > 0
}

// isWithinBusinessHours checks if the reservation time is within business hours
func (s *ReservationService) isWithinBusinessHours(startTime, endTime time.Time) bool {
	// Convert to local timezone for business hours validation
	loc, err := time.LoadLocation("America/Mexico_City") // GMT-6
	if err != nil {
		// Fallback to system timezone if location loading fails
		loc = time.Local
	}
	
	localStartTime := startTime.In(loc)
	localEndTime := endTime.In(loc)
	dayOfWeek := int(localStartTime.Weekday())

	var businessHour models.BusinessHour
	err = config.DB.Where("day_of_week = ?", dayOfWeek).First(&businessHour).Error

	if err != nil {
		return false
	}

	if businessHour.IsClosed {
		return false
	}

	startTimeStr := localStartTime.Format("15:04")
	endTimeStr := localEndTime.Format("15:04")

	// Check if reservation time is within business hours
	return startTimeStr >= businessHour.StartTime && endTimeStr <= businessHour.EndTime
}

func (s *ReservationService) CancelReservation(reservationID, userID uint, creditsToRefund *int) error {
	var reservation models.Reservation
	if err := config.DB.Where("id = ? AND user_id = ?", reservationID, userID).First(&reservation).Error; err != nil {
		return errors.New("Reservación no encontrada")
	}

	if reservation.Status == models.StatusCancelled {
		return errors.New("Reservación ya cancelada")
	}

	if reservation.Status == models.StatusCompleted {
		return errors.New("No se puede cancelar una reservación completada")
	}

	// Start transaction
	tx := config.DB.Begin()

	// Use a defer function to handle rollback in case of error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Store the original status before changing it
	originalStatus := reservation.Status

	// Update reservation status to Cancelled
	reservation.Status = models.StatusCancelled
	if err := tx.Save(&reservation).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Handle credit refund logic based on the original status
	if originalStatus == models.StatusConfirmed {
		var refundAmount int
		if creditsToRefund != nil {
			// New logic: use the provided refund amount
			refundAmount = *creditsToRefund
		} else {
			// Fallback to old logic: check for penalty
			now := time.Now()
			loc, _ := time.LoadLocation("America/Mexico_City")
			hoursUntilReservation := reservation.StartTime.Sub(now.In(loc)).Hours()
			if hoursUntilReservation >= 24 {
				refundAmount = reservation.CreditsUsed
			}
		}

		if refundAmount > 0 {
			if _, err := s.creditService.AddCredits(userID, refundAmount, "Reembolso por cancelación de usuario", reservationID, ""); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Commit the transaction
	return tx.Commit().Error
}

func (s *ReservationService) ApproveReservation(reservationID, adminID uint) error {
	var reservation models.Reservation
	if err := config.DB.First(&reservation, reservationID).Error; err != nil {
		return errors.New("Reservación no encontrada")
	}

	if reservation.Status != models.StatusPending {
		return errors.New("Reservación no pendiente de aprobación")
	}

	// Check for conflicts again
	if err := s.checkReservationConflicts(reservation.SpaceID, reservation.StartTime, reservation.EndTime, reservationID); err != nil {
		return err
	}

	// Deduct credits (only for user reservations, not external clients)
	if reservation.UserID != nil {
		if err := s.creditService.DeductCredits(*reservation.UserID, reservation.CreditsUsed); err != nil {
			return err
		}
	}

	// Update reservation
	now := time.Now()
	// Convert to local timezone for consistency
	loc, err := time.LoadLocation("America/Mexico_City") // GMT-6
	if err != nil {
		loc = time.Local
	}
	localNow := now.In(loc)
	reservation.Status = models.StatusConfirmed
	reservation.ApprovedBy = &adminID
	reservation.ApprovedAt = &localNow

	return config.DB.Save(&reservation).Error
}

func (s *ReservationService) GetUserReservations(userID uint) ([]models.Reservation, error) {
	var reservations []models.Reservation
	err := config.DB.Preload("Space").
		Where("user_id = ?", userID).
		Order("start_time ASC").
		Find(&reservations).Error

	return reservations, err
}

func (s *ReservationService) AdminCancelReservation(reservationID, adminID uint, reason string, penalty float64, notes string) error {
	var reservation models.Reservation
	if err := config.DB.First(&reservation, reservationID).Error; err != nil {
		return errors.New("Reservación no encontrada")
	}

	if reservation.Status == models.StatusCancelled {
		return errors.New("Reservación ya cancelada")
	}

	if reservation.Status == models.StatusCompleted {
		return errors.New("No se puede cancelar una reservación completada")
	}

	now := time.Now()
	// Convert to local timezone for consistency
	loc, err := time.LoadLocation("America/Mexico_City") // GMT-6
	if err != nil {
		loc = time.Local
	}
	localNow := now.In(loc)
	hoursUntilReservation := reservation.StartTime.Sub(localNow).Hours()

	tx := config.DB.Begin()

	cancellation := models.Cancellation{
		UserID:           *reservation.UserID,
		ReservationID:    reservationID,
		CancelledAt:      localNow,  // Use local time
		HoursBeforeStart: hoursUntilReservation,
		Reason:           reason,
		Notes:            notes,
		CancelledBy:      &adminID,
	}

	penaltyInt := int(penalty)
	if penalty > 0 {
		penaltyRecord := models.Penalty{
			UserID:        *reservation.UserID,
			ReservationID: reservationID,
			Amount:        penaltyInt,
			Status:        models.PenaltyPending,
			Reason:        reason,
		}

		if err := tx.Create(&penaltyRecord).Error; err != nil {
			tx.Rollback()
			return err
		}

		cancellation.PenaltyCredits = penaltyInt

		if reservation.Status == models.StatusConfirmed {
			refund := reservation.CreditsUsed - penaltyInt
			if refund < 0 {
				refund = 0
			}
			if refund > 0 {
				if _, err := s.creditService.AddCredits(*reservation.UserID, refund, "Reembolso por cancelación administrativa", reservationID, notes); err != nil {
					tx.Rollback()
					return err
				}
			}
			cancellation.Status = models.CancellationRefunded
			cancellation.RefundedCredits = refund
		} else {
			if err := s.creditService.DeductCredits(*reservation.UserID, penaltyInt); err != nil {
				tx.Rollback()
				return err
			}
			cancellation.Status = models.CancellationPenalized
		}
	} else {
		if reservation.Status == models.StatusConfirmed {
			if _, err := s.creditService.AddCredits(*reservation.UserID, reservation.CreditsUsed, "Reembolso por cancelación administrativa", reservationID, notes); err != nil {
				tx.Rollback()
				return err
			}
			cancellation.Status = models.CancellationRefunded
			cancellation.RefundedCredits = reservation.CreditsUsed
		} else {
			cancellation.Status = models.CancellationProcessed
		}
	}

	if err := tx.Create(&cancellation).Error; err != nil {
		tx.Rollback()
		return err
	}

	reservation.Status = models.StatusCancelled
	if err := tx.Save(&reservation).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

func (s *ReservationService) GetPendingReservations() ([]models.Reservation, error) {
	var reservations []models.Reservation
	err := config.DB.Preload("User").Preload("Space").
		Where("status = ?", models.StatusPending).
		Order("start_time ASC").
		Find(&reservations).Error

	return reservations, err
}
