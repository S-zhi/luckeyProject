package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	DB       DBConfig       `yaml:"db"`
	Redis    RedisConfig    `yaml:"redis"`
	BaiduPan BaiduPanConfig `yaml:"baidu_pan"`
	Log      LogConfig      `yaml:"log"`
}
type LogConfig struct {
	Path string `yaml:"path"`
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

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type BaiduPanConfig struct {
	AccessToken string `yaml:"access_token"`
	IsSVIP      bool   `yaml:"is_svip"`
	LogPath     string `yaml:"log_path"`
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

	if strings.TrimSpace(AppConfig.BaiduPan.LogPath) == "" {
		AppConfig.BaiduPan.LogPath = "logs/baiduPanSDK.log"
	}

	// 配置加载完成后，按配置路径初始化应用日志。
	InitLogger()

	return nil
}
