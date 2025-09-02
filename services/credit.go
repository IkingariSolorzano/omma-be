package services

import (
	"errors"
	"time"

	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
)

type CreditService struct{}

func NewCreditService() *CreditService {
	return &CreditService{}
}

func (s *CreditService) AddCredits(userID uint, amount int) (*models.Credit, error) {
	if amount <= 0 || amount%6 != 0 {
		return nil, errors.New("credit amount must be positive and multiple of 6")
	}

	credit := models.Credit{
		UserID:       userID,
		Amount:       amount,
		PurchaseDate: time.Now(),
		ExpiryDate:   time.Now().AddDate(0, 0, 30), // 30 days from now
		IsActive:     true,
	}

	if err := config.DB.Create(&credit).Error; err != nil {
		return nil, err
	}

	return &credit, nil
}

func (s *CreditService) GetActiveCredits(userID uint) (int, error) {
	var totalCredits int64
	
	err := config.DB.Model(&models.Credit{}).
		Where("user_id = ? AND is_active = ? AND expiry_date > ?", userID, true, time.Now()).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalCredits).Error

	if err != nil {
		return 0, err
	}

	return int(totalCredits), nil
}

func (s *CreditService) DeductCredits(userID uint, amount int) error {
	if amount <= 0 {
		return errors.New("deduction amount must be positive")
	}

	// Get active credits ordered by expiry date (FIFO)
	var credits []models.Credit
	err := config.DB.Where("user_id = ? AND is_active = ? AND expiry_date > ?", 
		userID, true, time.Now()).
		Order("expiry_date ASC").
		Find(&credits).Error

	if err != nil {
		return err
	}

	totalAvailable := 0
	for _, credit := range credits {
		totalAvailable += credit.Amount
	}

	if totalAvailable < amount {
		return errors.New("insufficient credits")
	}

	// Deduct credits starting from oldest
	remaining := amount
	for i := range credits {
		if remaining <= 0 {
			break
		}

		if credits[i].Amount <= remaining {
			// Use all credits from this record
			remaining -= credits[i].Amount
			credits[i].Amount = 0
			credits[i].IsActive = false
		} else {
			// Partial deduction
			credits[i].Amount -= remaining
			remaining = 0
		}

		config.DB.Save(&credits[i])
	}

	return nil
}

func (s *CreditService) ExpireCredits() error {
	return config.DB.Model(&models.Credit{}).
		Where("expiry_date <= ? AND is_active = ?", time.Now(), true).
		Update("is_active", false).Error
}

func (s *CreditService) GetUserCredits(userID uint) ([]models.Credit, error) {
	var credits []models.Credit
	err := config.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&credits).Error
	
	return credits, err
}
