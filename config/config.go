package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	DB     DBConfig     `yaml:"db"`
	Baidu  BaiduConfig  `yaml:"baidu"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DBConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type BaiduConfig struct {
	AccessToken string `yaml:"access_token"`
	ShardSize   int64  `yaml:"shard_size"`
}

var AppConfig *Config

func InitConfig() error {
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		return fmt.Errorf("read config file failed: %v", err)
	}

	AppConfig = &Config{}
	err = yaml.Unmarshal(data, AppConfig)
	if err != nil {
		return fmt.Errorf("unmarshal config failed: %v", err)
	}

	return nil
}
