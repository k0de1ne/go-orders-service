package repo

import (
	"context"
	"database/sql"

	"github.com/orders-service/internal/model"
)

type PostgresOrderRepository struct {
	db *sql.DB
}

func NewPostgresOrderRepository(db *sql.DB) *PostgresOrderRepository {
	return &PostgresOrderRepository{db: db}
}

func (r *PostgresOrderRepository) Create(ctx context.Context, order *model.Order) error {
	query := `INSERT INTO orders (id, product, quantity, status, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, order.ID, order.Product, order.Quantity, order.Status, order.CreatedAt)
	return err
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id string) (*model.Order, error) {
	query := `SELECT id, product, quantity, status, created_at FROM orders WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var order model.Order
	err := row.Scan(&order.ID, &order.Product, &order.Quantity, &order.Status, &order.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *PostgresOrderRepository) GetAll(ctx context.Context) ([]model.Order, error) {
	query := `SELECT id, product, quantity, status, created_at FROM orders ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var order model.Order
		if err := rows.Scan(&order.ID, &order.Product, &order.Quantity, &order.Status, &order.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}
