package usecase

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/akmazhito/assignment1_2/ap2/notification-service/internal/domain"
)

// NotificationUseCase processes payment events and simulates sending emails.
// Idempotency is enforced by tracking processed message IDs in memory
// (swap for DB/Redis in production).
type NotificationUseCase struct {
	mu          sync.Mutex
	processedIDs map[string]bool
}

func NewNotificationUseCase() *NotificationUseCase {
	return &NotificationUseCase{
		processedIDs: make(map[string]bool),
	}
}

// HandlePaymentEvent processes a payment event idempotently.
// Returns (alreadyProcessed, error).
func (uc *NotificationUseCase) HandlePaymentEvent(ctx context.Context, evt domain.PaymentEvent, messageID string) (bool, error) {
	// Idempotency check (A3 requirement)
	key := messageID
	if key == "" {
		key = evt.IdempotencyKey
	}
	if key == "" {
		key = evt.OrderID + ":" + evt.TransactionID
	}

	uc.mu.Lock()
	if uc.processedIDs[key] {
		uc.mu.Unlock()
		log.Printf("[Notification] duplicate message skipped: key=%s order=%s", key, evt.OrderID)
		return true, nil
	}
	uc.processedIDs[key] = true
	uc.mu.Unlock()

	// Simulate sending email
	amountDollars := float64(evt.Amount) / 100.0
	email := evt.CustomerEmail
	if email == "" {
		email = "majitova.aknur@gmail.com"
	}

	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%.2f Status: %s",
		email, evt.OrderID, amountDollars, evt.Status)

	_ = ctx
	return false, nil
}

// SimulateFailure deliberately returns an error for testing DLQ (A3 bonus).
// In tests you can inject an order_id that triggers this path.
func (uc *NotificationUseCase) SimulateFailure(evt domain.PaymentEvent) bool {
	return evt.OrderID == "dlq-test"
}

func (uc *NotificationUseCase) String() string {
	return fmt.Sprintf("NotificationUseCase(processed=%d)", len(uc.processedIDs))
}
