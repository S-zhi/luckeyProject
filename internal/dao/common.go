package dao

import (
	"context"
	"errors"
	"lucky_project/internal/entity"

	"gorm.io/gorm"
)

var (
	ErrDBNotInitialized = errors.New("gorm db is not initialized")
	ErrInvalidID        = errors.New("invalid id")
	ErrNilEntity        = errors.New("entity is nil")
)

const (
	defaultPageSize = 10
	maxPageSize     = 100
)

func withContext(dbConn *gorm.DB, ctx context.Context) (*gorm.DB, error) {
	if dbConn == nil {
		return nil, ErrDBNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return dbConn.WithContext(ctx), nil
}

func normalizeQueryParams(params entity.QueryParams) entity.QueryParams {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = defaultPageSize
	}
	if params.PageSize > maxPageSize {
		params.PageSize = maxPageSize
	}
	return params
}

func pagination(params entity.QueryParams) (offset, limit int) {
	p := normalizeQueryParams(params)
	return (p.Page - 1) * p.PageSize, p.PageSize
}
