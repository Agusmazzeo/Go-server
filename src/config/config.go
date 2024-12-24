package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Service         ServiceConfig        `mapstructure:"service"`
	Databases       DatabasesConfig      `mapstructure:"databases"`
	ExternalClients ExternalClientConfig `mapstructure:"externalClients"`
	Logger          LoggerConfig         `mapstructure:"logger"`
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
	SQL   SQLConfig   `mapstructure:"sql"`
	Redis RedisConfig `mapstructure:"redis"`
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

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
}

type ExternalClientConfig struct {
	ESCO ESCOConfig `mapstructure:"esco"`
	BCRA BCRAConfig `mapstructure:"bcra"`
}

type ESCOConfig struct {
	BaseURL         string `mapstructure:"baseUrl"`
	TokenURL        string `mapstructure:"tokenUrl"`
	CategoryMapFile string `mapstructure:"categoryMapFile"`
}

type BCRAConfig struct {
	BaseURL string `mapstructure:"baseUrl"`
}

type LoggerConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// LoadConfig loads the base appsettings file and the environment-specific settings file.
func LoadConfig(path string, environment string) (*Config, error) {
	var cfg Config

	// Load the base appsettings.yaml
	viper.AddConfigPath(path)
	viper.SetConfigName("appsettings")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading base config file: %w", err)
	}

	// Load the environment-specific appsettings.{env}.yaml if provided
	if environment != "" {
		viper.SetConfigName(fmt.Sprintf("appsettings.%s", environment))
		if err := viper.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("error reading environment-specific config file: %w", err)
		}
	}

	// Unmarshal the final config into the struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	return &cfg, nil
}
