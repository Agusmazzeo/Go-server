package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Databases DatabasesConfig `mapstructure:"databases"`
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

func LoadConfig() (*Config, error) {
	var cfg Config

	viper.AddConfigPath("settings")
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
