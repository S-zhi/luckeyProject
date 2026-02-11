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
	ErrInvalidAction    = errors.New("invalid action, must be one of: set/add/remove")
)

const (
	defaultPageSize = 10
	maxPageSize     = 1000

	StorageActionSet    = "set"
	StorageActionAdd    = "add"
	StorageActionRemove = "remove"
)

func daoLogger() *slog.Logger {
	logger := config.EnsureLoggerInitialized()
	if logger == nil {
		return slog.Default()
	}
	return logger.With("layer", "dao")
}

// withContext 安全增加上下文
func withContext(dbConn *gorm.DB, ctx context.Context) (*gorm.DB, error) {
	logger := daoLogger().With("func", "withContext")
	if dbConn == nil {
		logger.Error("db is nil")
		return nil, ErrDBNotInitialized
	}
	if ctx == nil {
		logger.Debug("context is nil, use background")
		ctx = context.Background()
	}
	logger.Debug("bind context to db")
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
	logger.Debug("query params normalized", "page", params.Page, "page_size", params.PageSize)
	return params
}

// 返回分页参数
func pagination(params entity.QueryParams) (offset, limit int) {
	logger := daoLogger().With("func", "pagination")
	p := normalizeQueryParams(params)
	offset, limit = (p.Page-1)*p.PageSize, p.PageSize
	logger.Debug("pagination generated", "offset", offset, "limit", limit)
	return offset, limit
}

func normalizeStorageServers(servers []string) []string {
	logger := daoLogger().With("func", "normalizeStorageServers")
	seen := make(map[string]struct{}, len(servers))
	result := make([]string, 0, len(servers))
	for _, server := range servers {
		value := strings.TrimSpace(server)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	logger.Debug("normalized storage servers", "input", len(servers), "output", len(result))
	return result
}

func parseStorageServerValue(raw string) []string {
	logger := daoLogger().With("func", "parseStorageServerValue")
	value := strings.TrimSpace(raw)
	if value == "" {
		return []string{}
	}

	var arrayValue []string
	if err := json.Unmarshal([]byte(value), &arrayValue); err == nil {
		normalized := normalizeStorageServers(arrayValue)
		logger.Debug("parsed storage server as array", "count", len(normalized))
		return normalized
	}

	var singleValue string
	if err := json.Unmarshal([]byte(value), &singleValue); err == nil {
		normalized := normalizeStorageServers([]string{singleValue})
		logger.Debug("parsed storage server as single json string", "count", len(normalized))
		return normalized
	}

	normalized := normalizeStorageServers([]string{value})
	logger.Debug("parsed storage server as plain string", "count", len(normalized))
	return normalized
}

func encodeStorageServerValue(servers []string) (string, error) {
	logger := daoLogger().With("func", "encodeStorageServerValue")
	normalized := normalizeStorageServers(servers)
	bytes, err := json.Marshal(normalized)
	if err != nil {
		logger.Error("encode storage server failed", "error", err)
		return "", err
	}
	logger.Debug("encoded storage server", "count", len(normalized))
	return string(bytes), nil
}

func applyStorageServerAction(current []string, action string, incoming []string) ([]string, error) {
	logger := daoLogger().With("func", "applyStorageServerAction")
	normalizedCurrent := normalizeStorageServers(current)
	normalizedIncoming := normalizeStorageServers(incoming)

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "", StorageActionSet:
		logger.Debug("apply action set", "incoming", len(normalizedIncoming))
		return normalizedIncoming, nil
	case StorageActionAdd:
		result := append(append([]string{}, normalizedCurrent...), normalizedIncoming...)
		merged := normalizeStorageServers(result)
		logger.Debug("apply action add", "before", len(normalizedCurrent), "incoming", len(normalizedIncoming), "after", len(merged))
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
		logger.Debug("apply action remove", "before", len(normalizedCurrent), "incoming", len(normalizedIncoming), "after", len(result))
		return result, nil
	default:
		logger.Warn("apply action failed: invalid action", "action", action)
		return nil, ErrInvalidAction
	}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate entry") ||
		strings.Contains(text, "error 1062") ||
		strings.Contains(text, "duplicated key")
}
