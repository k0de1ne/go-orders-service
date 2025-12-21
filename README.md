# Orders Service

Order management microservice with REST and gRPC APIs.
Demo project for **Junior+/Middle Go Developer** position in fintech/banking.

---

## Tech Stack

| Category | Technology |
|----------|------------|
| Language | Go 1.25 |
| HTTP | Gin |
| RPC | gRPC + Protocol Buffers |
| Database | PostgreSQL 18 |
| Messaging | Redis Streams |
| Containers | Docker, Docker Compose |
| CI/CD | GitHub Actions (self-hosted runner) |
| Logging | Zap (structured JSON) |
| Linter | golangci-lint |

---

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                     Clients                          │
└─────────────┬────────────────────────┬───────────────┘
              │                        │
       ┌──────▼──────┐          ┌──────▼──────┐
       │  REST API   │          │  gRPC API   │
       │  :8080      │          │  :9090      │
       └──────┬──────┘          └──────┬──────┘
              │                        │
              └───────────┬────────────┘
                          │
                ┌─────────▼─────────┐
                │   Service Layer   │
                │  (business logic) │
                └─────────┬─────────┘
                          │
         ┌────────────────┼────────────────┐
         │                │                │
  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐
  │ Repository  │  │  Publisher  │  │  Consumer   │
  │ (PostgreSQL)│  │   (Redis)   │  │   (Redis)   │
  └─────────────┘  └─────────────┘  └─────────────┘
```

**Key decisions:**

- **Layered architecture**: transport → service → repository
- **Shared service layer** for REST and gRPC — no business logic duplication
- **Event-driven** via Redis Streams with consumer groups
- **Graceful shutdown** for HTTP, gRPC, and Redis consumer
- **Structured logging** with request_id for request tracing

---

## Project Structure

```
cmd/api/           # Entry point, dependency initialization
internal/
  http/            # REST handlers (Gin)
  grpc/            # gRPC server
  service/         # Business logic
  repo/            # Repository (PostgreSQL)
  events/          # Publisher/Consumer (Redis Streams)
  logger/          # Zap logger + middleware
  model/           # Domain models
proto/             # .proto files and generated code
migrations/        # SQL migrations
build/             # Dockerfile, docker-compose
```

---

## API

### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/orders` | Create order |
| `GET` | `/orders/:id` | Get order |
| `GET` | `/orders` | List orders |
| `PUT` | `/orders/:id` | Update order |
| `DELETE` | `/orders/:id` | Delete order |
| `GET` | `/health` | Health check |

### gRPC Service

```protobuf
service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
  rpc UpdateOrder(UpdateOrderRequest) returns (UpdateOrderResponse);
  rpc DeleteOrder(DeleteOrderRequest) returns (DeleteOrderResponse);
}
```

**Idempotency**: pass `x-idempotency-key` in gRPC metadata for idempotent order creation.

### Events (Redis Streams)

| Event | Trigger |
|-------|---------|
| `order.created` | After order creation |
| `order.updated` | After order update |
| `order.deleted` | After order deletion |

Consumer automatically processes `order.created` and updates order status to `confirmed`.

---

## Running Locally

### Requirements

- Docker Desktop

### Pull Image

```bash
docker pull ghcr.io/k0de1ne/go-orders-service:main
```

### Start

```bash
cd build
docker compose up -d
```

### Verify

```bash
# Health check
curl http://localhost:8080/health

# Create order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"product":"Laptop","quantity":2}'

# List orders
curl http://localhost:8080/orders

# gRPC (requires grpcurl)
grpcurl -plaintext localhost:9090 orders.OrderService/ListOrders
```

### Stop

```bash
docker compose down
```

---

## CI/CD

Pipeline runs on **self-hosted runner** on push/PR to `main`.

### Stages

| Stage | Description |
|-------|-------------|
| **Lint** | golangci-lint + govulncheck |
| **Test** | `go test -race` with coverage |
| **Docker Build** | Multi-stage build, push to GHCR |
| **Security Scan** | Trivy (CRITICAL, HIGH) |
| **Integration Test** | docker-compose + health check |

### Docker Image

```
ghcr.io/k0de1ne/go-orders-service:latest
ghcr.io/k0de1ne/go-orders-service:sha-<commit>
```

---

## Production Patterns

**Implemented:**

- Graceful shutdown (HTTP, gRPC, Redis consumer)
- Structured logging (JSON) with request_id
- Health check endpoint
- Multi-stage Docker build (scratch image)
- Dependency health checks in docker-compose
- Consumer groups for Redis Streams
- Separation of transport/service/repository layers
- Unit tests with mock repository
- Performance profiling via pprof (CPU, heap, goroutines)
- Database connection pool configuration
- Database metrics endpoint
- Comprehensive benchmarking suite
- Load testing tool with configurable scenarios

**Intentional simplifications:**

- Migrations run at application startup
- Consumer group with single consumer
- No retry/DLQ for events
- Idempotency key is logged but not validated in DB

**Potential improvements:**

- Distributed tracing (OpenTelemetry)
- Metrics (Prometheus)
- Idempotency via key storage in Redis/PostgreSQL
- Outbox pattern for guaranteed event delivery
- Kubernetes manifests

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | HTTP server port |
| `GRPC_PORT` | 9090 | gRPC server port |
| `PPROF_PORT` | 6060 | pprof profiling port |
| `DATABASE_URL` | — | PostgreSQL connection string |
| `REDIS_URL` | — | Redis connection string |

---

## Development

### Basic Commands

```bash
# Run tests
make test

# Generate proto
make proto

# View logs
make logs

# Restart service
make restart

# Show all available commands
make help
```

### Performance Testing

**Benchmarks:**
```bash
make bench              # Run all benchmarks
make bench-service      # Service layer only
make bench-save         # Save baseline
make bench-compare      # Compare results
```

**Load Testing:**
```bash
make loadtest-create    # Test create operation
make loadtest-mixed     # Test mixed operations
make loadtest-duration  # 1-minute stress test
```

**Profiling:**
```bash
make pprof-cpu          # Collect CPU profile
make pprof-heap         # Collect heap profile
make pprof-web          # Open interactive web UI
```

**Monitoring:**
```bash
make health             # Health check
make db-metrics         # Database pool stats
```

**Custom load test:**
```bash
go run scripts/loadtest.go -requests 5000 -concurrency 50 -operation mixed
go run scripts/loadtest.go -duration 5m -concurrency 20 -operation create
```

Operations: `create`, `get`, `list`, `update`, `delete`, `mixed`

### Profiling Endpoints

| Endpoint | Description |
|----------|-------------|
| `:6060/debug/pprof/` | pprof index |
| `:6060/debug/pprof/profile?seconds=30` | CPU profile |
| `:6060/debug/pprof/heap` | Heap profile |
| `:6060/debug/pprof/goroutine` | Goroutine dump |
| `:8080/metrics/db` | Database pool metrics |
