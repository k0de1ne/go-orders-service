package grpc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/orders-service/internal/logger"
	"github.com/orders-service/internal/model"
	"github.com/orders-service/internal/service"
	pb "github.com/orders-service/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedOrderServiceServer
	orderService *service.OrderService
	log          *zap.Logger
}

func NewServer(orderService *service.OrderService, log *zap.Logger) *Server {
	return &Server{
		orderService: orderService,
		log:          log,
	}
}

func (s *Server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	ctx, log := s.setupContext(ctx)

	idempotencyKey := getIdempotencyKey(ctx)
	if idempotencyKey != "" {
		log = log.With(zap.String("idempotency_key", idempotencyKey))
		ctx = logger.WithContext(ctx, log)
	}

	createReq := service.CreateOrderRequest{
		Product:  req.Product,
		Quantity: int(req.Quantity),
	}

	order, err := s.orderService.CreateOrder(ctx, createReq)
	if err != nil {
		log.Error("failed to create order", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create order")
	}

	log.Info("order created via gRPC", zap.String("order_id", order.ID))
	return &pb.CreateOrderResponse{
		Order: modelToProto(order),
	}, nil
}

func (s *Server) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	ctx, log := s.setupContext(ctx)

	order, err := s.orderService.GetOrder(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("order not found", zap.String("order_id", req.Id))
			return nil, status.Error(codes.NotFound, "order not found")
		}
		log.Error("failed to get order", zap.String("order_id", req.Id), zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get order")
	}

	return &pb.GetOrderResponse{
		Order: modelToProto(order),
	}, nil
}

func (s *Server) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	ctx, log := s.setupContext(ctx)

	orders, err := s.orderService.GetOrders(ctx)
	if err != nil {
		log.Error("failed to list orders", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list orders")
	}

	pbOrders := make([]*pb.Order, len(orders))
	for i, o := range orders {
		pbOrders[i] = modelToProto(&o)
	}

	return &pb.ListOrdersResponse{
		Orders: pbOrders,
	}, nil
}

func (s *Server) UpdateOrder(ctx context.Context, req *pb.UpdateOrderRequest) (*pb.UpdateOrderResponse, error) {
	ctx, log := s.setupContext(ctx)

	updateReq := service.UpdateOrderRequest{
		Product:  req.Product,
		Quantity: int(req.Quantity),
		Status:   protoStatusToString(req.Status),
	}

	order, err := s.orderService.UpdateOrder(ctx, req.Id, updateReq)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("order not found", zap.String("order_id", req.Id))
			return nil, status.Error(codes.NotFound, "order not found")
		}
		log.Error("failed to update order", zap.String("order_id", req.Id), zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update order")
	}

	log.Info("order updated via gRPC", zap.String("order_id", order.ID))
	return &pb.UpdateOrderResponse{
		Order: modelToProto(order),
	}, nil
}

func (s *Server) DeleteOrder(ctx context.Context, req *pb.DeleteOrderRequest) (*pb.DeleteOrderResponse, error) {
	ctx, log := s.setupContext(ctx)

	err := s.orderService.DeleteOrder(ctx, req.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn("order not found", zap.String("order_id", req.Id))
			return nil, status.Error(codes.NotFound, "order not found")
		}
		log.Error("failed to delete order", zap.String("order_id", req.Id), zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to delete order")
	}

	log.Info("order deleted via gRPC", zap.String("order_id", req.Id))
	return &pb.DeleteOrderResponse{}, nil
}

func (s *Server) setupContext(ctx context.Context) (context.Context, *zap.Logger) {
	requestID := uuid.New().String()
	log := s.log.With(zap.String("request_id", requestID))
	ctx = logger.WithContext(ctx, log)
	return ctx, log
}

func getIdempotencyKey(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("x-idempotency-key")
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

func modelToProto(o *model.Order) *pb.Order {
	return &pb.Order{
		Id:        o.ID,
		Product:   o.Product,
		Quantity:  int64(o.Quantity),
		Status:    stringToProtoStatus(o.Status),
		CreatedAt: o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func stringToProtoStatus(s string) pb.OrderStatus {
	switch s {
	case "pending":
		return pb.OrderStatus_ORDER_STATUS_PENDING
	case "confirmed":
		return pb.OrderStatus_ORDER_STATUS_CONFIRMED
	case "cancelled":
		return pb.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return pb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func protoStatusToString(s pb.OrderStatus) string {
	switch s {
	case pb.OrderStatus_ORDER_STATUS_PENDING:
		return "pending"
	case pb.OrderStatus_ORDER_STATUS_CONFIRMED:
		return "confirmed"
	case pb.OrderStatus_ORDER_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "pending"
	}
}
