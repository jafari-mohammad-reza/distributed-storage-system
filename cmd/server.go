package main

import (
	"github.com/jafari-mohammad-reza/dotsync/server"
)

func main() {
	server.InitServer()
	// handle storage connections and version controll
	// there will be many replicas of server to prevent single point of failure

	// server waits for storage to sign as ready
	// it notifies other storages of that storage existence
	// storage will get an index and recieve latest data from previous index
}
