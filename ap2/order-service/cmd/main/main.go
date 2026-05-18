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
	"google.golang.org/grpc"

	"github.com/akmazhito/assignment1_2/ap2/order-service/internal/repository"
	grpcserver "github.com/akmazhito/assignment1_2/ap2/order-service/internal/transport/grpcserver"
	handler "github.com/akmazhito/assignment1_2/ap2/order-service/internal/transport/http"
	"github.com/akmazhito/assignment1_2/ap2/order-service/internal/usecase"
	
)

func main() {
	dsn := mustEnv("DB_DSN")
	paymentAddr := mustEnv("PAYMENT_GRPC_ADDR")
	httpPort := getEnv("HTTP_PORT", "8080")
	grpcPort := getEnv("GRPC_PORT", "9090")

	db := mustConnectDB(dsn)
	defer db.Close()

	mustRunMigrations(db)

	// Infrastructure layer
	orderRepo := repository.NewPostgresOrderRepository(db)
	paymentClient, err := repository.NewGRPCPaymentClient(paymentAddr)
	if err != nil {
		log.Fatalf("connecting to payment gRPC: %v", err)
	}

	// Use case layer (business logic)
	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient)

	// Delivery: HTTP (Gin)
	router := gin.Default()
	handler.NewOrderHandler(orderUC).RegisterRoutes(router)

	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: router,
	}

	// Delivery: gRPC streaming server (A2)
	grpcSrv := grpc.NewServer()
	grpcserver.RegisterOrderServiceServer(grpcSrv, db)

	// Start HTTP
	go func() {
		log.Printf("order-service HTTP listening on :%s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start gRPC
	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("gRPC listen: %v", err)
		}
		log.Printf("order-service gRPC streaming on :%s", grpcPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down order-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcSrv.GracefulStop()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
	log.Println("order-service stopped")
}

func mustConnectDB(dsn string) *sql.DB {
	var db *sql.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				log.Println("connected to orders DB")
				return db
			}
		}
		log.Printf("waiting for DB (%d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("cannot connect to orders DB: %v", err)
	return nil
}

func mustRunMigrations(db *sql.DB) {
	q := `
	CREATE TABLE IF NOT EXISTS orders (
		id               TEXT PRIMARY KEY,
		customer_id      TEXT NOT NULL,
		item_name        TEXT NOT NULL,
		amount           BIGINT NOT NULL,
		status           TEXT NOT NULL,
		idempotency_key  TEXT,
		created_at       TIMESTAMPTZ NOT NULL,
		updated_at       TIMESTAMPTZ NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_idempotency_key ON orders(idempotency_key) WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';
	`
	if _, err := db.Exec(q); err != nil {
		log.Fatalf("running migrations: %v", err)
	}
	log.Println("orders migrations applied")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}


