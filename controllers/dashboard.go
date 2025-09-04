package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/IkingariSolorzano/omma-be/config"
	"github.com/IkingariSolorzano/omma-be/models"
)

type DashboardController struct{}

func NewDashboardController() *DashboardController {
	return &DashboardController{}
}

type DashboardStats struct {
	// Users
	UsersRegistered           int     `json:"users_registered"`
	UsersWithCredits          int     `json:"users_with_credits"`
	UsersWithExpiringCredits  int     `json:"users_with_expiring_credits"`
	UsersWithExpiredCredits   int     `json:"users_with_expired_credits"`
	UsersWithoutCredits       int     `json:"users_without_credits"`
	
	// Spaces
	TotalSpaces               int     `json:"total_spaces"`
	TotalHoursPerDay          int     `json:"total_hours_per_day"`
	TotalSpacesPerWeek        int     `json:"total_spaces_per_week"`
	
	// Reservations
	SpacesReservedToday       int     `json:"spaces_reserved_today"`
	SpacesReservedThisWeek    int     `json:"spaces_reserved_this_week"`
	SpacesAvailableToday      int     `json:"spaces_available_today"`
	SpacesAvailableThisWeek   int     `json:"spaces_available_this_week"`
	PendingReservations       int     `json:"pending_reservations"`
	
	// Credits and Payments
	CreditsThisMonth          int     `json:"credits_this_month"`
	RevenueThisMonth          float64 `json:"revenue_this_month"`
	CreditsLastMonth          int     `json:"credits_last_month"`
	RevenueLastMonth          float64 `json:"revenue_last_month"`
	
	// Cancellations
	CancellationsThisMonth    int     `json:"cancellations_this_month"`
	CancellationsLastMonth    int     `json:"cancellations_last_month"`
	PenaltyCreditsThisMonth   int     `json:"penalty_credits_this_month"`
	PenaltyCreditsLastMonth   int     `json:"penalty_credits_last_month"`
}

