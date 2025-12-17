package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	handler "github.com/orders-service/internal/http"
	"github.com/orders-service/internal/repo"
	"github.com/orders-service/internal/service"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to database")

	if err := repo.RunMigrations(db, "migrations"); err != nil {
		log.Fatal(err)
	}
	log.Println("Migrations applied")

	orderRepo := repo.NewPostgresOrderRepository(db)
	orderService := service.NewOrderService(orderRepo)
	h := handler.NewHandler(orderService)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	h.RegisterRoutes(r)

	log.Printf("Starting server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
