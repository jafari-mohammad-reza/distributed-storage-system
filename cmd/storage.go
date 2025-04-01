package main

import (
	"log/slog"

	"github.com/jafari-mohammad-reza/dotsync/storage"
)

func main() {
	// storage first connects to server and gets its own index that will carry during its lifetime
	// create its own file system and starts to cosume exist data from previous index
	// subscribe to a channel using its index to give data to next index
	// iof its the 0 index it will just create its own file system
	if err := storage.InitStorage(); err != nil {
		slog.Error("Error init storage", "err", err.Error())
	}
	select {}
}
