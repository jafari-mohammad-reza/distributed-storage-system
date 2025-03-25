package main

import "github.com/jafari-mohammad-reza/dotsync/server"

func main() {
	if err := server.InitSql(); err != nil {
		panic(err) //TODO: will add error handling later
	}
	if err := server.InitHttpServer(); err != nil {
		panic(err) //TODO: will add error handling later
	}

}
