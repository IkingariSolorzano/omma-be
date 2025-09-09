package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
)

type CalendarController struct{}

func NewCalendarController() *CalendarController {
	return &CalendarController{}
}

type CalendarReservation struct {
	ID        uint      `json:"id"`
	SpaceID   uint      `json:"space_id"`
	SpaceName string    `json:"space_name"`
	UserID    uint      `json:"user_id"`
	UserName  string    `json:"user_name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

type CalendarResponse struct {
	Reservations []CalendarReservation `json:"reservations"`
	Period       string                `json:"period"`
	StartDate    time.Time             `json:"start_date"`
	EndDate      time.Time             `json:"end_date"`
	SpaceIDs     []uint                `json:"space_ids,omitempty"`
}

func (cc *CalendarController) GetCalendar(c *gin.Context) {
	// Parse query parameters
	periodType := c.DefaultQuery("period", "week") // week, month, day, custom
	startDateStr := c.Query("start_date")          // YYYY-MM-DD
	endDateStr := c.Query("end_date")              // YYYY-MM-DD
	spaceIDsStr := c.Query("space_ids")            // comma separated: 1,2,3

	var startDate, endDate time.Time
	var err error

	// Load local timezone for consistent handling
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		loc = time.Local
	}

	// Parse dates based on period type
	switch periodType {
	case "day":
		if startDateStr != "" {
			startDate, err = time.ParseInLocation("2006-01-02", startDateStr, loc)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Formato inválido de start_date. Use YYYY-MM-DD"})
				return
			}
		} else {
			startDate = time.Now().In(loc).Truncate(24 * time.Hour)
		}
		endDate = startDate.Add(24 * time.Hour)

	case "week":
		if startDateStr != "" {
			startDate, err = time.ParseInLocation("2006-01-02", startDateStr, loc)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Formato inválido de start_date. Use YYYY-MM-DD"})
				return
			}
		} else {
			now := time.Now().In(loc)
			startDate = now.AddDate(0, 0, -int(now.Weekday())).Truncate(24 * time.Hour)
		}
		endDate = startDate.Add(7 * 24 * time.Hour)

	case "month":
		if startDateStr != "" {
			startDate, err = time.ParseInLocation("2006-01-02", startDateStr, loc)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Formato inválido de start_date. Use YYYY-MM-DD"})
				return
			}
			startDate = time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, loc)
		} else {
			now := time.Now().In(loc)
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		}
		endDate = startDate.AddDate(0, 1, 0)

	case "custom":
		if startDateStr == "" || endDateStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_date y end_date son requeridos para el periodo personalizado"})
			return
		}
		// Parse dates in Mexico timezone to avoid UTC conversion issues
		startDate, err = time.ParseInLocation("2006-01-02", startDateStr, loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato inválido de start_date. Use YYYY-MM-DD"})
			return
		}
		endDate, err = time.ParseInLocation("2006-01-02", endDateStr, loc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Formato inválido de end_date. Use YYYY-MM-DD"})
			return
		}
		endDate = endDate.Add(24 * time.Hour) // Include end date

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Periodo inválido. Use: day, week, month, custom"})
		return
	}

	// Ensure dates are in local timezone for consistent querying
	startDate = startDate.In(loc)
	endDate = endDate.In(loc)

	// Debug logging for calendar queries
	fmt.Printf("[CALENDAR] Period: %s, StartDate: %s, EndDate: %s\n", periodType, startDate.Format("2006-01-02 15:04:05"), endDate.Format("2006-01-02 15:04:05"))

	// Build query
	query := config.DB.Table("reservations r").
		Select("r.id, r.space_id, s.name as space_name, r.user_id, u.name as user_name, r.start_time, r.end_time, r.status").
		Joins("LEFT JOIN spaces s ON r.space_id = s.id").
		Joins("LEFT JOIN users u ON r.user_id = u.id").
		Where("r.start_time >= ? AND r.start_time < ?", startDate, endDate).
		Where("r.status IN (?)", []models.ReservationStatus{models.StatusConfirmed, models.StatusPending})

	// Filter by space IDs if provided
	var spaceIDs []uint
	if spaceIDsStr != "" {
		spaceIDStrs := parseCommaSeparated(spaceIDsStr)
		for _, idStr := range spaceIDStrs {
			if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
				spaceIDs = append(spaceIDs, uint(id))
			}
		}
		if len(spaceIDs) > 0 {
			query = query.Where("r.space_id IN (?)", spaceIDs)
		}
	}

	var reservations []CalendarReservation
	if err := query.Order("r.start_time ASC").Find(&reservations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los datos del calendario"})
		return
	}

	// Debug logging for reservations found
	fmt.Printf("[CALENDAR] Found %d reservations:\n", len(reservations))
	for _, res := range reservations {
		fmt.Printf("  - ID: %d, Space: %s, Start: %s, Status: %s\n", 
			res.ID, res.SpaceName, res.StartTime.Format("2006-01-02 15:04:05"), res.Status)
	}

	response := CalendarResponse{
		Reservations: reservations,
		Period:       periodType,
		StartDate:    startDate,
		EndDate:      endDate,
		SpaceIDs:     spaceIDs,
	}

	c.JSON(http.StatusOK, response)
}

func (cc *CalendarController) GetAvailableSlots(c *gin.Context) {
	dateStr := c.Query("date")        // YYYY-MM-DD
	spaceIDStr := c.Query("space_id") // optional

	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetro date requerido (YYYY-MM-DD)"})
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato inválido de fecha. Use YYYY-MM-DD"})
		return
	}

	// Load local timezone for consistent handling
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		loc = time.Local
	}

	// Ensure date is in local timezone
	date = date.In(loc)

	dayOfWeek := int(date.Weekday())
	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Get schedules for the day
	scheduleQuery := config.DB.Where("day_of_week = ? AND is_active = ?", dayOfWeek, true)
	if spaceIDStr != "" {
		if spaceID, err := strconv.ParseUint(spaceIDStr, 10, 32); err == nil {
			scheduleQuery = scheduleQuery.Where("space_id = ?", uint(spaceID))
		}
	}

	var schedules []models.Schedule
	if err := scheduleQuery.Preload("Space").Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los horarios"})
		return
	}

	// Get existing reservations for the day
	reservationQuery := config.DB.Where("start_time >= ? AND start_time < ? AND status IN (?)",
		startOfDay, endOfDay, []models.ReservationStatus{models.StatusConfirmed, models.StatusPending})
	if spaceIDStr != "" {
		if spaceID, err := strconv.ParseUint(spaceIDStr, 10, 32); err == nil {
			reservationQuery = reservationQuery.Where("space_id = ?", uint(spaceID))
		}
	}

	var reservations []models.Reservation
	if err := reservationQuery.Find(&reservations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener las reservas"})
		return
	}

	// Calculate available slots
	type AvailableSlot struct {
		SpaceID   uint      `json:"space_id"`
		SpaceName string    `json:"space_name"`
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		Available bool      `json:"available"`
	}

	var slots []AvailableSlot

	for _, schedule := range schedules {
		// Parse schedule times
		startTime, _ := time.Parse("15:04", schedule.StartTime)
		endTime, _ := time.Parse("15:04", schedule.EndTime)

		// Create hourly slots
		current := time.Date(date.Year(), date.Month(), date.Day(), startTime.Hour(), startTime.Minute(), 0, 0, date.Location())
		scheduleEnd := time.Date(date.Year(), date.Month(), date.Day(), endTime.Hour(), endTime.Minute(), 0, 0, date.Location())

		for current.Before(scheduleEnd) {
			slotEnd := current.Add(time.Hour)
			if slotEnd.After(scheduleEnd) {
				break
			}

			// Check if slot is available
			available := true
			for _, res := range reservations {
				if res.SpaceID == schedule.SpaceID &&
					((current.Before(res.EndTime) && slotEnd.After(res.StartTime))) {
					available = false
					break
				}
			}

			slots = append(slots, AvailableSlot{
				SpaceID:   schedule.SpaceID,
				SpaceName: schedule.Space.Name,
				StartTime: current,
				EndTime:   slotEnd,
				Available: available,
			})

			current = slotEnd
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"date":  date,
		"slots": slots,
	})
}

func parseCommaSeparated(str string) []string {
	if str == "" {
		return []string{}
	}
	result := []string{}
	for _, s := range splitByComma(str) {
		if trimmed := trimSpace(s); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitByComma(str string) []string {
	result := []string{}
	current := ""
	for _, char := range str {
		if char == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimSpace(str string) string {
	start := 0
	end := len(str)
	for start < end && str[start] == ' ' {
		start++
	}
	for end > start && str[end-1] == ' ' {
		end--
	}
	return str[start:end]
}
