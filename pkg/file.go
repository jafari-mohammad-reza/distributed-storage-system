package pkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	Email       string
	Agent       string
	Application string
}
type TransferPacket struct {
	Command      string
	OriginalSize int64
	Compressed   []byte
	Meta         map[string]string
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
	meta := map[string]string{}
	meta["FileName"] = info.Name()
	meta["Dir"] = strings.Split(filePath, info.Name())[0]
	packet := &TransferPacket{
		OriginalSize: info.Size(),
		Compressed:   buf.Bytes(),
		Meta:         meta,
		SenderMeta:   senderMeta,
	}

	return packet, nil
}
func CompressDir(dirPath string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	gzipWriter, _ := gzip.NewWriterLevel(buf, gzip.BestCompression)
	tarWriter := tar.NewWriter(gzipWriter)

	err := filepath.Walk(dirPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing file %s: %w", filePath, err)
		}
		relPath, _ := filepath.Rel(dirPath, filePath)
		header, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", filePath, err)
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", filePath, err)
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filePath, err)
		}
		defer file.Close()

		if _, err := io.Copy(tarWriter, file); err != nil {
			return fmt.Errorf("failed to write file data for %s: %w", filePath, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	tarWriter.Close()
	gzipWriter.Close()

	return buf, nil
}

func DecompressBytes(data []byte, outputDir string) error {
	byteReader := bytes.NewReader(data)

	gzipReader, err := gzip.NewReader(byteReader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		targetPath := filepath.Join(outputDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directories: %w", err)
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			_, err = io.Copy(outFile, tarReader)
			outFile.Close()
			if err != nil {
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
		}
	}
	return nil
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

type PathKey struct {
	Pathname string
	Filename string
}

func HashPath(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashString := hex.EncodeToString(hash[:])
	blockSize := 5
	sliceLen := len(hashString) / blockSize
	paths := make([]string, sliceLen)
	for i := range sliceLen {
		from, to := i*blockSize, (i*blockSize)+blockSize
		paths[i] = hashString[from:to]
	}
	return PathKey{
		Pathname: strings.Join(paths, "/"),
		Filename: hashString,
	}
}

func AppendJson(path, content string) error {
	var data []string

	file, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			data = []string{content}
		} else {
			return fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		if err := json.Unmarshal(file, &data); err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
		data = append(data, content)
	}
	newData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	if err := os.WriteFile(path, newData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
