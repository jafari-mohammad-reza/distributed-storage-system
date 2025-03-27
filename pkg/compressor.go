package pkg

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"io"
	"os"
	"strings"
)

type SenderMeta struct {
	Email string
	Agent string
}
type TransferPacket struct {
	FileName     string
	Dir          string
	OriginalSize int64
	Compressed   []byte
	SenderMeta
}

// to send a file we first compress and then serialize packet and to recieve it we deserialize and decompress it
func CompressFile(filePath string, senderMeta SenderMeta) (*TransferPacket, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if _, err := io.Copy(gzipWriter, file); err != nil {
		return nil, err
	}
	gzipWriter.Close()
	packet := &TransferPacket{
		FileName:     info.Name(),
		Dir:          strings.Split(filePath, info.Name())[0],
		OriginalSize: info.Size(),
		Compressed:   buf.Bytes(),
		SenderMeta:   senderMeta,
	}

	return packet, nil
}

func SerializePacket(packet *TransferPacket) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(packet); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DeserializePacket(data []byte) (*TransferPacket, error) {
	var packet TransferPacket
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	if err := decoder.Decode(&packet); err != nil {
		return nil, err
	}
	return &packet, nil
}

func DecompressPacket(packet *TransferPacket) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(packet.Compressed))
	packet.Compressed = nil
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	var decompressed bytes.Buffer
	if _, err := io.Copy(&decompressed, gzipReader); err != nil {
		return nil, err
	}

	return decompressed.Bytes(), nil
}
