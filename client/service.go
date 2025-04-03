package client

import (
	"bytes"
	"encoding/json"
	"errors"
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
	packet.Command = "upload"

	serialized, err := pkg.SerializePacket(packet)
	if err != nil {
		slog.Error("error serializing file", "err", err)
	}
	conn, err := pkg.SendDataOverTcp(cfg.ServerTcpPort, int64(len(serialized)), serialized)
	if err != nil {
		return err
	}
	return conn.Close()
}

func ListUploads() ([]pkg.ListUploadsResult, error) {
	var result []pkg.ListUploadsResult
	token, err := loadTokenFromFile()
	if err != nil {
		return nil, err
	}
	// req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/api/upload-list", cfg.ServerAddr), nil)
	req, err := http.NewRequest("GET", "http://localhost:8080/api/upload-list", nil)
	req.Header.Add("Authorization", token)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
func DownloadFile(id, version, output string) error {
	token, err := loadTokenFromFile()
	if err != nil {
		return err
	}
	claims, err := pkg.DecodeToken(token)
	if err != nil {
		return err
	}
	meta := map[string]string{"FileID": id}
	if version != "" {
		meta["FileVersion"] = version
	}
	packet := pkg.TransferPacket{
		Command:    "download",
		Meta:       meta,
		SenderMeta: pkg.SenderMeta{Email: claims["email"].(string), Agent: claims["agent"].(string), Application: "client"},
	}

	serialized, err := pkg.SerializePacket(&packet)
	if err != nil {
		slog.Error("error serializing file", "err", err)
	}
	conn, err := pkg.SendDataOverTcp(cfg.ServerTcpPort, int64(len(serialized)), serialized)
	defer conn.Close()
	response, err := pkg.ReadConnBuffers(conn)
	if err != nil {
		slog.Error("error reading download response", "err", err.Error())
	}
	tr, err := pkg.DeserializePacket(response)
	if err != nil {
		return err
	}
	data, err := pkg.DecompressPacket(tr)

	if err != nil {
		return err
	}
	err = os.WriteFile(tr.Meta["FileName"], data, 0755)
	if err != nil {
		slog.Error("error writing file to output", "err", err.Error())
	}
	return nil
}
func Auth(email, password string) error {
	data, _ := json.Marshal(pkg.InvokeBody{Email: email, Password: password})
	// resp, err := http.Post(fmt.Sprintf("http://%s/api/invoke-token", cfg.ServerAddr), "application/json", bytes.NewBuffer(data))
	resp, err := http.Post("http://localhost:8080/api/invoke-token", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var responseBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&responseBody)
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
