FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /api ./cmd/api

FROM scratch
COPY --from=builder /api /api
COPY --from=builder /app/migrations /migrations
EXPOSE 8080 9090
CMD ["/api"]
