package pkg

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func GetFilesByte(files []string) (map[string][]byte, error) {
	result := make(map[string][]byte)

	for _, file := range files {
		data, err := GetFileBytes(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file, err)
		}
		result[file] = data
	}

	return result, nil
}

func GetFileBytes(file string) ([]byte, error) {
	info, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file %s does not exist", file)
		}
		return nil, err
	}

	if info.IsDir() {

		files, err := os.ReadDir(file)
		if err != nil {
			return nil, fmt.Errorf("error reading directory %s: %w", file, err)
		}

		var allData []byte
		for _, f := range files {
			fullPath := file + "/" + f.Name()
			data, err := GetFileBytes(fullPath)
			if err != nil {
				return nil, fmt.Errorf("error reading file %s: %w", fullPath, err)
			}
			allData = append(allData, data...)
		}
		return allData, nil
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", file, err)
	}
	return data, nil
}

func GetSize(file string) (int64, error) {
	stat, err := os.Stat(file)
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}

func isSerialized(data []byte) bool {
	var testPacket TransferPacket
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	err := decoder.Decode(&testPacket)
	return err != nil // If decoding fails, it means the data is serialized
}

type SenderMeta struct {
	Email string
	Agent string
	Application string
}
type TransferPacket struct {
	FileName     string
	Dir          string
	OriginalSize int64
	Compressed   []byte
	UploadedIn time.Time
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
