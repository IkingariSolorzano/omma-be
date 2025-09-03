package services

import (
	"errors"
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
		UserID:           userID,
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
	// Check if date is a closed date
	if s.isClosedDate(startTime) {
		return true // Closed date, requires approval
	}

	// Check business hours
	if !s.isWithinBusinessHours(startTime, endTime) {
		return true // Outside business hours, requires approval
	}

	// Check space-specific schedules
	var schedules []models.Schedule
	dayOfWeek := int(startTime.Weekday())

	config.DB.Where("space_id = ? AND day_of_week = ? AND is_active = ?",
		spaceID, dayOfWeek, true).Find(&schedules)

	if len(schedules) == 0 {
		return true // No schedule defined, requires approval
	}

	startTimeStr := startTime.Format("15:04")
	endTimeStr := endTime.Format("15:04")

	for _, schedule := range schedules {
		if startTimeStr >= schedule.StartTime && endTimeStr <= schedule.EndTime {
			return false // Within allowed schedule
		}
	}

	return true // Outside allowed schedule, requires approval
}

// isClosedDate checks if the given date is marked as closed
func (s *ReservationService) isClosedDate(date time.Time) bool {
	var count int64
	config.DB.Model(&models.ClosedDate{}).
		Where("date = ? AND is_active = ?", date.Format("2006-01-02"), true).
		Count(&count)
	return count > 0
}

// isWithinBusinessHours checks if the reservation time is within business hours
func (s *ReservationService) isWithinBusinessHours(startTime, endTime time.Time) bool {
	dayOfWeek := int(startTime.Weekday())
	
	var businessHour models.BusinessHour
	err := config.DB.Where("day_of_week = ?", dayOfWeek).First(&businessHour).Error
	
	if err != nil {
		// No business hours defined for this day, requires approval
		return false
	}
	
	if businessHour.IsClosed {
		// Business is closed on this day
		return false
	}
	
	startTimeStr := startTime.Format("15:04")
	endTimeStr := endTime.Format("15:04")
	
	// Check if reservation time is within business hours
	return startTimeStr >= businessHour.StartTime && endTimeStr <= businessHour.EndTime
}

func (s *ReservationService) CancelReservation(reservationID, userID uint) error {
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

	now := time.Now()
	hoursUntilReservation := reservation.StartTime.Sub(now).Hours()

	// Start transaction
	tx := config.DB.Begin()

	// Create cancellation record
	cancellation := models.Cancellation{
		UserID:           userID,
		ReservationID:    reservationID,
		CancelledAt:      now,
		HoursBeforeStart: hoursUntilReservation,
		Reason:           "Cancelación por usuario",
	}

	// Check if cancellation is within 24 hours
	if hoursUntilReservation < 24 {
		// Apply penalty (4 credits)
		penalty := models.Penalty{
			UserID:        userID,
			ReservationID: reservationID,
			Amount:        4, // 4 credits penalty
			Status:        models.PenaltyPending,
			Reason:        "Cancelación por usuario (menos de 24 horas)",
		}

		if err := tx.Create(&penalty).Error; err != nil {
			tx.Rollback()
			return err
		}

		cancellation.PenaltyCredits = 4

		if reservation.Status == models.StatusConfirmed {
			// If already charged, refund only (credits used - penalty)
			refund := reservation.CreditsUsed - 4
			if refund < 0 { refund = 0 }
			if _, err := s.creditService.AddCredits(userID, refund); err != nil {
				tx.Rollback()
				return err
			}
			cancellation.Status = models.CancellationRefunded
			cancellation.RefundedCredits = refund
		} else {
			// If not yet charged, deduct the penalty
			if err := s.creditService.DeductCredits(userID, 4); err != nil {
				tx.Rollback()
				return err
			}
			cancellation.Status = models.CancellationPenalized
		}
	} else {
		// No penalty, refund all credits
		if _, err := s.creditService.AddCredits(userID, reservation.CreditsUsed); err != nil {
			tx.Rollback()
			return err
		}
		cancellation.Status = models.CancellationRefunded
		cancellation.RefundedCredits = reservation.CreditsUsed
	}

	// Create cancellation record
	if err := tx.Create(&cancellation).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update reservation status
	reservation.Status = models.StatusCancelled
	if err := tx.Save(&reservation).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
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

	// Deduct credits
	if err := s.creditService.DeductCredits(reservation.UserID, reservation.CreditsUsed); err != nil {
		return err
	}

	// Update reservation
	now := time.Now()
	reservation.Status = models.StatusConfirmed
	reservation.ApprovedBy = &adminID
	reservation.ApprovedAt = &now

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
	hoursUntilReservation := reservation.StartTime.Sub(now).Hours()

	tx := config.DB.Begin()

	cancellation := models.Cancellation{
		UserID:           reservation.UserID,
		ReservationID:    reservationID,
		CancelledAt:      now,
		HoursBeforeStart: hoursUntilReservation,
		Reason:           reason,
		Notes:            notes,
		CancelledBy:      &adminID,
	}

	penaltyInt := int(penalty)
	if penalty > 0 {
		penaltyRecord := models.Penalty{
			UserID:        reservation.UserID,
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
				if _, err := s.creditService.AddCredits(reservation.UserID, refund); err != nil {
					tx.Rollback()
					return err
				}
			}
			cancellation.Status = models.CancellationRefunded
			cancellation.RefundedCredits = refund
		} else {
			if err := s.creditService.DeductCredits(reservation.UserID, penaltyInt); err != nil {
				tx.Rollback()
				return err
			}
			cancellation.Status = models.CancellationPenalized
		}
	} else {
		if reservation.Status == models.StatusConfirmed {
			if _, err := s.creditService.AddCredits(reservation.UserID, reservation.CreditsUsed); err != nil {
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

	tx.Commit()
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
