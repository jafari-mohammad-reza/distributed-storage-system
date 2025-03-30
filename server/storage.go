package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/redis/go-redis/v9"
)

var storages map[string]pkg.Storage // map storage id to storage
func InitStorageControll(serverId string, redisClient *redis.Client) error {
	storages = make(map[string]pkg.Storage)
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
			port := msg.Values["Port"].(string)
			portNum, _ := strconv.Atoi(port)
			if _, exists := storages[storageId]; !exists {
				storages[storageId] = pkg.Storage{
					Id:         storageId,
					Index:      len(storages) + 1,
					LastUpdate: time.Now(),
					Port:       portNum,
				}
				storagesMsg, _ := json.Marshal(storages)
				db.Publish(context.Background(), redisClient, "storage-update", string(storagesMsg))
			}
			if wg != nil {
				wg.Done()
			}
		}
	}()
}
func healthCheckStorages(redisClient *redis.Client) {
	// TODO: read times from env
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		for _, storage := range storages {
			if time.Since(storage.LastUpdate) > 10*time.Minute {
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
func HandleUploadedFile(tr *pkg.TransferPacket) error {
	ext := filepath.Ext(tr.FileName)
	dirPath := path.Join(tr.Dir, strings.ReplaceAll(tr.FileName, ext, ""))
	dirHash := pkg.HashPath(dirPath)
	uploadPath := path.Join(tr.Email, dirHash.Filename)
	uploadHash := pkg.HashPath(uploadPath)
	err := uploadFile(tr, uploadHash.Filename)
	if err != nil {
		slog.Error("error inserting upload", "err", err)
		return err
	}
	wg := sync.WaitGroup{}
	wg.Add(len(storages))
	for _, storage := range storages {
		go func(storage pkg.Storage) {
			tr.UploadedIn = time.Now()
			tr.SenderMeta.Application = "server"
			serialized, err := pkg.SerializePacket(tr)
			if err != nil {
				slog.Error("error serializing file", "err", err)
			}
			err = pkg.SendDataOverTcp(storage.Port, int64(len(serialized)), serialized)
			if err != nil {
				slog.Error("error sending data to storage", "err", err)
			}
			updateFileStorages(uploadHash.Filename, storage.Id)
			// updateUploadStorage(tr.FileName, tr.Dir, storage.Id)
			defer wg.Done()
		}(storage)
	}
	wg.Wait()
	return nil
}
