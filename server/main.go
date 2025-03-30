package server

import (
	"github.com/google/uuid"
	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
)

func InitServer() {
	id, _ := uuid.NewUUID()
	redisClient := db.NewRedisClient()
	go func() {
		if err := InitStorageControll(id.String(), redisClient); err != nil {
			panic(err)
		}
	}()
	go func() {
		if err := pkg.InitTcpListener(8000, HandleConnection); err != nil { // TODO: read port from config
			panic(err)
		}
	}()
	if err := InitHttpServer(); err != nil {
		panic(err) //TODO: will add error handling later
	}
}
