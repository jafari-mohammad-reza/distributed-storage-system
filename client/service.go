package client

import (
	"log/slog"

	"github.com/jafari-mohammad-reza/dotsync/pkg"
)

func UploadFile(filePath string) error {
	packet, err := pkg.CompressFile(filePath, pkg.SenderMeta{Email: "client@gmail.com", Agent: "client-agent"})
	if err != nil {
		slog.Error("error compressing file", "err", err)
		return err
	}

	serialized, err := pkg.SerializePacket(packet)
	if err != nil {
		slog.Error("error serializing file", "err", err)
	}
	return pkg.SendDataOverTcp(8000, int64(len(serialized)), serialized) // TODO: read server port from config
}
