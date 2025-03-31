package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/jafari-mohammad-reza/dotsync/storage"
)

func main() {
	id, _ := uuid.NewUUID()
	// storage first connects to server and gets its own index that will carry during its lifetime
	// create its own file system and starts to cosume exist data from previous index
	// subscribe to a channel using its index to give data to next index
	// iof its the 0 index it will just create its own file system
	redisClient := db.NewRedisClient()
	port := 8081
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

	go pkg.InitTcpListener(port, storage.HandleConnection)
	go storage.ConnectToService(id.String(), port, redisClient)
	go storage.HealthCheck(id.String(), redisClient)
	select {}
}
