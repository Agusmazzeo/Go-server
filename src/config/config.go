package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Service         ServiceConfig        `mapstructure:"service"`
	Databases       DatabasesConfig      `mapstructure:"databases"`
	ExternalClients ExternalClientConfig `mapstructure:"externalClients"`
}

type ServiceType string

const (
	API    ServiceType = "API"
	WORKER ServiceType = "WORKER"
)

type ServiceConfig struct {
	Type ServiceType `mapstructure:"type"`
	Port string      `mapstructure:"port"`
}

type DatabasesConfig struct {
	SQL SQLConfig `mapstructure:"sql"`
}

type SQLConfig struct {
	Host             string `mapstructure:"host"`
	Port             string `mapstructure:"port"`
	Username         string `mapstructure:"username"`
	Password         string `mapstructure:"password"`
	Driver           string `mapstructure:"driver"`
	Database         string `mapstructure:"database"`
	ConnectionString string `mapstructure:"connection_string"`
}

type ExternalClientConfig struct {
	ESCO ESCOConfig `mapstructure:"esco"`
	BCRA BCRAConfig `mapstructure:"bcra"`
}

type ESCOConfig struct {
	BaseURL  string `mapstructure:"baseUrl"`
	TokenURL string `mapstructure:"tokenUrl"`
}

type BCRAConfig struct {
	BaseURL string `mapstructure:"baseUrl"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	viper.AddConfigPath(path)
	viper.SetConfigName("appsettings")
	viper.SetConfigType("yaml")

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
