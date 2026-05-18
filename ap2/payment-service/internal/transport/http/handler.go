package http

import (
	"errors"
	"net/http"

	"github.com/akmazhito/assignment1_2/ap2/payment-service/internal/domain"
	"github.com/akmazhito/assignment1_2/ap2/payment-service/internal/usecase"
	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	uc *usecase.PaymentUseCase
}

func NewPaymentHandler(uc *usecase.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{uc: uc}
}

func (h *PaymentHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/payments", h.ProcessPayment)
	r.GET("/payments/:order_id", h.GetPayment)
}

type processPaymentRequest struct {
	OrderID       string `json:"order_id"       binding:"required"`
	Amount        int64  `json:"amount"         binding:"required,gt=0"`
	CustomerEmail string `json:"customer_email"`
}

func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	var req processPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.uc.ProcessPayment(c.Request.Context(), req.OrderID, req.Amount, req.CustomerEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"transaction_id": payment.TransactionID,
		"status":         payment.Status,
	})
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	payment, err := h.uc.GetByOrderID(c.Request.Context(), c.Param("order_id"))
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             payment.ID,
		"order_id":       payment.OrderID,
		"transaction_id": payment.TransactionID,
		"amount":         payment.Amount,
		"status":         payment.Status,
		"created_at":     payment.CreatedAt,
	})
}
