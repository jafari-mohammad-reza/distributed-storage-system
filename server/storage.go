package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/redis/go-redis/v9"
)

type Storage struct {
	Id         string
	Index      int
	LastUpdate time.Time
}

var storages map[string]Storage // map storage id to storage
func InitStorageControll(serverId string, redisClient *redis.Client) error {
	storages = make(map[string]Storage)
	go initRegisterSystem(serverId, redisClient, nil)
	go healthCheckStorages(redisClient)
	select {}
}
func initRegisterSystem(serverId string, redisClient *redis.Client, wg *sync.WaitGroup) {
	stream := "storage-stream"
	group := "storage-index" // Your consumer group handling index assignments
	consumer := serverId
	go db.CreateConsumerGroup(context.Background(), redisClient, stream, group)
	go func() {
		for msg := range db.Consume(context.Background(), redisClient, stream, group, consumer) {
			storageId := msg.Values["ID"].(string)
			storages[storageId] = Storage{
				Id:         storageId,
				Index:      len(storages) + 1,
				LastUpdate: time.Now(),
			}
			if wg != nil {
				wg.Done()
			}
		}
	}()
}
func healthCheckStorages(redisClient *redis.Client) {
	// TODO: read times from env
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		for _, storage := range storages {
			if time.Since(storage.LastUpdate) > 10*time.Second {
				channel := fmt.Sprintf("%s-health", storage.Id)
				redisClient.Publish(context.Background(), channel, "ping")

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				go func() {
					defer cancel()

					select {
					case msg := <-db.Subscribe(context.Background(), redisClient, channel):
						if msg.Payload == "pong" {
							storage.LastUpdate = time.Now()
						}
					case <-ctx.Done():
						delete(storages, storage.Id)
					}
				}()
			}
		}
	}
}
