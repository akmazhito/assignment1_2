package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/akmazhito/assignment1_2/ap2/notification-service/internal/domain"
	"github.com/akmazhito/assignment1_2/ap2/notification-service/internal/usecase"
)

const (
	QueueName  = "payment.completed"
	MaxRetries = 3
)

// RabbitMQConsumer consumes payment.completed events with manual ACKs.
type RabbitMQConsumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	uc   *usecase.NotificationUseCase
}

func NewRabbitMQConsumer(url string, uc *usecase.NotificationUseCase) (*RabbitMQConsumer, error) {
	var conn *amqp.Connection
	var err error
	for i := 0; i < 10; i++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("waiting for RabbitMQ (%d/10): %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("connecting: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("channel: %w", err)
	}

	// Prefetch 1: only receive next message after current is ACK'd
	if err := ch.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("qos: %w", err)
	}

	return &RabbitMQConsumer{conn: conn, ch: ch, uc: uc}, nil
}

// Consume starts consuming messages until ctx is cancelled.
func (c *RabbitMQConsumer) Consume(ctx context.Context) error {
	msgs, err := c.ch.Consume(
		QueueName,
		"notification-consumer",
		false, // autoAck=false — manual ACK required (A3 requirement)
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("registering consumer: %w", err)
	}

	log.Println("[Notification] consumer started, waiting for messages...")

	for {
		select {
		case <-ctx.Done():
			log.Println("[Notification] consumer shutting down gracefully")
			return nil

		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed")
			}
			c.handleMessage(ctx, msg)
		}
	}
}

func (c *RabbitMQConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	var evt domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &evt); err != nil {
		log.Printf("[Notification] invalid message body, sending to DLQ: %v", err)
		// Permanent failure — reject without requeue so DLQ picks it up
		_ = msg.Reject(false)
		return
	}

	// A3 bonus: simulate permanent failure for dlq-test orders
	if c.uc.SimulateFailure(evt) {
		retries := retryCount(msg)
		log.Printf("[Notification] simulated failure for order %s (attempt %d/%d)", evt.OrderID, retries+1, MaxRetries)
		if retries >= MaxRetries-1 {
			log.Printf("[Notification] max retries reached for order %s, rejecting to DLQ", evt.OrderID)
			_ = msg.Reject(false) // reject -> DLX -> DLQ
		} else {
			// Nack with requeue for retry
			_ = msg.Nack(false, true)
		}
		return
	}

	alreadyProcessed, err := c.uc.HandlePaymentEvent(ctx, evt, msg.MessageId)
	if err != nil {
		log.Printf("[Notification] processing error: %v, nacking", err)
		_ = msg.Nack(false, true)
		return
	}

	if alreadyProcessed {
		// Idempotency: already handled, ACK to remove from queue
		_ = msg.Ack(false)
		return
	}

	// Success: ACK only after log is printed (A3 requirement)
	_ = msg.Ack(false)
}

// retryCount reads x-death header to count delivery attempts.
func retryCount(msg amqp.Delivery) int {
	deaths, ok := msg.Headers["x-death"].([]interface{})
	if !ok || len(deaths) == 0 {
		return 0
	}
	table, ok := deaths[0].(amqp.Table)
	if !ok {
		return 0
	}
	count, _ := table["count"].(int64)
	return int(count)
}

func (c *RabbitMQConsumer) Close() {
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
