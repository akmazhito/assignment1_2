package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/akmazhito/assignment1_2/ap2/notification-service/internal/messaging"
	"github.com/akmazhito/assignment1_2/ap2/notification-service/internal/usecase"
)

func main() {
	rabbitURL := mustEnv("RABBITMQ_URL")

	uc := usecase.NewNotificationUseCase()

	consumer, err := messaging.NewRabbitMQConsumer(rabbitURL, uc)
	if err != nil {
		log.Fatalf("creating consumer: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("[Notification] signal received, shutting down...")
		cancel()
	}()

	if err := consumer.Consume(ctx); err != nil {
		log.Fatalf("consumer error: %v", err)
	}
	log.Println("[Notification] service stopped")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}
