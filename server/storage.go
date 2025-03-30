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

var storages map[string]pkg.Storage // map storage id to storage
func InitStorageControll(serverId string, redisClient *redis.Client) error {
	storages = make(map[string]pkg.Storage)
	go initRegisterSystem(serverId, redisClient, nil)
	go healthCheckStorages(redisClient)
	select {}
}
func initRegisterSystem(serverId string, redisClient *redis.Client, wg *sync.WaitGroup) {
	stream := "storage-stream"
	group := "storage-index"
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
				db.DeleteStream(context.Background(), redisClient, "storage-stream", msg.ID)
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
