package events

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

type Consumer struct {
	client *redis.Client
}

func NewConsumer(client *redis.Client) *Consumer {
	return &Consumer{client: client}
}

func (c *Consumer) Subscribe(ctx context.Context, channel string) {
	sub := c.client.Subscribe(ctx, channel)
	ch := sub.Channel()

	log.Printf("Subscribed to channel: %s", channel)

	for msg := range ch {
		log.Printf("[%s] %s", msg.Channel, msg.Payload)
	}
}
