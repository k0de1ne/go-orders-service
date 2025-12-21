.PHONY: build up down logs test proto bench loadtest pprof db-metrics clean help

COMPOSE_BASE := docker compose -f build/docker-compose.yml
COMPOSE_DEV := $(COMPOSE_BASE) -f build/docker-compose.dev.yml

proto:
	$(COMPOSE_DEV) run --rm test sh -c "\
		apk add --no-cache protobuf protobuf-dev && \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
		protoc -I=. --go_out=. --go_opt=paths=source_relative \
			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
			proto/orders.proto"

build:
	$(COMPOSE_DEV) build

up:
	$(COMPOSE_DEV) up -d

down:
	$(COMPOSE_DEV) down

logs:
	$(COMPOSE_DEV) logs -f

test:
	go test -v -race ./...

test-docker:
	$(COMPOSE_DEV) run --rm test go test -v -race ./...

# Benchmarks
bench:
	@echo "Running all benchmarks..."
	go test -bench=. -benchmem ./...

bench-service:
	@echo "Running service layer benchmarks..."
	go test -bench=. -benchmem ./internal/service

bench-repo:
	@echo "Running repository benchmarks..."
	go test -bench=. -benchmem ./internal/repo

bench-save:
	@echo "Saving benchmark results..."
	@mkdir -p benchmarks
	go test -bench=. -benchmem ./... > benchmarks/baseline.txt

bench-compare:
	@echo "Comparing benchmarks (requires: go install golang.org/x/perf/cmd/benchstat@latest)"
	benchstat benchmarks/baseline.txt benchmarks/optimized.txt

# Load testing
loadtest-create:
	go run scripts/loadtest.go -requests 1000 -concurrency 10 -operation create

loadtest-mixed:
	go run scripts/loadtest.go -requests 1000 -concurrency 10 -operation mixed

loadtest-duration:
	go run scripts/loadtest.go -duration 1m -concurrency 20 -operation mixed

# Performance profiling
pprof-cpu:
	@echo "Collecting CPU profile for 30 seconds..."
	@mkdir -p profiles
	curl http://localhost:6060/debug/pprof/profile?seconds=30 > profiles/cpu.prof
	@echo "Profile saved. Analyze with: go tool pprof -http=:8081 profiles/cpu.prof"

pprof-heap:
	@echo "Collecting heap profile..."
	@mkdir -p profiles
	curl http://localhost:6060/debug/pprof/heap > profiles/heap.prof
	@echo "Profile saved. Analyze with: go tool pprof -http=:8081 profiles/heap.prof"

pprof-web:
	@echo "Opening pprof web interface..."
	go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30

# Monitoring
db-metrics:
	@curl -s http://localhost:8080/metrics/db | python -m json.tool 2>/dev/null || curl -s http://localhost:8080/metrics/db

health:
	@curl -s http://localhost:8080/health

restart:
	$(COMPOSE_DEV) restart api

# Cleanup
clean:
	rm -rf benchmarks profiles results

# Help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Development:"
	@echo "  make build          - Build Docker images"
	@echo "  make up             - Start dev services"
	@echo "  make down           - Stop dev services"
	@echo "  make restart        - Restart API service"
	@echo "  make logs           - View dev logs"
	@echo ""
	@echo "Testing:"
	@echo "  make test           - Run tests with race detector"
	@echo "  make test-docker    - Run tests in Docker"
	@echo ""
	@echo "Benchmarks:"
	@echo "  make bench          - Run all benchmarks"
	@echo "  make bench-service  - Run service layer benchmarks"
	@echo "  make bench-repo     - Run repository benchmarks"
	@echo "  make bench-save     - Save benchmark baseline"
	@echo "  make bench-compare  - Compare baseline vs optimized"
	@echo ""
	@echo "Load Testing:"
	@echo "  make loadtest-create   - Test create operation (1000 requests)"
	@echo "  make loadtest-mixed    - Test mixed operations (1000 requests)"
	@echo "  make loadtest-duration - Test for 1 minute"
	@echo ""
	@echo "Profiling:"
	@echo "  make pprof-cpu      - Collect CPU profile (30s)"
	@echo "  make pprof-heap     - Collect heap profile"
	@echo "  make pprof-web      - Open pprof web UI"
	@echo ""
	@echo "Monitoring:"
	@echo "  make db-metrics     - Show database pool metrics"
	@echo "  make health         - Health check"
	@echo ""
	@echo "Other:"
	@echo "  make proto          - Generate protobuf code"
	@echo "  make clean          - Clean generated files"
