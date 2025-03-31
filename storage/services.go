package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
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
		fmt.Println("new storage subscribd", msg)
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
				go restoreData(&previousStorage)
			}
		}
	}
}
func restoreData(storage *pkg.Storage) {
	fmt.Println("restoring data", storage)
	tr := pkg.TransferPacket{
		Command:    "cacheup",
		Compressed: nil,
		SenderMeta: pkg.SenderMeta{},
	}
	packet, _ := pkg.SerializePacket(&tr)
	conn, err := pkg.SendDataOverTcp(storage.Port, int64(len(packet)), packet)
	if err != nil {
		fmt.Println("send cacheup err", err.Error())
		return
	}
	if conn != nil {
		defer conn.Close()
	}

	var dataSize int64
	err = binary.Read(conn, binary.BigEndian, &dataSize)
	if err != nil {
		fmt.Println("failed to read data size:", err.Error())
		return
	}

	buffer := make([]byte, dataSize)
	_, err = io.ReadFull(conn, buffer)
	if err != nil {
		fmt.Println("failed to read compressed data:", err.Error())
		return
	}
	err = pkg.DecompressBytes(buffer, path.Join("storage", "uploads"))
	if err != nil {
		fmt.Println("failed to decompress data:", err.Error())
		return
	}

}
func healthCheck(storageId string, redisClient *redis.Client) {
	channel := fmt.Sprintf("%s-health", storageId)
	msg := <-db.Subscribe(context.Background(), redisClient, channel)
	if msg.Payload == "ping" {
		db.Publish(context.Background(), redisClient, channel, "pong")
	}
}

func handleConnection(conn net.Conn) error {
	buf, err := pkg.GetIncomingBuf(conn)
	if err != nil {
		panic(err)
	}
	tr, err := pkg.DeserializePacket(buf.Bytes())
	if err != nil {
		panic(err)
	}
	switch tr.Command {
	case "upload":
		return handleUpload(tr)
	case "cacheup":
		return handleCacheUp(tr, conn)
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
func handleCacheUp(tr *pkg.TransferPacket, conn net.Conn) error {
	fmt.Println("Caching up")
	startSpan := tr.Meta["StartSpan"]
	if startSpan == "" {
	}

	dir, err := pkg.CompressDir(path.Join("storage", "uploads"))
	if err != nil {
		fmt.Println("compressing dir erro", err.Error())
		return err
	}
	err = binary.Write(conn, binary.BigEndian, int64(len(dir.Bytes())))
	if err != nil {
		fmt.Println("writing size erro", err.Error())
		return err
	}
	_, err = io.CopyN(conn, bytes.NewReader(dir.Bytes()), int64(len(dir.Bytes())))
	if err != nil {
		slog.Error("error copying", "error", err.Error())
		return err
	}
	return nil
}

func loadTransferLog() ([]pkg.TransferPacket, error) {
	return nil, nil
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
