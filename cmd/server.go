package main

import "github.com/jafari-mohammad-reza/dotsync/server"

func main() {
	if err := server.InitSql(); err != nil {
		panic(err) //TODO: will add error handling later
	}
	if err := server.InitHttpServer(); err != nil {
		panic(err) //TODO: will add error handling later
	}
	// handle storage connections and version controll
	// there will be many replicas of server to prevent single point of failure

}
