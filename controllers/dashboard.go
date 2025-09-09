package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
	"github.com/gin-gonic/gin"
)

type DashboardController struct{}

func NewDashboardController() *DashboardController {
	return &DashboardController{}
}

// Consolidated DashboardStats struct with clear naming
type DashboardStats struct {
	// Weekly Operational Stats
	WeeklyHoursPotential         float64 `json:"weekly_hours_potential"`
	WeeklyHoursReserved          float64 `json:"weekly_hours_reserved"`
	WeeklyHoursAvailable         float64 `json:"weekly_hours_available"`
	WeeklyReservationsCount      int     `json:"weekly_reservations_count"`
	WeeklyExternalReservations   int     `json:"weekly_external_reservations"`
	WeeklyCancellations          int     `json:"weekly_cancellations"`
	PendingReservationsCount     int     `json:"pending_reservations_count"`

	// Weekly Financial Stats
	WeeklyIncome                 float64 `json:"weekly_income"`
	WeeklyCreditsPurchased       int     `json:"weekly_credits_purchased"`
	WeeklyCreditsGranted         int     `json:"weekly_credits_granted"`
	WeeklyCreditsTotal           int     `json:"weekly_credits_total"`

	// General User Stats
	TotalUsersRegistered         int `json:"total_users_registered"`
	UsersWithActiveCredits       int `json:"users_with_active_credits"`
	UsersWithoutCredits          int `json:"users_without_credits"`
	UsersWithExpiringCredits     int `json:"users_with_expiring_credits"`
	UsersWithExpiredCredits      int `json:"users_with_expired_credits"`

	// General Space Stats
	TotalActiveSpaces            int `json:"total_active_spaces"`
}

func (dc *DashboardController) GetDashboardStats(c *gin.Context) {
	stats := DashboardStats{}
	now := time.Now()
	loc := now.Location()

	// --- Define Date Ranges ---
	// If today is Sunday, we look at the past week (Mon-Sun)
	var startOfWeek, endOfWeek time.Time
	if now.Weekday() == time.Sunday {
		startOfWeek = now.AddDate(0, 0, -6) // Previous Monday
		endOfWeek = now // Sunday
	} else {
		startOfWeek = now.AddDate(0, 0, -int(now.Weekday())+1) // This Monday
		endOfWeek = startOfWeek.AddDate(0, 0, 6)               // This Sunday
	}

	// Normalize time boundaries
	startOfWeek = time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, loc)
	endOfWeek = time.Date(endOfWeek.Year(), endOfWeek.Month(), endOfWeek.Day(), 23, 59, 59, 999999999, loc)
	
	// --- STATS CALCULATION ---

	// 1. Weekly Hours - Horas potenciales (suma de horas disponibles por consultorio)
	var totalHoursInWeek float64
	config.DB.Raw(`
		SELECT COALESCE(SUM(
			EXTRACT(EPOCH FROM (s.end_time::time - s.start_time::time)) / 3600
		), 0)
		FROM schedules s
		INNER JOIN spaces sp ON s.space_id = sp.id
		WHERE s.is_active = true AND sp.is_active = true
		AND s.day_of_week BETWEEN 1 AND 6
	`).Scan(&totalHoursInWeek)
	stats.WeeklyHoursPotential = totalHoursInWeek

	// Horas reservadas (suma de todas las reservas confirmadas y pendientes)
	var reservedHours float64
	config.DB.Model(&models.Reservation{}).
		Where("status IN ? AND start_time >= ? AND start_time < ?", []string{"confirmed", "pending"}, startOfWeek, endOfWeek).
		Select("COALESCE(SUM(EXTRACT(EPOCH FROM (end_time - start_time)) / 3600), 0)").
		Scan(&reservedHours)
	stats.WeeklyHoursReserved = reservedHours
	
	// Horas disponibles (potenciales - reservadas)
	stats.WeeklyHoursAvailable = stats.WeeklyHoursPotential - stats.WeeklyHoursReserved

	// 2. Weekly Reservations - Total de reservaciones en la semana
	var weeklyReservations int64
	config.DB.Model(&models.Reservation{}).
		Where("status IN ? AND start_time >= ? AND start_time < ?", []string{"confirmed", "pending"}, startOfWeek, endOfWeek).
		Count(&weeklyReservations)
	stats.WeeklyReservationsCount = int(weeklyReservations)

	// Reservaciones de usuarios externos
	var externalReservations int64
	config.DB.Model(&models.Reservation{}).
		Where("external_client_id IS NOT NULL AND start_time >= ? AND start_time < ?", startOfWeek, endOfWeek).
		Count(&externalReservations)
	stats.WeeklyExternalReservations = int(externalReservations)

	// Cancelaciones en la semana
	var weeklyCancellations int64
	config.DB.Model(&models.Reservation{}).
		Where("status = ? AND updated_at >= ? AND updated_at < ?", "cancelled", startOfWeek, endOfWeek).
		Count(&weeklyCancellations)
	stats.WeeklyCancellations = int(weeklyCancellations)

	// Reservaciones pendientes (total, no solo de la semana)
	var pendingReservations int64
	config.DB.Model(&models.Reservation{}).Where("status = ?", "pending").Count(&pendingReservations)
	stats.PendingReservationsCount = int(pendingReservations)

	// 3. Weekly Financials - Ingresos de la semana
	var weeklyIncome float64
	config.DB.Model(&models.Payment{}).
		Where("created_at >= ? AND created_at < ?", startOfWeek, endOfWeek).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&weeklyIncome)
	stats.WeeklyIncome = weeklyIncome

	// Créditos comprados (equivalente en créditos de los pagos)
	var creditsPurchased int64
	config.DB.Model(&models.Payment{}).
		Where("created_at >= ? AND created_at < ?", startOfWeek, endOfWeek).
		Select("COALESCE(SUM(credits_granted), 0)").
		Scan(&creditsPurchased)
	stats.WeeklyCreditsPurchased = int(creditsPurchased)

	// Créditos otorgados por administrador (usando tabla de historial)
	var creditsGranted int64
	config.DB.Model(&models.CreditHistory{}).
		Where("created_at >= ? AND created_at < ? AND action = ?", startOfWeek, endOfWeek, "granted").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&creditsGranted)
	stats.WeeklyCreditsGranted = int(creditsGranted)

	// Créditos totales (comprados + otorgados)
	stats.WeeklyCreditsTotal = stats.WeeklyCreditsPurchased + stats.WeeklyCreditsGranted

	// 4. General User Stats
	var totalUsers int64
	config.DB.Model(&models.User{}).Where("role = ?", models.RoleProfessional).Count(&totalUsers)
	stats.TotalUsersRegistered = int(totalUsers)

	var usersWithActiveCredits int64
	config.DB.Raw(`SELECT COUNT(DISTINCT user_id) FROM credits WHERE is_active = true AND expiry_date > ?`, now).Scan(&usersWithActiveCredits)
	stats.UsersWithActiveCredits = int(usersWithActiveCredits)

	// Usuarios con créditos por vencer (dentro de 7 días)
	var usersWithExpiringCredits int64
	config.DB.Raw(`SELECT COUNT(DISTINCT user_id) FROM credits WHERE is_active = true AND expiry_date BETWEEN ? AND ?`, now, now.AddDate(0, 0, 7)).Scan(&usersWithExpiringCredits)
	stats.UsersWithExpiringCredits = int(usersWithExpiringCredits)

	// Usuarios con créditos vencidos
	var usersWithExpiredCredits int64
	config.DB.Raw(`SELECT COUNT(DISTINCT user_id) FROM credits WHERE expiry_date <= ? AND user_id NOT IN (SELECT DISTINCT user_id FROM credits WHERE expiry_date > ?)`, now, now).Scan(&usersWithExpiredCredits)
	stats.UsersWithExpiredCredits = int(usersWithExpiredCredits)

	// Usuarios sin créditos (registrados - con créditos activos)
	stats.UsersWithoutCredits = stats.TotalUsersRegistered - stats.UsersWithActiveCredits

	// 5. General Space Stats - Total de consultorios
	var totalSpaces int64
	config.DB.Model(&models.Space{}).Where("is_active = ?", true).Count(&totalSpaces)
	stats.TotalActiveSpaces = int(totalSpaces)

	c.JSON(http.StatusOK, stats)
}

