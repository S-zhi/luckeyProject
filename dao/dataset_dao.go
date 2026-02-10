package dao

import (
	"context"
	"fmt"
	entity2 "lucky_project/entity"
	"lucky_project/infrastructure/db"
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

func (d *DatasetDAO) Save(ctx context.Context, dataset *entity2.Dataset) error {
	if dataset == nil {
		return ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return fmt.Errorf("save dataset failed: %w", err)
	}
	return dbConn.Create(dataset).Error
}

func (d *DatasetDAO) FindByID(ctx context.Context, id uint) (*entity2.Dataset, error) {
	if id == 0 {
		return nil, ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return nil, fmt.Errorf("find dataset by id failed: %w", err)
	}

	var dataset entity2.Dataset
	err = dbConn.First(&dataset, id).Error
	return &dataset, err
}

func (d *DatasetDAO) FindAll(ctx context.Context, params entity2.QueryParams) ([]entity2.Dataset, int64, error) {
	var datasets []entity2.Dataset
	var total int64

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("find datasets failed: %w", err)
	}

	dbConn = dbConn.Model(&entity2.Dataset{})

	// 1. 基础模糊搜索
	if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
		dbConn = dbConn.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 2. 指标组合过滤
	if name := strings.TrimSpace(params.Name); name != "" {
		dbConn = dbConn.Where("name = ?", name)
	}

	if storageServer := strings.TrimSpace(params.StorageServer); storageServer != "" {
		dbConn = dbConn.Where("storage_server = ?", storageServer)
	}
	if taskType := strings.TrimSpace(params.TaskType); taskType != "" {
		dbConn = dbConn.Where("task_type = ?", taskType)
	} else if params.DatasetType != nil {
		if mappedTaskType := mapDatasetTypeToTaskType(*params.DatasetType); mappedTaskType != "" {
			dbConn = dbConn.Where("task_type = ?", mappedTaskType)
		}
	}
	if datasetFormat := strings.TrimSpace(params.DatasetFormat); datasetFormat != "" {
		dbConn = dbConn.Where("dataset_format = ?", datasetFormat)
	}
	if configPath := strings.TrimSpace(params.ConfigPath); configPath != "" {
		dbConn = dbConn.Where("config_path = ?", configPath)
	}
	if version := strings.TrimSpace(params.Version); version != "" {
		dbConn = dbConn.Where("version = ?", version)
	}
	if params.NumClasses != nil {
		dbConn = dbConn.Where("num_classes = ?", *params.NumClasses)
	}

	// 3. 获取总数
	err = dbConn.Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("count datasets failed: %w", err)
	}

	// 4. 执行分页查询 (默认 ID 降序)
	orderStr := "id DESC"
	sortValue := strings.ToLower(strings.TrimSpace(params.SizeSort))
	if sortValue == "" {
		sortValue = strings.ToLower(strings.TrimSpace(params.WeightSort))
	}
	switch sortValue {
	case "asc":
		orderStr = "size_mb ASC"
	case "desc":
		orderStr = "size_mb DESC"
	}

	offset, limit := pagination(params)
	err = dbConn.Order(orderStr).Offset(offset).Limit(limit).Find(&datasets).Error
	if err != nil {
		return nil, 0, fmt.Errorf("query datasets failed: %w", err)
	}

	return datasets, total, err
}

func mapDatasetTypeToTaskType(datasetType int8) string {
	switch datasetType {
	case 1:
		return "detect"
	case 2:
		return "segment"
	case 3:
		return "classify"
	case 4:
		return "pose"
	case 5:
		return "obb"
	default:
		return ""
	}
}
