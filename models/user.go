package models

import (
	"time"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleAdmin        UserRole = "admin"
	RoleProfessional UserRole = "professional"
)

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null"`
	Password  string         `json:"-" gorm:"not null"`
	Name      string         `json:"name" gorm:"not null"`
	Phone     string         `json:"phone"`
	Role      UserRole       `json:"role" gorm:"default:'professional'"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Professional profile fields
	Specialty    string `json:"specialty"`
	Description  string `json:"description"`
	ProfileImage string `json:"profile_image"`

	// Relations
	Credits      []Credit      `json:"credits,omitempty"`
	Reservations []Reservation `json:"reservations,omitempty"`
}

type Space struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	Capacity    int            `json:"capacity" gorm:"default:1"`
	CostCredits int            `json:"cost_credits" gorm:"default:6"` // Usually 6 credits (60-100 pesos)
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Reservations []Reservation `json:"reservations,omitempty"`
	Schedules    []Schedule    `json:"schedules,omitempty"`
}

type Schedule struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	SpaceID   uint           `json:"space_id" gorm:"not null"`
	Space     Space          `json:"space,omitempty"`
	DayOfWeek int            `json:"day_of_week" gorm:"not null"` // 0=Sunday, 1=Monday, etc.
	StartTime string         `json:"start_time" gorm:"not null"`  // Format: "09:00"
	EndTime   string         `json:"end_time" gorm:"not null"`    // Format: "18:00"
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type ReservationStatus string

const (
	StatusPending   ReservationStatus = "pending"
	StatusConfirmed ReservationStatus = "confirmed"
	StatusCancelled ReservationStatus = "cancelled"
	StatusCompleted ReservationStatus = "completed"
)

type Reservation struct {
	ID              uint              `json:"id" gorm:"primaryKey"`
	UserID          *uint             `json:"user_id"`                        // Nullable for external clients
	User            *User             `json:"user,omitempty"`
	ExternalClientID *uint            `json:"external_client_id"`             // For clients without accounts
	ExternalClient  *ExternalClient   `json:"external_client,omitempty"`
	SpaceID         uint              `json:"space_id" gorm:"not null"`
	Space           Space             `json:"space,omitempty"`
	StartTime       time.Time         `json:"start_time" gorm:"not null"`
	EndTime         time.Time         `json:"end_time" gorm:"not null"`
	Status          ReservationStatus `json:"status" gorm:"default:'pending'"`
	CreditsUsed     int               `json:"credits_used" gorm:"default:0"`  // 0 for external clients (cash payment)
	RequiresApproval bool             `json:"requires_approval" gorm:"default:false"`
	ApprovedBy      *uint             `json:"approved_by"`
	ApprovedAt      *time.Time        `json:"approved_at"`
	CreatedBy       *uint             `json:"created_by"`                     // Admin who created the reservation
	CreatedByUser   *User             `json:"created_by_user,omitempty" gorm:"foreignKey:CreatedBy"`      // Relation to the admin who created it
	Notes           string            `json:"notes"`                          // Additional notes from admin
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	DeletedAt       gorm.DeletedAt    `json:"-" gorm:"index"`

	// Relations
	Penalties []Penalty `json:"penalties,omitempty"`
}

// GetReservantName returns the name of the person who made the reservation
// Checks User first, then ExternalClient, then returns a default
func (r *Reservation) GetReservantName() string {
	if r.User != nil && r.User.Name != "" {
		return r.User.Name
	}
	if r.ExternalClient != nil && r.ExternalClient.Name != "" {
		return r.ExternalClient.Name
	}
	return "Cliente externo"
}

// GetReservantPhone returns the phone of the person who made the reservation
func (r *Reservation) GetReservantPhone() string {
	if r.User != nil && r.User.Phone != "" {
		return r.User.Phone
	}
	if r.ExternalClient != nil && r.ExternalClient.Phone != "" {
		return r.ExternalClient.Phone
	}
	return ""
}

type PenaltyStatus string

const (
	PenaltyPending PenaltyStatus = "pending"
	PenaltyPaid    PenaltyStatus = "paid"
)

type Penalty struct {
	ID            uint          `json:"id" gorm:"primaryKey"`
	UserID        uint          `json:"user_id" gorm:"not null"`
	User          User          `json:"user,omitempty"`
	ReservationID uint          `json:"reservation_id" gorm:"not null"`
	Reservation   Reservation   `json:"reservation,omitempty"`
	Amount        int           `json:"amount" gorm:"not null"` // Credits deducted (2-4 credits)
	Status        PenaltyStatus `json:"status" gorm:"default:'pending'"`
	Reason        string        `json:"reason"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

type Payment struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	UserID          uint           `json:"user_id" gorm:"not null"`
	User            User           `json:"user,omitempty"`
	AdminID         uint           `json:"admin_id" gorm:"not null"`
	Admin           User           `json:"admin,omitempty" gorm:"foreignKey:AdminID"`
	Amount          float64        `json:"amount" gorm:"not null"` // Money paid
	CreditsGranted  int            `json:"credits_granted" gorm:"not null"`
	CreditCost      float64        `json:"credit_cost" gorm:"not null"` // Cost per credit
	PaymentMethod   string         `json:"payment_method"` // "transfer", "cash", "card"
	Reference       string         `json:"reference"` // Transaction reference
	Notes           string         `json:"notes"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

type CancellationStatus string

const (
	CancellationProcessed CancellationStatus = "processed"
	CancellationRefunded  CancellationStatus = "refunded"
	CancellationPenalized CancellationStatus = "penalized"
)

type Cancellation struct {
	ID               uint               `json:"id" gorm:"primaryKey"`
	UserID           uint               `json:"user_id" gorm:"not null"`
	User             User               `json:"user,omitempty"`
	ReservationID    uint               `json:"reservation_id" gorm:"not null"`
	Reservation      Reservation        `json:"reservation,omitempty"`
	CancelledAt      time.Time          `json:"cancelled_at" gorm:"not null"`
	HoursBeforeStart float64            `json:"hours_before_start" gorm:"not null"`
	Status           CancellationStatus `json:"status" gorm:"default:'processed'"`
	RefundedCredits  int                `json:"refunded_credits" gorm:"default:0"`
	PenaltyCredits   int                `json:"penalty_credits" gorm:"default:0"`
	Reason           string             `json:"reason"`
	Notes            string             `json:"notes"`
	CancelledBy      *uint              `json:"cancelled_by"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
	DeletedAt        gorm.DeletedAt     `json:"-" gorm:"index"`
}

type BusinessHour struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	DayOfWeek int            `json:"day_of_week" gorm:"not null;uniqueIndex"` // 0=Sunday, 1=Monday, ..., 6=Saturday
	StartTime string         `json:"start_time"`                              // Format: "09:00"
	EndTime   string         `json:"end_time"`                                // Format: "18:00"
	IsClosed  bool           `json:"is_closed" gorm:"default:false"`          // If true, the business is closed this day
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type ClosedDate struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Date      time.Time      `json:"date" gorm:"not null;uniqueIndex"` // Specific date when business is closed
	Reason    string         `json:"reason" gorm:"not null"`           // Holiday, maintenance, etc.
	IsActive  bool           `json:"is_active" gorm:"default:true"`    // Can be deactivated without deleting
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// ExternalClient represents clients without user accounts (for admin bookings)
type ExternalClient struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"not null"`
	Phone     string         `json:"phone" gorm:"not null"`
	Email     string         `json:"email"`
	Notes     string         `json:"notes"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Reservations []Reservation `json:"reservations,omitempty"`
}
