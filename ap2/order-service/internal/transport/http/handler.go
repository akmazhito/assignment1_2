package http

import (
	"errors"
	"net/http"

	"github.com/akmazhito/order-service/internal/domain"
	"github.com/akmazhito/order-service/internal/usecase"
	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	uc *usecase.OrderUseCase
}

func NewOrderHandler(uc *usecase.OrderUseCase) *OrderHandler {
	return &OrderHandler{uc: uc}
}

func (h *OrderHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/orders", h.CreateOrder)
	r.GET("/orders/:id", h.GetOrder)
	r.PATCH("/orders/:id/cancel", h.CancelOrder)
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name"   binding:"required"`
	Amount     int64  `json:"amount"      binding:"required,gt=0"`
}

// CreateOrder POST /orders
// Reads optional Idempotency-Key header for deduplication (A1 bonus).
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	order, err := h.uc.CreateOrder(c.Request.Context(), req.CustomerID, req.ItemName, req.Amount, idempotencyKey)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateRequest) {
			c.JSON(http.StatusOK, toOrderResponse(order))
			return
		}
		if errors.Is(err, domain.ErrPaymentUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment service unavailable"})
			return
		}
		if errors.Is(err, domain.ErrInvalidAmount) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toOrderResponse(order))
}

// GetOrder GET /orders/:id
func (h *OrderHandler) GetOrder(c *gin.Context) {
	order, err := h.uc.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toOrderResponse(order))
}

// CancelOrder PATCH /orders/:id/cancel
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	order, err := h.uc.CancelOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		if errors.Is(err, domain.ErrCannotCancel) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toOrderResponse(order))
}

type orderResponse struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	ItemName   string `json:"item_name"`
	Amount     int64  `json:"amount"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

func toOrderResponse(o *domain.Order) orderResponse {
	return orderResponse{
		ID:         o.ID,
		CustomerID: o.CustomerID,
		ItemName:   o.ItemName,
		Amount:     o.Amount,
		Status:     string(o.Status),
		CreatedAt:  o.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
