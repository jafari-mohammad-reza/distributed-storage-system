package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

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

func HandleUpload(tr *pkg.TransferPacket) error {
	packetBytes, err := pkg.DecompressPacket(tr)
	if err != nil {
		return err
	}
	ext := filepath.Ext(tr.FileName)
	dirPath := path.Join(tr.Dir, strings.ReplaceAll(tr.FileName, ext, ""))
	dirHash := pkg.HashPath(dirPath)
	uploadPath := path.Join(tr.Email, dirHash.Filename)
	uploadHash := pkg.HashPath(uploadPath)
	fmt.Printf("\n%+v", uploadHash)
	err = os.MkdirAll(uploadPath, 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(uploadPath, fmt.Sprintf("%s_%s%s", tr.UploadedIn.UTC().Format("20060102150405"), uploadHash.Filename, ext)), packetBytes, 0755)
	if err != nil {
		return err
	}
	return nil
}
