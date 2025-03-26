package main

import (
	"github.com/google/uuid"
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
	go storage.ConnectToService(id.String(), redisClient)
	go storage.HealthCheck(id.String(), redisClient)
	select {}
}
