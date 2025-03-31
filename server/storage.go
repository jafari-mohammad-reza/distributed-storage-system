package server

import (
	"bytes"
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

var storages map[string]pkg.Storage // map storage id to storage => if two storage connect at same time one will read from an empty storage we need to fix that
var mu sync.Mutex

func InitStorageControll(serverId string, redisClient *redis.Client) error {
	mu.Lock()
	defer mu.Unlock()
	storages = loadStoragesFromRedis(redisClient)
	go initRegisterSystem(serverId, redisClient, nil)
	go healthCheckStorages(redisClient)

	select {}
}
func loadStoragesFromRedis(redisClient *redis.Client) map[string]pkg.Storage {
	activeStorages := make(map[string]pkg.Storage)

	storageIds, err := redisClient.SMembers(context.Background(), "alive-storages").Result()
	if err != nil {
		slog.Error("Failed to fetch active storages", "err", err)
		return activeStorages
	}

	for _, storageId := range storageIds {
		port, err := redisClient.Get(context.Background(), fmt.Sprintf("storage:%s:port", storageId)).Int()
		if err != nil {
			slog.Warn("Missing port for storage", "storageId", storageId)
			continue
		}

		activeStorages[storageId] = pkg.Storage{
			Id:         storageId,
			Index:      len(activeStorages) + 1,
			LastUpdate: time.Now(),
			Port:       port,
		}
	}

	return activeStorages
}

func initRegisterSystem(serverId string, redisClient *redis.Client, wg *sync.WaitGroup) {
	stream := "storage-stream"
	disconnctStream := "disconnect-stream"
	group := "storage-index"
	consumer := serverId

	go db.CreateConsumerGroup(context.Background(), redisClient, stream, group)
	go db.CreateConsumerGroup(context.Background(), redisClient, disconnctStream, group)

	go func() {
		for msg := range db.Consume(context.Background(), redisClient, stream, group, consumer) {
			storageId := msg.Values["ID"].(string)
			port := msg.Values["Port"].(string)
			portNum, _ := strconv.Atoi(port)

			if _, exists := storages[storageId]; !exists {
				err := redisClient.SAdd(context.Background(), "alive-storages", storageId).Err()
				if err != nil {
					slog.Error("error adding new storage", "err", err.Error())
				}
				redisClient.Set(context.Background(), fmt.Sprintf("storage:%s:port", storageId), portNum, 0)
				mu.Lock()
				storages[storageId] = pkg.Storage{
					Id:         storageId,
					Index:      len(storages) + 1,
					LastUpdate: time.Now(),
					Port:       portNum,
				}
				mu.Unlock()
			}

			db.DeleteStream(context.Background(), redisClient, "storage-stream", msg.ID)

			storagesMsg, _ := json.Marshal(storages)
			db.Publish(context.Background(), redisClient, "storage-update", string(storagesMsg))

			if wg != nil {
				wg.Done()
			}
		}
	}()
	go func() {
		for msg := range db.Consume(context.Background(), redisClient, disconnctStream, group, consumer) {
			storageId := msg.Values["ID"].(string)

			if _, exists := storages[storageId]; exists {
				err := redisClient.SRem(context.Background(), "alive-storages", storageId).Err()
				if err != nil {
					slog.Error("error removing disconnected storage", "err", err.Error())
				}
				redisClient.Del(context.Background(), fmt.Sprintf("storage:%s:port", storageId))
				mu.Lock()

				delete(storages, storageId)
				mu.Unlock()
			}

			db.DeleteStream(context.Background(), redisClient, disconnctStream, msg.ID)
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
func HandleConnection(buf *bytes.Buffer) error {
	tr, err := pkg.DeserializePacket(buf.Bytes())
	if err != nil {
		panic(err)
	}
	switch tr.Command {
	case "upload":
		return handleUpload(tr)
	}
	return nil
}
func handleUpload(tr *pkg.TransferPacket) error {
	fileName := tr.Meta["FileName"]
	dir := tr.Meta["Dir"]
	ext := filepath.Ext(fileName)
	dirPath := path.Join(dir, strings.ReplaceAll(fileName, ext, ""))
	dirHash := pkg.HashPath(dirPath)
	uploadPath := path.Join(tr.Email, dirHash.Filename)
	uploadHash := pkg.HashPath(uploadPath)
	writeHash := fmt.Sprintf("%s_%s", time.Now().UTC().Format("20060102150405"), uploadHash.Filename)
	err := uploadFile(tr, writeHash)
	if err != nil {
		slog.Error("error inserting upload", "err", err)
		return err
	}
	wg := sync.WaitGroup{}
	wg.Add(len(storages))
	for _, storage := range storages {
		go func(storage pkg.Storage) {
			tr.Meta["UploadedIn"] = time.Now().String()
			tr.Meta["UploadPath"] = uploadPath
			tr.Meta["UploadHash"] = writeHash
			tr.SenderMeta.Application = "server"
			serialized, err := pkg.SerializePacket(tr)
			if err != nil {
				slog.Error("error serializing file", "err", err)
			}
			err = pkg.SendDataOverTcp(storage.Port, int64(len(serialized)), serialized)
			if err != nil {
				slog.Error("error sending data to storage", "err", err)
			}
			updateFileStorages(tr, writeHash, storage.Id)

			defer wg.Done()
		}(storage)
	}
	wg.Wait()
	return nil
}
