package usecase

import (
	"context"

	"github.com/akmazhito/payment-service/internal/domain"
)

// PaymentRepository is the persistence port.
type PaymentRepository interface {
	Save(ctx context.Context, p *domain.Payment) error
	FindByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}

// EventPublisher is the messaging port (A3).
type EventPublisher interface {
	PublishPaymentCompleted(ctx context.Context, evt PaymentCompletedEvent) error
}

// PaymentCompletedEvent is the event payload published after a successful payment.
type PaymentCompletedEvent struct {
	OrderID        string `json:"order_id"`
	TransactionID  string `json:"transaction_id"`
	Amount         int64  `json:"amount"`
	CustomerEmail  string `json:"customer_email"`
	Status         string `json:"status"`
	IdempotencyKey string `json:"idempotency_key"`
}
