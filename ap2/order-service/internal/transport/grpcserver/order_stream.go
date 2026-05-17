package grpcserver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/akmazhito/assignment1_2/order/v1"
	"github.com/akmazhito/order-service/internal/domain"
)

// OrderGRPCServer implements pb.OrderServiceServer for streaming order updates.
type OrderGRPCServer struct {
	pb.UnimplementedOrderServiceServer
	db *sql.DB
}

func NewOrderGRPCServer(db *sql.DB) *OrderGRPCServer {
	return &OrderGRPCServer{db: db}
}

func RegisterOrderServiceServer(s *grpc.Server, db *sql.DB) {
	pb.RegisterOrderServiceServer(s, NewOrderGRPCServer(db))
}

// SubscribeToOrderUpdates streams real-time order status changes from the DB.
// It polls every 500ms for a status change — tied to real DB state.
func (s *OrderGRPCServer) SubscribeToOrderUpdates(req *pb.OrderRequest, stream pb.OrderService_SubscribeToOrderUpdatesServer) error {
	ctx := stream.Context()
	orderID := req.OrderId

	var lastStatus domain.OrderStatus

	for {
		select {
		case <-ctx.Done():
			return status.FromContextError(ctx.Err()).Err()
		case <-time.After(500 * time.Millisecond):
		}

		o, err := s.queryOrder(ctx, orderID)
		if err != nil {
			return status.Errorf(codes.NotFound, "order %s not found", orderID)
		}

		if o.Status != lastStatus {
			lastStatus = o.Status
			update := &pb.OrderStatusUpdate{
				OrderId:   o.ID,
				Status:    string(o.Status),
				UpdatedAt: timestamppb.New(o.UpdatedAt),
			}
			if err := stream.Send(update); err != nil {
				return fmt.Errorf("sending update: %w", err)
			}
		}

		// Stop streaming on terminal states
		if o.Status == domain.StatusPaid || o.Status == domain.StatusFailed || o.Status == domain.StatusCancelled {
			return nil
		}
	}
}

func (s *OrderGRPCServer) queryOrder(ctx context.Context, id string) (*domain.Order, error) {
	q := `SELECT id, customer_id, item_name, amount, status, idempotency_key, created_at, updated_at FROM orders WHERE id=$1`
	row := s.db.QueryRowContext(ctx, q, id)
	var o domain.Order
	var st string
	if err := row.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &st, &o.IdempotencyKey, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, err
	}
	o.Status = domain.OrderStatus(st)
	return &o, nil
}
