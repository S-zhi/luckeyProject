package dao

import (
	"context"
	"errors"
	"log/slog"
	"lucky_project/config"
	"lucky_project/entity"

	"gorm.io/gorm"
)

var (
	ErrDBNotInitialized = errors.New("gorm db 没有初始化")
	ErrInvalidID        = errors.New("传入的 ID 不合法")
	ErrNilEntity        = errors.New("实体对象 为 nil")
	ErrAlreadyExists    = errors.New("记录已经存储在")
)

const (
	defaultPageSize = 10
	maxPageSize     = 1000
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
