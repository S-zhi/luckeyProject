package redisDao

import (
	"context"
	"errors"
	"fmt"
	"lucky_project/config"

	"github.com/redis/go-redis/v9"
)

var ErrRedisClientNotInitialized = errors.New("Redis 客户端未初始化")

const storageServiceListKey = "storage_service_list"

// getStorageServiceList 获取存储服务列表
func getStorageServiceList() []string {
	if config.RedisClient == nil {
		return []string{}
	}
	ctx := context.Background()
	storageServiceList := config.RedisClient.LRange(ctx, storageServiceListKey, 0, config.MaxReturnStorageServicesNumber)
	return storageServiceList.Val()
}

// StorageService 存储服务结构体
type StorageService struct {
	Name  string
	IP    string
	Port  string
	State string
}

func (s StorageService) String() string {
	return fmt.Sprintf("名称: %s, IP: %s, 端口: %s, 状态: %s", s.Name, s.IP, s.Port, s.State)
}

// checkSelectResult 检查查询结果

func checkSelectResult(strCmd *redis.StringCmd) string {
	if strCmd.Val() == "" || errors.Is(strCmd.Err(), redis.Nil) {
		return "unknown"
	}
	return strCmd.Val()
}

// getStorageCloudServiceIPAndPost 获取云存储服务IP和端口
func getStorageCloudServiceIPAndPostAndState(storageYunServiceName string) (string, string, string) {
	if config.RedisClient == nil {
		return "unknown", "unknown", "unknown"
	}
	storageCloudServiceIPKey := storageYunServiceName + "_ip"
	storageCloudServiceIP := config.RedisClient.Get(context.Background(), storageCloudServiceIPKey)
	storageCloudServicePostKey := storageYunServiceName + "_port"
	storageCloudServicePost := config.RedisClient.Get(context.Background(), storageCloudServicePostKey)
	storageCloudServiceStateKey := storageYunServiceName
	storageCloudServiceState := config.RedisClient.Get(context.Background(), storageCloudServiceStateKey)
	return checkSelectResult(storageCloudServiceIP), checkSelectResult(storageCloudServicePost), checkSelectResult(storageCloudServiceState)
}

// GetStorageStateList 获取存储服务状态列表
func GetStorageStateList() []StorageService {
	storageCloudServiceList := getStorageServiceList()
	storageServiceList := make([]StorageService, 0)
	for i := 0; i < len(storageCloudServiceList); i++ {
		storageCloudServiceIP, storageCloudServicePost, storageCloudServiceState := getStorageCloudServiceIPAndPostAndState(storageCloudServiceList[i])
		storageService := StorageService{
			Name:  storageCloudServiceList[i],
			IP:    storageCloudServiceIP,
			Port:  storageCloudServicePost,
			State: storageCloudServiceState,
		}
		storageServiceList = append(storageServiceList, storageService)
	}
	config.GetLogger().Info("存储服务列表:")
	for i := 0; i < len(storageServiceList); i++ {
		config.GetLogger().Info(storageServiceList[i].String())
	}
	return storageServiceList
}

// AddStorageService 添加存储服务
func AddStorageService(storageServiceName string, storageServiceIP string, storageServicePort string) error {
	if config.RedisClient == nil {
		return ErrRedisClientNotInitialized
	}
	ctx := context.Background()
	storageServiceIPKey := storageServiceName + "_ip"
	storageServicePortKey := storageServiceName + "_port"
	storageServiceStateKey := storageServiceName

	pipe := config.RedisClient.TxPipeline()
	pipe.RPush(ctx, storageServiceListKey, storageServiceName)
	pipe.Set(ctx, storageServiceIPKey, storageServiceIP, 0)
	pipe.Set(ctx, storageServicePortKey, storageServicePort, 0)
	pipe.Set(ctx, storageServiceStateKey, "unknown", 0)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	if logger := config.GetLogger(); logger != nil {
		logger.Info(
			"添加存储服务完成",
			"storage_service_name", storageServiceName,
			"storage_service_ip", storageServiceIP,
			"storage_service_port", storageServicePort,
		)
	}
	return nil
}

// UpdateStorageService 更新存储服务
func UpdateStorageService(oldStorageServiceName string, storageServiceName string, storageServiceIP string, storageServicePort string) error {
	if config.RedisClient == nil {
		return ErrRedisClientNotInitialized
	}
	ctx := context.Background()
	storageServiceIPKey := storageServiceName + "_ip"
	storageServicePortKey := storageServiceName + "_port"
	storageServiceStateKey := storageServiceName
	oldStorageServiceIPKey := oldStorageServiceName + "_ip"
	oldStorageServicePortKey := oldStorageServiceName + "_port"
	oldStorageServiceStateKey := oldStorageServiceName

	pipe := config.RedisClient.TxPipeline()
	pipe.LRem(ctx, storageServiceListKey, 0, oldStorageServiceName)
	pipe.RPush(ctx, storageServiceListKey, storageServiceName)
	pipe.Set(ctx, storageServiceIPKey, storageServiceIP, 0)
	pipe.Set(ctx, storageServicePortKey, storageServicePort, 0)
	pipe.Set(ctx, storageServiceStateKey, "unknown", 0)
	if oldStorageServiceName != storageServiceName {
		pipe.Del(ctx, oldStorageServiceIPKey, oldStorageServicePortKey, oldStorageServiceStateKey)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	if logger := config.GetLogger(); logger != nil {
		logger.Info(
			"更新存储服务完成",
			"old_storage_service_name", oldStorageServiceName,
			"storage_service_name", storageServiceName,
			"storage_service_ip", storageServiceIP,
			"storage_service_port", storageServicePort,
		)
	}
	return nil
}

// DeleteStorageService 删除存储服务
func DeleteStorageService(storageServiceName string) error {
	if config.RedisClient == nil {
		return ErrRedisClientNotInitialized
	}

	ctx := context.Background()
	storageServiceIPKey := storageServiceName + "_ip"
	storageServicePortKey := storageServiceName + "_port"
	storageServiceStateKey := storageServiceName

	pipe := config.RedisClient.TxPipeline()
	pipe.LRem(ctx, storageServiceListKey, 0, storageServiceName)
	pipe.Del(ctx, storageServiceIPKey, storageServicePortKey, storageServiceStateKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	if logger := config.GetLogger(); logger != nil {
		logger.Info(
			"删除存储服务完成",
			"storage_service_name", storageServiceName,
		)
	}
	return nil
}
