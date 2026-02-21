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
	daoLogger().With("dao", "DatasetDAO", "method", "NewDatasetDAO").Info("初始化数据集DAO")
	return &DatasetDAO{
		DB: config.DB,
	}
}

// Save 保存数据集
func (d *DatasetDAO) Save(ctx context.Context, dataset *entity2.Dataset) error {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "Save")
	if dataset == nil {
		logger.Warn("保存数据集已跳过：数据集为空")
		return ErrNilEntity
	}
	logger.Info("保存数据集开始", "name", dataset.Name)
	fileName := strings.TrimSpace(filepath.Base(dataset.FileName))
	if fileName == "" || fileName == "." || fileName == string(filepath.Separator) {
		legacy := strings.TrimSpace(strings.ReplaceAll(dataset.DatasetPath, "\\", "/"))
		if legacy != "" {
			derived := strings.TrimSpace(filepath.Base(legacy))
			if derived != "" && derived != "." && derived != string(filepath.Separator) {
				fileName = derived
			}
		}
	}
	if fileName == "" || fileName == "." || fileName == string(filepath.Separator) {
		logger.Warn("保存数据集已跳过：文件名为空")
		return ErrNilEntity
	}
	dataset.FileName = fileName

	normalizedStorageServer, err := encodeStorageServerValue(parseStorageServerValue(dataset.StorageServer))
	if err != nil {
		logger.Error("保存数据集失败：存储服务标准化失败", "name", dataset.Name, "error", err)
		return fmt.Errorf("保存数据集失败: %w", err)
	}
	dataset.StorageServer = normalizedStorageServer

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("保存数据集失败：绑定上下文失败", "error", err)
		return fmt.Errorf("保存数据集失败: %w", err)
	}
	if err := dbConn.Create(dataset).Error; err != nil {
		logger.Error("保存数据集失败：数据库创建失败", "error", err)
		return fmt.Errorf("保存数据集失败: %w", err)
	}
	logger.Info("保存数据集成功", "id", dataset.ID, "name", dataset.Name)
	return nil
}

func (d *DatasetDAO) FindByID(ctx context.Context, id uint) (*entity2.Dataset, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "FindByID")
	if id == 0 {
		logger.Warn("查询数据集已跳过：ID无效", "id", id)
		return nil, ErrInvalidID
	}
	logger.Info("查询数据集开始", "id", id)
	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("查询数据集失败：绑定上下文失败", "id", id, "error", err)
		return nil, fmt.Errorf("按 ID 查询数据集失败: %w", err)
	}
	var dataset entity2.Dataset
	err = dbConn.First(&dataset, id).Error
	if err != nil {
		logger.Error("查询数据集失败：数据库查询失败", "id", id, "error", err)
		return nil, err
	}
	logger.Info("查询数据集成功", "id", dataset.ID, "name", dataset.Name)
	return &dataset, nil
}

// FindFileNameByID 根据主键查询数据集文件名。
func (d *DatasetDAO) FindFileNameByID(ctx context.Context, id uint) (string, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "FindFileNameByID")
	if id == 0 {
		logger.Warn("查询数据集文件名已跳过：ID无效", "id", id)
		return "", ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("查询数据集文件名失败：绑定上下文失败", "id", id, "error", err)
		return "", fmt.Errorf("查询数据集 file_name 失败: %w", err)
	}

	var row struct {
		FileName string `gorm:"column:file_name"`
	}
	if err := dbConn.Model(&entity2.Dataset{}).Select("file_name").Where("id = ?", id).Take(&row).Error; err != nil {
		logger.Error("查询数据集文件名失败：数据库查询失败", "id", id, "error", err)
		return "", err
	}

	fileName := strings.TrimSpace(row.FileName)
	if fileName == "" {
		logger.Warn("查询数据集文件名为空", "id", id)
		return "", ErrNilEntity
	}
	logger.Info("查询数据集文件名成功", "id", id, "file_name", fileName)
	return fileName, nil
}

// FindByName 根据名称查询单条数据集记录。
func (d *DatasetDAO) FindByName(ctx context.Context, name string) (*entity2.Dataset, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "FindByName")
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		logger.Warn("按名称查询数据集已跳过：名称为空")
		return nil, ErrNilEntity
	}
	logger.Info("按名称查询数据集开始", "name", trimmed)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("按名称查询数据集失败：绑定上下文失败", "name", trimmed, "error", err)
		return nil, fmt.Errorf("按名称查询数据集失败: %w", err)
	}

	var dataset entity2.Dataset
	err = dbConn.Where("name = ?", trimmed).Take(&dataset).Error
	if err != nil {
		logger.Error("按名称查询数据集失败：数据库查询失败", "name", trimmed, "error", err)
		return nil, err
	}

	logger.Info("按名称查询数据集成功", "id", dataset.ID, "name", dataset.Name)
	return &dataset, nil
}

