package main

import (
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
	"github.com/jafari-mohammad-reza/dotsync/server"
)

func main() {
	if err := db.InitSqlite(); err != nil {
		panic(err) //TODO: will add error handling later
	}
	if err := server.InitDb(); err != nil {
		panic(err)
	}
	redisClient := db.NewRedisClient()
	go func() {
		if err := server.InitStorageControll(redisClient); err != nil {
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
