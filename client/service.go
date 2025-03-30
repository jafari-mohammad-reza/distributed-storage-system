package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jafari-mohammad-reza/dotsync/pkg"
)

func UploadFile(filePath string) error {
	token, err := loadTokenFromFile()
	if err != nil {
		return err
	}
	claims, err := pkg.DecodeToken(token)
	if err != nil {
		return err
	}
	packet, err := pkg.CompressFile(filePath, pkg.SenderMeta{Email: claims["email"].(string), Agent: claims["agent"].(string), Application: "client"})
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
func Auth(email, password string) error {
	data, _ := json.Marshal(pkg.InvokeBody{Email: email, Password: password})
	resp, err := http.Post("http://localhost:8080/api/invoke-token", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
	fmt.Println("responseBody", responseBody)
	if message, exist := responseBody["message"]; exist {
		return errors.New(message.(string))
	}
	if err := saveTokenToFile(responseBody["token"].(string)); err != nil {
		return err
	}
	return nil
}
func AuthGuard() error {
	token, err := loadTokenFromFile()
	if err != nil {
		return err
	}
	if token == "" {
		return errors.New("token not found")
	}
	return nil
}
func RevokeToken() error {
	token, err := loadTokenFromFile()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", "http://localhost:8080/api/revoke-token", nil)

	if err != nil {
		return err
	}
	req.Header.Add("Authorization", token)
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer removeTokenFromFile()

	return nil
}

type Config struct {
	Token string `json:"token"`
}

func saveTokenToFile(token string) error {
	configDir, _ := os.UserConfigDir()
	configPath := filepath.Join(configDir, "dss", "config.json")

	os.MkdirAll(filepath.Dir(configPath), 0700)

	data, _ := json.MarshalIndent(Config{Token: token}, "", "  ")
	return os.WriteFile(configPath, data, 0600)
}

func loadTokenFromFile() (string, error) {
	configDir, _ := os.UserConfigDir()
	configPath := filepath.Join(configDir, "dss", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	var config Config
	json.Unmarshal(data, &config)
	return config.Token, nil
}
func removeTokenFromFile() error {
	configDir, _ := os.UserConfigDir()
	configPath := filepath.Join(configDir, "dss", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	config.Token = ""
	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, updatedData, 0600)
}
