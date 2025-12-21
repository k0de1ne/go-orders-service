package service

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/orders-service/internal/model"
)

func BenchmarkCreateOrder(b *testing.B) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := NewOrderService(repo, pub)
	ctx := context.Background()

	req := CreateOrderRequest{
		Product:  "Benchmark Product",
		Quantity: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.CreateOrder(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCreateOrderParallel(b *testing.B) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := NewOrderService(repo, pub)
	ctx := context.Background()

	req := CreateOrderRequest{
		Product:  "Benchmark Product",
		Quantity: 100,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := svc.CreateOrder(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkGetOrder(b *testing.B) {
	repo := newMockRepo()
	svc := NewOrderService(repo, nil)
	ctx := context.Background()

	order := &model.Order{
		ID:        "bench-id",
		Product:   "Test",
		Quantity:  1,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	repo.orders[order.ID] = order

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.GetOrder(ctx, "bench-id")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAllOrders(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			repo := newMockRepo()
			svc := NewOrderService(repo, nil)
			ctx := context.Background()

			for i := 0; i < size; i++ {
				order := &model.Order{
					ID:        fmt.Sprintf("order-%d", i),
					Product:   fmt.Sprintf("Product %d", i),
					Quantity:  i + 1,
					Status:    "pending",
					CreatedAt: time.Now(),
				}
				repo.orders[order.ID] = order
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := svc.GetOrders(ctx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkUpdateOrder(b *testing.B) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := NewOrderService(repo, pub)
	ctx := context.Background()

	order := &model.Order{
		ID:        "bench-id",
		Product:   "Original",
		Quantity:  1,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	repo.orders[order.ID] = order

	req := UpdateOrderRequest{
		Product:  "Updated Product",
		Quantity: 10,
		Status:   "confirmed",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.UpdateOrder(ctx, "bench-id", req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeleteOrder(b *testing.B) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := NewOrderService(repo, pub)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		order := &model.Order{
			ID:        fmt.Sprintf("delete-%d", i),
			Product:   "Test",
			Quantity:  1,
			Status:    "pending",
			CreatedAt: time.Now(),
		}
		repo.orders[order.ID] = order
		b.StartTimer()

		err := svc.DeleteOrder(ctx, order.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateOrderStatus(b *testing.B) {
	repo := newMockRepo()
	svc := NewOrderService(repo, nil)
	ctx := context.Background()

	order := &model.Order{
		ID:        "bench-id",
		Product:   "Test",
		Quantity:  1,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	repo.orders[order.ID] = order

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := "confirmed"
		if i%2 == 0 {
			status = "pending"
		}
		err := svc.UpdateOrderStatus(ctx, "bench-id", status)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentReadWrite(b *testing.B) {
	repo := newMockRepo()
	pub := &mockPublisher{}
	svc := NewOrderService(repo, pub)
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		order := &model.Order{
			ID:        fmt.Sprintf("order-%d", i),
			Product:   fmt.Sprintf("Product %d", i),
			Quantity:  i + 1,
			Status:    "pending",
			CreatedAt: time.Now(),
		}
		repo.orders[order.ID] = order
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			switch i % 4 {
			case 0:
				req := CreateOrderRequest{
					Product:  fmt.Sprintf("Product %d", i),
					Quantity: i,
				}
				_, _ = svc.CreateOrder(ctx, req) // Errors ignored in mixed benchmark
			case 1:
				_, _ = svc.GetOrder(ctx, fmt.Sprintf("order-%d", i%100)) // May not exist, errors expected
			case 2:
				req := UpdateOrderRequest{
					Product:  fmt.Sprintf("Updated %d", i),
					Quantity: i,
					Status:   "confirmed",
				}
				_, _ = svc.UpdateOrder(ctx, fmt.Sprintf("order-%d", i%100), req) // May not exist, errors expected
			case 3:
				_, _ = svc.GetOrders(ctx) // Errors ignored in mixed benchmark
			}
		}
	})
}

type slowMockRepo struct {
	orders map[string]*model.Order
	delay  time.Duration
	mu     sync.RWMutex
}

func newSlowMockRepo(delay time.Duration) *slowMockRepo {
	return &slowMockRepo{
		orders: make(map[string]*model.Order),
		delay:  delay,
	}
}

func (m *slowMockRepo) Create(ctx context.Context, order *model.Order) error {
	time.Sleep(m.delay)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders[order.ID] = order
	return nil
}

func (m *slowMockRepo) GetByID(ctx context.Context, id string) (*model.Order, error) {
	time.Sleep(m.delay)
	m.mu.RLock()
	defer m.mu.RUnlock()
	order, ok := m.orders[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return order, nil
}

func (m *slowMockRepo) GetAll(ctx context.Context) ([]model.Order, error) {
	time.Sleep(m.delay)
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []model.Order
	for _, o := range m.orders {
		result = append(result, *o)
	}
	return result, nil
}

func (m *slowMockRepo) Update(ctx context.Context, order *model.Order) error {
	time.Sleep(m.delay)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.orders[order.ID]; !ok {
		return sql.ErrNoRows
	}
	m.orders[order.ID] = order
	return nil
}

func (m *slowMockRepo) Delete(ctx context.Context, id string) error {
	time.Sleep(m.delay)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.orders[id]; !ok {
		return sql.ErrNoRows
	}
	delete(m.orders, id)
	return nil
}

func BenchmarkSlowDB_CreateOrder(b *testing.B) {
	delays := []time.Duration{1 * time.Millisecond, 5 * time.Millisecond, 10 * time.Millisecond}

	for _, delay := range delays {
		b.Run(fmt.Sprintf("delay_%dms", delay.Milliseconds()), func(b *testing.B) {
			repo := newSlowMockRepo(delay)
			pub := &mockPublisher{}
			svc := NewOrderService(repo, pub)
			ctx := context.Background()

			req := CreateOrderRequest{
				Product:  "Benchmark Product",
				Quantity: 100,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := svc.CreateOrder(ctx, req)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkN1Problem_UpdateOrder(b *testing.B) {
	repo := newSlowMockRepo(5 * time.Millisecond)
	pub := &mockPublisher{}
	svc := NewOrderService(repo, pub)
	ctx := context.Background()

	order := &model.Order{
		ID:        "bench-id",
		Product:   "Original",
		Quantity:  1,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	repo.orders[order.ID] = order

	req := UpdateOrderRequest{
		Product:  "Updated",
		Quantity: 10,
		Status:   "confirmed",
	}

	b.ResetTimer()
	b.ReportMetric(float64(5*2), "expected_ms/op")

	for i := 0; i < b.N; i++ {
		_, err := svc.UpdateOrder(ctx, "bench-id", req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
