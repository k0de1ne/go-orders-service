package service

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/orders-service/internal/events"
	"github.com/orders-service/internal/model"
	"github.com/orders-service/internal/repo"
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
	order := &model.Order{
		ID:        uuid.New().String(),
		Product:   req.Product,
		Quantity:  req.Quantity,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, OrderCreatedChannel, order); err != nil {
			log.Printf("failed to publish order.created event: %v", err)
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
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	order.Product = req.Product
	order.Quantity = req.Quantity
	order.Status = req.Status

	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, OrderUpdatedChannel, order); err != nil {
			log.Printf("failed to publish order.updated event: %v", err)
		}
	}

	return order, nil
}

func (s *OrderService) DeleteOrder(ctx context.Context, id string) error {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, OrderDeletedChannel, order); err != nil {
			log.Printf("failed to publish order.deleted event: %v", err)
		}
	}

	return nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, id string, status string) error {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	order.Status = status
	return s.repo.Update(ctx, order)
}
