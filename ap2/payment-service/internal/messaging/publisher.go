package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/akmazhito/assignment1_2/ap2/payment-service/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "payments"
	QueueName    = "payment.completed"
	DLXName      = "payments.dlx" // Dead Letter Exchange (A3 bonus)
	DLQName      = "payment.completed.dlq"
	RoutingKey   = "payment.completed"
	MaxRetries   = 3
)

// RabbitMQPublisher implements usecase.EventPublisher.
type RabbitMQPublisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
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
		return nil, fmt.Errorf("connecting to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("opening channel: %w", err)
	}

	p := &RabbitMQPublisher{conn: conn, ch: ch}
	if err := p.declareTopology(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *RabbitMQPublisher) declareTopology() error {
	// Dead Letter Exchange (A3 bonus)
	if err := p.ch.ExchangeDeclare(DLXName, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declaring DLX: %w", err)
	}
	if _, err := p.ch.QueueDeclare(DLQName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declaring DLQ: %w", err)
	}
	if err := p.ch.QueueBind(DLQName, RoutingKey, DLXName, false, nil); err != nil {
		return fmt.Errorf("binding DLQ: %w", err)
	}

	// Main exchange
	if err := p.ch.ExchangeDeclare(ExchangeName, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declaring exchange: %w", err)
	}

	// Durable queue with DLX configured (messages exceeding x-delivery-count go to DLQ)
	args := amqp.Table{
		"x-dead-letter-exchange":    DLXName,
		"x-dead-letter-routing-key": RoutingKey,
		"x-message-ttl":             int32(60000), // 60s TTL for unacked messages
	}
	if _, err := p.ch.QueueDeclare(QueueName, true, false, false, false, args); err != nil {
		return fmt.Errorf("declaring queue: %w", err)
	}
	if err := p.ch.QueueBind(QueueName, RoutingKey, ExchangeName, false, nil); err != nil {
		return fmt.Errorf("binding queue: %w", err)
	}
	return nil
}

func (p *RabbitMQPublisher) PublishPaymentCompleted(_ context.Context, evt usecase.PaymentCompletedEvent) error {
	body, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshalling event: %w", err)
	}

	return p.ch.Publish(
		ExchangeName,
		RoutingKey,
		true, // mandatory — return if no queue bound
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // survive broker restart
			MessageId:    evt.IdempotencyKey,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

func (p *RabbitMQPublisher) Close() {
	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
