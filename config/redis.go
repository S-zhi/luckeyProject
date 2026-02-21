package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() error {
	if AppConfig == nil {
		return errors.New("应用配置未初始化")
	}

	cfg := AppConfig.Redis
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return errors.New("Redis host 为空")
	}

	port := cfg.Port
	if port == 0 {
		port = 6379
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return fmt.Errorf("Redis ping 失败 (host=%s port=%d db=%d): %w", host, port, cfg.DB, err)
	}

	RedisClient = client
	return nil
}

func CloseRedis() error {
	if RedisClient == nil {
		return nil
	}
	err := RedisClient.Close()
	RedisClient = nil
	return err
}
