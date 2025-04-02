package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
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
	go initRegisterSystem(serverId, redisClient)
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

func initRegisterSystem(serverId string, redisClient *redis.Client) {
	stream := "storage-stream"
	disconnctStream := "disconnect-stream"
	group := "storage-index"
	updateStream := "storage-update"
	consumer := serverId
	db.CreateConsumerGroup(context.Background(), redisClient, stream, group)
	db.CreateConsumerGroup(context.Background(), redisClient, disconnctStream, group)

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
				storages[storageId] = pkg.Storage{
					Id:         storageId,
					Index:      len(storages) + 1,
					LastUpdate: time.Now(),
					Port:       portNum,
				}
			}
			db.DeleteStream(context.Background(), redisClient, stream, msg.ID)
			storagesMsg, _ := json.Marshal(storages)
			db.Publish(context.Background(), redisClient, updateStream, string(storagesMsg))
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
				delete(storages, storageId)
			}

			db.DeleteStream(context.Background(), redisClient, disconnctStream, msg.ID)
		}
	}()
}
func healthCheckStorages(redisClient *redis.Client) {
	ticker := time.NewTicker(time.Duration(cfg.HealthcheckInterval) * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		for _, storage := range storages {
			if time.Since(storage.LastUpdate) > time.Duration(cfg.HealthCheckTimeout)*time.Minute {
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
func HandleConnection(conn net.Conn) error {
	buf, err := pkg.GetIncomingBuf(conn)
	if err != nil {
		slog.Error("Error getting incoming data", "err", err.Error())
	}
	tr, err := pkg.DeserializePacket(buf.Bytes())
	if err != nil {
		slog.Error("Error DeserializePacket", "err", err.Error())
	}
	switch tr.Command {
	case "upload":
		return handleUpload(tr)
	case "download":
		return handleDownload(tr, conn)
	}
	return nil
}
func handleDownload(tr *pkg.TransferPacket, conn net.Conn) error {
	sender := tr.SenderMeta.Email
	fileId := tr.Meta["FileID"]
	versionId, exist := tr.Meta["Version"]
	fmt.Println(sender, fileId, versionId, exist)
	user, err := findUser(tr.Email)
	if err != nil {
		return err
	}
	var file db.File
	for _, f := range user.Files {
		if f.ID == fileId {
			file = f
			break
		}
		continue
	}
	fmt.Println(file)
	var version db.FileVersion
	if exist {
		// download specific version
		for _, v := range file.Versions {
			if v.ID == versionId {
				version = v
				break
			}
			continue
		}
	} else {
		// download latest version
		version = file.Versions[len(file.Versions)-1]
	}
	fmt.Printf("version: %v\n", version)
	var storage pkg.Storage
	for _, st := range version.Storages {
		s, exist := storages[st]
		if exist {
			storage = s
			break
		}
		continue
	}
	tr.Meta["Hash"] = version.Hash
	ext := filepath.Ext(file.Name)
	dirPath := path.Join(file.Path, strings.ReplaceAll(file.Name, ext, ""))
	dirHash := pkg.HashPath(dirPath)
	uploadPath := path.Join(tr.Email, dirHash.Filename)
	tr.Meta["Path"] = uploadPath
	// TODO: save upload path + upload hash as hash
	serialized, err := pkg.SerializePacket(tr)
	if err != nil {
		return err
	}
	responseConn, err := pkg.SendDataOverTcp(storage.Port, int64(len(serialized)), serialized)
	fmt.Printf("responseConn: %v\n", responseConn)
	if err != nil {
		return err
	}
	data, err := pkg.ReadConnBuffers(responseConn)
	if err != nil {
		fmt.Println("failed to read respnse conn", err.Error())
		return err
	}
	pkg.DeserializePacket(data)
	err = pkg.SendByteToConn(conn, data)
	if err != nil {
		fmt.Println("failed to send data to conn")
		return err
	}
	defer responseConn.Close()
	return nil
}
func handleUpload(tr *pkg.TransferPacket) error {
	email := tr.SenderMeta.Email
	user, err := findUser(email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	fileName := tr.Meta["FileName"]
	dir := tr.Meta["Dir"]
	ext := filepath.Ext(fileName)
	dirPath := path.Join(dir, strings.ReplaceAll(fileName, ext, ""))
	dirHash := pkg.HashPath(dirPath)
	uploadPath := path.Join(tr.Email, dirHash.Filename)
	uploadHash := pkg.HashPath(uploadPath)
	writeHash := fmt.Sprintf("%s_%s", time.Now().UTC().Format("20060102150405"), uploadHash.Filename)
	err = uploadFile(tr, writeHash)
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
			conn, err := pkg.SendDataOverTcp(storage.Port, int64(len(serialized)), serialized)
			if err != nil {
				slog.Error("error sending data to storage", "err", err)
			}
			updateFileStorages(tr, writeHash, storage.Id)
			defer conn.Close()
			defer wg.Done()
		}(storage)
	}
	wg.Wait()
	return nil
}
