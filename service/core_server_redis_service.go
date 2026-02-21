package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"lucky_project/config"
	"sort"
	"strings"

	"github.com/redis/go-redis/v9"
)

const coreServersHashKey = "core-servers"

var ErrRedisNotInitialized = errors.New("Redis 客户端未初始化")
var ErrCoreServerKeyRequired = errors.New("核心服务器 key 不能为空")
var ErrCoreServerNotFound = errors.New("未找到核心服务器")

type CoreServer struct {
	Key  string `json:"key"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type coreServerValue struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func ListCoreServers(ctx context.Context) ([]CoreServer, error) {
	if config.RedisClient == nil {
		return nil, ErrRedisNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}

	rawMap, err := config.RedisClient.HGetAll(ctx, coreServersHashKey).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall %s 失败: %w", coreServersHashKey, err)
	}

	keys := make([]string, 0, len(rawMap))
	for key := range rawMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]CoreServer, 0, len(keys))
	for _, key := range keys {
		raw := strings.TrimSpace(rawMap[key])
		if raw == "" {
			continue
		}

		var value coreServerValue
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			return nil, fmt.Errorf("解析核心服务器失败（key=%s）: %w", key, err)
		}

		result = append(result, CoreServer{
			Key:  key,
			IP:   value.IP,
			Port: value.Port,
		})
	}

	return result, nil
}

func GetCoreServerByKey(ctx context.Context, key string) (CoreServer, error) {
	if config.RedisClient == nil {
		return CoreServer{}, ErrRedisNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}

	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return CoreServer{}, ErrCoreServerKeyRequired
	}

	raw, err := config.RedisClient.HGet(ctx, coreServersHashKey, trimmedKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return CoreServer{}, ErrCoreServerNotFound
		}
		return CoreServer{}, fmt.Errorf("hget %s 失败（key=%s）: %w", coreServersHashKey, trimmedKey, err)
	}

	payload := strings.TrimSpace(raw)
	if payload == "" {
		return CoreServer{}, ErrCoreServerNotFound
	}

	var value coreServerValue
	if err := json.Unmarshal([]byte(payload), &value); err != nil {
		return CoreServer{}, fmt.Errorf("解析核心服务器失败（key=%s）: %w", trimmedKey, err)
	}

	return CoreServer{
		Key:  trimmedKey,
		IP:   value.IP,
		Port: value.Port,
	}, nil
}
