package service

import (
	"context"
	"errors"
	"lucky_project/config"
	"lucky_project/dao/redisDao"
	"sort"
	"strconv"
	"strings"
)

var ErrRedisNotInitialized = errors.New("Redis 客户端未初始化")
var ErrCoreServerKeyRequired = errors.New("核心服务器 key 不能为空")
var ErrCoreServerIPRequired = errors.New("核心服务器 ip 不能为空")
var ErrCoreServerPortRequired = errors.New("核心服务器 port 不能为空")
var ErrCoreServerPortInvalid = errors.New("核心服务器 port 必须是 1-65535 之间的整数")
var ErrCoreServerNotFound = errors.New("未找到核心服务器")
var ErrCoreServerAlreadyExists = errors.New("核心服务器已存在")

func ListStorageServers(ctx context.Context) ([]redisDao.StorageService, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if config.RedisClient == nil {
		return nil, ErrRedisNotInitialized
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
	key := strings.TrimSpace(name)
	if key == "" {
		return redisDao.StorageService{}, ErrCoreServerKeyRequired
	}

	servers, err := ListStorageServers(ctx)
	if err != nil {
		return redisDao.StorageService{}, err
	}

	for _, server := range servers {
		if server.Name == key {
			return server, nil
		}
	}
	return redisDao.StorageService{}, ErrCoreServerNotFound
}

func CreateCoreServer(ctx context.Context, key string, ip string, port string) (redisDao.StorageService, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	coreServerKey := strings.TrimSpace(key)
	if coreServerKey == "" {
		return redisDao.StorageService{}, ErrCoreServerKeyRequired
	}

	coreServerIP := strings.TrimSpace(ip)
	if coreServerIP == "" {
		return redisDao.StorageService{}, ErrCoreServerIPRequired
	}

	coreServerPort, err := normalizeCoreServerPort(port)
	if err != nil {
		return redisDao.StorageService{}, err
	}

	if config.RedisClient == nil {
		return redisDao.StorageService{}, ErrRedisNotInitialized
	}

	_, err = GetCoreServerByKey(ctx, coreServerKey)
	switch {
	case err == nil:
		return redisDao.StorageService{}, ErrCoreServerAlreadyExists
	case errors.Is(err, ErrCoreServerNotFound):
	default:
		return redisDao.StorageService{}, err
	}

	if err := redisDao.AddStorageService(coreServerKey, coreServerIP, coreServerPort); err != nil {
		return redisDao.StorageService{}, err
	}

	return redisDao.StorageService{
		Name:  coreServerKey,
		IP:    coreServerIP,
		Port:  coreServerPort,
		State: "unknown",
	}, nil
}

func UpdateCoreServer(ctx context.Context, oldKey string, newKey string, ip string, port string) (redisDao.StorageService, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	coreServerOldKey := strings.TrimSpace(oldKey)
	if coreServerOldKey == "" {
		return redisDao.StorageService{}, ErrCoreServerKeyRequired
	}

	if config.RedisClient == nil {
		return redisDao.StorageService{}, ErrRedisNotInitialized
	}

	current, err := GetCoreServerByKey(ctx, coreServerOldKey)
	if err != nil {
		return redisDao.StorageService{}, err
	}

	coreServerNewKey := strings.TrimSpace(newKey)
	if coreServerNewKey == "" {
		coreServerNewKey = current.Name
	}

	coreServerIP := strings.TrimSpace(ip)
	if coreServerIP == "" {
		coreServerIP = current.IP
	}
	if coreServerIP == "" {
		return redisDao.StorageService{}, ErrCoreServerIPRequired
	}

	coreServerPort := strings.TrimSpace(port)
	if coreServerPort == "" {
		coreServerPort = current.Port
	} else {
		normalizedPort, err := normalizeCoreServerPort(coreServerPort)
		if err != nil {
			return redisDao.StorageService{}, err
		}
		coreServerPort = normalizedPort
	}
	if coreServerPort == "" {
		return redisDao.StorageService{}, ErrCoreServerPortRequired
	}

	if coreServerNewKey != coreServerOldKey {
		_, err = GetCoreServerByKey(ctx, coreServerNewKey)
		switch {
		case err == nil:
			return redisDao.StorageService{}, ErrCoreServerAlreadyExists
		case errors.Is(err, ErrCoreServerNotFound):
		default:
			return redisDao.StorageService{}, err
		}
	}

	if err := redisDao.UpdateStorageService(coreServerOldKey, coreServerNewKey, coreServerIP, coreServerPort); err != nil {
		return redisDao.StorageService{}, err
	}

	return redisDao.StorageService{
		Name:  coreServerNewKey,
		IP:    coreServerIP,
		Port:  coreServerPort,
		State: current.State,
	}, nil
}

func DeleteCoreServer(ctx context.Context, key string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	coreServerKey := strings.TrimSpace(key)
	if coreServerKey == "" {
		return ErrCoreServerKeyRequired
	}

	if config.RedisClient == nil {
		return ErrRedisNotInitialized
	}

	if _, err := GetCoreServerByKey(ctx, coreServerKey); err != nil {
		return err
	}

	return redisDao.DeleteStorageService(coreServerKey)
}

func normalizeCoreServerPort(port string) (string, error) {
	value := strings.TrimSpace(port)
	if value == "" {
		return "", ErrCoreServerPortRequired
	}

	number, err := strconv.Atoi(value)
	if err != nil || number < 1 || number > 65535 {
		return "", ErrCoreServerPortInvalid
	}
	return strconv.Itoa(number), nil
}
