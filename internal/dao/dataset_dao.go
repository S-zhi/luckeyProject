package dao

import (
	"context"
	"fmt"
	"lucky_project/internal/entity"
	"lucky_project/pkg/db"
	"strings"

	"gorm.io/gorm"
)

type DatasetDAO struct {
	DB *gorm.DB
}

func NewDatasetDAO() *DatasetDAO {
	return &DatasetDAO{
		DB: db.DB,
	}
}

func (d *DatasetDAO) Save(ctx context.Context, dataset *entity.Dataset) error {
	if dataset == nil {
		return ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return fmt.Errorf("save dataset failed: %w", err)
	}
	return dbConn.Create(dataset).Error
}

func (d *DatasetDAO) FindByID(ctx context.Context, id uint) (*entity.Dataset, error) {
	if id == 0 {
		return nil, ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return nil, fmt.Errorf("find dataset by id failed: %w", err)
	}

	var dataset entity.Dataset
	err = dbConn.First(&dataset, id).Error
	return &dataset, err
}

func (d *DatasetDAO) FindAll(ctx context.Context, params entity.QueryParams) ([]entity.Dataset, int64, error) {
	var datasets []entity.Dataset
	var total int64

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("find datasets failed: %w", err)
	}

	dbConn = dbConn.Model(&entity.Dataset{})

	// 1. 基础模糊搜索
	if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
		dbConn = dbConn.Where("dataset_name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 2. 指标组合过滤
	if name := strings.TrimSpace(params.Name); name != "" {
		dbConn = dbConn.Where("dataset_name = ?", name)
	}
	if params.DatasetType != nil {
		dbConn = dbConn.Where("dataset_type = ?", *params.DatasetType)
	}
	if params.IsLatest != nil {
		dbConn = dbConn.Where("is_latest = ?", *params.IsLatest)
	}
	if params.StorageType != nil {
		dbConn = dbConn.Where("storage_type = ?", *params.StorageType)
	}
	if params.AnnotationType != nil {
		dbConn = dbConn.Where("annotation_type = ?", *params.AnnotationType)
	}

	// 3. 获取总数
	err = dbConn.Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("count datasets failed: %w", err)
	}

	// 4. 执行分页查询 (默认 ID 降序)
	offset, limit := pagination(params)
	err = dbConn.Order("id DESC").Offset(offset).Limit(limit).Find(&datasets).Error
	if err != nil {
		return nil, 0, fmt.Errorf("query datasets failed: %w", err)
	}

	return datasets, total, err
}
