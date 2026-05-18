package usecase

import (
	"context"
	"fmt"

	"github.com/akmazhito/assignment1_2/ap2/payment-service/internal/domain"
	"github.com/google/uuid"
)

type PaymentUseCase struct {
	repo      PaymentRepository
	publisher EventPublisher
}

func NewPaymentUseCase(repo PaymentRepository, publisher EventPublisher) *PaymentUseCase {
	return &PaymentUseCase{repo: repo, publisher: publisher}
}

// ProcessPayment authorizes (or declines) a payment, persists it, then publishes an event.
func (uc *PaymentUseCase) ProcessPayment(ctx context.Context, orderID string, amount int64, customerEmail string) (*domain.Payment, error) {
	payment, st := domain.NewPayment(orderID, amount)
	payment.ID = uuid.NewString()
	payment.TransactionID = uuid.NewString()

	if err := uc.repo.Save(ctx, payment); err != nil {
		return nil, fmt.Errorf("saving payment: %w", err)
	}

	// Publish event to message broker after DB commit (A3)
	evt := PaymentCompletedEvent{
		OrderID:        orderID,
		TransactionID:  payment.TransactionID,
		Amount:         amount,
		CustomerEmail:  customerEmail,
		Status:         string(st),
		IdempotencyKey: payment.ID, // use payment ID as dedup key
	}
	if err := uc.publisher.PublishPaymentCompleted(ctx, evt); err != nil {
		// Log but don't fail the payment — at-least-once delivery via broker
		fmt.Printf("[WARN] failed to publish payment event: %v\n", err)
	}

	return payment, nil
}

// GetByOrderID fetches a payment record by order ID.
func (uc *PaymentUseCase) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return uc.repo.FindByOrderID(ctx, orderID)
}
