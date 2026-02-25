package service

import (
	"context"
	"errors"
	"lucky_project/config"
	"lucky_project/dao/redisDao"
	"sort"
	"strconv"
)

var ErrRedisNotInitialized = errors.New("Redis 客户端未初始化")
var ErrCoreServerKeyRequired = errors.New("核心服务器 key 不能为空")
var ErrCoreServerNotFound = errors.New("未找到核心服务器")

type CoreServer struct {
	Key  string `json:"key"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func buildCoreServerFromStorageState(item redisDao.StorageService) (CoreServer, bool) {
	port, _ := strconv.Atoi(item.Port)
	return CoreServer{
		Key:  item.Name,
		IP:   item.IP,
		Port: port,
	}, true
}

func ListStorageServers(ctx context.Context) ([]redisDao.StorageService, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	storageServices := redisDao.GetStorageStateList()
	result := make([]redisDao.StorageService, 0, len(storageServices))
	for _, item := range storageServices {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == config.StorageBackEndLabel {
			return true
		} else if result[j].Name == config.StorageBackEndLabel {
			return false
		} else if result[i].Name == config.StorageBaiduNetdiskLabel {
			return true
		} else if result[j].Name == config.StorageBaiduNetdiskLabel {
			return false
		} else {
			return result[i].Name < result[j].Name
		}
	})

	return result, nil
}

func GetCoreServerByKey(ctx context.Context, name string) (redisDao.StorageService, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	servers, err := ListStorageServers(ctx)
	if err != nil {
		return redisDao.StorageService{}, err
	}

	for _, server := range servers {
		if server.Name == name {
			return server, nil
		}
	}
	return redisDao.StorageService{}, ErrCoreServerNotFound
}
