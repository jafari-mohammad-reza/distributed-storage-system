package main

import "github.com/jafari-mohammad-reza/dotsync/client"

func main() {
	if err := client.InitCli(); err != nil {
		panic(err)
	}
}
