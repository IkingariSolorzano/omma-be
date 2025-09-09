package models

import (
	"time"
	"gorm.io/gorm"
)

type CreditHistory struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"`
	AdminID     uint           `json:"admin_id" gorm:"not null;index"`
	Amount      int            `json:"amount" gorm:"not null"`
	Action      string         `json:"action" gorm:"not null"` // "granted", "deducted", "transferred_in", "transferred_out"
	Description string         `json:"description"`
	Notes       string         `json:"notes"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships
	User  User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Admin User `json:"admin,omitempty" gorm:"foreignKey:AdminID"`
}

func (CreditHistory) TableName() string {
	return "credit_history"
}
