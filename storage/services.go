package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/redis/go-redis/v9"
)

func ConnectToService(storageId string, port int, redisClient *redis.Client) {
	var strorages map[string]pkg.Storage
	go db.Produce(context.Background(), redisClient, "storage-stream", map[string]interface{}{"ID": storageId, "Port": port})
	for msg := range db.Subscribe(context.Background(), redisClient, "storage-update") {
		json.Unmarshal([]byte(msg.Payload), &strorages)
		currentStorage := strorages[storageId]
		if len(strorages) > 1 {
			var previousStorage pkg.Storage
			for _, storage := range strorages {
				if storage.Index == currentStorage.Index-1 {
					previousStorage = storage
					break
				}
				continue
			}
			if previousStorage.Index != 0 {
				// fetch exist data from previous index
				go restoreData(previousStorage.Id)
			}
		}
	}
}
func restoreData(storageId string) {}
func HealthCheck(storageId string, redisClient *redis.Client) {
	channel := fmt.Sprintf("%s-health", storageId)
	msg := <-db.Subscribe(context.Background(), redisClient, channel)
	if msg.Payload == "ping" {
		db.Publish(context.Background(), redisClient, channel, "pong")
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
	uploadPath := tr.Meta["UploadPath"]
	uploadHash := tr.Meta["UploadHash"]
	err := os.MkdirAll(uploadPath, 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(uploadPath, uploadHash), tr.Compressed, 0755)
	if err != nil {
		return err
	}
	return nil

}