func (dc *DashboardController) GetRecentActivity(c *gin.Context) {
	var activities []RecentActivity

	// Últimas reservaciones (últimas 10)
	var reservations []struct {
		ID        uint      `json:"id"`
		UserName  string    `json:"user_name"`
		SpaceName string    `json:"space_name"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	}

	config.DB.Table("reservations r").
		Select("r.id, u.name as user_name, s.name as space_name, r.status, r.created_at").
		Joins("LEFT JOIN users u ON r.user_id = u.id").
		Joins("LEFT JOIN spaces s ON r.space_id = s.id").
		Where("r.status <> ?", "cancelled").
		Order("r.created_at DESC").
		Limit(5).
		Scan(&reservations)

	for _, res := range reservations {
		activities = append(activities, RecentActivity{
			ID:          res.ID,
			Type:        "reservation",
			Description: res.UserName + " reservó " + res.SpaceName + " (" + res.Status + ")",
			UserName:    res.UserName,
			CreatedAt:   res.CreatedAt,
		})
	}

	// Últimos créditos asignados (últimos 5)
	var credits []struct {
		ID        uint      `json:"id"`
		UserName  string    `json:"user_name"`
		Amount    int       `json:"amount"`
		CreatedAt time.Time `json:"created_at"`
	}

	config.DB.Table("credits c").
		Select("c.id, u.name as user_name, c.amount, c.created_at").
		Joins("LEFT JOIN users u ON c.user_id = u.id").
		Order("c.created_at DESC").
		Limit(5).
		Scan(&credits)

	for _, credit := range credits {
		activities = append(activities, RecentActivity{
			ID:          credit.ID,
			Type:        "credit",
			Description: fmt.Sprintf("Se asignaron %d créditos a %s", credit.Amount, credit.UserName),
			UserName:    credit.UserName,
			CreatedAt:   credit.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
	})
}

type RecentActivity struct {
	ID          uint      `json:"id"`
	Type        string    `json:"type"` // "reservation", "credit", "user"
	Description string    `json:"description"`
	UserName    string    `json:"user_name"`
	CreatedAt   time.Time `json:"created_at"`
}