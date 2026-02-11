package dao

import (
	"context"
	"fmt"
	"lucky_project/config"
	entity2 "lucky_project/entity"
	"strings"

	"gorm.io/gorm"
)

type DatasetDAO struct {
	DB *gorm.DB
}

func NewDatasetDAO() *DatasetDAO {
	daoLogger().With("dao", "DatasetDAO", "method", "NewDatasetDAO").Info("init dataset dao")
	return &DatasetDAO{
		DB: config.DB,
	}
}

// Save 保存数据集
func (d *DatasetDAO) Save(ctx context.Context, dataset *entity2.Dataset) error {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "Save")
	if dataset == nil {
		logger.Warn("save dataset skipped: dataset is nil")
		return ErrNilEntity
	}
	logger.Info("save dataset begin", "name", dataset.Name)
	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("save dataset failed: with context", "error", err)
		return fmt.Errorf("save dataset failed: %w", err)
	}
	if err := dbConn.Create(dataset).Error; err != nil {
		logger.Error("save dataset failed: db create", "error", err)
		return fmt.Errorf("save dataset failed: %w", err)
	}
	logger.Info("save dataset success", "id", dataset.ID, "name", dataset.Name)
	return nil
}

func (d *DatasetDAO) FindByID(ctx context.Context, id uint) (*entity2.Dataset, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "FindByID")
	if id == 0 {
		logger.Warn("find dataset skipped: invalid id", "id", id)
		return nil, ErrInvalidID
	}
	logger.Info("find dataset begin", "id", id)
	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("find dataset failed: with context", "id", id, "error", err)
		return nil, fmt.Errorf("find dataset by id failed: %w", err)
	}
	var dataset entity2.Dataset
	err = dbConn.First(&dataset, id).Error
	if err != nil {
		logger.Error("find dataset failed: db query", "id", id, "error", err)
		return nil, err
	}
	logger.Info("find dataset success", "id", dataset.ID, "name", dataset.Name)
	return &dataset, nil
}

func (d *DatasetDAO) FindAll(ctx context.Context, params entity2.QueryParams) ([]entity2.Dataset, int64, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "FindAll")
	var datasets []entity2.Dataset
	var total int64
	logger.Info("find datasets begin",
		"page", params.Page,
		"page_size", params.PageSize,
		"name", params.Name,
		"keyword", params.Keyword,
		"storage_server", params.StorageServer,
		"task_type", params.TaskType,
	)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("find datasets failed: with context", "error", err)
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
		logger.Error("count datasets failed", "error", err)
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
		logger.Error("query datasets failed", "error", err)
		return nil, 0, fmt.Errorf("query datasets failed: %w", err)
	}

	logger.Info("find datasets success", "total", total, "returned", len(datasets))
	return datasets, total, err
}

func mapDatasetTypeToTaskType(datasetType int8) string {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "mapDatasetTypeToTaskType")
	switch datasetType {
	case 1:
		logger.Debug("map dataset type", "dataset_type", datasetType, "task_type", "detect")
		return "detect"
	case 2:
		logger.Debug("map dataset type", "dataset_type", datasetType, "task_type", "segment")
		return "segment"
	case 3:
		logger.Debug("map dataset type", "dataset_type", datasetType, "task_type", "classify")
		return "classify"
	case 4:
		logger.Debug("map dataset type", "dataset_type", datasetType, "task_type", "pose")
		return "pose"
	case 5:
		logger.Debug("map dataset type", "dataset_type", datasetType, "task_type", "obb")
		return "obb"
	default:
		logger.Warn("unknown dataset type", "dataset_type", datasetType)
		return "unknown"
	}
}
