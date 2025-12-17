package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/orders-service/internal/events"
	handler "github.com/orders-service/internal/http"
	"github.com/orders-service/internal/repo"
	"github.com/orders-service/internal/service"
	"github.com/redis/go-redis/v9"
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

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL is required")
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

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal(err)
	}
	redisClient := redis.NewClient(opt)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to Redis")

	publisher := events.NewRedisPublisher(redisClient)
	consumer := events.NewConsumer(redisClient)

	go consumer.Subscribe(context.Background(), service.OrderCreatedChannel)

	orderRepo := repo.NewPostgresOrderRepository(db)
	orderService := service.NewOrderService(orderRepo, publisher)
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
