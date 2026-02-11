package dao

import (
	"context"
	"fmt"
	"lucky_project/config"
	entity2 "lucky_project/entity"
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

// Save 按名称保存模型；若名称已存在则执行更新（upsert）。
func (d *ModelDAO) Save(ctx context.Context, model *entity2.Model) error {
	logger := daoLogger().With("dao", "ModelDAO", "method", "Save")
	if model == nil {
		logger.Warn("save model skipped: model is nil")
		return ErrNilEntity
	}
	logger.Info("save model begin", "name", model.Name)

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		logger.Error("save model failed: with context", "name", model.Name, "error", err)
		return fmt.Errorf("save model failed: %w", err)
	}

	if err := dbConn.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns(updatableModelColumns()),
	}).Create(model).Error; err != nil {
		logger.Error("save model failed: create/upsert", "name", model.Name, "error", err)
		return fmt.Errorf("save model failed: %w", err)
	}

	if strings.TrimSpace(model.Name) != "" {
		if err := dbConn.Where("name = ?", model.Name).First(model).Error; err != nil {
			logger.Error("save model failed: reload by name", "name", model.Name, "error", err)
			return fmt.Errorf("save model failed: %w", err)
		}
	}

	logger.Info("save model success", "id", model.ID, "name", model.Name)
	return nil
}

// updatableModelColumns 返回 upsert 时允许更新的字段列表。
func updatableModelColumns() []string {
	columns := []string{
		"storage_server",
		"paper",
		"params_url",
		"model_path",
		"base_model_id",
		"impl_type",
		"dataset_id",
		"size_mb",
		"version",
		"train_task_id",
		"description",
		"task_type",
	}
	daoLogger().With("dao", "ModelDAO", "method", "updatableModelColumns").Debug("updatable columns prepared", "count", len(columns))
	return columns
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
		dbConn = dbConn.Where("storage_server = ?", storageServer)
	}
	if taskType := strings.TrimSpace(params.TaskType); taskType != "" {
		dbConn = dbConn.Where("task_type = ?", taskType)
	}
	if implType := strings.TrimSpace(params.ImplType); implType != "" {
		dbConn = dbConn.Where("impl_type = ?", implType)
	} else if algorithm := strings.TrimSpace(params.Algorithm); algorithm != "" {
		// 兼容旧参数 algorithm
		dbConn = dbConn.Where("impl_type = ?", algorithm)
	}
	if version := strings.TrimSpace(params.Version); version != "" {
		dbConn = dbConn.Where("version = ?", version)
	}
	if params.DatasetID != nil {
		dbConn = dbConn.Where("dataset_id = ?", *params.DatasetID)
	}
	if params.TrainTaskID != nil {
		dbConn = dbConn.Where("train_task_id = ?", *params.TrainTaskID)
	}
	if params.BaseModelID != nil {
		dbConn = dbConn.Where("base_model_id = ?", *params.BaseModelID)
	}

	// 3. 排序规则 (根据 size_mb)
	orderStr := "id DESC" // 默认按 ID 降序
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
