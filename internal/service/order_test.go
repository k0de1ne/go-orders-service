package service

import (
	"context"
	"testing"
	"time"

	"github.com/orders-service/internal/model"
)

type mockRepo struct {
	orders map[string]*model.Order
}

func newMockRepo() *mockRepo {
	return &mockRepo{orders: make(map[string]*model.Order)}
}

func (m *mockRepo) Create(ctx context.Context, order *model.Order) error {
	m.orders[order.ID] = order
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*model.Order, error) {
	return m.orders[id], nil
}

func (m *mockRepo) GetAll(ctx context.Context) ([]model.Order, error) {
	var result []model.Order
	for _, o := range m.orders {
		result = append(result, *o)
	}
	return result, nil
}

type mockPublisher struct {
	published []interface{}
}

func (m *mockPublisher) Publish(ctx context.Context, channel string, message interface{}) error {
	m.published = append(m.published, message)
	return nil
}

func TestCreateOrder(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := NewOrderService(repo, pub)

	req := CreateOrderRequest{
		Product:  "Test Product",
		Quantity: 5,
	}

	order, err := svc.CreateOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order.Product != req.Product {
		t.Errorf("expected product %s, got %s", req.Product, order.Product)
	}

	if order.Quantity != req.Quantity {
		t.Errorf("expected quantity %d, got %d", req.Quantity, order.Quantity)
	}

	if order.Status != "pending" {
		t.Errorf("expected status pending, got %s", order.Status)
	}

	if order.ID == "" {
		t.Error("expected order ID to be set")
	}

	if order.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if len(pub.published) != 1 {
		t.Errorf("expected 1 event published, got %d", len(pub.published))
	}
}

func TestGetOrder(t *testing.T) {
	repo := newMockRepo()
	svc := NewOrderService(repo, nil)

	expected := &model.Order{
		ID:        "test-id",
		Product:   "Test",
		Quantity:  1,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	repo.orders[expected.ID] = expected

	order, err := svc.GetOrder(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order.ID != expected.ID {
		t.Errorf("expected ID %s, got %s", expected.ID, order.ID)
	}
}
