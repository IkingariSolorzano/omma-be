package models

import (
	"time"

	"gorm.io/gorm"
)

type TransactionType string

const (
	TransactionTypePurchase   TransactionType = "purchase"
	TransactionTypeRefund     TransactionType = "refund"
	TransactionTypeDeduction  TransactionType = "deduction"
	TransactionTypeCorrection TransactionType = "correction"
)

type Credit struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	UserID       uint           `json:"user_id"`
	User         User           `json:"-"`
	Amount       int            `json:"amount"`
	PurchaseDate time.Time      `json:"purchase_date"`
	ExpiryDate   time.Time      `json:"expiry_date"`
	IsActive     bool           `json:"is_active"`
}

type CreditTransaction struct {
	ID            uint            `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time       `json:"created_at"`
	UserID        uint            `json:"user_id"`
	User          User            `json:"-"`
	Amount        int             `json:"amount"`
	Type          TransactionType `json:"type"`
	Reason        string          `json:"reason"`
	Notes         string          `json:"notes"`
	ReservationID *uint           `json:"reservation_id,omitempty"`
	Reservation   *Reservation    `json:"-"`
}
