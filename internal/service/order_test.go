package service

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/orders-service/internal/model"
)

type mockRepo struct {
	orders map[string]*model.Order
	mu     sync.RWMutex
}

func newMockRepo() *mockRepo {
	return &mockRepo{orders: make(map[string]*model.Order)}
}

func (m *mockRepo) Create(ctx context.Context, order *model.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders[order.ID] = order
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*model.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	order, ok := m.orders[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return order, nil
}

func (m *mockRepo) GetAll(ctx context.Context) ([]model.Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []model.Order
	for _, o := range m.orders {
		result = append(result, *o)
	}
	return result, nil
}

func (m *mockRepo) Update(ctx context.Context, order *model.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.orders[order.ID]; !ok {
		return sql.ErrNoRows
	}
	m.orders[order.ID] = order
	return nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.orders[id]; !ok {
		return sql.ErrNoRows
	}
	delete(m.orders, id)
	return nil
}

type mockPublisher struct {
	published []interface{}
	mu        sync.Mutex
}

func (m *mockPublisher) Publish(ctx context.Context, channel string, message interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func TestUpdateOrder(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := NewOrderService(repo, pub)

	existing := &model.Order{
		ID:        "test-id",
		Product:   "Original",
		Quantity:  1,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	repo.orders[existing.ID] = existing

	req := UpdateOrderRequest{
		Product:  "Updated Product",
		Quantity: 10,
		Status:   "shipped",
	}

	order, err := svc.UpdateOrder(context.Background(), "test-id", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order.Product != req.Product {
		t.Errorf("expected product %s, got %s", req.Product, order.Product)
	}

	if order.Quantity != req.Quantity {
		t.Errorf("expected quantity %d, got %d", req.Quantity, order.Quantity)
	}

	if order.Status != req.Status {
		t.Errorf("expected status %s, got %s", req.Status, order.Status)
	}

	if len(pub.published) != 1 {
		t.Errorf("expected 1 event published, got %d", len(pub.published))
	}
}

func TestUpdateOrderNotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewOrderService(repo, nil)

	req := UpdateOrderRequest{
		Product:  "Updated",
		Quantity: 1,
		Status:   "shipped",
	}

	_, err := svc.UpdateOrder(context.Background(), "nonexistent", req)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteOrder(t *testing.T) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := NewOrderService(repo, pub)

	existing := &model.Order{
		ID:        "test-id",
		Product:   "Test",
		Quantity:  1,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	repo.orders[existing.ID] = existing

	err := svc.DeleteOrder(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := repo.orders["test-id"]; ok {
		t.Error("expected order to be deleted")
	}

	if len(pub.published) != 1 {
		t.Errorf("expected 1 event published, got %d", len(pub.published))
	}
}

func TestDeleteOrderNotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewOrderService(repo, nil)

	err := svc.DeleteOrder(context.Background(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestUpdateOrderStatus(t *testing.T) {
	repo := newMockRepo()
	svc := NewOrderService(repo, nil)

	existing := &model.Order{
		ID:        "test-id",
		Product:   "Test",
		Quantity:  1,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	repo.orders[existing.ID] = existing

	err := svc.UpdateOrderStatus(context.Background(), "test-id", "confirmed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.orders["test-id"].Status != "confirmed" {
		t.Errorf("expected status confirmed, got %s", repo.orders["test-id"].Status)
	}
}