// UpdateMetadataByID 按主键更新数据集元信息，updates 仅包含允许更新的字段。
func (d *DatasetDAO) UpdateMetadataByID(ctx context.Context, id uint, updates map[string]interface{}) (*entity2.Dataset, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "UpdateMetadataByID")
	if id == 0 {
		logger.Warn("更新数据集元数据已跳过：ID无效", "id", id)
		return nil, ErrInvalidID
	}
	if len(updates) == 0 {
		logger.Warn("更新数据集元数据已跳过：更新字段为空", "id", id)
		return nil, ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("更新数据集元数据失败：绑定上下文失败", "id", id, "error", err)
		return nil, fmt.Errorf("更新数据集元数据失败: %w", err)
	}

	var current entity2.Dataset
	if err := dbConn.First(&current, id).Error; err != nil {
		logger.Error("更新数据集元数据失败：查询当前记录失败", "id", id, "error", err)
		return nil, err
	}

	result := dbConn.Model(&entity2.Dataset{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		logger.Error("更新数据集元数据失败：数据库更新失败", "id", id, "error", result.Error)
		if isDuplicateKeyError(result.Error) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("更新数据集元数据失败: %w", result.Error)
	}

	var updated entity2.Dataset
	if err := dbConn.First(&updated, id).Error; err != nil {
		logger.Error("更新数据集元数据失败：回查失败", "id", id, "error", err)
		return nil, err
	}

	logger.Info("更新数据集元数据成功", "id", id, "updated_fields", len(updates))
	return &updated, nil
}

// GetStorageServersByID 查询数据集的 storage_server 列并统一返回数组格式。
func (d *DatasetDAO) GetStorageServersByID(ctx context.Context, id uint) ([]string, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "GetStorageServersByID")
	if id == 0 {
		logger.Warn("查询数据集存储服务已跳过：ID无效", "id", id)
		return nil, ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("查询数据集存储服务失败：绑定上下文失败", "id", id, "error", err)
		return nil, fmt.Errorf("获取数据集存储服务失败: %w", err)
	}

	var row struct {
		StorageServer string `gorm:"column:storage_server"`
	}
	if err := dbConn.Model(&entity2.Dataset{}).Select("storage_server").Where("id = ?", id).Take(&row).Error; err != nil {
		logger.Error("查询数据集存储服务失败：数据库查询失败", "id", id, "error", err)
		return nil, err
	}

	servers := parseStorageServerValue(row.StorageServer)
	logger.Info("查询数据集存储服务成功", "id", id, "count", len(servers))
	return servers, nil
}

// UpdateStorageServersByID 按 action(set/add/remove) 更新数据集 storage_server。
func (d *DatasetDAO) UpdateStorageServersByID(ctx context.Context, id uint, action string, servers []string) ([]string, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "UpdateStorageServersByID")
	if id == 0 {
		logger.Warn("更新数据集存储服务已跳过：ID无效", "id", id)
		return nil, ErrInvalidID
	}

	current, err := d.GetStorageServersByID(ctx, id)
	if err != nil {
		logger.Error("更新数据集存储服务失败：加载当前值失败", "id", id, "error", err)
		return nil, err
	}

	next, err := applyStorageServerAction(current, action, servers)
	if err != nil {
		logger.Error("更新数据集存储服务失败：应用动作失败", "id", id, "action", action, "error", err)
		return nil, err
	}

	encoded, err := encodeStorageServerValue(next)
	if err != nil {
		logger.Error("更新数据集存储服务失败：编码失败", "id", id, "error", err)
		return nil, fmt.Errorf("更新数据集存储服务失败: %w", err)
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("更新数据集存储服务失败：绑定上下文失败", "id", id, "error", err)
		return nil, fmt.Errorf("更新数据集存储服务失败: %w", err)
	}

	if err := dbConn.Model(&entity2.Dataset{}).Where("id = ?", id).Update("storage_server", encoded).Error; err != nil {
		logger.Error("更新数据集存储服务失败：数据库更新失败", "id", id, "error", err)
		return nil, fmt.Errorf("更新数据集存储服务失败: %w", err)
	}

	logger.Info("更新数据集存储服务成功", "id", id, "action", action, "count", len(next))
	return next, nil
}

