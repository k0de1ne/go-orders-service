.PHONY: build up down logs test proto

proto:
	docker run --rm -v "$(shell pwd):/app" -w /app golang:1.25.5-alpine sh -c "\
		apk add --no-cache protobuf protobuf-dev && \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
		protoc -I=. --go_out=. --go_opt=paths=source_relative \
			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
			proto/orders.proto"

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

test:
	docker compose run --rm api go test ./...
