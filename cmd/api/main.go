package main

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/orders-service/internal/events"
	grpcserver "github.com/orders-service/internal/grpc"
	handler "github.com/orders-service/internal/http"
	"github.com/orders-service/internal/logger"
	"github.com/orders-service/internal/repo"
	"github.com/orders-service/internal/service"
	pb "github.com/orders-service/proto"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	log, err := logger.New()
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	pprofPort := os.Getenv("PPROF_PORT")
	if pprofPort == "" {
		pprofPort = "6060"
	}
	go func() {
		log.Info("starting pprof server", zap.String("port", pprofPort))
		if err := http.ListenAndServe(":"+pprofPort, nil); err != nil {
			log.Error("pprof server error", zap.Error(err))
		}
	}()

	log.Info("runtime configuration",
		zap.Int("GOMAXPROCS", runtime.GOMAXPROCS(0)),
		zap.Int("NumCPU", runtime.NumCPU()),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
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
		log.Fatal("failed to open database", zap.Error(err))
	}
	defer db.Close()

	maxOpenConns := 25
	maxIdleConns := 5
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	log.Info("database pool configured",
		zap.Int("max_open_conns", maxOpenConns),
		zap.Int("max_idle_conns", maxIdleConns),
	)

	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping database", zap.Error(err))
	}
	log.Info("connected to database")

	if err := repo.RunMigrations(db, "migrations"); err != nil {
		log.Fatal("failed to run migrations", zap.Error(err))
	}
	log.Info("migrations applied")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("failed to parse redis URL", zap.Error(err))
	}
	redisClient := redis.NewClient(opt)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatal("failed to ping redis", zap.Error(err))
	}
	log.Info("connected to redis")

	publisher := events.NewRedisPublisher(redisClient)

	orderRepo := repo.NewPostgresOrderRepository(db)
	orderService := service.NewOrderService(orderRepo, publisher)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := events.NewConsumer(redisClient, orderService, log)
	go consumer.Subscribe(ctx, service.OrderCreatedChannel)
	h := handler.NewHandler(orderService)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(logger.Middleware(log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/metrics/db", func(c *gin.Context) {
		stats := db.Stats()
		c.JSON(http.StatusOK, gin.H{
			"max_open_connections": stats.MaxOpenConnections,
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
			"wait_count":           stats.WaitCount,
			"wait_duration":        stats.WaitDuration.String(),
			"max_idle_closed":      stats.MaxIdleClosed,
			"max_idle_time_closed": stats.MaxIdleTimeClosed,
			"max_lifetime_closed":  stats.MaxLifetimeClosed,
		})
	})

	h.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Info("starting HTTP server", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	grpcSrv := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcSrv, grpcserver.NewServer(orderService, log))

	grpcLis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatal("failed to listen for gRPC", zap.Error(err))
	}

	go func() {
		log.Info("starting gRPC server", zap.String("port", grpcPort))
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down servers")

	cancel()

	grpcSrv.GracefulStop()
	log.Info("gRPC server stopped")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("HTTP server forced to shutdown", zap.Error(err))
	}

	if err := redisClient.Close(); err != nil {
		log.Error("error closing redis connection", zap.Error(err))
	}

	log.Info("servers exited")
}
