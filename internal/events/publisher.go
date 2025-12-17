package events

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

const StreamName = "orders"

type Publisher interface {
	Publish(ctx context.Context, channel string, message interface{}) error
}

type RedisPublisher struct {
	client *redis.Client
}

func NewRedisPublisher(client *redis.Client) *RedisPublisher {
	return &RedisPublisher{client: client}
}

func (p *RedisPublisher) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamName,
		Values: map[string]interface{}{
			"event":   channel,
			"payload": string(data),
		},
	}).Err()
}
