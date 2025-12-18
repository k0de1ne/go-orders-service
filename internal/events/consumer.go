package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/orders-service/internal/logger"
	"github.com/orders-service/internal/model"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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
	log     *zap.Logger
}

func NewConsumer(client *redis.Client, updater OrderStatusUpdater, log *zap.Logger) *Consumer {
	return &Consumer{client: client, updater: updater, log: log}
}

func (c *Consumer) Subscribe(ctx context.Context, channel string) {
	err := c.client.XGroupCreateMkStream(ctx, StreamName, ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		c.log.Error("redis: failed to create consumer group", zap.Error(err))
	}

	c.log.Info("subscribed to stream", zap.String("stream", StreamName), zap.String("group", ConsumerGroup))

	for {
		select {
		case <-ctx.Done():
			c.log.Info("consumer shutting down")
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
			c.log.Error("redis: failed to read from stream", zap.Error(err))
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
		c.log.Warn("invalid event type in message", zap.String("message_id", message.ID))
		c.ackMessage(ctx, message.ID)
		return
	}

	payload, ok := message.Values["payload"].(string)
	if !ok {
		c.log.Warn("invalid payload in message", zap.String("message_id", message.ID))
		c.ackMessage(ctx, message.ID)
		return
	}

	c.log.Info("event received", zap.String("event", event), zap.String("message_id", message.ID))

	msgCtx := logger.WithContext(ctx, c.log.With(zap.String("event", event), zap.String("message_id", message.ID)))

	if event == "order.created" {
		c.handleOrderCreated(msgCtx, payload)
	}

	c.ackMessage(ctx, message.ID)
}

func (c *Consumer) ackMessage(ctx context.Context, messageID string) {
	if err := c.client.XAck(ctx, StreamName, ConsumerGroup, messageID).Err(); err != nil {
		c.log.Error("redis: failed to ack message", zap.String("message_id", messageID), zap.Error(err))
	}
}

func (c *Consumer) handleOrderCreated(ctx context.Context, payload string) {
	log := logger.FromContext(ctx)

	var order model.Order
	if err := json.Unmarshal([]byte(payload), &order); err != nil {
		log.Error("failed to unmarshal order", zap.Error(err))
		return
	}

	time.Sleep(2 * time.Second)

	if c.updater != nil {
		if err := c.updater.UpdateOrderStatus(ctx, order.ID, "confirmed"); err != nil {
			log.Error("failed to update order status", zap.String("order_id", order.ID), zap.Error(err))
			return
		}
		log.Info("order confirmed", zap.String("order_id", order.ID))
	}
}
