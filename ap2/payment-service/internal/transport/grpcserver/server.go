package grpcserver

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/akmazhito/assignment1_2/ap2/payment-service/internal/domain"
	"github.com/akmazhito/assignment1_2/ap2/payment-service/internal/usecase"
	pb "github.com/akmazhito/assignment1_2/ap2/proto/payment/v1"
)

// PaymentGRPCServer implements pb.PaymentServiceServer.
type PaymentGRPCServer struct {
	pb.UnimplementedPaymentServiceServer
	uc *usecase.PaymentUseCase
}

func NewPaymentGRPCServer(uc *usecase.PaymentUseCase) *PaymentGRPCServer {
	return &PaymentGRPCServer{uc: uc}
}

func (s *PaymentGRPCServer) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	// Business rule: amount must be positive
	if req.Amount <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "amount must be > 0")
	}

	payment, err := s.uc.ProcessPayment(ctx, req.OrderId, req.Amount, req.CustomerEmail)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "processing payment: %v", err)
	}

	return &pb.PaymentResponse{
		TransactionId: payment.TransactionID,
		Status:        string(payment.Status),
	}, nil
}

func (s *PaymentGRPCServer) GetPaymentByOrder(ctx context.Context, req *pb.GetPaymentRequest) (*pb.PaymentResponse, error) {
	payment, err := s.uc.GetByOrderID(ctx, req.OrderId)
	if err != nil {
		if err == domain.ErrPaymentNotFound {
			return nil, status.Errorf(codes.NotFound, "payment not found for order %s", req.OrderId)
		}
		return nil, status.Errorf(codes.Internal, "fetching payment: %v", err)
	}
	return &pb.PaymentResponse{
		TransactionId: payment.TransactionID,
		Status:        string(payment.Status),
	}, nil
}

// LoggingInterceptor is the A2 bonus: logs every incoming gRPC call with method name and duration.
func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)

	code := codes.OK
	if err != nil {
		if st, ok := status.FromError(err); ok {
			code = st.Code()
		}
	}
	log.Printf("[gRPC] method=%s duration=%s code=%s", info.FullMethod, duration, code)
	return resp, err
}

// NewGRPCServer creates a gRPC server with the logging interceptor pre-registered.
func NewGRPCServer(uc *usecase.PaymentUseCase) *grpc.Server {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(LoggingInterceptor),
	)
	pb.RegisterPaymentServiceServer(srv, NewPaymentGRPCServer(uc))
	return srv
}