// UpdateSizeByFileName 按 file_name 更新数据集大小（MB）。
func (d *DatasetDAO) UpdateSizeByFileName(ctx context.Context, fileName string, sizeMB float64) (int64, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "UpdateSizeByFileName")

	name := strings.TrimSpace(filepath.Base(fileName))
	if name == "" || name == "." || name == string(filepath.Separator) {
		logger.Warn("更新数据集大小已跳过：文件名无效", "file_name", fileName)
		return 0, ErrNilEntity
	}
	if sizeMB < 0 {
		logger.Warn("更新数据集大小已跳过：大小MB无效", "file_name", name, "size_mb", sizeMB)
		return 0, ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("更新数据集大小失败：绑定上下文失败", "file_name", name, "error", err)
		return 0, fmt.Errorf("更新数据集大小失败: %w", err)
	}

	result := dbConn.Model(&entity2.Dataset{}).Where("file_name = ?", name).Update("size_mb", sizeMB)
	if result.Error != nil {
		logger.Error("更新数据集大小失败：数据库更新失败", "file_name", name, "size_mb", sizeMB, "error", result.Error)
		return 0, fmt.Errorf("更新数据集大小失败: %w", result.Error)
	}

	logger.Info("更新数据集大小成功", "file_name", name, "size_mb", sizeMB, "rows_affected", result.RowsAffected)
	return result.RowsAffected, nil
}

// DeleteByFileName 根据数据集文件名删除数据集记录。
func (d *DatasetDAO) DeleteByFileName(ctx context.Context, fileName string) (int64, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "DeleteByFileName")

	name := strings.TrimSpace(filepath.Base(fileName))
	if name == "" || name == "." || name == string(filepath.Separator) {
		logger.Warn("按文件名删除数据集已跳过：文件名无效", "file_name", fileName)
		return 0, ErrNilEntity
	}
	logger.Info("按文件名删除数据集开始", "file_name", name)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("按文件名删除数据集失败：绑定上下文失败", "file_name", name, "error", err)
		return 0, fmt.Errorf("按 file_name 删除数据集失败: %w", err)
	}

	result := dbConn.Where("file_name = ?", name).Delete(&entity2.Dataset{})
	if result.Error != nil {
		logger.Error("按文件名删除数据集失败：数据库删除失败", "file_name", name, "error", result.Error)
		return 0, fmt.Errorf("按 file_name 删除数据集失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		logger.Warn("按文件名删除数据集未找到记录", "file_name", name)
		return 0, gorm.ErrRecordNotFound
	}

	logger.Info("按文件名删除数据集成功", "file_name", name, "rows_affected", result.RowsAffected)
	return result.RowsAffected, nil
}

func (d *DatasetDAO) FindAll(ctx context.Context, params entity2.QueryParams) ([]entity2.Dataset, int64, error) {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "FindAll")
	var datasets []entity2.Dataset
	var total int64
	logger.Info("分页查询数据集开始",
		"page", params.Page,
		"page_size", params.PageSize,
		"name", params.Name,
		"keyword", params.Keyword,
		"storage_server", params.StorageServer,
		"task_type", params.TaskType,
	)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("分页查询数据集失败：绑定上下文失败", "error", err)
		return nil, 0, fmt.Errorf("查询数据集列表失败: %w", err)
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
		logger.Error("统计数据集总数失败", "error", err)
		return nil, 0, fmt.Errorf("统计数据集数量失败: %w", err)
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
		logger.Error("查询数据集列表失败", "error", err)
		return nil, 0, fmt.Errorf("执行数据集查询失败: %w", err)
	}

	logger.Info("分页查询数据集成功", "total", total, "returned", len(datasets))
	return datasets, total, err
}

func mapDatasetTypeToTaskType(datasetType int8) string {
	logger := daoLogger().With("dao", "DatasetDAO", "method", "mapDatasetTypeToTaskType")
	switch datasetType {
	case 1:
		logger.Debug("映射数据集类型", "dataset_type", datasetType, "task_type", "detect")
		return "detect"
	case 2:
		logger.Debug("映射数据集类型", "dataset_type", datasetType, "task_type", "segment")
		return "segment"
	case 3:
		logger.Debug("映射数据集类型", "dataset_type", datasetType, "task_type", "classify")
		return "classify"
	case 4:
		logger.Debug("映射数据集类型", "dataset_type", datasetType, "task_type", "pose")
		return "pose"
	case 5:
		logger.Debug("映射数据集类型", "dataset_type", datasetType, "task_type", "obb")
		return "obb"
	default:
		logger.Warn("未知数据集类型", "dataset_type", datasetType)
		return "unknown"
	}
}
