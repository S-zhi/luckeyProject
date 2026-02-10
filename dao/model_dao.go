package dao

import (
	"context"
	"fmt"
	entity2 "lucky_project/entity"
	"lucky_project/infrastructure/db"
	"strings"

	"gorm.io/gorm"
)

type ModelDAO struct {
	DB *gorm.DB
}

func NewModelDAO() *ModelDAO {
	return &ModelDAO{
		DB: db.DB,
	}
}

func (d *ModelDAO) Save(ctx context.Context, model *entity2.Model) error {
	if model == nil {
		return ErrNilEntity
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return fmt.Errorf("save model failed: %w", err)
	}
	return dbConn.Create(model).Error
}

func (d *ModelDAO) DeleteByID(ctx context.Context, id uint) error {
	if id == 0 {
		return ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return fmt.Errorf("delete model by id failed: %w", err)
	}

	result := dbConn.Delete(&entity2.Model{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete model by id failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (d *ModelDAO) FindByID(ctx context.Context, id uint) (*entity2.Model, error) {
	if id == 0 {
		return nil, ErrInvalidID
	}

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
		return nil, fmt.Errorf("find model by id failed: %w", err)
	}

	var model entity2.Model
	err = dbConn.First(&model, id).Error
	return &model, err
}

func (d *ModelDAO) FindAll(ctx context.Context, params entity2.QueryParams) ([]entity2.Model, int64, error) {
	var models []entity2.Model
	var total int64

	dbConn, err := withContext(d.DB, ctx)
	if err != nil {
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
		return nil, 0, fmt.Errorf("count models failed: %w", err)
	}

	// 5. 执行分页查询
	offset, limit := pagination(params)
	err = dbConn.Order(orderStr).Offset(offset).Limit(limit).Find(&models).Error
	if err != nil {
		return nil, 0, fmt.Errorf("query models failed: %w", err)
	}

	return models, total, err
}
