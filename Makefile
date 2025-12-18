.PHONY: build up down logs test proto

proto:
	docker compose -f build/docker-compose.dev.yml run --rm test sh -c "\
		apk add --no-cache protobuf protobuf-dev && \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
		protoc -I=. --go_out=. --go_opt=paths=source_relative \
			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
			proto/orders.proto"

build:
	docker compose -f build/docker-compose.dev.yml build

up:
	docker compose -f build/docker-compose.dev.yml up -d

down:
	docker compose -f build/docker-compose.dev.yml down

logs:
	docker compose -f build/docker-compose.dev.yml logs -f

test:
	docker compose -f build/docker-compose.dev.yml run --rm test go test ./...
