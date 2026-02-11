package dao

import (
	"context"
	"fmt"
	"lucky_project/config"
	entity2 "lucky_project/entity"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ModelDAO struct {
	DB *gorm.DB
}

// NewModelDAO 创建 ModelDAO，并注入全局数据库连接。
func NewModelDAO() *ModelDAO {
	logger := daoLogger().With("dao", "ModelDAO", "method", "NewModelDAO")
	logger.Info("init model dao")
	return &ModelDAO{
		DB: config.DB,
	}
}

// Save 按 (name, version) 保存模型；若唯一键冲突则执行更新（upsert）。
func (d *ModelDAO) Save(ctx context.Context, model *entity2.Model) error {
	logger := daoLogger().With("dao", "ModelDAO", "method", "Save")
	if model == nil {
		logger.Warn("save model skipped: model is nil")
		return ErrNilEntity
	}
	logger.Info("save model begin", "name", model.Name, "version", model.Version)

	weightName := strings.TrimSpace(filepath.Base(model.WeightName))
	if weightName == "" || weightName == "." || weightName == string(filepath.Separator) {
		legacyFileName := strings.TrimSpace(model.LegacyFileName)
		if legacyFileName != "" {
			weightName = strings.TrimSpace(filepath.Base(legacyFileName))
		}
	}
	if weightName == "" || weightName == "." || weightName == string(filepath.Separator) {
		legacyPath := strings.TrimSpace(strings.ReplaceAll(model.LegacyModelPath, "\\", "/"))
		if legacyPath != "" {
			derived := strings.TrimSpace(filepath.Base(legacyPath))
			if derived != "" && derived != "." && derived != string(filepath.Separator) {
				weightName = derived
			}
		}
	}
	if weightName == "" || weightName == "." || weightName == string(filepath.Separator) {
		logger.Warn("save model skipped: weight_name is empty")
		return ErrNilEntity
	}
	model.WeightName = weightName

	normalizedStorageServer, err := encodeStorageServerValue(parseStorageServerValue(model.StorageServer))
	if err != nil {
		logger.Error("save model failed: normalize storage server", "name", model.Name, "error", err)
		return fmt.Errorf("save model failed: %w", err)
	}
	model.StorageServer = normalizedStorageServer

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("save model failed: with context", "name", model.Name, "error", err)
		return fmt.Errorf("save model failed: %w", err)
	}

	if err := dbConn.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}, {Name: "version"}},
		DoUpdates: clause.AssignmentColumns(updatableModelColumns()),
	}).Create(model).Error; err != nil {
		logger.Error("save model failed: create/upsert", "name", model.Name, "error", err)
		return fmt.Errorf("save model failed: %w", err)
	}

	if strings.TrimSpace(model.Name) != "" {
		if err := dbConn.Where("name = ? AND version = ?", model.Name, model.Version).First(model).Error; err != nil {
			logger.Error("save model failed: reload by unique key", "name", model.Name, "version", model.Version, "error", err)
			return fmt.Errorf("save model failed: %w", err)
		}
	}

	logger.Info("save model success", "id", model.ID, "name", model.Name, "version", model.Version)
	return nil
}

// GetStorageServersByID 查询模型的 storage_server 列并统一返回数组格式。
func (d *ModelDAO) GetStorageServersByID(ctx context.Context, id uint) ([]string, error) {
	logger := daoLogger().With("dao", "ModelDAO", "method", "GetStorageServersByID")
	if id == 0 {
		logger.Warn("get model storage server skipped: invalid id", "id", id)
		return nil, ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("get model storage server failed: with context", "id", id, "error", err)
		return nil, fmt.Errorf("get model storage server failed: %w", err)
	}

	var row struct {
		StorageServer string `gorm:"column:storage_server"`
	}
	if err := dbConn.Model(&entity2.Model{}).Select("storage_server").Where("id = ?", id).Take(&row).Error; err != nil {
		logger.Error("get model storage server failed: db query", "id", id, "error", err)
		return nil, err
	}

	servers := parseStorageServerValue(row.StorageServer)
	logger.Info("get model storage server success", "id", id, "count", len(servers))
	return servers, nil
}

