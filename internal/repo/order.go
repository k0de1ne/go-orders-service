package repo

import (
	"context"

	"github.com/orders-service/internal/model"
)

type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	GetByID(ctx context.Context, id string) (*model.Order, error)
	GetAll(ctx context.Context) ([]model.Order, error)
	Update(ctx context.Context, order *model.Order) error
	Delete(ctx context.Context, id string) error
}
