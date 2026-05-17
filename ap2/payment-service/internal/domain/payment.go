package domain

import (
	"errors"
	"time"
)

const MaxPaymentAmount int64 = 100000 // amounts above this are declined

type PaymentStatus string

const (
	StatusAuthorized PaymentStatus = "Authorized"
	StatusDeclined   PaymentStatus = "Declined"
)

type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	Amount        int64
	Status        PaymentStatus
	CreatedAt     time.Time
}

var ErrPaymentNotFound = errors.New("payment not found")

func NewPayment(orderID string, amount int64) (*Payment, PaymentStatus) {
	st := StatusAuthorized
	if amount > MaxPaymentAmount {
		st = StatusDeclined
	}
	return &Payment{
		OrderID:   orderID,
		Amount:    amount,
		Status:    st,
		CreatedAt: time.Now().UTC(),
	}, st
}
