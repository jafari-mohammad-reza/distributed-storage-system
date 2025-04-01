package pkg

import (
	"errors"
	"log"

	"github.com/spf13/viper"
)

type ClientConfig struct{
	ServerAddr string
	ServerPort int
	ServerTcpPort int 
}
type ServerConfig struct{
	HttpPort int
	TcpPort int
	HealthcheckInterval int
	HealthCheckTimeout int
}
type StorageConfig struct{
	Port int
}

func InitConfig(name string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("yml")
	v.SetConfigName(name)
	v.AddConfigPath(".")
	v.AutomaticEnv()
	v.WatchConfig()
	err := v.ReadInConfig()
	if err != nil {
		log.Printf("Unable to read config: %v", err)
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return nil, errors.New("config file not found")
		}
		return nil, err
	}
	return v, nil
}
func GetClientConfig() (*ClientConfig, error) {
	v, err := InitConfig("client.yml")
	if err != nil {
		return nil, err
	}
	var cfg ClientConfig
	err = v.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func GetServerConfig() (*ServerConfig, error) {
	v, err := InitConfig("server.yml")
	if err != nil {
		return nil, err
	}
	var cfg ServerConfig
	err = v.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func GetStorageConfig() (*StorageConfig, error) {
	v, err := InitConfig("storage.yml")
	if err != nil {
		return nil, err
	}
	var cfg StorageConfig
	err = v.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}