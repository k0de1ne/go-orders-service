package repo

import "github.com/orders-service/internal/model"

type OrderRepository interface {
	Create(order *model.Order) error
	GetByID(id string) (*model.Order, error)
	GetAll() ([]model.Order, error)
}
