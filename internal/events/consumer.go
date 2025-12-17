package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/orders-service/internal/model"
	"github.com/redis/go-redis/v9"
)

const (
	ConsumerGroup = "order-processors"
	ConsumerName  = "processor-1"
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
	err := c.client.XGroupCreateMkStream(ctx, StreamName, ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		log.Printf("failed to create consumer group: %v", err)
	}

	log.Printf("Subscribed to stream: %s (group: %s)", StreamName, ConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer shutting down")
			return
		default:
		}

		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    ConsumerGroup,
			Consumer: ConsumerName,
			Streams:  []string{StreamName, ">"},
			Count:    10,
			Block:    time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			log.Printf("failed to read from stream: %v", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				c.processMessage(ctx, message)
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, message redis.XMessage) {
	event, ok := message.Values["event"].(string)
	if !ok {
		log.Printf("invalid event type in message %s", message.ID)
		c.ackMessage(ctx, message.ID)
		return
	}

	payload, ok := message.Values["payload"].(string)
	if !ok {
		log.Printf("invalid payload in message %s", message.ID)
		c.ackMessage(ctx, message.ID)
		return
	}

	log.Printf("[%s] %s", event, payload)

	if event == "order.created" {
		c.handleOrderCreated(ctx, payload)
	}

	c.ackMessage(ctx, message.ID)
}

func (c *Consumer) ackMessage(ctx context.Context, messageID string) {
	if err := c.client.XAck(ctx, StreamName, ConsumerGroup, messageID).Err(); err != nil {
		log.Printf("failed to ack message %s: %v", messageID, err)
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
