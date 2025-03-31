package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/redis/go-redis/v9"
)

func connectToService(storageId string, port int, redisClient *redis.Client) {
	var storages map[string]pkg.Storage
	go db.Produce(context.Background(), redisClient, "storage-stream", map[string]interface{}{"ID": storageId, "Port": port})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("Received termination signal, disconnecting storage:", storageId)
		db.Produce(context.Background(), redisClient, "disconnect-stream", map[string]interface{}{
			"ID":   storageId,
			"Port": port,
		})
		os.Exit(0)
	}()
	for msg := range db.Subscribe(context.Background(), redisClient, "storage-update") {
		json.Unmarshal([]byte(msg.Payload), &storages)
		currentStorage := storages[storageId]
		if len(storages) > 1 {
			var previousStorage pkg.Storage
			for _, storage := range storages {
				if storage.Index == currentStorage.Index-1 {
					previousStorage = storage
					break
				}
			}
			if previousStorage.Index != 0 {
				// Fetch existing data from previous storage
				go restoreData(previousStorage.Id)
			}
		}
	}
}
func restoreData(storageId string) {}
func healthCheck(storageId string, redisClient *redis.Client) {
	channel := fmt.Sprintf("%s-health", storageId)
	msg := <-db.Subscribe(context.Background(), redisClient, channel)
	if msg.Payload == "ping" {
		db.Publish(context.Background(), redisClient, channel, "pong")
	}
}

func handleConnection(buf *bytes.Buffer) error {
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
	err := os.MkdirAll(path.Join("storage", "uploads", uploadPath), 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join("storage", "uploads", uploadPath, uploadHash), tr.Compressed, 0755)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	err = recordTransferLog(tr)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil

}

func recordTransferLog(tr *pkg.TransferPacket) error {
	today := time.Now().Format(time.DateOnly)
	logPath := path.Join("storage", "logs", fmt.Sprintf("%s.json", today))
	if _, err := os.Stat(path.Join("storage", "logs")); os.IsExist(err) {
		fmt.Println("reating shit")
		if err := os.MkdirAll(path.Join("storage", "logs"), 0755); err != nil {
			return err
		}
	}
	tr.Compressed = nil
	tr.Meta["UploadedIn"] = time.Now().Format(time.DateOnly)
	data, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	err = pkg.AppendJson(logPath, string(data))
	if err != nil {
		return err
	}
	return nil
}
