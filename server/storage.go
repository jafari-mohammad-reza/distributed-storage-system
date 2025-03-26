package server

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/redis/go-redis/v9"
)

type Storage struct {
	Id         uuid.UUID
	Index      int
	LastUpdate time.Time
}

var storages map[string]Storage // map storage id to storage
func InitStorageControll(serverId string, redisClient *redis.Client) error {
	storages = make(map[string]Storage)
	go initRegisterSystem(serverId, redisClient)
	select {}
}
func initRegisterSystem(serverId string, redisClient *redis.Client) {
	stream := "storage-stream"
	group := "storage-index" // Your consumer group handling index assignments
	consumer := serverId
	go db.CreateConsumerGroup(context.Background(), redisClient, stream, group)
	go func() {
		for msg := range db.Consume(context.Background(), redisClient, stream, group, consumer) {
			fmt.Println("message", msg)
		}
	}()
}
