package redisDao

import (
	"context"
	"errors"
	"fmt"
	"lucky_project/config"

	"github.com/redis/go-redis/v9"
)

// getStorageServiceList 获取存储服务列表
func getStorageServiceList() []string {
	if config.RedisClient == nil {
		return []string{}
	}
	ctx := context.Background()
	storageServiceList := config.RedisClient.LRange(ctx, "storage_service_list", 0, config.MaxReturnStorageServicesNumber)
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
func AddStorageService(storageServiceName string, storageServiceIP string, storageServicePort string) {
	ctx := context.Background()
	storageServiceIPKey := storageServiceName + "_ip"
	storageServicePortKey := storageServiceName + "_port"
	storageServiceStateKey := storageServiceName
	config.RedisClient.RPush(ctx, "storage_service_list", storageServiceName)
	config.RedisClient.Set(ctx, storageServiceIPKey, storageServiceIP, 0)
	config.RedisClient.Set(ctx, storageServicePortKey, storageServicePort, 0)
	config.RedisClient.Set(ctx, storageServiceStateKey, "unknown", 0)
	config.GetLogger().Info("添加存储服务完成: storageServiceName: %s , storageServiceIP:%s , storageServicePort:%s", storageServiceName, storageServiceIP, storageServicePort)
}

// UpdateStorageService 更新存储服务
func UpdateStorageService(oldStorageServiceName string, storageServiceName string, storageServiceIP string, storageServicePort string) {
	ctx := context.Background()
	storageServiceIPKey := storageServiceName + "_ip"
	storageServicePortKey := storageServiceName + "_port"
	storageServiceStateKey := storageServiceName
	config.RedisClient.LRem(ctx, "storage_service_list", 1, oldStorageServiceName)
	config.RedisClient.RPush(ctx, "storage_service_list", storageServiceName)
	config.RedisClient.Set(ctx, storageServiceIPKey, storageServiceIP, 0)
	config.RedisClient.Set(ctx, storageServicePortKey, storageServicePort, 0)
	config.RedisClient.Set(ctx, storageServiceStateKey, "unknown", 0)
	config.GetLogger().Info("更新存储服务完成: storageServiceName: %s , storageServiceIP:%s , storageServicePort:%s", storageServiceName, storageServiceIP, storageServicePort)
}
