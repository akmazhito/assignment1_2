package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"github.com/akmazhito/assignment1_2/payment-service/internal/messaging"
	"github.com/akmazhito/assignment1_2/payment-service/internal/repository"
	grpcserver "github.com/akmazhito/assignment1_2/payment-service/internal/transport/grpcserver"
	handler "github.com/akmazhito/assignment1_2/payment-service/internal/transport/http"
	"github.com/akmazhito/assignment1_2/payment-service/internal/usecase"
)

func main() {
	dsn := mustEnv("DB_DSN")
	rabbitURL := mustEnv("RABBITMQ_URL")
	httpPort := getEnv("HTTP_PORT", "8081")
	grpcPort := getEnv("GRPC_PORT", "9091")

	db := mustConnectDB(dsn)
	defer db.Close()
	mustRunMigrations(db)

	// Infrastructure
	paymentRepo := repository.NewPostgresPaymentRepository(db)
	publisher, err := messaging.NewRabbitMQPublisher(rabbitURL)
	if err != nil {
		log.Fatalf("RabbitMQ publisher: %v", err)
	}
	defer publisher.Close()

	// Business logic
	paymentUC := usecase.NewPaymentUseCase(paymentRepo, publisher)

	// Delivery: HTTP
	router := gin.Default()
	handler.NewPaymentHandler(paymentUC).RegisterRoutes(router)
	httpSrv := &http.Server{Addr: ":" + httpPort, Handler: router}

	// Delivery: gRPC (with logging interceptor - A2 bonus)
	grpcSrv := grpcserver.NewGRPCServer(paymentUC)

	go func() {
		log.Printf("payment-service HTTP on :%s", httpPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP error: %v", err)
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("gRPC listen: %v", err)
		}
		log.Printf("payment-service gRPC on :%s", grpcPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("gRPC error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down payment-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	grpcSrv.GracefulStop()
	_ = httpSrv.Shutdown(ctx)
	log.Println("payment-service stopped")
}

func mustConnectDB(dsn string) *sql.DB {
	var db *sql.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				log.Println("connected to payments DB")
				return db
			}
		}
		log.Printf("waiting for DB (%d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("cannot connect to payments DB: %v", err)
	return nil
}

func mustRunMigrations(db *sql.DB) {
	q := `
	CREATE TABLE IF NOT EXISTS payments (
		id             TEXT PRIMARY KEY,
		order_id       TEXT NOT NULL UNIQUE,
		transaction_id TEXT NOT NULL,
		amount         BIGINT NOT NULL,
		status         TEXT NOT NULL,
		created_at     TIMESTAMPTZ NOT NULL
	);`
	if _, err := db.Exec(q); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Println("payments migrations applied")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
