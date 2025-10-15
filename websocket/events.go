package websocket

// Event types for WebSocket messages
const (
	// Reservation events
	EventReservationCreated   = "reservation:created"
	EventReservationUpdated   = "reservation:updated"
	EventReservationCancelled = "reservation:cancelled"
	EventReservationApproved  = "reservation:approved"

	// User events
	EventUserStatusChanged = "user:status_changed"

	// Space events
	EventSpaceUpdated = "space:updated"

	// General events
	EventCalendarRefresh = "calendar:refresh"
)

// ReservationEvent represents a reservation-related event
type ReservationEvent struct {
	ReservationID uint   `json:"reservation_id"`
	SpaceID       uint   `json:"space_id"`
	SpaceName     string `json:"space_name"`
	UserName      string `json:"user_name"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	Status        string `json:"status"`
	Action        string `json:"action"` // created, updated, cancelled, approved
}

// UserStatusEvent represents a user status change event
type UserStatusEvent struct {
	UserID   uint   `json:"user_id"`
	UserName string `json:"user_name"`
	IsActive bool   `json:"is_active"`
}

// SpaceEvent represents a space-related event
type SpaceEvent struct {
	SpaceID   uint   `json:"space_id"`
	SpaceName string `json:"space_name"`
	Action    string `json:"action"` // updated, deleted
}

// CalendarRefreshEvent signals that the calendar should be refreshed
type CalendarRefreshEvent struct {
	Reason string `json:"reason"`
}
