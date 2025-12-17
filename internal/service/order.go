package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/orders-service/internal/events"
	"github.com/orders-service/internal/model"
	"github.com/orders-service/internal/repo"
)

const OrderCreatedChannel = "order.created"

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
		_ = s.publisher.Publish(ctx, OrderCreatedChannel, order)
	}

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (*model.Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *OrderService) GetOrders(ctx context.Context) ([]model.Order, error) {
	return s.repo.GetAll(ctx)
}
