package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/akmazhito/order-service/internal/domain"
	_ "github.com/lib/pq"
)

type PostgresOrderRepository struct {
	db *sql.DB
}

func NewPostgresOrderRepository(db *sql.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
}

func (r *PostgresOrderRepository) Save(ctx context.Context, o *domain.Order) error {
	q := `INSERT INTO orders (id, customer_id, item_name, amount, status, idempotency_key, created_at, updated_at)
	      VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := r.db.ExecContext(ctx, q, o.ID, o.CustomerID, o.ItemName, o.Amount, string(o.Status), o.IdempotencyKey, o.CreatedAt, o.UpdatedAt)
	if err != nil {
		return fmt.Errorf("inserting order: %w", err)
	}
	return nil
}

func (r *PostgresOrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	q := `SELECT id, customer_id, item_name, amount, status, idempotency_key, created_at, updated_at FROM orders WHERE id=$1`
	row := r.db.QueryRowContext(ctx, q, id)
	return scanOrder(row)
}

func (r *PostgresOrderRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	if key == "" {
		return nil, domain.ErrOrderNotFound
	}
	q := `SELECT id, customer_id, item_name, amount, status, idempotency_key, created_at, updated_at FROM orders WHERE idempotency_key=$1`
	row := r.db.QueryRowContext(ctx, q, key)
	return scanOrder(row)
}

func (r *PostgresOrderRepository) Update(ctx context.Context, o *domain.Order) error {
	q := `UPDATE orders SET status=$1, updated_at=$2 WHERE id=$3`
	_, err := r.db.ExecContext(ctx, q, string(o.Status), o.UpdatedAt, o.ID)
	if err != nil {
		return fmt.Errorf("updating order: %w", err)
	}
	return nil
}

func scanOrder(row *sql.Row) (*domain.Order, error) {
	var o domain.Order
	var status string
	err := row.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &status, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("scanning order: %w", err)
	}
	o.Status = domain.OrderStatus(status)
	return &o, nil
}
