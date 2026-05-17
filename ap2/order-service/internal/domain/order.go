package domain

import (
	"errors"
	"time"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "Pending"
	StatusPaid      OrderStatus = "Paid"
	StatusFailed    OrderStatus = "Failed"
	StatusCancelled OrderStatus = "Cancelled"
)

type Order struct {
	ID             string
	CustomerID     string
	ItemName       string
	Amount         int64
	Status         OrderStatus
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidAmount      = errors.New("amount must be greater than 0")
	ErrCannotCancel       = errors.New("only pending orders can be cancelled")
	ErrDuplicateRequest   = errors.New("duplicate request: order already exists for this idempotency key")
	ErrPaymentUnavailable = errors.New("payment service unavailable")
)

func NewOrder(customerID, itemName string, amount int64, idempotencyKey string) (*Order, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if customerID == "" {
		return nil, errors.New("customer_id is required")
	}
	if itemName == "" {
		return nil, errors.New("item_name is required")
	}
	return &Order{
		CustomerID:     customerID,
		ItemName:       itemName,
		Amount:         amount,
		Status:         StatusPending,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}, nil
}

func (o *Order) MarkPaid() {
	o.Status = StatusPaid
	o.UpdatedAt = time.Now().UTC()
}

func (o *Order) MarkFailed() {
	o.Status = StatusFailed
	o.UpdatedAt = time.Now().UTC()
}

func (o *Order) Cancel() error {
	if o.Status != StatusPending {
		return ErrCannotCancel
	}
	o.Status = StatusCancelled
	o.UpdatedAt = time.Now().UTC()
	return nil
}
