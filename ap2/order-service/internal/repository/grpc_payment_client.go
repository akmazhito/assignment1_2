package repository

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/akmazhito/assignment1_2/ap2/order-service/internal/domain"
	paymentv1 "github.com/akmazhito/assignment1_2/ap2/proto/payment/v1"
)

// GRPCPaymentClient wraps a assignment1_2 gRPC stub and satisfies the PaymentClient port.
type GRPCPaymentClient struct {
	client paymentv1.PaymentServiceClient
}

func NewGRPCPaymentClient(addr string) (*GRPCPaymentClient, error) {
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("connecting to payment service: %w", err)
	}
	return &GRPCPaymentClient{client: paymentv1.NewPaymentServiceClient(conn)}, nil
}

func (c *GRPCPaymentClient) ProcessPayment(ctx context.Context, orderID string, amount int64) (string, string, error) {
	resp, err := c.client.ProcessPayment(ctx, &paymentv1.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded) {
			return "", "", domain.ErrPaymentUnavailable
		}
		return "", "", fmt.Errorf("grpc ProcessPayment: %w", err)
	}
	return resp.TransactionId, resp.Status, nil
}
