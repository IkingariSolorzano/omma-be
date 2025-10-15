package services

import (
	"errors"
	"time"

	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
)

type PaymentService struct {
	creditService *CreditService
}

func NewPaymentService() *PaymentService {
	return &PaymentService{
		creditService: NewCreditService(),
	}
}

func (s *PaymentService) RegisterPayment(userID, adminID uint, amount float64, paymentMethod, reference, notes string) (*models.Payment, error) {
	if amount <= 0 {
		return nil, errors.New("El monto debe ser positivo")
	}

	// 1 crédito = 10 pesos. Exigir múltiplos de 10 para evitar fracciones de crédito
	if int(amount)%10 != 0 {
		return nil, errors.New("El monto debe ser múltiplo de 10 (1 crédito = 10 pesos)")
	}

	// Calcular créditos desde el monto
	credits := int(amount) / 10
	creditCost := 10.0

	// Start transaction
	tx := config.DB.Begin()

	// Create payment record
	payment := models.Payment{
		UserID:         userID,
		AdminID:        adminID,
		Amount:         amount,
		CreditsGranted: credits,
		CreditCost:     creditCost,
		PaymentMethod:  paymentMethod,
		Reference:      reference,
		Notes:          notes,
	}

	if err := tx.Create(&payment).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Add credits to user
	credit := models.Credit{
		UserID:       userID,
		Amount:       credits,
		PurchaseDate: time.Now(),
		ExpiryDate:   time.Now().AddDate(0, 0, 30), // 30 days
		IsActive:     true,
	}

	if err := tx.Create(&credit).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (s *PaymentService) GetPaymentHistory(userID uint) ([]models.Payment, error) {
	var payments []models.Payment
	err := config.DB.Preload("Admin").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&payments).Error
	
	return payments, err
}

func (s *PaymentService) GetAllPayments() ([]models.Payment, error) {
	var payments []models.Payment
	err := config.DB.Preload("User").Preload("Admin").
		Order("created_at DESC").
		Find(&payments).Error
	
	return payments, err
}
