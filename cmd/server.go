package main

import (
	"github.com/google/uuid"
	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/jafari-mohammad-reza/dotsync/server"
)

func main() {
	id, _ := uuid.NewUUID()
	redisClient := db.NewRedisClient()
	go func() {
		if err := server.InitStorageControll(id.String(), redisClient); err != nil {
			panic(err)
		}
	}()
	go func() {
		if err := pkg.InitTcpListener(8000, server.HandleUploadedFile); err != nil { // TODO: read port from config
			panic(err)
		}
	}()
	if err := server.InitHttpServer(); err != nil {
		panic(err) //TODO: will add error handling later
	}
	// handle storage connections and version controll
	// there will be many replicas of server to prevent single point of failure

	// server waits for storage to sign as ready
	// it notifies other storages of that storage existence
	// storage will get an index and recieve latest data from previous index
}