func (dc *DashboardController) GetStats(c *gin.Context) {
	var stats DashboardStats
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	startOfWeek := today.AddDate(0, 0, -int(today.Weekday()))
	endOfWeek := startOfWeek.Add(7 * 24 * time.Hour)
	startOfMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	startOfLastMonth := startOfMonth.AddDate(0, -1, 0)

	// Users
	var totalUsers int64
	config.DB.Model(&models.User{}).Where("role = ?", models.RoleProfessional).Count(&totalUsers)
	stats.UsersRegistered = int(totalUsers)

	// Users with active credits
	activeUsersQuery := `
		SELECT COUNT(DISTINCT u.id) 
		FROM users u 
		INNER JOIN credits c ON u.id = c.user_id 
		WHERE u.role = ? AND c.is_active = true AND c.expiry_date > ?
	`
	var activeUsers int64
	config.DB.Raw(activeUsersQuery, models.RoleProfessional, now).Scan(&activeUsers)
	stats.UsersWithCredits = int(activeUsers)

	// Users with expiring credits (next 7 days)
	nextWeek := now.AddDate(0, 0, 7)
	expiringUsersQuery := `
		SELECT COUNT(DISTINCT u.id) 
		FROM users u 
		INNER JOIN credits c ON u.id = c.user_id 
		WHERE u.role = ? AND c.is_active = true AND c.expiry_date > ? AND c.expiry_date <= ?
	`
	var expiringUsers int64
	config.DB.Raw(expiringUsersQuery, models.RoleProfessional, now, nextWeek).Scan(&expiringUsers)
	stats.UsersWithExpiringCredits = int(expiringUsers)

	// Users with expired credits
	expiredUsersQuery := `
		SELECT COUNT(DISTINCT u.id) 
		FROM users u 
		INNER JOIN credits c ON u.id = c.user_id 
		WHERE u.role = ? AND c.expiry_date <= ? AND c.created_at >= ?
	`
	var expiredUsers int64
	config.DB.Raw(expiredUsersQuery, models.RoleProfessional, now, now.AddDate(0, -3, 0)).Scan(&expiredUsers)
	stats.UsersWithExpiredCredits = int(expiredUsers)

	// Users without credits
	stats.UsersWithoutCredits = stats.UsersRegistered - stats.UsersWithCredits - stats.UsersWithExpiredCredits

	// Spaces
	var totalSpaces int64
	config.DB.Model(&models.Space{}).Where("is_active = ?", true).Count(&totalSpaces)
	stats.TotalSpaces = int(totalSpaces)

	// Calculate total hours per day (spaces * working hours)
	var totalHours float64
	config.DB.Raw(`
		SELECT COALESCE(SUM(
			EXTRACT(EPOCH FROM (end_time::time - start_time::time)) / 3600
		), 0) * COUNT(DISTINCT space_id)
		FROM schedules s
		INNER JOIN spaces sp ON s.space_id = sp.id
		WHERE s.is_active = true AND sp.is_active = true
		AND s.day_of_week = ?
	`, int(today.Weekday())).Scan(&totalHours)
	stats.TotalHoursPerDay = int(totalHours)
	stats.TotalSpacesPerWeek = stats.TotalHoursPerDay * 7

	// Reservations
	var pendingReservations int64
	config.DB.Model(&models.Reservation{}).Where("status = ?", models.StatusPending).Count(&pendingReservations)
	stats.PendingReservations = int(pendingReservations)

	// Spaces reserved today
	var reservedToday int64
	config.DB.Model(&models.Reservation{}).
		Where("start_time >= ? AND start_time < ? AND status IN (?)", 
			today, tomorrow, []models.ReservationStatus{models.StatusConfirmed, models.StatusPending}).
		Count(&reservedToday)
	stats.SpacesReservedToday = int(reservedToday)

	// Spaces reserved this week
	var reservedThisWeek int64
	config.DB.Model(&models.Reservation{}).
		Where("start_time >= ? AND start_time < ? AND status IN (?)", 
			startOfWeek, endOfWeek, []models.ReservationStatus{models.StatusConfirmed, models.StatusPending}).
		Count(&reservedThisWeek)
	stats.SpacesReservedThisWeek = int(reservedThisWeek)

	// Available spaces
	stats.SpacesAvailableToday = stats.TotalHoursPerDay - stats.SpacesReservedToday
	stats.SpacesAvailableThisWeek = stats.TotalSpacesPerWeek - stats.SpacesReservedThisWeek

	// Credits and payments this month
	var creditsThisMonth int64
	var revenueThisMonthFloat float64
	config.DB.Model(&models.Payment{}).
		Where("created_at >= ? AND created_at < ?", startOfMonth, endOfMonth).
		Select("COALESCE(SUM(credits_granted), 0)").
		Scan(&creditsThisMonth)
	config.DB.Model(&models.Payment{}).
		Where("created_at >= ? AND created_at < ?", startOfMonth, endOfMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&revenueThisMonthFloat)
	stats.CreditsThisMonth = int(creditsThisMonth)
	stats.RevenueThisMonth = revenueThisMonthFloat

	// Credits and payments last month
	var creditsLastMonth int64
	var revenueLastMonthFloat float64
	config.DB.Model(&models.Payment{}).
		Where("created_at >= ? AND created_at < ?", startOfLastMonth, startOfMonth).
		Select("COALESCE(SUM(credits_granted), 0)").
		Scan(&creditsLastMonth)
	config.DB.Model(&models.Payment{}).
		Where("created_at >= ? AND created_at < ?", startOfLastMonth, startOfMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&revenueLastMonthFloat)
	stats.CreditsLastMonth = int(creditsLastMonth)
	stats.RevenueLastMonth = revenueLastMonthFloat

	// Cancellations this month
	var cancellationsThisMonth int64
	config.DB.Model(&models.Cancellation{}).
		Where("created_at >= ? AND created_at < ?", startOfMonth, endOfMonth).
		Count(&cancellationsThisMonth)
	stats.CancellationsThisMonth = int(cancellationsThisMonth)

	// Cancellations last month
	var cancellationsLastMonth int64
	config.DB.Model(&models.Cancellation{}).
		Where("created_at >= ? AND created_at < ?", startOfLastMonth, startOfMonth).
		Count(&cancellationsLastMonth)
	stats.CancellationsLastMonth = int(cancellationsLastMonth)

	// Penalty credits this month
	var penaltyThisMonth int64
	config.DB.Model(&models.Cancellation{}).
		Where("created_at >= ? AND created_at < ?", startOfMonth, endOfMonth).
		Select("COALESCE(SUM(penalty_credits), 0)").
		Scan(&penaltyThisMonth)
	stats.PenaltyCreditsThisMonth = int(penaltyThisMonth)

	// Penalty credits last month
	var penaltyLastMonth int64
	config.DB.Model(&models.Cancellation{}).
		Where("created_at >= ? AND created_at < ?", startOfLastMonth, startOfMonth).
		Select("COALESCE(SUM(penalty_credits), 0)").
		Scan(&penaltyLastMonth)
	stats.PenaltyCreditsLastMonth = int(penaltyLastMonth)

	// Transform to match frontend expectations
	response := gin.H{
		"usuarios_registrados":                           stats.UsersRegistered,
		"usuarios_con_creditos":                         stats.UsersWithCredits,
		"usuarios_con_creditos_por_vencer":              stats.UsersWithExpiringCredits,
		"usuarios_con_creditos_vencidos":                stats.UsersWithExpiredCredits,
		"usuarios_sin_creditos":                         stats.UsersWithoutCredits,
		"total_consultorios":                            stats.TotalSpaces,
		"total_horas_del_dia":                          stats.TotalHoursPerDay,
		"total_espacios_semana":                        stats.TotalSpacesPerWeek,
		"espacios_reservados_del_dia":                  stats.SpacesReservedToday,
		"espacios_reservados_de_la_semana":             stats.SpacesReservedThisWeek,
		"espacios_disponibles_del_dia":                 stats.SpacesAvailableToday,
		"espacios_disponibles_de_la_semana":            stats.SpacesAvailableThisWeek,
		"reservaciones_pendientes":                     stats.PendingReservations,
		"creditos_comprados_este_mes":                  stats.CreditsThisMonth,
		"dinero_total_por_venta_de_creditos_este_mes":  stats.RevenueThisMonth,
		"creditos_comprados_el_mes_pasado":             stats.CreditsLastMonth,
		"dinero_total_por_venta_de_creditos_el_mes_pasado": stats.RevenueLastMonth,
		"total_cancelaciones_en_el_mes":                stats.CancellationsThisMonth,
		"total_cancelaciones_el_mes_pasado":            stats.CancellationsLastMonth,
		"creditos_de_penalizacion_este_mes":            stats.PenaltyCreditsThisMonth,
		"creditos_de_penalizacion_el_mes_pasado":       stats.PenaltyCreditsLastMonth,
	}

	c.JSON(http.StatusOK, response)
}

type RecentActivity struct {
	ID          uint      `json:"id"`
	Type        string    `json:"type"` // "reservation", "credit", "user"
	Description string    `json:"description"`
	UserName    string    `json:"user_name"`
	CreatedAt   time.Time `json:"created_at"`
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
		ID       uint      `json:"id"`
		UserName string    `json:"user_name"`
		Amount   int       `json:"amount"`
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
