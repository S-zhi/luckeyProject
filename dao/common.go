package dao

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"lucky_project/config"
	"lucky_project/entity"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrDBNotInitialized = errors.New("gorm db 没有初始化")
	ErrInvalidID        = errors.New("传入的 ID 不合法")
	ErrNilEntity        = errors.New("实体对象 为 nil")
	ErrAlreadyExists    = errors.New("记录已经存储在")
	ErrInvalidAction    = errors.New("无效的 action，必须是 set/add/remove 之一")
)

const (
	// 默认分页参数
	defaultPageSize = 10
	// 最大分页参数
	maxPageSize = 1000

	StorageActionSet    = "set"
	StorageActionAdd    = "add"
	StorageActionRemove = "remove"
	// 本地存储
	storageLabelLocal = "backend"
	// 百度网盘
	storageBaiduNetDisk = "baidu_netdisk"
)

// daoLogger 获取Dao层日志器
func daoLogger() *slog.Logger {
	logger := config.GetLogger()
	if logger == nil {
		return slog.Default()
	}
	return logger.With("所属文件夹", "dao")
}

// withContext 安全增加上下文
func withContext(dbConn *gorm.DB, ctx context.Context) (*gorm.DB, error) {
	logger := daoLogger().With("func", "withContext")
	if dbConn == nil {
		logger.Error("数据库连接为空")
		return nil, ErrDBNotInitialized
	}
	if ctx == nil {
		logger.Debug("上下文为空，使用后台上下文")
		ctx = context.Background()
	}
	logger.Debug("绑定上下文到数据库")
	return dbConn.WithContext(ctx), nil
}

// normalizeQueryParams 规范查询参数
func normalizeQueryParams(params entity.QueryParams) entity.QueryParams {
	logger := daoLogger().With("func", "normalizeQueryParams")
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = defaultPageSize
	}
	if params.PageSize > maxPageSize {
		params.PageSize = maxPageSize
	}
	logger.Debug("查询参数已规范化", "page", params.Page, "page_size", params.PageSize)
	return params
}

// pagination 返回分页参数 offset, limit
func pagination(params entity.QueryParams) (offset, limit int) {
	logger := daoLogger().With("func", "pagination")
	p := normalizeQueryParams(params)
	offset, limit = (p.Page-1)*p.PageSize, p.PageSize
	logger.Debug("分页参数已生成", "offset", offset, "limit", limit)
	return offset, limit
}

// normalizeStorageServers 对存储服务列表进行“标准化 + 去重 + 过滤无效项”
func normalizeStorageServers(servers []string) []string {
	logger := daoLogger().With("func", "normalizeStorageServers")
	seen := make(map[string]struct{}, len(servers))
	result := make([]string, 0, len(servers))
	for _, server := range servers {
		label, ok := toStorageLabel(server)
		if !ok {
			continue
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		result = append(result, label)
	}
	logger.Debug("存储服务已规范化", "input", len(servers), "output", len(result))
	return result
}

// toStorageLabel 获取存储服务标签
func toStorageLabel(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}

	switch strings.ToLower(value) {
	case storageBaiduNetDisk, "baidu", "baidu-pan", "baidu_pan", "baidupan", "pan.baidu", "百度网盘":
		return storageBaiduNetDisk, true
	case storageLabelLocal, "本地存储", "Local", "LOCAL":
		return storageLabelLocal, true
	default:
		return value, true
	}
}

// parseStorageServerValue 解析存储服务为数组
func parseStorageServerValue(raw string) []string {
	logger := daoLogger().With("func", "parseStorageServerValue")
	value := strings.TrimSpace(raw)
	if value == "" {
		return []string{}
	}

	var arrayValue []string
	if err := json.Unmarshal([]byte(value), &arrayValue); err == nil {
		normalized := normalizeStorageServers(arrayValue)
		logger.Debug("存储服务解析为数组", "count", len(normalized))
		return normalized
	}

	var singleValue string
	if err := json.Unmarshal([]byte(value), &singleValue); err == nil {
		normalized := normalizeStorageServers([]string{singleValue})
		logger.Debug("存储服务解析为单个JSON字符串", "count", len(normalized))
		return normalized
	}

	normalized := normalizeStorageServers([]string{value})
	logger.Debug("存储服务解析为普通字符串", "count", len(normalized))
	return normalized
}

// encodeStorageServerValue 编码存储服务为JSON字符串
func encodeStorageServerValue(servers []string) (string, error) {
	logger := daoLogger().With("func", "encodeStorageServerValue")
	normalized := normalizeStorageServers(servers)
	bytes, err := json.Marshal(normalized)
	if err != nil {
		logger.Error("编码存储服务失败", "error", err)
		return "", err
	}
	logger.Debug("存储服务编码完成", "count", len(normalized))
	return string(bytes), nil
}

// applyStorageServerAction 检查存储服务器是否存活，或者有无重新拉活的服务
func applyStorageServerAction(current []string, action string, incoming []string) ([]string, error) {
	logger := daoLogger().With("func", "applyStorageServerAction")
	normalizedCurrent := normalizeStorageServers(current)
	normalizedIncoming := normalizeStorageServers(incoming)

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "", StorageActionSet:
		logger.Debug("应用存储动作：设置", "incoming", len(normalizedIncoming))
		return normalizedIncoming, nil
	case StorageActionAdd:
		result := append(append([]string{}, normalizedCurrent...), normalizedIncoming...)
		merged := normalizeStorageServers(result)
		logger.Debug("应用存储动作：追加", "before", len(normalizedCurrent), "incoming", len(normalizedIncoming), "after", len(merged))
		return merged, nil
	case StorageActionRemove:
		removeSet := make(map[string]struct{}, len(normalizedIncoming))
		for _, server := range normalizedIncoming {
			removeSet[server] = struct{}{}
		}

		result := make([]string, 0, len(normalizedCurrent))
		for _, server := range normalizedCurrent {
			if _, ok := removeSet[server]; ok {
				continue
			}
			result = append(result, server)
		}
		logger.Debug("应用存储动作：移除", "before", len(normalizedCurrent), "incoming", len(normalizedIncoming), "after", len(result))
		return result, nil
	default:
		logger.Warn("应用存储动作失败：动作无效", "action", action)
		return nil, ErrInvalidAction
	}
}

// isDuplicateKeyError 判断是否为重复键错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate entry") ||
		strings.Contains(text, "error 1062") ||
		strings.Contains(text, "duplicated key")
}