// UpdateStorageServersByID 按 action(set/add/remove) 更新模型 storage_server。
func (d *ModelDAO) UpdateStorageServersByID(ctx context.Context, id uint, action string, servers []string) ([]string, error) {
	logger := daoLogger().With("dao", "ModelDAO", "method", "UpdateStorageServersByID")
	if id == 0 {
		logger.Warn("update model storage server skipped: invalid id", "id", id)
		return nil, ErrInvalidID
	}

	current, err := d.GetStorageServersByID(ctx, id)
	if err != nil {
		logger.Error("update model storage server failed: load current", "id", id, "error", err)
		return nil, err
	}

	next, err := applyStorageServerAction(current, action, servers)
	if err != nil {
		logger.Error("update model storage server failed: apply action", "id", id, "action", action, "error", err)
		return nil, err
	}

	encoded, err := encodeStorageServerValue(next)
	if err != nil {
		logger.Error("update model storage server failed: encode", "id", id, "error", err)
		return nil, fmt.Errorf("update model storage server failed: %w", err)
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("update model storage server failed: with context", "id", id, "error", err)
		return nil, fmt.Errorf("update model storage server failed: %w", err)
	}

	if err := dbConn.Model(&entity2.Model{}).Where("id = ?", id).Update("storage_server", encoded).Error; err != nil {
		logger.Error("update model storage server failed: db update", "id", id, "error", err)
		return nil, fmt.Errorf("update model storage server failed: %w", err)
	}

	logger.Info("update model storage server success", "id", id, "action", action, "count", len(next))
	return next, nil
}

// updatableModelColumns 返回 upsert 时允许更新的字段列表。
func updatableModelColumns() []string {
	columns := []string{
		"storage_server",
		"base_model_id",
		"algorithm_id",
		"task_type",
		"description",
		"framework",
		"weight_size_mb",
		"paper",
		"params_url",
		"weight_name",
	}
	daoLogger().With("dao", "ModelDAO", "method", "updatableModelColumns").Debug("updatable columns prepared", "count", len(columns))
	return columns
}

// FindWeightNameByID 根据主键查询模型权重文件名。
func (d *ModelDAO) FindWeightNameByID(ctx context.Context, id uint) (string, error) {
	logger := daoLogger().With("dao", "ModelDAO", "method", "FindWeightNameByID")
	if id == 0 {
		logger.Warn("find model weight_name skipped: invalid id", "id", id)
		return "", ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("find model weight_name failed: with context", "id", id, "error", err)
		return "", fmt.Errorf("find model weight_name failed: %w", err)
	}

	var row struct {
		WeightName string `gorm:"column:weight_name"`
	}
	if err := dbConn.Model(&entity2.Model{}).Select("weight_name").Where("id = ?", id).Take(&row).Error; err != nil {
		logger.Error("find model weight_name failed: db query", "id", id, "error", err)
		return "", err
	}

	weightName := strings.TrimSpace(row.WeightName)
	if weightName == "" {
		logger.Warn("find model weight_name empty", "id", id)
		return "", ErrNilEntity
	}
	logger.Info("find model weight_name success", "id", id, "weight_name", weightName)
	return weightName, nil
}

// FindFileNameByID 兼容旧调用，内部映射到 weight_name。
func (d *ModelDAO) FindFileNameByID(ctx context.Context, id uint) (string, error) {
	return d.FindWeightNameByID(ctx, id)
}

// UpdateWeightSizeByWeightName 按权重文件名更新模型文件大小（MB）。
func (d *ModelDAO) UpdateWeightSizeByWeightName(ctx context.Context, weightName string, weightSizeMB float64) (int64, error) {
	logger := daoLogger().With("dao", "ModelDAO", "method", "UpdateWeightSizeByWeightName")

	name := strings.TrimSpace(filepath.Base(weightName))
	if name == "" || name == "." || name == string(filepath.Separator) {
		logger.Warn("update model weight size skipped: invalid weight_name", "weight_name", weightName)
		return 0, ErrNilEntity
	}
	if weightSizeMB < 0 {
		logger.Warn("update model weight size skipped: invalid weight_size_mb", "weight_name", name, "weight_size_mb", weightSizeMB)
		return 0, ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("update model weight size failed: with context", "weight_name", name, "error", err)
		return 0, fmt.Errorf("update model weight size failed: %w", err)
	}

	result := dbConn.Model(&entity2.Model{}).Where("weight_name = ?", name).Update("weight_size_mb", weightSizeMB)
	if result.Error != nil {
		logger.Error("update model weight size failed: db update", "weight_name", name, "weight_size_mb", weightSizeMB, "error", result.Error)
		return 0, fmt.Errorf("update model weight size failed: %w", result.Error)
	}

	logger.Info("update model weight size success", "weight_name", name, "weight_size_mb", weightSizeMB, "rows_affected", result.RowsAffected)
	return result.RowsAffected, nil
}

