# Orders Service

Event-driven REST backend service built with Go, Gin, PostgreSQL, and Redis.

## Stack

- Go 1.25
- Gin (HTTP framework)
- PostgreSQL (persistence)
- Redis (Pub/Sub events)
- Docker + docker-compose

## Architecture

```
cmd/api/main.go       # Application entry point
internal/
  http/               # HTTP handlers
  service/            # Business logic
  repo/               # Data access layer
  events/             # Redis pub/sub
  model/              # Domain models
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

### List All Orders
```bash
curl http://localhost:8080/orders
```

### Health Check
```bash
curl http://localhost:8080/health
```

## Events

When an order is created, an `order.created` event is published to Redis Pub/Sub. The built-in consumer logs events to stdout.

## Running Tests

```bash
docker compose run --rm api go test ./...
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | HTTP server port |
| DATABASE_URL | - | PostgreSQL connection string |
| REDIS_URL | - | Redis connection string |
