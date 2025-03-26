package pkg

import (
	"fmt"
	"os"
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
