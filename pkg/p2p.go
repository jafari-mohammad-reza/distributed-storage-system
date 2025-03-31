package pkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
)

func InitTcpListener(port int, connectionHandler func(conn net.Conn) error) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		slog.Error("Error in listening", "port", port, "err", err.Error())
		return err
	}
	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			if err != nil {
				slog.Error("Error in accepting connection", "err", err.Error())
				continue
			}
			go handleConnection(conn, connectionHandler)
		}
	}()
	return nil
}
func GetIncomingBuf(conn net.Conn) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	var size int64
	err := binary.Read(conn, binary.BigEndian, &size)
	if err != nil {
		slog.Error("Failed to read file size", "err", err)
		return nil, err
	}
	slog.Info("Size received", "size", size)
	_, err = io.CopyN(buf, conn, size)
	if err != nil {
		slog.Error("File reception error", "err", err)
		return nil, err
	}
	return buf, nil
}
func handleConnection(conn net.Conn, connectionHandler func(conn net.Conn) error) error {
	defer conn.Close()

	if err := connectionHandler(conn); err != nil {
		return err
	}
	return nil
}
func SendDataOverTcp(port int, size int64, dataBytes []byte) (net.Conn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		slog.Error("error dialing", "port", port, "error", err.Error())
		return nil, err
	}
	err = binary.Write(conn, binary.BigEndian, size)
	if err != nil {
		slog.Error("error sending", "port", port, "error", err.Error())
		return nil, err
	}
	_, err = io.CopyN(conn, bytes.NewReader(dataBytes), size)
	if err != nil {
		slog.Error("error copying", "port", port, "error", err.Error())
		return nil, err
	}
	return conn, nil
}
