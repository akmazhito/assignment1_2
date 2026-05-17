package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/akmazhito/assignment1_2/ap2/order-service/internal/domain"
	"github.com/google/uuid"
)

type OrderUseCase struct {
	repo    OrderRepository
	payment PaymentClient
}

func NewOrderUseCase(repo OrderRepository, payment PaymentClient) *OrderUseCase {
	return &OrderUseCase{repo: repo, payment: payment}
}

// CreateOrder creates a new order and synchronously authorizes payment via gRPC.
// Idempotency: if an order with the same key already exists, it is returned as-is.
func (uc *OrderUseCase) CreateOrder(ctx context.Context, customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error) {
	// Idempotency check (A1 bonus)
	if idempotencyKey != "" {
		existing, err := uc.repo.FindByIdempotencyKey(ctx, idempotencyKey)
		if err == nil && existing != nil {
			return existing, domain.ErrDuplicateRequest
		}
	}

	order, err := domain.NewOrder(customerID, itemName, amount, idempotencyKey)
	if err != nil {
		return nil, err
	}
	order.ID = uuid.NewString()

	if err := uc.repo.Save(ctx, order); err != nil {
		return nil, fmt.Errorf("saving order: %w", err)
	}

	_, status, err := uc.payment.ProcessPayment(ctx, order.ID, order.Amount)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentUnavailable) {
			// Leave order as Failed — chosen over Pending so retries use idempotency key
			order.MarkFailed()
			_ = uc.repo.Update(ctx, order)
			return order, domain.ErrPaymentUnavailable
		}
		order.MarkFailed()
		_ = uc.repo.Update(ctx, order)
		return order, fmt.Errorf("payment processing: %w", err)
	}

	if status == "Authorized" {
		order.MarkPaid()
	} else {
		order.MarkFailed()
	}

	if err := uc.repo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("updating order status: %w", err)
	}

	return order, nil
}

// GetOrder retrieves an order by ID.
func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return order, nil
}

// CancelOrder cancels a Pending order.
func (uc *OrderUseCase) CancelOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := order.Cancel(); err != nil {
		return nil, err
	}
	if err := uc.repo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("cancelling order: %w", err)
	}
	return order, nil
}
