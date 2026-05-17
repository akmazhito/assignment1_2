package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/akmazhito/payment-service/internal/domain"
	_ "github.com/lib/pq"
)

type PostgresPaymentRepository struct {
	db *sql.DB
}

func NewPostgresPaymentRepository(db *sql.DB) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{db: db}
}

func (r *PostgresPaymentRepository) Save(ctx context.Context, p *domain.Payment) error {
	q := `INSERT INTO payments (id, order_id, transaction_id, amount, status, created_at)
	      VALUES ($1,$2,$3,$4,$5,$6)`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.OrderID, p.TransactionID, p.Amount, string(p.Status), p.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting payment: %w", err)
	}
	return nil
}

func (r *PostgresPaymentRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	q := `SELECT id, order_id, transaction_id, amount, status, created_at FROM payments WHERE order_id=$1`
	row := r.db.QueryRowContext(ctx, q, orderID)
	var p domain.Payment
	var st string
	err := row.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &st, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("scanning payment: %w", err)
	}
	p.Status = domain.PaymentStatus(st)
	return &p, nil
}
