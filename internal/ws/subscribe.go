package ws

import (
	"context"
	"github.com/redis/go-redis/v9"
)

const (
	RedisEvalChannel = "eval_task_results"
)

func StartRedisSubscriber(ctx context.Context, rdb *redis.Client, hub *Hub) {
	pubsub := rdb.Subscribe(ctx, RedisEvalChannel)
	ch := pubsub.Channel()

	go func() {
		for msg := range ch {
			hub.broadcast <- []byte(msg.Payload) // 转发给 WebSocket
		}
	}()
}
