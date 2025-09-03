package services

import (
	"errors"
	"time"

	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
	"gorm.io/gorm"
)

type CreditService struct{}

func NewCreditService() *CreditService {
	return &CreditService{}
}

func (s *CreditService) AddCredits(userID uint, amount int) (*models.Credit, error) {
	if amount <= 0 {
		return nil, errors.New("El monto de créditos debe ser positivo")
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
		return errors.New("El monto de la deducción debe ser positivo")
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
		return errors.New("Créditos insuficientes")
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

func (s *CreditService) GetUserCreditCounts(userID uint) (int, int) {
	var activeCredits, totalCredits int
	
	// Count active credits
	config.DB.Model(&models.Credit{}).
		Where("user_id = ? AND is_active = ? AND amount > 0", userID, true).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&activeCredits)
	
	// Count total credits
	config.DB.Model(&models.Credit{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalCredits)
	
	return activeCredits, totalCredits
}

// ExtendExpiry extends the expiry date of all active credits for a user by the given number of days
func (s *CreditService) ExtendExpiry(userID uint, days int) error {
    if days <= 0 {
        return errors.New("Los días a extender deben ser positivos")
    }

    var credits []models.Credit
    if err := config.DB.Where("user_id = ? AND is_active = ? AND amount > 0", userID, true).
        Find(&credits).Error; err != nil {
        return err
    }
    for i := range credits {
        credits[i].ExpiryDate = credits[i].ExpiryDate.AddDate(0, 0, days)
        if err := config.DB.Save(&credits[i]).Error; err != nil {
            return err
        }
    }
    return nil
}

// ReactivateExpired reactivates all expired (inactive) credits with a new expiry date
func (s *CreditService) ReactivateExpired(userID uint, newExpiry time.Time) (int64, error) {
	tx := config.DB.Model(&models.Credit{}).
		Where("user_id = ? AND is_active = ? AND amount > 0", userID, false).
		Updates(map[string]interface{}{
			"is_active":   true,
			"expiry_date": newExpiry,
		})
	return tx.RowsAffected, tx.Error
}

// TransferCredits deducts from origin and creates a new credit lot for the destination user
func (s *CreditService) TransferCredits(fromUserID, toUserID uint, amount int) error {
    if amount <= 0 {
        return errors.New("El monto debe ser positivo")
    }
    if fromUserID == toUserID {
        return errors.New("No se puede transferir al mismo usuario")
    }

	return config.DB.Transaction(func(tx *gorm.DB) error {
		// Deduct from origin
		// Load active credits FIFO within the transaction context
		var credits []models.Credit
		if err := tx.Where("user_id = ? AND is_active = ? AND expiry_date > ?", fromUserID, true, time.Now()).
			Order("expiry_date ASC").
			Find(&credits).Error; err != nil {
			return err
		}

		totalAvailable := 0
		for _, c := range credits {
			totalAvailable += c.Amount
		}
		if totalAvailable < amount {
			return errors.New("Creditos insuficientes para transferir")
		}

		remaining := amount
		for i := range credits {
			if remaining <= 0 {
				break
			}
			if credits[i].Amount <= remaining {
				remaining -= credits[i].Amount
				credits[i].Amount = 0
				credits[i].IsActive = false
			} else {
				credits[i].Amount -= remaining
				remaining = 0
			}
			if err := tx.Save(&credits[i]).Error; err != nil {
				return err
			}
		}

		// Credit the destination user as a new lot with default 30 days expiry
		newCredit := models.Credit{
			UserID:       toUserID,
			Amount:       amount,
			PurchaseDate: time.Now(),
			ExpiryDate:   time.Now().AddDate(0, 0, 30),
			IsActive:     true,
		}
		if err := tx.Create(&newCredit).Error; err != nil {
			return err
		}
		return nil
	})
}

// AdminDeduct allows an admin to deduct credits directly from a user
func (s *CreditService) AdminDeduct(userID uint, amount int) error {
    return s.DeductCredits(userID, amount)
}

// ExtendCreditLot extends expiry for a specific credit lot
func (s *CreditService) ExtendCreditLot(creditID uint, days int) error {
    if days <= 0 {
        return errors.New("Los días a extender deben ser positivos")
    }
    var credit models.Credit
    if err := config.DB.First(&credit, creditID).Error; err != nil {
        return err
    }
    if !credit.IsActive || credit.Amount <= 0 {
        return errors.New("El lote no está activo o no tiene créditos disponibles")
    }
    credit.ExpiryDate = credit.ExpiryDate.AddDate(0, 0, days)
    return config.DB.Save(&credit).Error
}

// ReactivateCreditLot reactivates a specific expired credit lot with a new expiry date
func (s *CreditService) ReactivateCreditLot(creditID uint, newExpiry time.Time) error {
    var credit models.Credit
    if err := config.DB.First(&credit, creditID).Error; err != nil {
        return err
    }
    if credit.IsActive {
        return errors.New("El lote ya está activo")
    }
    if credit.Amount <= 0 {
        return errors.New("El lote no tiene créditos disponibles")
    }
    credit.IsActive = true
    credit.ExpiryDate = newExpiry
    return config.DB.Save(&credit).Error
}

// AdminDeductFromLot deducts credits from a specific lot
func (s *CreditService) AdminDeductFromLot(creditID uint, amount int) error {
    if amount <= 0 {
        return errors.New("El monto de la deducción debe ser positivo")
    }
    var credit models.Credit
    if err := config.DB.First(&credit, creditID).Error; err != nil {
        return err
    }
    if !credit.IsActive || credit.ExpiryDate.Before(time.Now()) {
        return errors.New("El lote no está activo o ya expiró")
    }
    if credit.Amount < amount {
        return errors.New("Créditos insuficientes en el lote")
    }
    credit.Amount -= amount
    if credit.Amount == 0 {
        credit.IsActive = false
    }
    return config.DB.Save(&credit).Error
}

// TransferFromLot transfers credits from a specific lot to another user
func (s *CreditService) TransferFromLot(creditID, toUserID uint, amount int) error {
    if amount <= 0 {
        return errors.New("El monto debe ser positivo")
    }
    return config.DB.Transaction(func(tx *gorm.DB) error {
        var credit models.Credit
        if err := tx.First(&credit, creditID).Error; err != nil {
            return err
        }
        if !credit.IsActive || credit.ExpiryDate.Before(time.Now()) {
            return errors.New("El lote no está activo o ya expiró")
        }
        if credit.Amount < amount {
            return errors.New("Créditos insuficientes en el lote")
        }
        // Deduct from lot
        credit.Amount -= amount
        if credit.Amount == 0 {
            credit.IsActive = false
        }
        if err := tx.Save(&credit).Error; err != nil {
            return err
        }
        // Create destination lot (30 days by default)
        newCredit := models.Credit{
            UserID:       toUserID,
            Amount:       amount,
            PurchaseDate: time.Now(),
            ExpiryDate:   time.Now().AddDate(0, 0, 30),
            IsActive:     true,
        }
        if err := tx.Create(&newCredit).Error; err != nil {
            return err
        }
        return nil
    })
}
