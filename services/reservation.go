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
		return nil, errors.New("space not found")
	}

	// Check if user has enough credits
	availableCredits, err := s.creditService.GetActiveCredits(userID)
	if err != nil {
		return nil, err
	}

	if availableCredits < space.CostCredits {
		return nil, errors.New("insufficient credits")
	}

	// Check for conflicts
	if err := s.checkReservationConflicts(spaceID, startTime, endTime, 0); err != nil {
		return nil, err
	}

	// Check if reservation is within allowed schedule
	requiresApproval := s.requiresApproval(spaceID, startTime, endTime)

	reservation := models.Reservation{
		UserID:           userID,
		SpaceID:          spaceID,
		StartTime:        startTime,
		EndTime:          endTime,
		Status:           models.StatusPending,
		CreditsUsed:      space.CostCredits,
		RequiresApproval: requiresApproval,
	}

	if !requiresApproval {
		reservation.Status = models.StatusConfirmed
		// Deduct credits immediately for confirmed reservations
		if err := s.creditService.DeductCredits(userID, space.CostCredits); err != nil {
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
		return errors.New("time slot already reserved")
	}

	return nil
}

func (s *ReservationService) requiresApproval(spaceID uint, startTime, endTime time.Time) bool {
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

func (s *ReservationService) CancelReservation(reservationID, userID uint) error {
	var reservation models.Reservation
	if err := config.DB.Where("id = ? AND user_id = ?", reservationID, userID).First(&reservation).Error; err != nil {
		return errors.New("reservation not found")
	}

	if reservation.Status == models.StatusCancelled {
		return errors.New("reservation already cancelled")
	}

	if reservation.Status == models.StatusCompleted {
		return errors.New("cannot cancel completed reservation")
	}

	now := time.Now()
	hoursUntilReservation := reservation.StartTime.Sub(now).Hours()

	// Check if cancellation is within 24 hours
	if hoursUntilReservation < 24 {
		// Apply penalty (2-4 credits, let's use 2 for now)
		penalty := models.Penalty{
			UserID:        userID,
			ReservationID: reservationID,
			Amount:        2, // 2 credits penalty
			Status:        models.PenaltyPending,
			Reason:        "Late cancellation (less than 24 hours)",
		}

		if err := config.DB.Create(&penalty).Error; err != nil {
			return err
		}

		// Deduct penalty credits if user has confirmed reservation
		if reservation.Status == models.StatusConfirmed {
			s.creditService.DeductCredits(userID, 2)
		}
	} else {
		// No penalty, refund credits if they were deducted
		if reservation.Status == models.StatusConfirmed {
			s.creditService.AddCredits(userID, reservation.CreditsUsed)
		}
	}

	// Update reservation status
	reservation.Status = models.StatusCancelled
	return config.DB.Save(&reservation).Error
}

func (s *ReservationService) ApproveReservation(reservationID, adminID uint) error {
	var reservation models.Reservation
	if err := config.DB.First(&reservation, reservationID).Error; err != nil {
		return errors.New("reservation not found")
	}

	if reservation.Status != models.StatusPending {
		return errors.New("reservation is not pending approval")
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
		Order("start_time DESC").
		Find(&reservations).Error

	return reservations, err
}

func (s *ReservationService) GetPendingReservations() ([]models.Reservation, error) {
	var reservations []models.Reservation
	err := config.DB.Preload("User").Preload("Space").
		Where("status = ?", models.StatusPending).
		Order("created_at ASC").
		Find(&reservations).Error

	return reservations, err
}
