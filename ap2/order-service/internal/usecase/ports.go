package usecase

import (
	"context"

	"github.com/akmazhito/order-service/internal/domain"
)

// OrderRepository is the port for order persistence.
type OrderRepository interface {
	Save(ctx context.Context, order *domain.Order) error
	FindByID(ctx context.Context, id string) (*domain.Order, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
	Update(ctx context.Context, order *domain.Order) error
}

// PaymentClient is the port for inter-service payment calls.
type PaymentClient interface {
	ProcessPayment(ctx context.Context, orderID string, amount int64) (transactionID string, status string, err error)
}

// OrderStatusSubscriber allows streaming order status updates.
type OrderStatusSubscriber interface {
	Subscribe(ctx context.Context, orderID string) (<-chan *domain.Order, error)
}
