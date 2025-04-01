package main

import (
	"log/slog"

	"github.com/jafari-mohammad-reza/dotsync/client"
)

func main() {
	if err := client.InitCli(); err != nil {
		slog.Error("Error init cli", "err", err.Error())
	}
}
