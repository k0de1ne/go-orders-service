# Orders Service

This project is an order management microservice featuring REST and gRPC APIs.

---

## Tech Stack

The service is built with the following technologies:

| Category | Technology | Version |
|----------|------------|---------|
| Language | Go | `1.25` |
| HTTP Framework | Gin | `v1.11.0` |
| RPC Framework | gRPC | `v1.77.0` |
| Database | PostgreSQL | `18.1-alpine` |
| Messaging | Redis | `8.4.0-alpine` |
| Containers | Docker, Docker Compose | - |
| CI/CD | GitHub Actions | - |
| Logging | Zap | `v1.27.1` |
| Linter | golangci-lint | `v2.7.2` |

---

## Architecture

The application follows a standard layered architecture:

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
          ┌────────────────┼────────────────────┐
          │                │                    │
   ┌──────▼───────┐ ┌──────▼──────────┐  ┌──────▼──────────┐
   │  Repository  │ │    Publisher    │  │    Consumer     │
   │ (PostgreSQL) │ │ (Redis Streams) │  │ (Redis Streams) │
   └──────────────┘ └─────────────────┘  └─────────────────┘
```

### Key Architectural Points:

- **Layered Design**: A clear separation between transport (HTTP/gRPC), business logic (service), and data access (repository) layers.
- **Shared Logic**: Both REST and gRPC APIs utilize the same core `service` layer, preventing code duplication.
- **Event-Driven**: The service uses Redis Streams for asynchronous event handling. For example, after an order is created, an `order.created` event is published. A background consumer process listens for these events and updates the order status to `confirmed`.
- **Graceful Shutdown**: The application gracefully shuts down HTTP, gRPC, and the Redis consumer upon receiving a `SIGINT` or `SIGTERM` signal.
- **Structured Logging**: All logs are structured (JSON) and enriched with a `request_id` for easier tracing and debugging.
- **Database Migrations**: SQL migrations are automatically applied at application startup.

---

## Project Structure

```
.
├── build/             # Docker configuration
├── cmd/api/           # Application entry point and initialization
├── internal/
│   ├── events/        # Redis Streams publisher and consumer
│   ├── grpc/          # gRPC server implementation
│   ├── http/          # REST API handlers (Gin)
│   ├── logger/        # Zap logger configuration and middleware
│   ├── model/         # Core domain models
│   ├── repo/          # PostgreSQL repository implementation
│   └── service/       # Business logic layer
├── migrations/        # SQL database migrations
└── proto/             # Protocol Buffers definitions and generated Go code
```

---

## API Reference

### REST API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/orders` | Create a new order |
| `GET` | `/orders/:id` | Get an order by its ID |
| `GET` | `/orders` | List all orders |
| `PUT` | `/orders/:id` | Update an existing order |
| `DELETE` | `/orders/:id` | Delete an order |
| `GET` | `/health` | Health check endpoint |
| `GET` | `/metrics/db`| Database connection pool statistics |

### gRPC API

The following RPCs are defined in `proto/orders.proto`:

```protobuf
service OrderService {
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
  rpc UpdateOrder(UpdateOrderRequest) returns (UpdateOrderResponse);
  rpc DeleteOrder(DeleteOrderRequest) returns (DeleteOrderResponse);
}
```

---

## How to Run

### Prerequisites

- Docker Desktop
- A `git` client
- A terminal or command prompt

### Steps

1.  **Clone the repository:**
    ```bash
    git clone <repository-url>
    cd go-orders-service
    ```

2.  **Start the services:**
    This command builds the Docker images (if not already built) and starts the `api`, `postgres`, and `redis` containers.
    ```bash
    make up
    ```
    *The services run in detached mode (`-d`). You can view logs using `make logs`.*

3.  **Verify the application is running:**
    Check the health endpoint:
    ```bash
    curl http://localhost:8080/health
    # Expected output: {"status":"ok"}
    ```

4.  **Interact with the API:**

    *   **Create an order (REST):**
        ```bash
        curl -X POST http://localhost:8080/orders \
          -H "Content-Type: application/json" \
          -d '{"product":"Laptop","quantity":1}'
        ```

    *   **List orders (REST):**
        ```bash
        curl http://localhost:8080/orders
        ```

    *   **List orders (gRPC):**
        *(Requires [grpcurl](https://github.com/fullstorydev/grpcurl))*
        ```bash
        grpcurl -plaintext localhost:9090 orders.OrderService/ListOrders
        ```

5.  **Stop the services:**
    This command stops and removes the containers.
    ```bash
    make down
    ```

---

## Development & Testing

A `Makefile` provides commands for common development tasks.

### Running Tests

Run unit and integration tests with the race detector enabled:
```bash
make test
```

### Generating Protobuf Code

To regenerate the Go code from the `.proto` files, run:
```bash
make proto
```
*This command runs inside a temporary Docker container to ensure the correct versions of `protoc` and its plugins are used.*

### Benchmarking & Performance

The project includes a suite of benchmarks and load testing scripts.

-   **Run all benchmarks:** `make bench`
-   **Run a load test:** `make loadtest-mixed`
-   **Collect a CPU profile:** `make pprof-cpu`
-   **Collect a heap profile:** `make pprof-heap`

For a full list of commands, run `make help`.

---

## CI/CD

The CI pipeline is defined in `.github/workflows/ci.yml` and runs on a self-hosted runner. It includes the following stages:

1.  **Lint**: Runs `golangci-lint` and `govulncheck` to check for code style issues and vulnerabilities.
2.  **Test**: Executes the test suite using `go test -race`.
3.  **Docker Build**: Builds the application's Docker image and pushes it to GitHub Container Registry (GHCR). A `sha-<commit>` tag and a branch tag are created. The `latest` tag is applied only to the `main` branch.
4.  **Security Scan**: Uses `Trivy` to scan the built Docker image for `CRITICAL` and `HIGH` severity vulnerabilities.
5.  **Integration Test**: Starts the application stack using `docker-compose` and verifies the API health endpoint.