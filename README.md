# Orders Service

Event-driven backend service with REST and gRPC APIs, built with Go, Gin, PostgreSQL, and Redis Streams.

## Stack

- Go 1.25
- Gin (HTTP framework)
- gRPC (binary protocol)
- PostgreSQL (persistence)
- Redis Streams (guaranteed event delivery)
- Docker + docker-compose

## Architecture

```
cmd/api/main.go       # Application entry point
internal/
  http/               # HTTP handlers
  grpc/               # gRPC handlers
  service/            # Business logic
  repo/               # Data access layer
  events/             # Redis Streams publisher/consumer
  model/              # Domain models
proto/                # Protocol Buffers definitions
migrations/           # SQL migrations
```

## Running on Windows

Prerequisites: Docker Desktop

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f api

# Stop services
docker compose down
```

## API Endpoints

### Create Order
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d "{\"product\":\"Laptop\",\"quantity\":2}"
```

### Get Order by ID
```bash
curl http://localhost:8080/orders/{id}
```

### Update Order
```bash
curl -X PUT http://localhost:8080/orders/{id} \
  -H "Content-Type: application/json" \
  -d "{\"product\":\"Laptop Pro\",\"quantity\":3,\"status\":\"shipped\"}"
```

### Delete Order
```bash
curl -X DELETE http://localhost:8080/orders/{id}
```

### List All Orders
```bash
curl http://localhost:8080/orders
```

### Health Check
```bash
curl http://localhost:8080/health
```

## gRPC API

The service exposes a gRPC API on port `9090` with the following methods:

| Method | Description |
|--------|-------------|
| `CreateOrder` | Create a new order |
| `GetOrder` | Get order by ID |
| `ListOrders` | List all orders |
| `UpdateOrder` | Update an existing order |
| `DeleteOrder` | Delete an order |

### Idempotency

Pass `x-idempotency-key` in gRPC metadata to ensure idempotent order creation.

### Example with grpcurl

```bash
# List orders
grpcurl -plaintext localhost:9090 orders.OrderService/ListOrders

# Create order
grpcurl -plaintext -d '{"product":"Laptop","quantity":2}' \
  localhost:9090 orders.OrderService/CreateOrder

# Get order
grpcurl -plaintext -d '{"id":"<order-id>"}' \
  localhost:9090 orders.OrderService/GetOrder
```

## Events

Events are published to Redis Streams for guaranteed delivery:

| Event | Description |
|-------|-------------|
| `order.created` | Published when a new order is created |
| `order.updated` | Published when an order is updated |
| `order.deleted` | Published when an order is deleted |

The built-in consumer automatically updates order status to `confirmed` after receiving `order.created` (with 2-second processing delay).

## Running Tests

```bash
docker compose run --rm api go test ./...
```

## Generating Proto Files

```bash
make proto
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | HTTP server port |
| GRPC_PORT | 9090 | gRPC server port |
| DATABASE_URL | - | PostgreSQL connection string |
| REDIS_URL | - | Redis connection string |