// DeleteByID 根据主键删除模型记录。
func (d *ModelDAO) DeleteByID(ctx context.Context, id uint) error {
	logger := daoLogger().With("dao", "ModelDAO", "method", "DeleteByID")
	if id == 0 {
		logger.Warn("delete model skipped: invalid id", "id", id)
		return ErrInvalidID
	}
	logger.Info("delete model begin", "id", id)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("delete model failed: with context", "id", id, "error", err)
		return fmt.Errorf("delete model by id failed: %w", err)
	}

	result := dbConn.Delete(&entity2.Model{}, id)
	if result.Error != nil {
		logger.Error("delete model failed: db delete", "id", id, "error", result.Error)
		return fmt.Errorf("delete model by id failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		logger.Warn("delete model not found", "id", id)
		return gorm.ErrRecordNotFound
	}

	logger.Info("delete model success", "id", id)
	return nil
}

// DeleteByWeightName 根据权重文件名删除模型记录。
func (d *ModelDAO) DeleteByWeightName(ctx context.Context, weightName string) (int64, error) {
	logger := daoLogger().With("dao", "ModelDAO", "method", "DeleteByWeightName")

	name := strings.TrimSpace(filepath.Base(weightName))
	if name == "" || name == "." || name == string(filepath.Separator) {
		logger.Warn("delete model by weight_name skipped: invalid weight_name", "weight_name", weightName)
		return 0, ErrNilEntity
	}
	logger.Info("delete model by weight_name begin", "weight_name", name)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("delete model by weight_name failed: with context", "weight_name", name, "error", err)
		return 0, fmt.Errorf("delete model by weight_name failed: %w", err)
	}

	result := dbConn.Where("weight_name = ?", name).Delete(&entity2.Model{})
	if result.Error != nil {
		logger.Error("delete model by weight_name failed: db delete", "weight_name", name, "error", result.Error)
		return 0, fmt.Errorf("delete model by weight_name failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		logger.Warn("delete model by weight_name not found", "weight_name", name)
		return 0, gorm.ErrRecordNotFound
	}

	logger.Info("delete model by weight_name success", "weight_name", name, "rows_affected", result.RowsAffected)
	return result.RowsAffected, nil
}

// FindByID 根据主键查询单条模型记录。
func (d *ModelDAO) FindByID(ctx context.Context, id uint) (*entity2.Model, error) {
	logger := daoLogger().With("dao", "ModelDAO", "method", "FindByID")
	if id == 0 {
		logger.Warn("find model by id skipped: invalid id", "id", id)
		return nil, ErrInvalidID
	}
	logger.Info("find model by id begin", "id", id)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("find model by id failed: with context", "id", id, "error", err)
		return nil, fmt.Errorf("find model by id failed: %w", err)
	}

	var model entity2.Model
	err = dbConn.First(&model, id).Error
	if err != nil {
		logger.Error("find model by id failed: db query", "id", id, "error", err)
		return &model, err
	}
	logger.Info("find model by id success", "id", model.ID, "name", model.Name)
	return &model, err
}

// FindByName 根据名称查询单条模型记录。
func (d *ModelDAO) FindByName(ctx context.Context, name string) (*entity2.Model, error) {
	logger := daoLogger().With("dao", "ModelDAO", "method", "FindByName")
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		logger.Warn("find model by name skipped: empty name")
		return nil, ErrNilEntity
	}
	logger.Info("find model by name begin", "name", trimmed)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("find model by name failed: with context", "name", trimmed, "error", err)
		return nil, fmt.Errorf("find model by name failed: %w", err)
	}

	var model entity2.Model
	err = dbConn.Where("name = ?", trimmed).Order("version DESC, id DESC").Take(&model).Error
	if err != nil {
		logger.Error("find model by name failed: db query", "name", trimmed, "error", err)
		return nil, err
	}

	logger.Info("find model by name success", "id", model.ID, "name", model.Name)
	return &model, nil
}

