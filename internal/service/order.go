package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/orders-service/internal/events"
	"github.com/orders-service/internal/logger"
	"github.com/orders-service/internal/model"
	"github.com/orders-service/internal/repo"
	"go.uber.org/zap"
)

const (
	OrderCreatedChannel = "order.created"
	OrderUpdatedChannel = "order.updated"
	OrderDeletedChannel = "order.deleted"
)

type OrderService struct {
	repo      repo.OrderRepository
	publisher events.Publisher
}

func NewOrderService(repo repo.OrderRepository, publisher events.Publisher) *OrderService {
	return &OrderService{repo: repo, publisher: publisher}
}

type CreateOrderRequest struct {
	Product  string `json:"product"`
	Quantity int    `json:"quantity"`
}

type UpdateOrderRequest struct {
	Product  string `json:"product"`
	Quantity int    `json:"quantity"`
	Status   string `json:"status"`
}

func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*model.Order, error) {
	log := logger.FromContext(ctx)

	order := &model.Order{
		ID:        uuid.New().String(),
		Product:   req.Product,
		Quantity:  req.Quantity,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, order); err != nil {
		log.Error("postgres: failed to create order", zap.Error(err))
		return nil, err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, OrderCreatedChannel, order); err != nil {
			log.Error("failed to publish order.created event", zap.Error(err))
		} else {
			log.Info("event published", zap.String("channel", OrderCreatedChannel), zap.String("order_id", order.ID))
		}
	}

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (*model.Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *OrderService) GetOrders(ctx context.Context) ([]model.Order, error) {
	return s.repo.GetAll(ctx)
}

func (s *OrderService) UpdateOrder(ctx context.Context, id string, req UpdateOrderRequest) (*model.Order, error) {
	log := logger.FromContext(ctx)

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("postgres: failed to get order", zap.String("order_id", id), zap.Error(err))
		return nil, err
	}

	order.Product = req.Product
	order.Quantity = req.Quantity
	order.Status = req.Status

	if err := s.repo.Update(ctx, order); err != nil {
		log.Error("postgres: failed to update order", zap.String("order_id", id), zap.Error(err))
		return nil, err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, OrderUpdatedChannel, order); err != nil {
			log.Error("failed to publish order.updated event", zap.Error(err))
		} else {
			log.Info("event published", zap.String("channel", OrderUpdatedChannel), zap.String("order_id", order.ID))
		}
	}

	return order, nil
}

func (s *OrderService) DeleteOrder(ctx context.Context, id string) error {
	log := logger.FromContext(ctx)

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("postgres: failed to get order", zap.String("order_id", id), zap.Error(err))
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		log.Error("postgres: failed to delete order", zap.String("order_id", id), zap.Error(err))
		return err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, OrderDeletedChannel, order); err != nil {
			log.Error("failed to publish order.deleted event", zap.Error(err))
		} else {
			log.Info("event published", zap.String("channel", OrderDeletedChannel), zap.String("order_id", order.ID))
		}
	}

	return nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, id string, status string) error {
	log := logger.FromContext(ctx)

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("postgres: failed to get order", zap.String("order_id", id), zap.Error(err))
		return err
	}

	order.Status = status
	if err := s.repo.Update(ctx, order); err != nil {
		log.Error("postgres: failed to update order status", zap.String("order_id", id), zap.Error(err))
		return err
	}

	log.Info("order status updated", zap.String("order_id", id), zap.String("status", status))
	return nil
}
