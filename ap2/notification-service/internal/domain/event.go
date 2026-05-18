package domain

// PaymentEvent is the notification service's own internal model.
// It deliberately does NOT import from payment-service — decoupled bounded context.
type PaymentEvent struct {
	OrderID        string `json:"order_id"`
	TransactionID  string `json:"transaction_id"`
	Amount         int64  `json:"amount"`
	CustomerEmail  string `json:"customer_email"`
	Status         string `json:"status"`
	IdempotencyKey string `json:"idempotency_key"`
}
