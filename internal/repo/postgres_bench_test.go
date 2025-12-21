package repo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/orders-service/internal/model"
)

func BenchmarkPostgresCreate(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	repo := NewPostgresOrderRepository(db)
	ctx := context.Background()

	order := &model.Order{
		ID:        "test-id",
		Product:   "Test Product",
		Quantity:  10,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock.ExpectExec("INSERT INTO orders").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		b.StartTimer()

		err := repo.Create(ctx, order)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostgresGetByID(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	repo := NewPostgresOrderRepository(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create new rows for each iteration - rows cannot be reused
		rows := sqlmock.NewRows([]string{"id", "product", "quantity", "status", "created_at"}).
			AddRow("test-id", "Test Product", 10, "pending", time.Now())
		mock.ExpectQuery("SELECT (.+) FROM orders WHERE id").
			WithArgs("test-id").
			WillReturnRows(rows)
		b.StartTimer()

		_, err := repo.GetByID(ctx, "test-id")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostgresGetAll(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("rows_%d", size), func(b *testing.B) {
			db, mock, err := sqlmock.New()
			if err != nil {
				b.Fatal(err)
			}
			defer db.Close()

			repo := NewPostgresOrderRepository(db)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				// Create fresh rows for each iteration
				rows := sqlmock.NewRows([]string{"id", "product", "quantity", "status", "created_at"})
				for j := 0; j < size; j++ {
					rows.AddRow(
						fmt.Sprintf("id-%d", j),
						fmt.Sprintf("Product %d", j),
						j+1,
						"pending",
						time.Now(),
					)
				}
				mock.ExpectQuery("SELECT (.+) FROM orders ORDER BY created_at DESC").
					WillReturnRows(rows)
				b.StartTimer()

				_, err := repo.GetAll(ctx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkPostgresUpdate(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	repo := NewPostgresOrderRepository(db)
	ctx := context.Background()

	order := &model.Order{
		ID:       "test-id",
		Product:  "Updated Product",
		Quantity: 20,
		Status:   "confirmed",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock.ExpectExec("UPDATE orders").
			WithArgs(order.Product, order.Quantity, order.Status, order.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		b.StartTimer()

		err := repo.Update(ctx, order)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostgresDelete(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	repo := NewPostgresOrderRepository(db)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		mock.ExpectExec("DELETE FROM orders").
			WithArgs("test-id").
			WillReturnResult(sqlmock.NewResult(0, 1))
		b.StartTimer()

		err := repo.Delete(ctx, "test-id")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPostgresGetByIDParallel is REMOVED due to sqlmock not being thread-safe.
// sqlmock cannot be used safely in parallel benchmarks as it causes race conditions.
// For parallel benchmarks, use real database connections or thread-safe mocks.
// See BenchmarkCreateOrderParallel in service layer for proper parallel benchmark example.

func BenchmarkUpdateWithRowsAffectedCheck(b *testing.B) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	repo := NewPostgresOrderRepository(db)
	ctx := context.Background()

	order := &model.Order{
		ID:       "test-id",
		Product:  "Test",
		Quantity: 1,
		Status:   "pending",
	}

	b.Run("WithCheck", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			mock.ExpectExec("UPDATE orders").
				WillReturnResult(sqlmock.NewResult(0, 1))
			b.StartTimer()

			err := repo.Update(ctx, order)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithoutCheck", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			mock.ExpectExec("UPDATE orders").
				WillReturnResult(sqlmock.NewResult(0, 1))
			b.StartTimer()

			query := `UPDATE orders SET product = $1, quantity = $2, status = $3 WHERE id = $4`
			_, err := db.ExecContext(ctx, query, order.Product, order.Quantity, order.Status, order.ID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
