package db

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	addr := "localhost:6379"
	if os.Getenv("MODE") == "test" {
		addr = "localhost:6380"
	}
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})
}
func Publish(ctx context.Context, client *redis.Client, channel, message string) {
	err := client.Publish(ctx, channel, message).Err()
	if err != nil {
		slog.Error("could not publish:", "Error", err)
	}
	slog.Info("Published message:", "message", message)
}
func Subscribe(ctx context.Context, client *redis.Client, channel string) <-chan *redis.Message {
	sub := client.Subscribe(ctx, channel)
	return sub.Channel()
}
func Produce(ctx context.Context, client *redis.Client, stream string, data map[string]interface{}) {
	id, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: data,
	}).Result()
	if err != nil {
		slog.Error("Failed to add message:", "Error", err)
	}
	slog.Info("Produced message with", "ID", id)
}
func Consume(ctx context.Context, client *redis.Client, stream, group, consumer string) <-chan *redis.XMessage {
	out := make(chan *redis.XMessage)

	go func() {
		defer close(out)

		for {
			entries, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: consumer,
				Streams:  []string{stream, ">"},
				Count:    1,
				Block:    5 * time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					continue
				}
				slog.Error("Failed to read from stream:", "Error", err)
				continue
			}

			for _, streamEntry := range entries {
				for _, message := range streamEntry.Messages {
					slog.Info("Consumed message:", "ID", message.ID, "Values", message.Values)

					if ackErr := client.XAck(ctx, stream, group, message.ID).Err(); ackErr != nil {
						slog.Error("Failed to ACK message:", "Error", ackErr)
					}

					select {
					case out <- &message:
					case <-ctx.Done():
						slog.Warn("Context canceled, stopping consumer")
						return
					}
				}
			}
		}
	}()

	return out
}
func CreateConsumerGroup(ctx context.Context, client *redis.Client, stream, group string) {
	err := client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		slog.Error("Failed to create consumer group:", "error", err)
	} else if err == nil {
		slog.Info("Consumer group created")
	}
}
func FlushRedis(ctx context.Context, client *redis.Client) error {
	return client.FlushAll(ctx).Err()
}
