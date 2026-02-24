package config

import (
	"fmt"
	"log"
	"os"

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
	data, err := os.ReadFile(DefaultConfigFilePath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}
	AppConfig = &Config{}
	err = yaml.Unmarshal(data, AppConfig)
	if err != nil {
		return fmt.Errorf("解析配置失败: %v", err)
	}

	return nil
}

func InitDeploy() {
	// 初始化配置文件
	if err := InitConfig(); err != nil {
		log.Fatalf("初始化配置失败：%v", err)
	}
	// 初始化日志文件
	InitLogger()
	// 初始化 Mysql数据库
	if err := InitDB(); err != nil {
		log.Fatalf("初始化数据库失败：%v", err)
	}

	// 初始化 Redis数据库
	if err := InitRedis(); err != nil {
		log.Fatalf("初始化Redis失败：%v", err)
	}

}
