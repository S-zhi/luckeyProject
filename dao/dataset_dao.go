package dao

import (
	"context"
	"fmt"
	"lucky_project/config"
	entity2 "lucky_project/entity"
	"path/filepath"
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
	if strings.TrimSpace(dataset.FileName) == "" {
		legacy := strings.TrimSpace(strings.ReplaceAll(dataset.DatasetPath, "\\", "/"))
		if legacy != "" {
			derived := strings.TrimSpace(filepath.Base(legacy))
			if derived != "" && derived != "." && derived != string(filepath.Separator) {
				dataset.FileName = derived
			}
		}
	}

	normalizedStorageServer, err := encodeStorageServerValue(parseStorageServerValue(dataset.StorageServer))
	if err != nil {
		logger.Error("save dataset failed: normalize storage server", "name", dataset.Name, "error", err)
		return fmt.Errorf("save dataset failed: %w", err)
	}
	dataset.StorageServer = normalizedStorageServer

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

// FindFileNameByID 根据主键查询数据集文件名。
func (d *DatasetDAO) FindFileNameByID(ctx context.Context, id uint) (string, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "FindFileNameByID")
	if id == 0 {
		logger.Warn("find dataset file_name skipped: invalid id", "id", id)
		return "", ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("find dataset file_name failed: with context", "id", id, "error", err)
		return "", fmt.Errorf("find dataset file_name failed: %w", err)
	}

	var row struct {
		FileName string `gorm:"column:file_name"`
	}
	if err := dbConn.Model(&entity2.Dataset{}).Select("file_name").Where("id = ?", id).Take(&row).Error; err != nil {
		logger.Error("find dataset file_name failed: db query", "id", id, "error", err)
		return "", err
	}

	fileName := strings.TrimSpace(row.FileName)
	if fileName == "" {
		logger.Warn("find dataset file_name empty", "id", id)
		return "", ErrNilEntity
	}
	logger.Info("find dataset file_name success", "id", id, "file_name", fileName)
	return fileName, nil
}

// FindByName 根据名称查询单条数据集记录。
func (d *DatasetDAO) FindByName(ctx context.Context, name string) (*entity2.Dataset, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "FindByName")
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		logger.Warn("find dataset by name skipped: empty name")
		return nil, ErrNilEntity
	}
	logger.Info("find dataset by name begin", "name", trimmed)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("find dataset by name failed: with context", "name", trimmed, "error", err)
		return nil, fmt.Errorf("find dataset by name failed: %w", err)
	}

	var dataset entity2.Dataset
	err = dbConn.Where("name = ?", trimmed).Take(&dataset).Error
	if err != nil {
		logger.Error("find dataset by name failed: db query", "name", trimmed, "error", err)
		return nil, err
	}

	logger.Info("find dataset by name success", "id", dataset.ID, "name", dataset.Name)
	return &dataset, nil
}

// GetStorageServersByID 查询数据集的 storage_server 列并统一返回数组格式。
func (d *DatasetDAO) GetStorageServersByID(ctx context.Context, id uint) ([]string, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "GetStorageServersByID")
	if id == 0 {
		logger.Warn("get dataset storage server skipped: invalid id", "id", id)
		return nil, ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("get dataset storage server failed: with context", "id", id, "error", err)
		return nil, fmt.Errorf("get dataset storage server failed: %w", err)
	}

	var row struct {
		StorageServer string `gorm:"column:storage_server"`
	}
	if err := dbConn.Model(&entity2.Dataset{}).Select("storage_server").Where("id = ?", id).Take(&row).Error; err != nil {
		logger.Error("get dataset storage server failed: db query", "id", id, "error", err)
		return nil, err
	}

	servers := parseStorageServerValue(row.StorageServer)
	logger.Info("get dataset storage server success", "id", id, "count", len(servers))
	return servers, nil
}

// UpdateStorageServersByID 按 action(set/add/remove) 更新数据集 storage_server。
func (d *DatasetDAO) UpdateStorageServersByID(ctx context.Context, id uint, action string, servers []string) ([]string, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "UpdateStorageServersByID")
	if id == 0 {
		logger.Warn("update dataset storage server skipped: invalid id", "id", id)
		return nil, ErrInvalidID
	}

	current, err := d.GetStorageServersByID(ctx, id)
	if err != nil {
		logger.Error("update dataset storage server failed: load current", "id", id, "error", err)
		return nil, err
	}

	next, err := applyStorageServerAction(current, action, servers)
	if err != nil {
		logger.Error("update dataset storage server failed: apply action", "id", id, "action", action, "error", err)
		return nil, err
	}

	encoded, err := encodeStorageServerValue(next)
	if err != nil {
		logger.Error("update dataset storage server failed: encode", "id", id, "error", err)
		return nil, fmt.Errorf("update dataset storage server failed: %w", err)
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("update dataset storage server failed: with context", "id", id, "error", err)
		return nil, fmt.Errorf("update dataset storage server failed: %w", err)
	}

	if err := dbConn.Model(&entity2.Dataset{}).Where("id = ?", id).Update("storage_server", encoded).Error; err != nil {
		logger.Error("update dataset storage server failed: db update", "id", id, "error", err)
		return nil, fmt.Errorf("update dataset storage server failed: %w", err)
	}

	logger.Info("update dataset storage server success", "id", id, "action", action, "count", len(next))
	return next, nil
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
		dbConn = dbConn.Where(
			"(storage_server = ?) OR (JSON_VALID(storage_server) AND JSON_CONTAINS(storage_server, JSON_QUOTE(?)))",
			storageServer,
			storageServer,
		)
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
