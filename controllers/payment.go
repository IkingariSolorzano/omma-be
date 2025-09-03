package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/IkingariSolorzano/omma-be/services"
)

type PaymentController struct {
	paymentService *services.PaymentService
}

func NewPaymentController() *PaymentController {
	return &PaymentController{
		paymentService: services.NewPaymentService(),
	}
}

type RegisterPaymentRequest struct {
	UserID        uint    `json:"user_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,min=0.01"`
	Credits       *int    `json:"credits,omitempty"` // opcional, ignorado por backend (se calcula desde Amount)
	PaymentMethod string  `json:"payment_method" binding:"required"`
	Reference     string  `json:"reference"`
	Notes         string  `json:"notes"`
}

func (pc *PaymentController) RegisterPayment(c *gin.Context) {
	var req RegisterPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminID, _ := c.Get("user_id")

	payment, err := pc.paymentService.RegisterPayment(
		req.UserID,
		adminID.(uint),
		req.Amount,
		req.PaymentMethod,
		req.Reference,
		req.Notes,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pago registrado exitosamente",
		"payment": payment,
	})
}

func (pc *PaymentController) GetPaymentHistory(c *gin.Context) {
	userIDStr := c.Query("user_id")
	
	if userIDStr != "" {
		// Get specific user's payment history
		userID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id"})
			return
		}

		payments, err := pc.paymentService.GetPaymentHistory(uint(userID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener el historial de pagos"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"payments": payments})
	} else {
		// Get all payments
		payments, err := pc.paymentService.GetAllPayments()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los pagos"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"payments": payments})
	}
}
