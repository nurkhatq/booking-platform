package services

import (
    "context"
    
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    
    "booking-platform/shared/config"
    pb "booking-platform/payment-service/proto"
)

type PaymentService struct {
    pb.UnimplementedPaymentServiceServer
    config *config.Config
}

func NewPaymentService(cfg *config.Config) *PaymentService {
    return &PaymentService{
        config: cfg,
    }
}

func (s *PaymentService) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Payment processing not implemented yet")
}

func (s *PaymentService) RefundPayment(ctx context.Context, req *pb.RefundPaymentRequest) (*pb.RefundPaymentResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Payment refund not implemented yet")
}

func (s *PaymentService) GetPaymentStatus(ctx context.Context, req *pb.GetPaymentStatusRequest) (*pb.GetPaymentStatusResponse, error) {
    return nil, status.Error(codes.Unimplemented, "Payment status check not implemented yet")
}
