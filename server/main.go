package server

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/jafari-mohammad-reza/dotsync/pkg"
	"github.com/jafari-mohammad-reza/dotsync/pkg/db"
)

var cfg *pkg.ServerConfig

func InitServer() {
	config, err := pkg.GetServerConfig()
	if err != nil {
		slog.Error("Error getting server config", "err", err.Error())
	}
	cfg = config
	id, _ := uuid.NewUUID()
	redisClient := db.NewRedisClient()
	go func() {
		if err := InitStorageControll(id.String(), redisClient); err != nil {
			slog.Error("Error init storage controller", "err", err.Error())
		}
	}()
	go func() {
		if err := pkg.InitTcpListener(cfg.TcpPort, HandleConnection); err != nil {
			slog.Error("Error init tcp listener", "err", err.Error())
		}
	}()
	if err := InitHttpServer(); err != nil {
		slog.Error("Error init http server", "err", err.Error())
	}
}
