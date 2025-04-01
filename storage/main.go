package storage

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
)

func InitStorage() error {
	id, _ := uuid.NewUUID()
	redisClient := db.NewRedisClient()
	port := rand.IntN(9000-8080) + 8080
	fmt.Println("Storage port", port)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Application crashed, disconnecting storage:", id.String())
			db.Produce(context.Background(), redisClient, "disconnect-stream", map[string]interface{}{
				"ID":   id.String(),
				"Port": port,
			})
			os.Exit(1)
		}
	}()
	if err := initFileSystem(); err != nil {
		slog.Error("init storage file system", "err", err.Error())
	}
	go pkg.InitTcpListener(port, handleConnection)
	go connectToService(id.String(), port, redisClient)
	go healthCheck(id.String(), redisClient)
	return nil
}
func initFileSystem() error {
	dirs := []string{ "logs", "uploads", "backups"}
	for _, dir := range dirs {
		if err := os.MkdirAll(path.Join("storage", dir), 0755); err != nil {
			return err
		}
	}
	return nil
}
