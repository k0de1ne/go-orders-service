package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/orders-service/internal/model"
	"github.com/redis/go-redis/v9"
)

type OrderStatusUpdater interface {
	UpdateOrderStatus(ctx context.Context, id string, status string) error
}

type Consumer struct {
	client  *redis.Client
	updater OrderStatusUpdater
}

func NewConsumer(client *redis.Client, updater OrderStatusUpdater) *Consumer {
	return &Consumer{client: client, updater: updater}
}

func (c *Consumer) Subscribe(ctx context.Context, channel string) {
	sub := c.client.Subscribe(ctx, channel)
	ch := sub.Channel()

	log.Printf("Subscribed to channel: %s", channel)

	for msg := range ch {
		log.Printf("[%s] %s", msg.Channel, msg.Payload)

		if msg.Channel == "order.created" {
			c.handleOrderCreated(ctx, msg.Payload)
		}
	}
}

func (c *Consumer) handleOrderCreated(ctx context.Context, payload string) {
	var order model.Order
	if err := json.Unmarshal([]byte(payload), &order); err != nil {
		log.Printf("failed to unmarshal order: %v", err)
		return
	}

	time.Sleep(2 * time.Second)

	if c.updater != nil {
		if err := c.updater.UpdateOrderStatus(ctx, order.ID, "confirmed"); err != nil {
			log.Printf("failed to update order status: %v", err)
			return
		}
		log.Printf("order %s status updated to confirmed", order.ID)
	}
}