// FindAll 按查询参数分页获取模型列表与总数。
func (d *ModelDAO) FindAll(ctx context.Context, params entity2.QueryParams) ([]entity2.Model, int64, error) {
	logger := daoLogger().With("dao", "ModelDAO", "method", "FindAll")
	var models []entity2.Model
	var total int64
	logger.Info("find models begin",
		"page", params.Page,
		"page_size", params.PageSize,
		"name", params.Name,
		"keyword", params.Keyword,
		"storage_server", params.StorageServer,
		"task_type", params.TaskType,
		"algorithm_id", params.AlgorithmID,
		"framework", params.Framework,
	)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("find models failed: with context", "error", err)
		return nil, 0, fmt.Errorf("find models failed: %w", err)
	}

	dbConn = dbConn.Model(&entity2.Model{})

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
	}
	algorithmID := strings.TrimSpace(params.AlgorithmID)
	if algorithmID == "" {
		algorithmID = strings.TrimSpace(params.ImplType)
	}
	if algorithmID == "" {
		algorithmID = strings.TrimSpace(params.Algorithm)
	}
	if algorithmID != "" {
		dbConn = dbConn.Where("algorithm_id = ?", algorithmID)
	}
	if framework := strings.TrimSpace(params.Framework); framework != "" {
		dbConn = dbConn.Where("framework = ?", framework)
	}
	if version := strings.TrimSpace(params.Version); version != "" {
		dbConn = dbConn.Where("version = ?", version)
	}
	if params.BaseModelID != nil {
		dbConn = dbConn.Where("base_model_id = ?", *params.BaseModelID)
	}

	// 兼容旧参数（旧表字段已删除），仅记录日志并忽略。
	if params.DatasetID != nil || params.TrainTaskID != nil {
		logger.Warn("legacy model filters ignored for new schema",
			"dataset_id_set", params.DatasetID != nil,
			"train_task_id_set", params.TrainTaskID != nil,
		)
	}

	// 3. 排序规则 (根据 weight_size_mb)
	orderStr := "id DESC" // 默认按 ID 降序
	sortValue := strings.ToLower(strings.TrimSpace(params.SizeSort))
	if sortValue == "" {
		sortValue = strings.ToLower(strings.TrimSpace(params.WeightSort))
	}
	switch sortValue {
	case "asc":
		orderStr = "weight_size_mb ASC"
	case "desc":
		orderStr = "weight_size_mb DESC"
	}

	// 4. 获取总数
	err = dbConn.Count(&total).Error
	if err != nil {
		logger.Error("count models failed", "error", err)
		return nil, 0, fmt.Errorf("count models failed: %w", err)
	}

	// 5. 执行分页查询
	offset, limit := pagination(params)
	err = dbConn.Order(orderStr).Offset(offset).Limit(limit).Find(&models).Error
	if err != nil {
		logger.Error("query models failed", "error", err)
		return nil, 0, fmt.Errorf("query models failed: %w", err)
	}

	logger.Info("find models success", "total", total, "returned", len(models))
	return models, total, err
}

// UpdateMetadataByID 按主键更新模型元信息，updates 仅包含允许更新的字段。
func (d *ModelDAO) UpdateMetadataByID(ctx context.Context, id uint, updates map[string]interface{}) (*entity2.Model, error) {
	logger := daoLogger().With("dao", "ModelDAO", "method", "UpdateMetadataByID")
	if id == 0 {
		logger.Warn("update model metadata skipped: invalid id", "id", id)
		return nil, ErrInvalidID
	}
	if len(updates) == 0 {
		logger.Warn("update model metadata skipped: empty updates", "id", id)
		return nil, ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("update model metadata failed: with context", "id", id, "error", err)
		return nil, fmt.Errorf("update model metadata failed: %w", err)
	}

	var current entity2.Model
	if err := dbConn.First(&current, id).Error; err != nil {
		logger.Error("update model metadata failed: query current", "id", id, "error", err)
		return nil, err
	}

	result := dbConn.Model(&entity2.Model{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		logger.Error("update model metadata failed: db update", "id", id, "error", result.Error)
		if isDuplicateKeyError(result.Error) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("update model metadata failed: %w", result.Error)
	}

	var updated entity2.Model
	if err := dbConn.First(&updated, id).Error; err != nil {
		logger.Error("update model metadata failed: reload", "id", id, "error", err)
		return nil, err
	}

	logger.Info("update model metadata success", "id", id, "updated_fields", len(updates))
	return &updated, nil
}
